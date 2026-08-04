package compiler_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
)

func createValidGraph() *compiler.XYFlowGraph {
	return &compiler.XYFlowGraph{
		DraftID:  "draft-valid-1",
		TenantID: "tenant-test",
		Name:     "Valid Journey Graph",
		Version:  1,
		Nodes: []compiler.XYFlowNode{
			{
				ID:   "node-trigger",
				Type: "trigger",
				Name: "Start Trigger",
			},
			{
				ID:         "node_action",
				Type:       "action",
				Name:       "Send Welcome Email",
				CatalogRef: "catalog:send_email_v1",
			},
			{
				ID:           "node-condition",
				Type:         "condition",
				Name:         "Check Engagement",
				BranchLabels: []string{"engaged", "default"},
			},
			{
				ID:   "node-exit-1",
				Type: "exit",
				Name: "Workflow Completed",
			},
			{
				ID:   "node-exit-2",
				Type: "exit",
				Name: "Workflow Fallback Exit",
			},
		},
		Edges: []compiler.XYFlowEdge{
			{
				ID:     "edge-1",
				Source: "node-trigger",
				Target: "node_action",
			},
			{
				ID:                  "edge-2",
				Source:              "node_action",
				Target:              "node-condition",
				ConditionExpression: "node_output.node_action.status == 'sent'",
			},
			{
				ID:          "edge-3",
				Source:      "node-condition",
				Target:      "node-exit-1",
				BranchLabel: "engaged",
			},
			{
				ID:          "edge-4",
				Source:      "node-condition",
				Target:      "node-exit-2",
				BranchLabel: "default",
			},
		},
	}
}

func TestValidGraphPassesValidation(t *testing.T) {
	graph := createValidGraph()
	res, err := compiler.ValidateGraph(graph)
	if err != nil {
		t.Fatalf("unexpected error validating valid graph: %v", err)
	}
	if !res.IsValid {
		t.Fatalf("expected valid graph, got invalid with issues: %+v", res.Issues)
	}
}

