package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

// Projector maintains read-model projections for journey workflow runs.
type Projector struct {
	mu          sync.RWMutex
	projections map[string]*domain.RunProjection
	repo        postgres.Repository
}

// NewProjector creates a new RunProjection read-model projector.
func NewProjector(repo postgres.Repository) *Projector {
	return &Projector{
		projections: make(map[string]*domain.RunProjection),
		repo:        repo,
	}
}

func makeKey(tenantID, runID string) string {
	if tenantID == "" {
		tenantID = "default"
	}
	return tenantID + ":" + runID
}

// ProjectLifecycleEvent processes a workflow lifecycle event and updates the run projection.
func (p *Projector) ProjectLifecycleEvent(ctx context.Context, le *postgres.LifecycleEvent) (*domain.RunProjection, error) {
	if le == nil {
		return nil, fmt.Errorf("lifecycle event is nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := makeKey(le.TenantID, le.EntityID)
	proj, exists := p.projections[key]
	now := time.Now()

	if !exists {
		proj = &domain.RunProjection{
			SchemaVersion: domain.DefaultSchemaVersion,
			RunID:         le.EntityID,
			TenantID:      le.TenantID,
			WorkflowID:    le.EntityID,
			Status:        domain.RunStatusRunning,
			CurrentNodes:  []string{"start"},
			Variables:     make(map[string]interface{}),
			StartedAt:     le.CreatedAt,
			UpdatedAt:     now,
		}
		p.projections[key] = proj
	}

	// Parse payload if present
	var payload map[string]interface{}
	if len(le.Payload) > 0 {
		_ = json.Unmarshal(le.Payload, &payload)
	}

	switch le.EventName {
	case "workflow_started", "WorkflowStarted":
		proj.Status = domain.RunStatusRunning
		if node, ok := payload["node_id"].(string); ok && node != "" {
			proj.CurrentNodes = []string{node}
		}
	case "node_visited", "node_entered", "NodeVisited", "NodeEntered":
		if node, ok := payload["node_id"].(string); ok && node != "" {
			proj.CurrentNodes = append([]string{node}, proj.CurrentNodes...)
			// Keep unique nodes, max 10 recent
			proj.CurrentNodes = deduplicateNodes(proj.CurrentNodes)
		}
	case "variables_updated", "VariablesUpdated":
		if vars, ok := payload["variables"].(map[string]interface{}); ok {
			for k, v := range vars {
				proj.Variables[k] = v
			}
		}
	case "workflow_completed", "WorkflowCompleted":
		proj.Status = domain.RunStatusCompleted
		proj.CompletedAt = &now
	case "workflow_failed", "WorkflowFailed":
		proj.Status = domain.RunStatusFailed
		proj.CompletedAt = &now
	case "workflow_terminated", "WorkflowTerminated":
		proj.Status = domain.RunStatusTerminated
		proj.CompletedAt = &now
	}

	proj.UpdatedAt = now
	return proj, nil
}

// UpsertProjection directly sets or updates a RunProjection.
func (p *Projector) UpsertProjection(ctx context.Context, proj *domain.RunProjection) error {
	if proj == nil || proj.RunID == "" {
		return fmt.Errorf("invalid projection")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := makeKey(proj.TenantID, proj.RunID)
	proj.UpdatedAt = time.Now()
	p.projections[key] = proj
	return nil
}

// GetRunProjection retrieves a run projection by tenant and run ID.
func (p *Projector) GetRunProjection(tenantID, runID string) (*domain.RunProjection, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	key := makeKey(tenantID, runID)
	proj, ok := p.projections[key]
	if !ok {
		return nil, fmt.Errorf("run projection not found for run_id=%s", runID)
	}

	// Return deep copy
	clone := *proj
	clone.CurrentNodes = append([]string(nil), proj.CurrentNodes...)
	clone.Variables = make(map[string]interface{})
	for k, v := range proj.Variables {
		clone.Variables[k] = v
	}
	return &clone, nil
}

// ListRunProjections lists all run projections for a tenant.
func (p *Projector) ListRunProjections(tenantID string) ([]*domain.RunProjection, error) {
	if tenantID == "" {
		tenantID = "default"
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	var results []*domain.RunProjection
	for _, proj := range p.projections {
		if proj.TenantID == tenantID {
			clone := *proj
			results = append(results, &clone)
		}
	}
	return results, nil
}

func deduplicateNodes(nodes []string) []string {
	seen := make(map[string]bool)
	var res []string
	for _, n := range nodes {
		if !seen[n] && n != "" {
			seen[n] = true
			res = append(res, n)
		}
	}
	if len(res) > 10 {
		return res[:10]
	}
	return res
}
