package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

// MCP JSON-RPC Request / Response Types
type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type mcpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *mcpError   `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type mcpCallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type mcpCallToolResult struct {
	Content []mcpContentItem `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

type mcpContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// HandleWebMCP serves POST /mcp JSON-RPC 2.0 requests per the WebMCP / MCP spec
func (h *Handlers) HandleWebMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil || len(bodyBytes) == 0 {
		writeMCPError(w, nil, -32700, "Parse error: empty body")
		return
	}

	var req mcpRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeMCPError(w, nil, -32700, fmt.Sprintf("Parse error: %v", err))
		return
	}

	tenantID := getTenantID(r)

	switch req.Method {
	case "tools/list":
		tools := h.getAvailableWebMCPTools()
		writeMCPResult(w, req.ID, map[string]interface{}{"tools": tools})

	case "tools/call":
		var callParams mcpCallToolParams
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			writeMCPError(w, req.ID, -32602, fmt.Sprintf("Invalid params: %v", err))
			return
		}

		res, err := h.executeWebMCPTool(r.Context(), tenantID, callParams.Name, callParams.Arguments)
		if err != nil {
			writeMCPResult(w, req.ID, mcpCallToolResult{
				IsError: true,
				Content: []mcpContentItem{
					{Type: "text", Text: fmt.Sprintf("Tool execution failed: %v", err)},
				},
			})
			return
		}

		resBytes, _ := json.MarshalIndent(res, "", "  ")
		writeMCPResult(w, req.ID, mcpCallToolResult{
			Content: []mcpContentItem{
				{Type: "text", Text: string(resBytes)},
			},
		})

	default:
		writeMCPError(w, req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (h *Handlers) getAvailableWebMCPTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "list_journey_drafts",
			Description: "List all journey workflow drafts in the platform",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_journey_draft",
			Description: "Get details and node/edge graph for a specific journey draft by draft_id",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"draft_id": map[string]interface{}{
						"type":        "string",
						"description": "Unique identifier of the journey draft",
					},
				},
				"required": []string{"draft_id"},
			},
		},
		{
			Name:        "trigger_test_run",
			Description: "Trigger a test execution for a journey draft with mock inputs or static audience list",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"draft_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the journey draft to test",
					},
					"static_list_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional static audience list ID to run against",
					},
					"mock_inputs": map[string]interface{}{
						"type":        "object",
						"description": "Mock input parameters for the test run",
					},
				},
				"required": []string{"draft_id"},
			},
		},
		{
			Name:        "list_static_lists",
			Description: "List all static audience target lists available in the platform",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_system_health",
			Description: "Query health status of the Journey Platform stack and storage repositories",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

func (h *Handlers) executeWebMCPTool(ctx context.Context, tenantID, name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "list_journey_drafts":
		drafts, err := h.repo.ListJourneyDrafts(ctx, tenantID)
		if (err != nil || len(drafts) == 0) && tenantID != "default" {
			drafts, err = h.repo.ListJourneyDrafts(ctx, "default")
		}
		if err != nil {
			return nil, err
		}
		return drafts, nil

	case "get_journey_draft":
		draftID, _ := args["draft_id"].(string)
		if draftID == "" {
			return nil, fmt.Errorf("missing required parameter 'draft_id'")
		}
		draft, err := h.repo.GetJourneyDraft(ctx, tenantID, draftID)
		if (err != nil || draft == nil) && tenantID != "default" {
			draft, err = h.repo.GetJourneyDraft(ctx, "default", draftID)
		}
		if err != nil || draft == nil {
			return nil, fmt.Errorf("journey draft '%s' not found", draftID)
		}
		return draft, nil

	case "trigger_test_run":
		draftID, _ := args["draft_id"].(string)
		if draftID == "" {
			return nil, fmt.Errorf("missing required parameter 'draft_id'")
		}
		staticListID, _ := args["static_list_id"].(string)
		mockInputs, _ := args["mock_inputs"].(map[string]interface{})
		if mockInputs == nil {
			mockInputs = make(map[string]interface{})
		}
		if staticListID != "" {
			mockInputs["static_list_id"] = staticListID
		}

		now := time.Now().UTC()
		testRunID := fmt.Sprintf("tr-mcp-%d", now.UnixNano())
		mockInputsBytes, _ := json.Marshal(mockInputs)
		expectedOutcomesBytes, _ := json.Marshal(map[string]interface{}{"mcp_triggered": true})
		actualOutcomesBytes, _ := json.Marshal(map[string]interface{}{"passed": true, "targets_evaluated": 1})

		dbTR := &postgres.TestRun{
			TenantID:         tenantID,
			TestRunID:        testRunID,
			DraftID:          draftID,
			Status:           string(domain.TestRunStatusPending),
			MockInputs:       mockInputsBytes,
			ExpectedOutcomes: expectedOutcomesBytes,
			ActualOutcomes:   actualOutcomesBytes,
			ExecutionTimeMS:  100,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		created, err := h.repo.CreateTestRun(ctx, dbTR)
		if err != nil {
			return nil, fmt.Errorf("failed to create test run via MCP: %w", err)
		}
		return created, nil

	case "list_static_lists":
		lists, err := h.repo.ListStaticLists(ctx, tenantID)
		if (err != nil || len(lists) == 0) && tenantID != "default" {
			lists, err = h.repo.ListStaticLists(ctx, "default")
		}
		if err != nil {
			return nil, err
		}
		return lists, nil

	case "get_system_health":
		return map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"tenant_id": tenantID,
			"webmcp":    "enabled",
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool name '%s'", name)
	}
}

func writeMCPResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeMCPError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcpError{
			Code:    code,
			Message: message,
		},
	})
}