func TestCycleDetection(t *testing.T) {
	graph := createValidGraph()
	// Create a cycle: node-condition -> node_action
	graph.Edges = append(graph.Edges, compiler.XYFlowEdge{
		ID:     "edge-cycle",
		Source: "node-condition",
		Target: "node_action",
	})

	res, err := compiler.ValidateGraph(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsValid {
		t.Fatal("expected graph with cycle to be invalid")
	}

	foundCycleIssue := false
	for _, issue := range res.Issues {
		if issue.Code == compiler.CodeCycleDetected {
			foundCycleIssue = true
			if issue.NodeID == "" {
				t.Error("expected node_id to be populated on cycle issue")
			}
			break
		}
	}
	if !foundCycleIssue {
		t.Fatalf("expected ERR_CYCLE_DETECTED code in issues, got: %+v", res.Issues)
	}
}

func TestDisconnectedNode(t *testing.T) {
	graph := createValidGraph()
	// Add an isolated, unreachable node
	graph.Nodes = append(graph.Nodes, compiler.XYFlowNode{
		ID:   "node-orphan",
		Type: "action",
		Name: "Orphaned Action Node",
	})

	res, err := compiler.ValidateGraph(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsValid {
		t.Fatal("expected graph with unreachable node to be invalid")
	}

	foundUnreachable := false
	for _, issue := range res.Issues {
		if issue.Code == compiler.CodeUnreachableNode && issue.NodeID == "node-orphan" {
			foundUnreachable = true
			break
		}
	}
	if !foundUnreachable {
		t.Fatalf("expected ERR_UNREACHABLE_NODE for node-orphan, got issues: %+v", res.Issues)
	}
}

func TestMissingDefaultBranch(t *testing.T) {
	graph := &compiler.XYFlowGraph{
		DraftID: "draft-missing-default",
		Nodes: []compiler.XYFlowNode{
			{ID: "start", Type: "trigger", Name: "Start"},
			{ID: "cond", Type: "condition", Name: "Check VIP"},
			{ID: "exit1", Type: "exit", Name: "VIP Path"},
			{ID: "exit2", Type: "exit", Name: "Regular Path"},
		},
		Edges: []compiler.XYFlowEdge{
			{ID: "e1", Source: "start", Target: "cond"},
			{
				ID:                  "e2",
				Source:              "cond",
				Target:              "exit1",
				ConditionExpression: "parameter.is_vip == true",
				BranchLabel:         "vip",
			},
			{
				ID:                  "e3",
				Source:              "cond",
				Target:              "exit2",
				ConditionExpression: "parameter.is_vip == false",
				BranchLabel:         "regular",
			},
		},
	}

	res, err := compiler.ValidateGraph(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsValid {
		t.Fatal("expected condition node without default branch to be invalid")
	}

	foundMissingDefault := false
	for _, issue := range res.Issues {
		if issue.Code == compiler.CodeMissingDefaultBranch && issue.NodeID == "cond" {
			foundMissingDefault = true
			break
		}
	}
	if !foundMissingDefault {
		t.Fatalf("expected ERR_MISSING_DEFAULT_BRANCH for node 'cond', got issues: %+v", res.Issues)
	}
}

func TestDuplicateEdge(t *testing.T) {
	graph := createValidGraph()
	// Duplicate edge-1 (node-trigger -> node_action)
	graph.Edges = append(graph.Edges, compiler.XYFlowEdge{
		ID:     "edge-1-duplicate",
		Source: "node-trigger",
		Target: "node_action",
	})

	res, err := compiler.ValidateGraph(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsValid {
		t.Fatal("expected graph with duplicate edges to be invalid")
	}

	foundDuplicate := false
	for _, issue := range res.Issues {
		if issue.Code == compiler.CodeDuplicateEdge {
			foundDuplicate = true
			break
		}
	}
	if !foundDuplicate {
		t.Fatalf("expected ERR_DUPLICATE_EDGE in issues, got: %+v", res.Issues)
	}
}

func TestInvalidDelay(t *testing.T) {
	t.Run("Negative Delay", func(t *testing.T) {
		graph := &compiler.XYFlowGraph{
			DraftID: "draft-invalid-delay-1",
			Nodes: []compiler.XYFlowNode{
				{ID: "start", Type: "trigger", Name: "Start"},
				{
					ID:   "delay-node",
					Type: "delay",
					Name: "Wait Negative Time",
					Config: map[string]interface{}{
						"delay_seconds": -500,
					},
				},
				{ID: "exit", Type: "exit", Name: "End"},
			},
			Edges: []compiler.XYFlowEdge{
				{ID: "e1", Source: "start", Target: "delay-node"},
				{ID: "e2", Source: "delay-node", Target: "exit"},
			},
		}

		res, err := compiler.ValidateGraph(graph)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsValid {
			t.Fatal("expected graph with negative delay to be invalid")
		}

		foundInvalidDelay := false
		for _, issue := range res.Issues {
			if issue.Code == compiler.CodeInvalidDelay && issue.NodeID == "delay-node" {
				foundInvalidDelay = true
				break
			}
		}
		if !foundInvalidDelay {
			t.Fatalf("expected ERR_INVALID_DELAY for negative delay, got: %+v", res.Issues)
		}
	})

	t.Run("Exceeded Single Delay Limit", func(t *testing.T) {
		graph := &compiler.XYFlowGraph{
			DraftID: "draft-invalid-delay-2",
			Nodes: []compiler.XYFlowNode{
				{ID: "start", Type: "trigger", Name: "Start"},
				{
					ID:   "delay-node",
					Type: "delay",
					Name: "Wait 100 Days",
					Config: map[string]interface{}{
						"delay_seconds": 100 * 24 * 3600, // 100 days > 30 days max
					},
				},
				{ID: "exit", Type: "exit", Name: "End"},
			},
			Edges: []compiler.XYFlowEdge{
				{ID: "e1", Source: "start", Target: "delay-node"},
				{ID: "e2", Source: "delay-node", Target: "exit"},
			},
		}

		res, err := compiler.ValidateGraph(graph)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsValid {
			t.Fatal("expected graph with single delay > 30 days to be invalid")
		}

		foundInvalidDelay := false
		for _, issue := range res.Issues {
			if issue.Code == compiler.CodeInvalidDelay && issue.NodeID == "delay-node" {
				foundInvalidDelay = true
				break
			}
		}
		if !foundInvalidDelay {
			t.Fatalf("expected ERR_INVALID_DELAY for exceeded delay limit, got: %+v", res.Issues)
		}
	})
}

func TestInvalidCatalogReference(t *testing.T) {
	registry := map[string]domain.CatalogRecord{
		"catalog:valid_action_v1": {
			RecordID:      "catalog:valid_action_v1",
			Name:          "Valid Action",
			ComponentType: domain.ComponentTypeAction,
			IsDeprecated:  false,
		},
		"catalog:deprecated_action": {
			RecordID:      "catalog:deprecated_action",
			Name:          "Deprecated Action",
			ComponentType: domain.ComponentTypeAction,
			IsDeprecated:  true,
		},
	}

	t.Run("Non-existent Catalog Ref", func(t *testing.T) {
		graph := &compiler.XYFlowGraph{
			DraftID: "draft-cat-ref-1",
			Nodes: []compiler.XYFlowNode{
				{ID: "start", Type: "trigger", Name: "Start"},
				{
					ID:         "act",
					Type:       "action",
					Name:       "Unknown Action",
					CatalogRef: "catalog:non_existent",
				},
				{ID: "exit", Type: "exit", Name: "End"},
			},
			Edges: []compiler.XYFlowEdge{
				{ID: "e1", Source: "start", Target: "act"},
				{ID: "e2", Source: "act", Target: "exit"},
			},
		}

		res, err := compiler.ValidateGraph(graph, compiler.WithCatalogRegistry(registry))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsValid {
			t.Fatal("expected non-existent catalog ref to fail validation")
		}

		foundInvalidCatRef := false
		for _, issue := range res.Issues {
			if issue.Code == compiler.CodeInvalidCatalogRef && issue.NodeID == "act" {
				foundInvalidCatRef = true
				break
			}
		}
		if !foundInvalidCatRef {
			t.Fatalf("expected ERR_INVALID_CATALOG_REF, got issues: %+v", res.Issues)
		}
	})

	t.Run("Deprecated Catalog Ref Warning", func(t *testing.T) {
		graph := &compiler.XYFlowGraph{
			DraftID: "draft-cat-ref-2",
			Nodes: []compiler.XYFlowNode{
				{ID: "start", Type: "trigger", Name: "Start"},
				{
					ID:         "act",
					Type:       "action",
					Name:       "Old Action",
					CatalogRef: "catalog:deprecated_action",
				},
				{ID: "exit", Type: "exit", Name: "End"},
			},
			Edges: []compiler.XYFlowEdge{
				{ID: "e1", Source: "start", Target: "act"},
				{ID: "e2", Source: "act", Target: "exit"},
			},
		}

		res, err := compiler.ValidateGraph(graph, compiler.WithCatalogRegistry(registry))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		foundWarning := false
		for _, issue := range res.Issues {
			if issue.Code == compiler.CodeDeprecatedCatalogRef && issue.Severity == domain.ValidationSeverityWarning {
				foundWarning = true
				break
			}
		}
		if !foundWarning {
			t.Fatalf("expected WARN_DEPRECATED_CATALOG_REF warning, got issues: %+v", res.Issues)
		}
	})
}

func TestForwardReferenceInExpression(t *testing.T) {
	// Node A -> Node B -> Node C
	// Edge Node A -> Node B condition references node_output.node_c (which hasn't executed yet!)
	graph := &compiler.XYFlowGraph{
		DraftID: "draft-forward-ref",
		Nodes: []compiler.XYFlowNode{
			{ID: "start", Type: "trigger", Name: "Start"},
			{ID: "node_a", Type: "action", Name: "First Action"},
			{ID: "node_b", Type: "action", Name: "Second Action"},
			{ID: "node_c", Type: "exit", Name: "Exit"},
		},
		Edges: []compiler.XYFlowEdge{
			{ID: "e1", Source: "start", Target: "node_a"},
			{
				ID:                  "e2",
				Source:              "node_a",
				Target:              "node_b",
				ConditionExpression: "node_output.node_c.result == 'ok'", // Forward reference!
			},
			{ID: "e3", Source: "node_b", Target: "node_c"},
		},
	}

	res, err := compiler.ValidateGraph(graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsValid {
		t.Fatal("expected forward reference expression to be invalid")
	}

	foundForwardRef := false
	for _, issue := range res.Issues {
		if issue.Code == compiler.CodeForwardReference {
			foundForwardRef = true
			break
		}
	}
	if !foundForwardRef {
		t.Fatalf("expected ERR_FORWARD_REFERENCE, got issues: %+v", res.Issues)
	}
}

func TestNoStartNodeAndMultipleStartNodes(t *testing.T) {
	t.Run("No Start Node", func(t *testing.T) {
		graph := &compiler.XYFlowGraph{
			DraftID: "draft-no-start",
			Nodes: []compiler.XYFlowNode{
				{ID: "act1", Type: "action", Name: "Action 1"},
				{ID: "exit1", Type: "exit", Name: "Exit 1"},
			},
			Edges: []compiler.XYFlowEdge{
				{ID: "e1", Source: "act1", Target: "exit1"},
			},
		}

		res, err := compiler.ValidateGraph(graph)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsValid {
			t.Fatal("expected graph without start node to be invalid")
		}

		foundNoStart := false
		for _, issue := range res.Issues {
			if issue.Code == compiler.CodeNoStartNode {
				foundNoStart = true
				break
			}
		}
		if !foundNoStart {
			t.Fatalf("expected ERR_NO_START_NODE, got issues: %+v", res.Issues)
		}
	})

	t.Run("Multiple Start Nodes", func(t *testing.T) {
		graph := &compiler.XYFlowGraph{
			DraftID: "draft-multi-start",
			Nodes: []compiler.XYFlowNode{
				{ID: "trig1", Type: "trigger", Name: "Trigger 1"},
				{ID: "trig2", Type: "trigger", Name: "Trigger 2"},
				{ID: "exit1", Type: "exit", Name: "Exit 1"},
			},
			Edges: []compiler.XYFlowEdge{
				{ID: "e1", Source: "trig1", Target: "exit1"},
				{ID: "e2", Source: "trig2", Target: "exit1"},
			},
		}

		res, err := compiler.ValidateGraph(graph)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsValid {
			t.Fatal("expected graph with multiple start nodes to be invalid")
		}

		foundMultiStart := false
		for _, issue := range res.Issues {
			if issue.Code == compiler.CodeMultipleStartNodes {
				foundMultiStart = true
				break
			}
		}
		if !foundMultiStart {
			t.Fatalf("expected ERR_MULTIPLE_START_NODES, got issues: %+v", res.Issues)
		}
	})
}

func TestRapidPropertyPanicFreedom(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	nodeTypes := []string{"trigger", "start", "action", "activity", "condition", "split", "delay", "wait", "exit", "end", "invalid_kind", ""}
	expressions := []string{
		"parameter.count > 10",
		"node_output.node-1.val == 'test'",
		"invalid expr +++ syntax",
		"",
		"user.is_active == true",
		"100 + 'invalid'",
	}

	for i := 0; i < 500; i++ {
		numNodes := rng.Intn(20)
		nodes := make([]compiler.XYFlowNode, numNodes)
		for n := 0; n < numNodes; n++ {
			nodeID := fmt.Sprintf("node-%d", n)
			if rng.Intn(10) == 0 {
				nodeID = "" // test empty ID
			} else if rng.Intn(10) == 0 {
				nodeID = "duplicate-id"
			}
			nodes[n] = compiler.XYFlowNode{
				ID:   nodeID,
				Type: nodeTypes[rng.Intn(len(nodeTypes))],
				Name: fmt.Sprintf("Random Node %d", n),
				Config: map[string]interface{}{
					"delay_seconds": rng.Intn(1000) - 500,
					"catalog_ref":   fmt.Sprintf("cat-%d", rng.Intn(5)),
				},
				BranchLabels: []string{"branch-a", "branch-b"},
			}
		}

		numEdges := rng.Intn(30)
		edges := make([]compiler.XYFlowEdge, numEdges)
		for e := 0; e < numEdges; e++ {
			src := fmt.Sprintf("node-%d", rng.Intn(numNodes+1))
			tgt := fmt.Sprintf("node-%d", rng.Intn(numNodes+1))
			edges[e] = compiler.XYFlowEdge{
				ID:                  fmt.Sprintf("edge-%d", e),
				Source:              src,
				Target:              tgt,
				ConditionExpression: expressions[rng.Intn(len(expressions))],
				BranchLabel:         fmt.Sprintf("branch-%d", rng.Intn(3)),
			}
		}

		graph := &compiler.XYFlowGraph{
			DraftID: fmt.Sprintf("fuzz-draft-%d", i),
			Nodes:   nodes,
			Edges:   edges,
		}

		// Validation must run smoothly and NEVER panic
		res, err := compiler.ValidateGraph(graph)
		if err != nil {
			t.Fatalf("iteration %d returned unexpected error: %v", i, err)
		}
		if res == nil {
			t.Fatalf("iteration %d returned nil ValidationResult", i)
		}
	}
}
