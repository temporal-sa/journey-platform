package integration

import (
	"fmt"
	"strings"
)

// TraceCorrelation maintains correlated identity tracking across all system stores and boundaries.
type TraceCorrelation struct {
	EventID           string `json:"event_id"`
	TargetID          string `json:"target_id"`
	WorkflowID        string `json:"workflow_id"`
	NodeVisit         string `json:"node_visit"`
	AssignmentID      string `json:"assignment_id"`
	ExposureID        string `json:"exposure_id"`
	ActionID          string `json:"action_id"`
	ProviderRequestID string `json:"provider_request_id"`
	OutcomeID         string `json:"outcome_id"`
	ReportID          string `json:"report_id"`
}

// NewTraceCorrelation creates a new TraceCorrelation instance.
func NewTraceCorrelation(
	eventID, targetID, workflowID, nodeVisit, assignmentID,
	exposureID, actionID, providerReqID, outcomeID, reportID string,
) *TraceCorrelation {
	return &TraceCorrelation{
		EventID:           strings.TrimSpace(eventID),
		TargetID:          strings.TrimSpace(targetID),
		WorkflowID:        strings.TrimSpace(workflowID),
		NodeVisit:         strings.TrimSpace(nodeVisit),
		AssignmentID:      strings.TrimSpace(assignmentID),
		ExposureID:        strings.TrimSpace(exposureID),
		ActionID:          strings.TrimSpace(actionID),
		ProviderRequestID: strings.TrimSpace(providerReqID),
		OutcomeID:         strings.TrimSpace(outcomeID),
		ReportID:          strings.TrimSpace(reportID),
	}
}

// Verify Correlation validates that all 10 required trace correlation fields are non-empty and well-formed.
func (tc *TraceCorrelation) VerifyCorrelation() error {
	if tc == nil {
		return fmt.Errorf("trace correlation is nil")
	}

	fields := map[string]string{
		"event_id":            tc.EventID,
		"target_id":           tc.TargetID,
		"workflow_id":         tc.WorkflowID,
		"node_visit":          tc.NodeVisit,
		"assignment_id":       tc.AssignmentID,
		"exposure_id":         tc.ExposureID,
		"action_id":           tc.ActionID,
		"provider_request_id": tc.ProviderRequestID,
		"outcome_id":          tc.OutcomeID,
		"report_id":           tc.ReportID,
	}

	var missing []string
	for k, v := range fields {
		if v == "" {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("trace correlation incomplete, missing fields: %s", strings.Join(missing, ", "))
	}

	return nil
}

// Map returns trace correlation fields as a map for logging/tracing headers.
func (tc *TraceCorrelation) Map() map[string]string {
	if tc == nil {
		return map[string]string{}
	}
	return map[string]string{
		"event_id":            tc.EventID,
		"target_id":           tc.TargetID,
		"workflow_id":         tc.WorkflowID,
		"node_visit":          tc.NodeVisit,
		"assignment_id":       tc.AssignmentID,
		"exposure_id":         tc.ExposureID,
		"action_id":           tc.ActionID,
		"provider_request_id": tc.ProviderRequestID,
		"outcome_id":          tc.OutcomeID,
		"report_id":           tc.ReportID,
	}
}
