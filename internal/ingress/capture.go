package ingress

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/ingress/codec"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Outcome constants for event processing
const (
	OutcomeProcessed        = "processed"
	OutcomeNoTarget         = "no-target"
	OutcomeTombstonedSubject = "tombstoned-subject"
	OutcomeInvalidSchema    = "invalid-schema"
	OutcomeQuarantined      = "quarantined"
)

// KafkaMessage represents an incoming message from Kafka or simulated queue.
type KafkaMessage struct {
	Topic     string            `json:"topic"`
	Partition int32             `json:"partition"`
	Offset    int64             `json:"offset"`
	Key       string            `json:"key"`
	Value     []byte            `json:"value"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// InjectKafkaHeaders injects W3C traceparent headers into Kafka record headers map using otel.GetTextMapPropagator().
func InjectKafkaHeaders(ctx context.Context, headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	ctx = middleware.EnsureOTelSpanContext(ctx)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))
	if headers["traceparent"] == "" {
		tID := middleware.GetTraceID(ctx)
		sID := middleware.GetSpanID(ctx)
		if tID != "" {
			if sID == "" {
				sID = fmt.Sprintf("%016x", time.Now().UnixNano()&0x7FFFFFFFFFFFFFFF)
			}
			headers["traceparent"] = fmt.Sprintf("00-%s-%s-01", tID, sID)
		}
	}
	return headers
}

// ExtractKafkaHeaders extracts W3C traceparent context from Kafka record headers map into context.Context.
func ExtractKafkaHeaders(ctx context.Context, headers map[string]string) context.Context {
	if headers == nil {
		return ctx
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
	if tp := headers["traceparent"]; tp != "" {
		parts := strings.Split(tp, "-")
		if len(parts) >= 2 && len(parts[1]) == 32 {
			ctx = middleware.WithTraceID(ctx, parts[1])
		}
	}
	return middleware.EnsureOTelSpanContext(ctx)
}

// OffsetCommitter interface defines Kafka offset commit behavior.
type OffsetCommitter interface {
	CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error
}

// CaptureConfig defines configuration options for the capture consumer.
type CaptureConfig struct {
	BatchSize    int
	BatchTimeout time.Duration
	MaxInFlight  int
}

// DefaultCaptureConfig returns standard config defaults.
func DefaultCaptureConfig() CaptureConfig {
	return CaptureConfig{
		BatchSize:    50,
		BatchTimeout: 100 * time.Millisecond,
		MaxInFlight:  100,
	}
}

// CaptureResult details the outcome of processing a single message.
type CaptureResult struct {
	Inbox           *postgres.KafkaInbox
	TargetManifest  *postgres.TargetManifest
	Outcome         string
	IsDuplicate     bool
	OffsetCommitted bool
}

// Option configures CaptureConsumer.
type Option func(*CaptureConsumer)

// WithCodec sets custom codec for payload validation.
func WithCodec(c *codec.Codec) Option {
	return func(cc *CaptureConsumer) {
		if c != nil {
			cc.codec = c
		}
	}
}

// WithConfig sets custom capture configuration.
func WithConfig(cfg CaptureConfig) Option {
	return func(cc *CaptureConsumer) {
		cc.config = cfg
		if cfg.MaxInFlight > 0 {
			cc.inFlightSem = make(chan struct{}, cfg.MaxInFlight)
		}
	}
}

// CaptureConsumer manages durable Kafka message capture and target manifest freezing.
type CaptureConsumer struct {
	repo        postgres.Repository
	codec       *codec.Codec
	committer   OffsetCommitter
	config      CaptureConfig
	watermarks  map[string]map[int32]int64
	watermarkMu sync.RWMutex
	inFlightSem chan struct{}
}

// NewCaptureConsumer initializes a new CaptureConsumer.
func NewCaptureConsumer(repo postgres.Repository, committer OffsetCommitter, opts ...Option) *CaptureConsumer {
	cc := &CaptureConsumer{
		repo:       repo,
		codec:      codec.NewCodec(),
		committer:  committer,
		config:     DefaultCaptureConfig(),
		watermarks: make(map[string]map[int32]int64),
	}

	for _, opt := range opts {
		opt(cc)
	}

	if cc.config.MaxInFlight > 0 && cc.inFlightSem == nil {
		cc.inFlightSem = make(chan struct{}, cc.config.MaxInFlight)
	}

	return cc
}

// ProcessMessage ingests a single Kafka message into database inbox and freezes target manifest inside a single transaction.
func (c *CaptureConsumer) ProcessMessage(ctx context.Context, msg KafkaMessage) (*CaptureResult, error) {
	ctx = ExtractKafkaHeaders(ctx, msg.Headers)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Backpressure control
	if c.inFlightSem != nil {
		select {
		case c.inFlightSem <- struct{}{}:
			defer func() { <-c.inFlightSem }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 1. Unmarshal basic header info from raw bytes
	var raw map[string]interface{}
	_ = json.Unmarshal(msg.Value, &raw)

	tenantID, _ := raw["tenant_id"].(string)
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	eventID, _ := raw["event_id"].(string)
	if eventID == "" {
		eventID = msg.Key
	}
	if eventID == "" {
		eventID = fmt.Sprintf("evt-%s-%d-%d", msg.Topic, msg.Partition, msg.Offset)
	}

	messageID := fmt.Sprintf("msg-%s-%s", tenantID, eventID)

	evtType, _ := raw["event_type"].(string)
	logging.Info().
		Str("topic", msg.Topic).
		Int32("partition", msg.Partition).
		Int64("offset", msg.Offset).
		Str("event_type", evtType).
		Str("event_id", eventID).
		Str("tenant_id", tenantID).
		Msg("captured kafka event")

	var result CaptureResult

	// 2. Perform DB operations inside a single database transaction (WithTx)
	err := c.repo.WithTx(ctx, func(txRepo postgres.Repository) error {
		// A. Check for duplicate event (idempotency check)
		existingInbox, err := txRepo.GetKafkaInboxMessage(ctx, tenantID, messageID)
		if err == nil && existingInbox != nil {
			manifestID := fmt.Sprintf("man-%s", existingInbox.EventID)
			existingManifest, _ := txRepo.GetTargetManifest(ctx, tenantID, manifestID)

			result.Inbox = existingInbox
			result.TargetManifest = existingManifest
			result.Outcome = existingInbox.Status
			result.IsDuplicate = true
			return nil
		}

		// B. Decode & validate envelope
		env, decodeErr := c.codec.Decode(msg.Value)
		if decodeErr != nil {
			status := OutcomeInvalidSchema
			if isQuarantinedPayload(msg.Value) {
				status = OutcomeQuarantined
			}

			inbox := &postgres.KafkaInbox{
				TenantID:  tenantID,
				MessageID: messageID,
				EventID:   eventID,
				Topic:     msg.Topic,
				Partition: msg.Partition,
				OffsetVal: msg.Offset,
				Payload:   msg.Value,
				Status:    status,
				CreatedAt: time.Now().UTC(),
			}

			savedInbox, saveErr := txRepo.SaveKafkaInboxMessage(ctx, inbox)
			if saveErr != nil {
				if errors.Is(saveErr, postgres.ErrAlreadyExists) || errors.Is(saveErr, postgres.ErrConflict) {
					if existing, getErr := txRepo.GetKafkaInboxMessage(ctx, tenantID, messageID); getErr == nil {
						result.Inbox = existing
						result.Outcome = existing.Status
						result.IsDuplicate = true
						return nil
					}
				}
				return saveErr
			}

			result.Inbox = savedInbox
			result.Outcome = status
			return nil
		}

		// Check if valid envelope contains poison pill or quarantine flag
		if isQuarantinedEnvelope(env) {
			inbox := &postgres.KafkaInbox{
				TenantID:  env.TenantID,
				MessageID: messageID,
				EventID:   env.EventID,
				Topic:     msg.Topic,
				Partition: msg.Partition,
				OffsetVal: msg.Offset,
				Payload:   msg.Value,
				Status:    OutcomeQuarantined,
				CreatedAt: time.Now().UTC(),
			}

			savedInbox, saveErr := txRepo.SaveKafkaInboxMessage(ctx, inbox)
			if saveErr != nil {
				return saveErr
			}

			result.Inbox = savedInbox
			result.Outcome = OutcomeQuarantined
			return nil
		}

		// C. Check if subject is tombstoned
		tombstone, _ := txRepo.GetTombstoneByEntity(ctx, env.TenantID, "subject", env.SubjectRef)
		if tombstone == nil {
			tombstone, _ = txRepo.GetTombstoneByEntity(ctx, env.TenantID, "user", env.SubjectRef)
		}
		if tombstone != nil {
			inbox := &postgres.KafkaInbox{
				TenantID:  env.TenantID,
				MessageID: messageID,
				EventID:   env.EventID,
				Topic:     msg.Topic,
				Partition: msg.Partition,
				OffsetVal: msg.Offset,
				Payload:   msg.Value,
				Status:    OutcomeTombstonedSubject,
				CreatedAt: time.Now().UTC(),
			}

			savedInbox, saveErr := txRepo.SaveKafkaInboxMessage(ctx, inbox)
			if saveErr != nil {
				return saveErr
			}

			result.Inbox = savedInbox
			result.Outcome = OutcomeTombstonedSubject
			return nil
		}

		// D. Resolve active routing metadata (journey version)
		targets, journeyVersionID, err := resolveRoutingMetadata(ctx, txRepo, env)
		if err != nil || len(targets) == 0 {
			inbox := &postgres.KafkaInbox{
				TenantID:  env.TenantID,
				MessageID: messageID,
				EventID:   env.EventID,
				Topic:     msg.Topic,
				Partition: msg.Partition,
				OffsetVal: msg.Offset,
				Payload:   msg.Value,
				Status:    OutcomeNoTarget,
				CreatedAt: time.Now().UTC(),
			}

			savedInbox, saveErr := txRepo.SaveKafkaInboxMessage(ctx, inbox)
			if saveErr != nil {
				return saveErr
			}

			result.Inbox = savedInbox
			result.Outcome = OutcomeNoTarget
			return nil
		}

		// E. Active targets resolved! Freeze target manifest
		manifestID := fmt.Sprintf("man-%s", env.EventID)
		querySpecMap := map[string]interface{}{
			"event_id":           env.EventID,
			"event_type":         env.EventType,
			"journey_version_id": journeyVersionID,
			"subject_ref":        env.SubjectRef,
			"targets":            targets,
		}
		querySpecData, _ := json.Marshal(querySpecMap)
		contentHash := sha256Hex(querySpecData)

		tm := &postgres.TargetManifest{
			TenantID:    env.TenantID,
			ManifestID:  manifestID,
			Name:        fmt.Sprintf("manifest-%s", env.EventID),
			QuerySpec:   querySpecData,
			TotalCount:  int64(len(targets)),
			ContentHash: contentHash,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		createdManifest, createErr := txRepo.CreateTargetManifest(ctx, tm)
		if createErr != nil {
			return createErr
		}

		// Record dispatch ledger entries
		for _, targetSubj := range targets {
			dl := &postgres.DispatchLedger{
				TenantID:         env.TenantID,
				DispatchID:       fmt.Sprintf("disp-%s-%s", manifestID, targetSubj),
				ManifestID:       manifestID,
				JourneyVersionID: journeyVersionID,
				SubjectID:        targetSubj,
				Status:           "dispatched",
				DispatchedAt:     time.Now().UTC(),
				CreatedAt:        time.Now().UTC(),
			}
			_, _ = txRepo.CreateDispatchLedger(ctx, dl)
		}

		// Save KafkaInbox with status OutcomeProcessed
		inbox := &postgres.KafkaInbox{
			TenantID:  env.TenantID,
			MessageID: messageID,
			EventID:   env.EventID,
			Topic:     msg.Topic,
			Partition: msg.Partition,
			OffsetVal: msg.Offset,
			Payload:   msg.Value,
			Status:    OutcomeProcessed,
			CreatedAt: time.Now().UTC(),
		}

		savedInbox, saveErr := txRepo.SaveKafkaInboxMessage(ctx, inbox)
		if saveErr != nil {
			return saveErr
		}

		result.Inbox = savedInbox
		result.TargetManifest = createdManifest
		result.Outcome = OutcomeProcessed
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 3. Commit Kafka offsets ONLY after database transaction commits
	if c.committer != nil {
		if commitErr := c.committer.CommitOffset(ctx, msg.Topic, msg.Partition, msg.Offset); commitErr != nil {
			return &result, fmt.Errorf("transaction committed but offset commit failed: %w", commitErr)
		}
		result.OffsetCommitted = true
	}

	// 4. Record offset watermark
	c.recordWatermark(msg.Topic, msg.Partition, msg.Offset)

	return &result, nil
}

// ProcessBatch processes a batch of Kafka messages respecting context cancellation and backpressure limits.
func (c *CaptureConsumer) ProcessBatch(ctx context.Context, batch []KafkaMessage) ([]*CaptureResult, error) {
	results := make([]*CaptureResult, 0, len(batch))
	for _, msg := range batch {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		res, err := c.ProcessMessage(ctx, msg)
		if err != nil {
			return results, fmt.Errorf("failed processing msg offset %d: %w", msg.Offset, err)
		}
		results = append(results, res)
	}
	return results, nil
}

// Watermark returns the highest committed offset for a given topic and partition.
func (c *CaptureConsumer) Watermark(topic string, partition int32) int64 {
	c.watermarkMu.RLock()
	defer c.watermarkMu.RUnlock()

	if partitions, ok := c.watermarks[topic]; ok {
		if offset, found := partitions[partition]; found {
			return offset
		}
	}
	return -1
}

// GetWatermarks returns a snapshot of recorded offset watermarks.
func (c *CaptureConsumer) GetWatermarks() map[string]map[int32]int64 {
	c.watermarkMu.RLock()
	defer c.watermarkMu.RUnlock()

	snapshot := make(map[string]map[int32]int64)
	for t, pMap := range c.watermarks {
		snapshot[t] = make(map[int32]int64)
		for p, off := range pMap {
			snapshot[t][p] = off
		}
	}
	return snapshot
}

func (c *CaptureConsumer) recordWatermark(topic string, partition int32, offset int64) {
	c.watermarkMu.Lock()
	defer c.watermarkMu.Unlock()

	if c.watermarks[topic] == nil {
		c.watermarks[topic] = make(map[int32]int64)
	}
	if offset > c.watermarks[topic][partition] {
		c.watermarks[topic][partition] = offset
	}
}

func isQuarantinedPayload(data []byte) bool {
	s := strings.ToLower(string(data))
	return strings.Contains(s, "poison_pill") ||
		strings.Contains(s, "poison.pill") ||
		strings.Contains(s, `"quarantined":true`) ||
		strings.Contains(s, `"quarantined": true`)
}

func isQuarantinedEnvelope(env *codec.EventEnvelope) bool {
	if env == nil {
		return false
	}
	if strings.Contains(strings.ToLower(env.EventType), "poison") {
		return true
	}
	if env.Payload != nil {
		if q, ok := env.Payload["quarantined"].(bool); ok && q {
			return true
		}
		if p, ok := env.Payload["poison_pill"].(bool); ok && p {
			return true
		}
	}
	return false
}

func resolveRoutingMetadata(ctx context.Context, repo postgres.Repository, env *codec.EventEnvelope) ([]string, string, error) {
	// Check if explicit journey version specified in payload
	if versionID, ok := env.Payload["journey_version_id"].(string); ok && versionID != "" {
		jv, err := repo.GetJourneyVersion(ctx, env.TenantID, versionID)
		if err == nil && jv != nil {
			return []string{env.SubjectRef}, jv.VersionID, nil
		}
	}

	// Check if draft_id specified
	draftID, _ := env.Payload["draft_id"].(string)
	if draftID == "" {
		draftID = "draft-default"
	}

	versions, err := repo.ListJourneyVersionsByDraft(ctx, env.TenantID, draftID)
	if err == nil && len(versions) > 0 {
		return []string{env.SubjectRef}, versions[0].VersionID, nil
	}

	return nil, "", nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
