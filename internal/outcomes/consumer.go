package outcomes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/ingress"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Consumer aliases OutcomeConsumer for backward compatibility and concise access.
type Consumer = OutcomeConsumer

// OutcomeConsumer manages Kafka outcome event ingestion with OpenTelemetry trace context extraction.
type OutcomeConsumer struct {
	handler *Handler
}

// NewConsumer initializes a new OutcomeConsumer for outcome-ingress.
func NewConsumer(handler *Handler) *OutcomeConsumer {
	if handler == nil {
		handler = New()
	}
	return &OutcomeConsumer{handler: handler}
}

// ExtractKafkaTraceHeaders extracts W3C traceparent headers from Kafka record headers using otel.GetTextMapPropagator().
func ExtractKafkaTraceHeaders(ctx context.Context, headers map[string]string) context.Context {
	if headers == nil {
		return middleware.EnsureOTelSpanContext(ctx)
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

// ProcessMessage ingests a single Kafka record message, extracting W3C traceparent headers.
func (c *OutcomeConsumer) ProcessMessage(ctx context.Context, msg ingress.KafkaMessage) (*IngestResult, error) {
	ctx = ExtractKafkaTraceHeaders(ctx, msg.Headers)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if len(msg.Value) == 0 {
		return nil, fmt.Errorf("empty Kafka message payload")
	}

	var event OutcomeEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal outcome event: %w", err)
	}

	return c.handler.Ingest(ctx, &event)
}

// ProcessBatch processes a slice of Kafka outcome messages, propagating trace context for each record.
func (c *OutcomeConsumer) ProcessBatch(ctx context.Context, batch []ingress.KafkaMessage) ([]*IngestResult, error) {
	results := make([]*IngestResult, 0, len(batch))
	for _, msg := range batch {
		res, err := c.ProcessMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}
