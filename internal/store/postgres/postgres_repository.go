package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PostgresRepository implements the Repository interface backed by a relational PostgreSQL database via DBTX.
type PostgresRepository struct {
	q  *Queries
	db DBTX
}

// NewPostgresRepository creates a new PostgresRepository wrapping a DBTX connection or transaction.
func NewPostgresRepository(db DBTX) *PostgresRepository {
	return &PostgresRepository{
		q:  NewQueries(db),
		db: db,
	}
}

// TxBeginner optional interface to begin transactions if db is a *sql.DB.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// WithTx executes fn inside a database transaction.
func (r *PostgresRepository) WithTx(ctx context.Context, fn func(repo Repository) error) error {
	if beginner, ok := r.db.(TxBeginner); ok {
		tx, err := beginner.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		txRepo := NewPostgresRepository(tx)
		err = fn(txRepo)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}
	// Fallback if db already is a transaction
	return fn(r)
}

// 1. Catalogs
func (r *PostgresRepository) CreateCatalog(ctx context.Context, c *Catalog) (*Catalog, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO catalogs (tenant_id, record_id, name, component_type, version, description, schema_definition, content_hash, tags, is_deprecated, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING tenant_id, record_id, name, component_type, version, description, schema_definition, content_hash, tags, is_deprecated, created_at, updated_at
	`, c.TenantID, c.RecordID, c.Name, c.ComponentType, c.Version, c.Description, c.SchemaDefinition, c.ContentHash, c.Tags, c.IsDeprecated, c.CreatedAt, c.UpdatedAt)

	var res Catalog
	err := row.Scan(&res.TenantID, &res.RecordID, &res.Name, &res.ComponentType, &res.Version, &res.Description, &res.SchemaDefinition, &res.ContentHash, &res.Tags, &res.IsDeprecated, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog: %w", err)
	}
	return &res, nil
}

func (r *PostgresRepository) GetCatalog(ctx context.Context, tenantID, recordID string) (*Catalog, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, record_id, name, component_type, version, description, schema_definition, content_hash, tags, is_deprecated, created_at, updated_at
		FROM catalogs WHERE tenant_id = $1 AND record_id = $2
	`, tenantID, recordID)

	var res Catalog
	err := row.Scan(&res.TenantID, &res.RecordID, &res.Name, &res.ComponentType, &res.Version, &res.Description, &res.SchemaDefinition, &res.ContentHash, &res.Tags, &res.IsDeprecated, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListCatalogs(ctx context.Context, tenantID string) ([]Catalog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, record_id, name, component_type, version, description, schema_definition, content_hash, tags, is_deprecated, created_at, updated_at
		FROM catalogs WHERE tenant_id = $1 ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Catalog
	for rows.Next() {
		var c Catalog
		if err := rows.Scan(&c.TenantID, &c.RecordID, &c.Name, &c.ComponentType, &c.Version, &c.Description, &c.SchemaDefinition, &c.ContentHash, &c.Tags, &c.IsDeprecated, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *PostgresRepository) UpdateCatalog(ctx context.Context, c *Catalog) (*Catalog, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE catalogs
		SET name = $3, component_type = $4, version = $5, description = $6, schema_definition = $7, content_hash = $8, tags = $9, is_deprecated = $10, updated_at = $11
		WHERE tenant_id = $1 AND record_id = $2
		RETURNING tenant_id, record_id, name, component_type, version, description, schema_definition, content_hash, tags, is_deprecated, created_at, updated_at
	`, c.TenantID, c.RecordID, c.Name, c.ComponentType, c.Version, c.Description, c.SchemaDefinition, c.ContentHash, c.Tags, c.IsDeprecated, c.UpdatedAt)

	var res Catalog
	err := row.Scan(&res.TenantID, &res.RecordID, &res.Name, &res.ComponentType, &res.Version, &res.Description, &res.SchemaDefinition, &res.ContentHash, &res.Tags, &res.IsDeprecated, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) DeleteCatalog(ctx context.Context, tenantID, recordID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM catalogs WHERE tenant_id = $1 AND record_id = $2`, tenantID, recordID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// 2. Journey Drafts
func (r *PostgresRepository) CreateJourneyDraft(ctx context.Context, d *JourneyDraft) (*JourneyDraft, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO journey_drafts (tenant_id, draft_id, name, description, version, nodes, edges, content_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING tenant_id, draft_id, name, description, version, nodes, edges, content_hash, created_at, updated_at
	`, d.TenantID, d.DraftID, d.Name, d.Description, d.Version, d.Nodes, d.Edges, d.ContentHash, d.CreatedAt, d.UpdatedAt)

	var res JourneyDraft
	err := row.Scan(&res.TenantID, &res.DraftID, &res.Name, &res.Description, &res.Version, &res.Nodes, &res.Edges, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetJourneyDraft(ctx context.Context, tenantID, draftID string) (*JourneyDraft, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, draft_id, name, description, version, nodes, edges, content_hash, created_at, updated_at
		FROM journey_drafts WHERE tenant_id = $1 AND draft_id = $2
	`, tenantID, draftID)

	var res JourneyDraft
	err := row.Scan(&res.TenantID, &res.DraftID, &res.Name, &res.Description, &res.Version, &res.Nodes, &res.Edges, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListJourneyDrafts(ctx context.Context, tenantID string) ([]JourneyDraft, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, draft_id, name, description, version, nodes, edges, content_hash, created_at, updated_at
		FROM journey_drafts WHERE tenant_id = $1 ORDER BY updated_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []JourneyDraft
	for rows.Next() {
		var d JourneyDraft
		if err := rows.Scan(&d.TenantID, &d.DraftID, &d.Name, &d.Description, &d.Version, &d.Nodes, &d.Edges, &d.ContentHash, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *PostgresRepository) UpdateJourneyDraft(ctx context.Context, d *JourneyDraft) (*JourneyDraft, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE journey_drafts
		SET name = $3, description = $4, version = $5, nodes = $6, edges = $7, content_hash = $8, updated_at = $9
		WHERE tenant_id = $1 AND draft_id = $2
		RETURNING tenant_id, draft_id, name, description, version, nodes, edges, content_hash, created_at, updated_at
	`, d.TenantID, d.DraftID, d.Name, d.Description, d.Version, d.Nodes, d.Edges, d.ContentHash, d.UpdatedAt)

	var res JourneyDraft
	err := row.Scan(&res.TenantID, &res.DraftID, &res.Name, &res.Description, &res.Version, &res.Nodes, &res.Edges, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) DeleteJourneyDraft(ctx context.Context, tenantID, draftID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM journey_drafts WHERE tenant_id = $1 AND draft_id = $2`, tenantID, draftID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// 3. Journey Versions
func (r *PostgresRepository) CreateJourneyVersion(ctx context.Context, v *JourneyVersion) (*JourneyVersion, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO journey_versions (tenant_id, version_id, draft_id, version, entry_node_id, nodes, edges, content_hash, compiled_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING tenant_id, version_id, draft_id, version, entry_node_id, nodes, edges, content_hash, compiled_at, created_at
	`, v.TenantID, v.VersionID, v.DraftID, v.Version, v.EntryNodeID, v.Nodes, v.Edges, v.ContentHash, v.CompiledAt, v.CreatedAt)

	var res JourneyVersion
	err := row.Scan(&res.TenantID, &res.VersionID, &res.DraftID, &res.Version, &res.EntryNodeID, &res.Nodes, &res.Edges, &res.ContentHash, &res.CompiledAt, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetJourneyVersion(ctx context.Context, tenantID, versionID string) (*JourneyVersion, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, version_id, draft_id, version, entry_node_id, nodes, edges, content_hash, compiled_at, created_at
		FROM journey_versions WHERE tenant_id = $1 AND version_id = $2
	`, tenantID, versionID)

	var res JourneyVersion
	err := row.Scan(&res.TenantID, &res.VersionID, &res.DraftID, &res.Version, &res.EntryNodeID, &res.Nodes, &res.Edges, &res.ContentHash, &res.CompiledAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetJourneyVersionByDraftAndVersion(ctx context.Context, tenantID, draftID string, version int32) (*JourneyVersion, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, version_id, draft_id, version, entry_node_id, nodes, edges, content_hash, compiled_at, created_at
		FROM journey_versions WHERE tenant_id = $1 AND draft_id = $2 AND version = $3
	`, tenantID, draftID, version)

	var res JourneyVersion
	err := row.Scan(&res.TenantID, &res.VersionID, &res.DraftID, &res.Version, &res.EntryNodeID, &res.Nodes, &res.Edges, &res.ContentHash, &res.CompiledAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListJourneyVersionsByDraft(ctx context.Context, tenantID, draftID string) ([]JourneyVersion, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, version_id, draft_id, version, entry_node_id, nodes, edges, content_hash, compiled_at, created_at
		FROM journey_versions WHERE tenant_id = $1 AND draft_id = $2 ORDER BY version DESC
	`, tenantID, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []JourneyVersion
	for rows.Next() {
		var v JourneyVersion
		if err := rows.Scan(&v.TenantID, &v.VersionID, &v.DraftID, &v.Version, &v.EntryNodeID, &v.Nodes, &v.Edges, &v.ContentHash, &v.CompiledAt, &v.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

// 4. Idempotency Keys
func (r *PostgresRepository) CreateIdempotencyKey(ctx context.Context, ik *IdempotencyKey) (*IdempotencyKey, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO idempotency_keys (tenant_id, key, scope, status, response_payload, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING tenant_id, key, scope, status, response_payload, expires_at, created_at, updated_at
	`, ik.TenantID, ik.Key, ik.Scope, ik.Status, ik.ResponsePayload, ik.ExpiresAt, ik.CreatedAt, ik.UpdatedAt)

	var res IdempotencyKey
	err := row.Scan(&res.TenantID, &res.Key, &res.Scope, &res.Status, &res.ResponsePayload, &res.ExpiresAt, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetIdempotencyKey(ctx context.Context, tenantID, scope, key string) (*IdempotencyKey, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, key, scope, status, response_payload, expires_at, created_at, updated_at
		FROM idempotency_keys WHERE tenant_id = $1 AND scope = $2 AND key = $3
	`, tenantID, scope, key)

	var res IdempotencyKey
	err := row.Scan(&res.TenantID, &res.Key, &res.Scope, &res.Status, &res.ResponsePayload, &res.ExpiresAt, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) UpdateIdempotencyKey(ctx context.Context, ik *IdempotencyKey) (*IdempotencyKey, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE idempotency_keys
		SET status = $4, response_payload = $5, updated_at = $6
		WHERE tenant_id = $1 AND scope = $2 AND key = $3
		RETURNING tenant_id, key, scope, status, response_payload, expires_at, created_at, updated_at
	`, ik.TenantID, ik.Scope, ik.Key, ik.Status, ik.ResponsePayload, ik.UpdatedAt)

	var res IdempotencyKey
	err := row.Scan(&res.TenantID, &res.Key, &res.Scope, &res.Status, &res.ResponsePayload, &res.ExpiresAt, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 5. Kafka Inbox
func (r *PostgresRepository) SaveKafkaInboxMessage(ctx context.Context, msg *KafkaInbox) (*KafkaInbox, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO kafka_inbox (tenant_id, message_id, event_id, topic, partition, offset_val, payload, status, processed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING tenant_id, message_id, event_id, topic, partition, offset_val, payload, status, processed_at, created_at
	`, msg.TenantID, msg.MessageID, msg.EventID, msg.Topic, msg.Partition, msg.OffsetVal, msg.Payload, msg.Status, msg.ProcessedAt, msg.CreatedAt)

	var res KafkaInbox
	err := row.Scan(&res.TenantID, &res.MessageID, &res.EventID, &res.Topic, &res.Partition, &res.OffsetVal, &res.Payload, &res.Status, &res.ProcessedAt, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetKafkaInboxMessage(ctx context.Context, tenantID, messageID string) (*KafkaInbox, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, message_id, event_id, topic, partition, offset_val, payload, status, processed_at, created_at
		FROM kafka_inbox WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID)

	var res KafkaInbox
	err := row.Scan(&res.TenantID, &res.MessageID, &res.EventID, &res.Topic, &res.Partition, &res.OffsetVal, &res.Payload, &res.Status, &res.ProcessedAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) MarkKafkaInboxProcessed(ctx context.Context, tenantID, messageID string, processedAt time.Time) (*KafkaInbox, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE kafka_inbox SET status = 'processed', processed_at = $3
		WHERE tenant_id = $1 AND message_id = $2
		RETURNING tenant_id, message_id, event_id, topic, partition, offset_val, payload, status, processed_at, created_at
	`, tenantID, messageID, processedAt)

	var res KafkaInbox
	err := row.Scan(&res.TenantID, &res.MessageID, &res.EventID, &res.Topic, &res.Partition, &res.OffsetVal, &res.Payload, &res.Status, &res.ProcessedAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 6. Target Manifests
func (r *PostgresRepository) CreateTargetManifest(ctx context.Context, m *TargetManifest) (*TargetManifest, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO target_manifests (tenant_id, manifest_id, name, query_spec, total_count, content_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING tenant_id, manifest_id, name, query_spec, total_count, content_hash, created_at, updated_at
	`, m.TenantID, m.ManifestID, m.Name, m.QuerySpec, m.TotalCount, m.ContentHash, m.CreatedAt, m.UpdatedAt)

	var res TargetManifest
	err := row.Scan(&res.TenantID, &res.ManifestID, &res.Name, &res.QuerySpec, &res.TotalCount, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetTargetManifest(ctx context.Context, tenantID, manifestID string) (*TargetManifest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, manifest_id, name, query_spec, total_count, content_hash, created_at, updated_at
		FROM target_manifests WHERE tenant_id = $1 AND manifest_id = $2
	`, tenantID, manifestID)

	var res TargetManifest
	err := row.Scan(&res.TenantID, &res.ManifestID, &res.Name, &res.QuerySpec, &res.TotalCount, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 7. Dispatch Ledger
func (r *PostgresRepository) CreateDispatchLedger(ctx context.Context, dl *DispatchLedger) (*DispatchLedger, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO dispatch_ledger (tenant_id, dispatch_id, manifest_id, journey_version_id, subject_id, status, dispatched_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING tenant_id, dispatch_id, manifest_id, journey_version_id, subject_id, status, dispatched_at, created_at
	`, dl.TenantID, dl.DispatchID, dl.ManifestID, dl.JourneyVersionID, dl.SubjectID, dl.Status, dl.DispatchedAt, dl.CreatedAt)

	var res DispatchLedger
	err := row.Scan(&res.TenantID, &res.DispatchID, &res.ManifestID, &res.JourneyVersionID, &res.SubjectID, &res.Status, &res.DispatchedAt, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetDispatchLedger(ctx context.Context, tenantID, dispatchID string) (*DispatchLedger, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, dispatch_id, manifest_id, journey_version_id, subject_id, status, dispatched_at, created_at
		FROM dispatch_ledger WHERE tenant_id = $1 AND dispatch_id = $2
	`, tenantID, dispatchID)

	var res DispatchLedger
	err := row.Scan(&res.TenantID, &res.DispatchID, &res.ManifestID, &res.JourneyVersionID, &res.SubjectID, &res.Status, &res.DispatchedAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 8. Enrollments
func (r *PostgresRepository) CreateEnrollment(ctx context.Context, e *Enrollment) (*Enrollment, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO enrollments (tenant_id, enrollment_id, journey_version_id, subject_id, status, current_node_id, state_data, enrolled_at, updated_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING tenant_id, enrollment_id, journey_version_id, subject_id, status, current_node_id, state_data, enrolled_at, updated_at, completed_at
	`, e.TenantID, e.EnrollmentID, e.JourneyVersionID, e.SubjectID, e.Status, e.CurrentNodeID, e.StateData, e.EnrolledAt, e.UpdatedAt, e.CompletedAt)

	var res Enrollment
	err := row.Scan(&res.TenantID, &res.EnrollmentID, &res.JourneyVersionID, &res.SubjectID, &res.Status, &res.CurrentNodeID, &res.StateData, &res.EnrolledAt, &res.UpdatedAt, &res.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetEnrollment(ctx context.Context, tenantID, enrollmentID string) (*Enrollment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, enrollment_id, journey_version_id, subject_id, status, current_node_id, state_data, enrolled_at, updated_at, completed_at
		FROM enrollments WHERE tenant_id = $1 AND enrollment_id = $2
	`, tenantID, enrollmentID)

	var res Enrollment
	err := row.Scan(&res.TenantID, &res.EnrollmentID, &res.JourneyVersionID, &res.SubjectID, &res.Status, &res.CurrentNodeID, &res.StateData, &res.EnrolledAt, &res.UpdatedAt, &res.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListEnrollments(ctx context.Context, tenantID string) ([]Enrollment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, enrollment_id, journey_version_id, subject_id, status, current_node_id, state_data, enrolled_at, updated_at, completed_at
		FROM enrollments WHERE tenant_id = $1 ORDER BY enrolled_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Enrollment
	for rows.Next() {
		var e Enrollment
		if err := rows.Scan(&e.TenantID, &e.EnrollmentID, &e.JourneyVersionID, &e.SubjectID, &e.Status, &e.CurrentNodeID, &e.StateData, &e.EnrolledAt, &e.UpdatedAt, &e.CompletedAt); err != nil {
			return nil, err
		}
		res = append(res, e)
	}
	return res, rows.Err()
}


func (r *PostgresRepository) UpdateEnrollmentStatus(ctx context.Context, tenantID, enrollmentID, status, currentNodeID string, stateData []byte, completedAt *time.Time) (*Enrollment, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE enrollments
		SET status = $3, current_node_id = $4, state_data = $5, updated_at = NOW(), completed_at = $6
		WHERE tenant_id = $1 AND enrollment_id = $2
		RETURNING tenant_id, enrollment_id, journey_version_id, subject_id, status, current_node_id, state_data, enrolled_at, updated_at, completed_at
	`, tenantID, enrollmentID, status, currentNodeID, stateData, completedAt)

	var res Enrollment
	err := row.Scan(&res.TenantID, &res.EnrollmentID, &res.JourneyVersionID, &res.SubjectID, &res.Status, &res.CurrentNodeID, &res.StateData, &res.EnrolledAt, &res.UpdatedAt, &res.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 9. Subscriptions
func (r *PostgresRepository) CreateSubscription(ctx context.Context, s *Subscription) (*Subscription, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO subscriptions (tenant_id, subscription_id, enrollment_id, event_type, condition_expr, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING tenant_id, subscription_id, enrollment_id, event_type, condition_expr, status, created_at, updated_at
	`, s.TenantID, s.SubscriptionID, s.EnrollmentID, s.EventType, s.ConditionExpr, s.Status, s.CreatedAt, s.UpdatedAt)

	var res Subscription
	err := row.Scan(&res.TenantID, &res.SubscriptionID, &res.EnrollmentID, &res.EventType, &res.ConditionExpr, &res.Status, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListSubscriptionsByEnrollment(ctx context.Context, tenantID, enrollmentID string) ([]Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, subscription_id, enrollment_id, event_type, condition_expr, status, created_at, updated_at
		FROM subscriptions WHERE tenant_id = $1 AND enrollment_id = $2 AND status = 'active'
	`, tenantID, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.TenantID, &s.SubscriptionID, &s.EnrollmentID, &s.EventType, &s.ConditionExpr, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

// 10. Action Ledger
func (r *PostgresRepository) CreateActionLedger(ctx context.Context, al *ActionLedger) (*ActionLedger, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO action_ledger (tenant_id, action_id, enrollment_id, node_id, activity_type, status, output, error_message, execution_duration_ms, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING tenant_id, action_id, enrollment_id, node_id, activity_type, status, output, error_message, execution_duration_ms, completed_at, created_at
	`, al.TenantID, al.ActionID, al.EnrollmentID, al.NodeID, al.ActivityType, al.Status, al.Output, al.ErrorMessage, al.ExecutionDurationMS, al.CompletedAt, al.CreatedAt)

	var res ActionLedger
	err := row.Scan(&res.TenantID, &res.ActionID, &res.EnrollmentID, &res.NodeID, &res.ActivityType, &res.Status, &res.Output, &res.ErrorMessage, &res.ExecutionDurationMS, &res.CompletedAt, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetActionLedger(ctx context.Context, tenantID, actionID string) (*ActionLedger, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, action_id, enrollment_id, node_id, activity_type, status, output, error_message, execution_duration_ms, completed_at, created_at
		FROM action_ledger WHERE tenant_id = $1 AND action_id = $2
	`, tenantID, actionID)

	var res ActionLedger
	err := row.Scan(&res.TenantID, &res.ActionID, &res.EnrollmentID, &res.NodeID, &res.ActivityType, &res.Status, &res.Output, &res.ErrorMessage, &res.ExecutionDurationMS, &res.CompletedAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) UpdateActionLedgerStatus(ctx context.Context, tenantID, actionID, status string, output []byte, errMsg string, durationMS int64, completedAt *time.Time) (*ActionLedger, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE action_ledger
		SET status = $3, output = $4, error_message = $5, execution_duration_ms = $6, completed_at = $7
		WHERE tenant_id = $1 AND action_id = $2
		RETURNING tenant_id, action_id, enrollment_id, node_id, activity_type, status, output, error_message, execution_duration_ms, completed_at, created_at
	`, tenantID, actionID, status, output, errMsg, durationMS, completedAt)

	var res ActionLedger
	err := row.Scan(&res.TenantID, &res.ActionID, &res.EnrollmentID, &res.NodeID, &res.ActivityType, &res.Status, &res.Output, &res.ErrorMessage, &res.ExecutionDurationMS, &res.CompletedAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 11. Experiment Definitions
func (r *PostgresRepository) CreateExperimentDefinition(ctx context.Context, exp *ExperimentDefinition) (*ExperimentDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO experiment_definitions (tenant_id, experiment_id, name, description, status, variants, target_audience, content_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING tenant_id, experiment_id, name, description, status, variants, target_audience, content_hash, created_at, updated_at
	`, exp.TenantID, exp.ExperimentID, exp.Name, exp.Description, exp.Status, exp.Variants, exp.TargetAudience, exp.ContentHash, exp.CreatedAt, exp.UpdatedAt)

	var res ExperimentDefinition
	err := row.Scan(&res.TenantID, &res.ExperimentID, &res.Name, &res.Description, &res.Status, &res.Variants, &res.TargetAudience, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetExperimentDefinition(ctx context.Context, tenantID, experimentID string) (*ExperimentDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, experiment_id, name, description, status, variants, target_audience, content_hash, created_at, updated_at
		FROM experiment_definitions WHERE tenant_id = $1 AND experiment_id = $2
	`, tenantID, experimentID)

	var res ExperimentDefinition
	err := row.Scan(&res.TenantID, &res.ExperimentID, &res.Name, &res.Description, &res.Status, &res.Variants, &res.TargetAudience, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListExperimentDefinitions(ctx context.Context, tenantID string) ([]ExperimentDefinition, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, experiment_id, name, description, status, variants, target_audience, content_hash, created_at, updated_at
		FROM experiment_definitions WHERE tenant_id = $1 ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []ExperimentDefinition
	for rows.Next() {
		var exp ExperimentDefinition
		if err := rows.Scan(&exp.TenantID, &exp.ExperimentID, &exp.Name, &exp.Description, &exp.Status, &exp.Variants, &exp.TargetAudience, &exp.ContentHash, &exp.CreatedAt, &exp.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, exp)
	}
	return res, rows.Err()
}


func (r *PostgresRepository) UpdateExperimentDefinition(ctx context.Context, exp *ExperimentDefinition) (*ExperimentDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE experiment_definitions
		SET name = $3, description = $4, status = $5, variants = $6, target_audience = $7, content_hash = $8, updated_at = $9
		WHERE tenant_id = $1 AND experiment_id = $2
		RETURNING tenant_id, experiment_id, name, description, status, variants, target_audience, content_hash, created_at, updated_at
	`, exp.TenantID, exp.ExperimentID, exp.Name, exp.Description, exp.Status, exp.Variants, exp.TargetAudience, exp.ContentHash, exp.UpdatedAt)

	var res ExperimentDefinition
	err := row.Scan(&res.TenantID, &res.ExperimentID, &res.Name, &res.Description, &res.Status, &res.Variants, &res.TargetAudience, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 12. Assignments
func (r *PostgresRepository) CreateAssignment(ctx context.Context, a *Assignment) (*Assignment, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO assignments (tenant_id, assignment_id, experiment_id, subject_id, variant_id, weight_basis_points, assigned_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING tenant_id, assignment_id, experiment_id, subject_id, variant_id, weight_basis_points, assigned_at, created_at
	`, a.TenantID, a.AssignmentID, a.ExperimentID, a.SubjectID, a.VariantID, a.WeightBasisPoints, a.AssignedAt, a.CreatedAt)

	var res Assignment
	err := row.Scan(&res.TenantID, &res.AssignmentID, &res.ExperimentID, &res.SubjectID, &res.VariantID, &res.WeightBasisPoints, &res.AssignedAt, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetAssignment(ctx context.Context, tenantID, assignmentID string) (*Assignment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, assignment_id, experiment_id, subject_id, variant_id, weight_basis_points, assigned_at, created_at
		FROM assignments WHERE tenant_id = $1 AND assignment_id = $2
	`, tenantID, assignmentID)

	var res Assignment
	err := row.Scan(&res.TenantID, &res.AssignmentID, &res.ExperimentID, &res.SubjectID, &res.VariantID, &res.WeightBasisPoints, &res.AssignedAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetAssignmentByExperimentSubject(ctx context.Context, tenantID, experimentID, subjectID string) (*Assignment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, assignment_id, experiment_id, subject_id, variant_id, weight_basis_points, assigned_at, created_at
		FROM assignments WHERE tenant_id = $1 AND experiment_id = $2 AND subject_id = $3
	`, tenantID, experimentID, subjectID)

	var res Assignment
	err := row.Scan(&res.TenantID, &res.AssignmentID, &res.ExperimentID, &res.SubjectID, &res.VariantID, &res.WeightBasisPoints, &res.AssignedAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}
func (r *PostgresRepository) ListAssignmentsByExperiment(ctx context.Context, tenantID, experimentID string) ([]Assignment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, assignment_id, experiment_id, subject_id, variant_id, weight_basis_points, assigned_at, created_at
		FROM assignments WHERE tenant_id = $1 AND experiment_id = $2
	`, tenantID, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.TenantID, &a.AssignmentID, &a.ExperimentID, &a.SubjectID, &a.VariantID, &a.WeightBasisPoints, &a.AssignedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, a)
	}
	return res, nil
}

// 13. Exposures
func (r *PostgresRepository) CreateExposure(ctx context.Context, ex *Exposure) (*Exposure, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO exposures (tenant_id, exposure_id, experiment_id, assignment_id, subject_id, variant_id, weight_basis_points, context, exposed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING tenant_id, exposure_id, experiment_id, assignment_id, subject_id, variant_id, weight_basis_points, context, exposed_at, created_at
	`, ex.TenantID, ex.ExposureID, ex.ExperimentID, ex.AssignmentID, ex.SubjectID, ex.VariantID, ex.WeightBasisPoints, ex.Context, ex.ExposedAt, ex.CreatedAt)

	var res Exposure
	err := row.Scan(&res.TenantID, &res.ExposureID, &res.ExperimentID, &res.AssignmentID, &res.SubjectID, &res.VariantID, &res.WeightBasisPoints, &res.Context, &res.ExposedAt, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetExposure(ctx context.Context, tenantID, exposureID string) (*Exposure, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, exposure_id, experiment_id, assignment_id, subject_id, variant_id, weight_basis_points, context, exposed_at, created_at
		FROM exposures WHERE tenant_id = $1 AND exposure_id = $2
	`, tenantID, exposureID)

	var res Exposure
	err := row.Scan(&res.TenantID, &res.ExposureID, &res.ExperimentID, &res.AssignmentID, &res.SubjectID, &res.VariantID, &res.WeightBasisPoints, &res.Context, &res.ExposedAt, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}
func (r *PostgresRepository) ListExposuresByExperiment(ctx context.Context, tenantID, experimentID string) ([]Exposure, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, exposure_id, experiment_id, assignment_id, subject_id, variant_id, weight_basis_points, context, exposed_at, created_at
		FROM exposures WHERE tenant_id = $1 AND experiment_id = $2
	`, tenantID, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Exposure
	for rows.Next() {
		var ex Exposure
		if err := rows.Scan(&ex.TenantID, &ex.ExposureID, &ex.ExperimentID, &ex.AssignmentID, &ex.SubjectID, &ex.VariantID, &ex.WeightBasisPoints, &ex.Context, &ex.ExposedAt, &ex.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, ex)
	}
	return res, nil
}

// 14. Outbox
func (r *PostgresRepository) CreateOutboxEvent(ctx context.Context, o *Outbox) (*Outbox, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO outbox (tenant_id, id, aggregate_type, aggregate_id, event_type, payload, headers, status, retry_count, created_at, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING tenant_id, id, aggregate_type, aggregate_id, event_type, payload, headers, status, retry_count, created_at, processed_at
	`, o.TenantID, o.ID, o.AggregateType, o.AggregateID, o.EventType, o.Payload, o.Headers, o.Status, o.RetryCount, o.CreatedAt, o.ProcessedAt)

	var res Outbox
	err := row.Scan(&res.TenantID, &res.ID, &res.AggregateType, &res.AggregateID, &res.EventType, &res.Payload, &res.Headers, &res.Status, &res.RetryCount, &res.CreatedAt, &res.ProcessedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetOutboxEvent(ctx context.Context, tenantID, id string) (*Outbox, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, id, aggregate_type, aggregate_id, event_type, payload, headers, status, retry_count, created_at, processed_at
		FROM outbox WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)

	var res Outbox
	err := row.Scan(&res.TenantID, &res.ID, &res.AggregateType, &res.AggregateID, &res.EventType, &res.Payload, &res.Headers, &res.Status, &res.RetryCount, &res.CreatedAt, &res.ProcessedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListPendingOutboxEvents(ctx context.Context, tenantID string, limit int) ([]Outbox, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, id, aggregate_type, aggregate_id, event_type, payload, headers, status, retry_count, created_at, processed_at
		FROM outbox WHERE tenant_id = $1 AND status = 'pending' ORDER BY created_at ASC LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Outbox
	for rows.Next() {
		var o Outbox
		if err := rows.Scan(&o.TenantID, &o.ID, &o.AggregateType, &o.AggregateID, &o.EventType, &o.Payload, &o.Headers, &o.Status, &o.RetryCount, &o.CreatedAt, &o.ProcessedAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, nil
}

func (r *PostgresRepository) MarkOutboxProcessed(ctx context.Context, tenantID, id string, processedAt time.Time) (*Outbox, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE outbox SET status = 'processed', processed_at = $3
		WHERE tenant_id = $1 AND id = $2
		RETURNING tenant_id, id, aggregate_type, aggregate_id, event_type, payload, headers, status, retry_count, created_at, processed_at
	`, tenantID, id, processedAt)

	var res Outbox
	err := row.Scan(&res.TenantID, &res.ID, &res.AggregateType, &res.AggregateID, &res.EventType, &res.Payload, &res.Headers, &res.Status, &res.RetryCount, &res.CreatedAt, &res.ProcessedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 15. Static Lists
func (r *PostgresRepository) CreateStaticList(ctx context.Context, l *StaticList) (*StaticList, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO static_lists (tenant_id, list_id, name, description, item_count, data_classification, items, content_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING tenant_id, list_id, name, description, item_count, data_classification, items, content_hash, created_at, updated_at
	`, l.TenantID, l.ListID, l.Name, l.Description, l.ItemCount, l.DataClassification, l.Items, l.ContentHash, l.CreatedAt, l.UpdatedAt)

	var res StaticList
	err := row.Scan(&res.TenantID, &res.ListID, &res.Name, &res.Description, &res.ItemCount, &res.DataClassification, &res.Items, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetStaticList(ctx context.Context, tenantID, listID string) (*StaticList, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, list_id, name, description, item_count, data_classification, items, content_hash, created_at, updated_at
		FROM static_lists WHERE tenant_id = $1 AND list_id = $2
	`, tenantID, listID)

	var res StaticList
	err := row.Scan(&res.TenantID, &res.ListID, &res.Name, &res.Description, &res.ItemCount, &res.DataClassification, &res.Items, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListStaticLists(ctx context.Context, tenantID string) ([]StaticList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, list_id, name, description, item_count, data_classification, items, content_hash, created_at, updated_at
		FROM static_lists WHERE tenant_id = $1 ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []StaticList
	for rows.Next() {
		var l StaticList
		if err := rows.Scan(&l.TenantID, &l.ListID, &l.Name, &l.Description, &l.ItemCount, &l.DataClassification, &l.Items, &l.ContentHash, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, l)
	}
	return res, rows.Err()
}


func (r *PostgresRepository) UpdateStaticList(ctx context.Context, l *StaticList) (*StaticList, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE static_lists
		SET name = $3, description = $4, item_count = $5, data_classification = $6, items = $7, content_hash = $8, updated_at = $9
		WHERE tenant_id = $1 AND list_id = $2
		RETURNING tenant_id, list_id, name, description, item_count, data_classification, items, content_hash, created_at, updated_at
	`, l.TenantID, l.ListID, l.Name, l.Description, l.ItemCount, l.DataClassification, l.Items, l.ContentHash, l.UpdatedAt)

	var res StaticList
	err := row.Scan(&res.TenantID, &res.ListID, &res.Name, &res.Description, &res.ItemCount, &res.DataClassification, &res.Items, &res.ContentHash, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) DeleteStaticList(ctx context.Context, tenantID, listID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM static_lists WHERE tenant_id = $1 AND list_id = $2`, tenantID, listID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// 16. Test Runs
func (r *PostgresRepository) CreateTestRun(ctx context.Context, tr *TestRun) (*TestRun, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO test_runs (tenant_id, test_run_id, draft_id, ir_id, status, mock_inputs, expected_outcomes, actual_outcomes, execution_time_ms, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING tenant_id, test_run_id, draft_id, ir_id, status, mock_inputs, expected_outcomes, actual_outcomes, execution_time_ms, created_at, updated_at
	`, tr.TenantID, tr.TestRunID, tr.DraftID, tr.IRID, tr.Status, tr.MockInputs, tr.ExpectedOutcomes, tr.ActualOutcomes, tr.ExecutionTimeMS, tr.CreatedAt, tr.UpdatedAt)

	var res TestRun
	err := row.Scan(&res.TenantID, &res.TestRunID, &res.DraftID, &res.IRID, &res.Status, &res.MockInputs, &res.ExpectedOutcomes, &res.ActualOutcomes, &res.ExecutionTimeMS, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetTestRun(ctx context.Context, tenantID, testRunID string) (*TestRun, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, test_run_id, draft_id, ir_id, status, mock_inputs, expected_outcomes, actual_outcomes, execution_time_ms, created_at, updated_at
		FROM test_runs WHERE tenant_id = $1 AND test_run_id = $2
	`, tenantID, testRunID)

	var res TestRun
	err := row.Scan(&res.TenantID, &res.TestRunID, &res.DraftID, &res.IRID, &res.Status, &res.MockInputs, &res.ExpectedOutcomes, &res.ActualOutcomes, &res.ExecutionTimeMS, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListTestRuns(ctx context.Context, tenantID string) ([]TestRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, test_run_id, draft_id, ir_id, status, mock_inputs, expected_outcomes, actual_outcomes, execution_time_ms, created_at, updated_at
		FROM test_runs WHERE tenant_id = $1 ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []TestRun
	for rows.Next() {
		var tr TestRun
		if err := rows.Scan(&tr.TenantID, &tr.TestRunID, &tr.DraftID, &tr.IRID, &tr.Status, &tr.MockInputs, &tr.ExpectedOutcomes, &tr.ActualOutcomes, &tr.ExecutionTimeMS, &tr.CreatedAt, &tr.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, tr)
	}
	return res, rows.Err()
}


func (r *PostgresRepository) UpdateTestRun(ctx context.Context, tenantID, testRunID, status string, actualOutcomes []byte, durationMS int64) (*TestRun, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE test_runs
		SET status = $3, actual_outcomes = $4, execution_time_ms = $5, updated_at = NOW()
		WHERE tenant_id = $1 AND test_run_id = $2
		RETURNING tenant_id, test_run_id, draft_id, ir_id, status, mock_inputs, expected_outcomes, actual_outcomes, execution_time_ms, created_at, updated_at
	`, tenantID, testRunID, status, actualOutcomes, durationMS)

	var res TestRun
	err := row.Scan(&res.TenantID, &res.TestRunID, &res.DraftID, &res.IRID, &res.Status, &res.MockInputs, &res.ExpectedOutcomes, &res.ActualOutcomes, &res.ExecutionTimeMS, &res.CreatedAt, &res.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// 17. Lifecycle Events
func (r *PostgresRepository) RecordLifecycleEvent(ctx context.Context, le *LifecycleEvent) (*LifecycleEvent, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO lifecycle_events (tenant_id, event_id, entity_type, entity_id, event_name, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING tenant_id, event_id, entity_type, entity_id, event_name, payload, created_at
	`, le.TenantID, le.EventID, le.EntityType, le.EntityID, le.EventName, le.Payload, le.CreatedAt)

	var res LifecycleEvent
	err := row.Scan(&res.TenantID, &res.EventID, &res.EntityType, &res.EntityID, &res.EventName, &res.Payload, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) ListLifecycleEventsByEntity(ctx context.Context, tenantID, entityType, entityID string) ([]LifecycleEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tenant_id, event_id, entity_type, entity_id, event_name, payload, created_at
		FROM lifecycle_events WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3 ORDER BY created_at ASC
	`, tenantID, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []LifecycleEvent
	for rows.Next() {
		var le LifecycleEvent
		if err := rows.Scan(&le.TenantID, &le.EventID, &le.EntityType, &le.EntityID, &le.EventName, &le.Payload, &le.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, le)
	}
	return list, nil
}

// 18. Tombstones
func (r *PostgresRepository) CreateTombstone(ctx context.Context, t *Tombstone) (*Tombstone, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO tombstones (tenant_id, tombstone_id, entity_type, entity_id, deleted_by, reason, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING tenant_id, tombstone_id, entity_type, entity_id, deleted_by, reason, deleted_at
	`, t.TenantID, t.TombstoneID, t.EntityType, t.EntityID, t.DeletedBy, t.Reason, t.DeletedAt)

	var res Tombstone
	err := row.Scan(&res.TenantID, &res.TombstoneID, &res.EntityType, &res.EntityID, &res.DeletedBy, &res.Reason, &res.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetTombstone(ctx context.Context, tenantID, tombstoneID string) (*Tombstone, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, tombstone_id, entity_type, entity_id, deleted_by, reason, deleted_at
		FROM tombstones WHERE tenant_id = $1 AND tombstone_id = $2
	`, tenantID, tombstoneID)

	var res Tombstone
	err := row.Scan(&res.TenantID, &res.TombstoneID, &res.EntityType, &res.EntityID, &res.DeletedBy, &res.Reason, &res.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) GetTombstoneByEntity(ctx context.Context, tenantID, entityType, entityID string) (*Tombstone, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT tenant_id, tombstone_id, entity_type, entity_id, deleted_by, reason, deleted_at
		FROM tombstones WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3
	`, tenantID, entityType, entityID)

	var res Tombstone
	err := row.Scan(&res.TenantID, &res.TombstoneID, &res.EntityType, &res.EntityID, &res.DeletedBy, &res.Reason, &res.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}
