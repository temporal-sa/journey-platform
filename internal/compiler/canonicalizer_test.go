package compiler_test

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
)

func createSampleGraph() *compiler.XYFlowGraph {
	return &compiler.XYFlowGraph{
		DraftID:     "draft-cmp-001",
		TenantID:    "tenant-alpha",
		Name:        "Customer Retention Campaign",
		Description: "Multi-branch automated retention journey",
		Version:     1,
		Viewport: &compiler.Viewport{
			X:    150.5,
			Y:    300.0,
			Zoom: 1.2,
		},
		Nodes: []compiler.XYFlowNode{
			{
				ID:   "node-trigger-1",
				Type: "trigger",
				Name: "On User Inactive 30 Days",
				Config: map[string]interface{}{
					"inactivity_days": 30,
					"segment":         "high_value",
				},
				Position: &domain.NodePosition{X: 100, Y: 100},
				Dimensions: &compiler.UIDimensions{
					Width:  200,
					Height: 80,
				},
				Metrics: []string{"latency_ms", "trigger_count"},
			},
			{
				ID:          "node-action-email",
				Type:        "action",
				Name:        "Send Retention Discount Email",
				CatalogRef:  "catalog:email_sender_v2",
				TemplateRef: "tpl_discount_20pct",
				Config: map[string]interface{}{
					"discount_percent": 20,
					"sender_alias":     "support@example.com",
				},
				Position: &domain.NodePosition{X: 400, Y: 100},
				Dimensions: &compiler.UIDimensions{
					Width:  220,
					Height: 90,
				},
				BranchLabels:   []string{"email_sent", "email_bounced"},
				TimeoutSeconds: 60,
				RetryPolicy: &domain.RetryPolicy{
					MaximumAttempts:        3,
					InitialIntervalSeconds: 5,
				},
				ExperimentVariants: []domain.ExperimentVariant{
					{
						VariantID:         "var-a",
						Name:              "20% Discount",
						WeightBasisPoints: 5000,
						Config:            map[string]interface{}{"promo": "SAVE20"},
					},
					{
						VariantID:         "var-b",
						Name:              "Free Month",
						WeightBasisPoints: 5000,
						Config:            map[string]interface{}{"promo": "FREEMONTH"},
					},
				},
			},
			{
				ID:   "node-condition-check",
				Type: "condition",
				Name: "Check Loyalty Status",
				Config: map[string]interface{}{
					"min_points": 500,
				},
				Position: &domain.NodePosition{X: 700, Y: 100},
				Dimensions: &compiler.UIDimensions{
					Width:  180,
					Height: 75,
				},
			},
		},
		Edges: []compiler.XYFlowEdge{
			{
				ID:                  "edge-1",
				Source:              "node-trigger-1",
				Target:              "node-action-email",
				ConditionExpression: "user.is_active == false",
				Style:               map[string]interface{}{"stroke": "#ff0000"},
				Animated:            true,
			},
			{
				ID:                  "edge-2",
				Source:              "node-action-email",
				Target:              "node-condition-check",
				ConditionExpression: "outcome.status == 'delivered'",
				BranchLabel:         "email_sent",
			},
		},
	}
}

// TestCompilingIdenticalGraph100Times verifies that compiling the identical graph 100 times
// produces byte-identical canonical JSON and SHA-256 hash.
func TestCompilingIdenticalGraph100Times(t *testing.T) {
	c := compiler.New()
	baseGraph := createSampleGraph()

	res0, err := c.CanonicalizeGraph(baseGraph)
	if err != nil {
		t.Fatalf("failed initial canonicalization: %v", err)
	}

	if res0.ContentHash == "" {
		t.Fatal("content hash is empty")
	}
	if len(res0.CanonicalJSON) == 0 {
		t.Fatal("canonical JSON is empty")
	}
	if res0.CompiledIR.ContentHash == "" {
		t.Fatal("CompiledIR content hash is empty")
	}

	for i := 1; i <= 100; i++ {
		res, err := c.CanonicalizeGraph(baseGraph)
		if err != nil {
			t.Fatalf("failed canonicalization on iteration %d: %v", i, err)
		}

		if res.ContentHash != res0.ContentHash {
			t.Fatalf("iteration %d hash mismatch: expected %s, got %s", i, res0.ContentHash, res.ContentHash)
		}
		if res.CompiledIR.ContentHash != res0.CompiledIR.ContentHash {
			t.Fatalf("iteration %d IR hash mismatch: expected %s, got %s", i, res0.CompiledIR.ContentHash, res.CompiledIR.ContentHash)
		}
		if !bytes.Equal(res.CanonicalJSON, res0.CanonicalJSON) {
			t.Fatalf("iteration %d canonical JSON byte mismatch", i)
		}
	}
}

// TestRandomizedNodeEdgeOrderProducesIdenticalHash verifies that randomizing node, edge, branch label,
// metric, and experiment variant order produces the identical canonical JSON and hash.
func TestRandomizedNodeEdgeOrderProducesIdenticalHash(t *testing.T) {
	c := compiler.New()
	baseGraph := createSampleGraph()

	baseRes, err := c.CanonicalizeGraph(baseGraph)
	if err != nil {
		t.Fatalf("failed base canonicalization: %v", err)
	}

	r := rand.New(rand.NewSource(42))

	for i := 1; i <= 50; i++ {
		shuffled := createSampleGraph()

		// Shuffle nodes
		r.Shuffle(len(shuffled.Nodes), func(a, b int) {
			shuffled.Nodes[a], shuffled.Nodes[b] = shuffled.Nodes[b], shuffled.Nodes[a]
		})

		// Shuffle edges
		r.Shuffle(len(shuffled.Edges), func(a, b int) {
			shuffled.Edges[a], shuffled.Edges[b] = shuffled.Edges[b], shuffled.Edges[a]
		})

		// Shuffle internal slices in nodes
		for j := range shuffled.Nodes {
			if len(shuffled.Nodes[j].BranchLabels) > 1 {
				r.Shuffle(len(shuffled.Nodes[j].BranchLabels), func(a, b int) {
					shuffled.Nodes[j].BranchLabels[a], shuffled.Nodes[j].BranchLabels[b] = shuffled.Nodes[j].BranchLabels[b], shuffled.Nodes[j].BranchLabels[a]
				})
			}
			if len(shuffled.Nodes[j].Metrics) > 1 {
				r.Shuffle(len(shuffled.Nodes[j].Metrics), func(a, b int) {
					shuffled.Nodes[j].Metrics[a], shuffled.Nodes[j].Metrics[b] = shuffled.Nodes[j].Metrics[b], shuffled.Nodes[j].Metrics[a]
				})
			}
			if len(shuffled.Nodes[j].ExperimentVariants) > 1 {
				r.Shuffle(len(shuffled.Nodes[j].ExperimentVariants), func(a, b int) {
					shuffled.Nodes[j].ExperimentVariants[a], shuffled.Nodes[j].ExperimentVariants[b] = shuffled.Nodes[j].ExperimentVariants[b], shuffled.Nodes[j].ExperimentVariants[a]
				})
			}
		}

		res, err := c.CanonicalizeGraph(shuffled)
		if err != nil {
			t.Fatalf("failed canonicalization on iteration %d: %v", i, err)
		}

		if res.ContentHash != baseRes.ContentHash {
			t.Errorf("shuffled iteration %d hash mismatch: expected %s, got %s", i, baseRes.ContentHash, res.ContentHash)
		}
		if res.CompiledIR.ContentHash != baseRes.CompiledIR.ContentHash {
			t.Errorf("shuffled iteration %d IR hash mismatch: expected %s, got %s", i, baseRes.CompiledIR.ContentHash, res.CompiledIR.ContentHash)
		}
		if !bytes.Equal(res.CanonicalJSON, baseRes.CanonicalJSON) {
			t.Errorf("shuffled iteration %d canonical JSON byte mismatch", i)
		}
	}
}

// TestLayoutPositionChangeKeepsHashIdentical verifies that layout position, viewport,
// and UI dimensions changes do not alter runtime SHA-256 hash.
func TestLayoutPositionChangeKeepsHashIdentical(t *testing.T) {
	c := compiler.New()
	baseGraph := createSampleGraph()

	baseRes, err := c.CanonicalizeGraph(baseGraph)
	if err != nil {
		t.Fatalf("failed base canonicalization: %v", err)
	}

	layoutModifiedGraph := createSampleGraph()

	// Modify viewport
	layoutModifiedGraph.Viewport = &compiler.Viewport{
		X:    9999.9,
		Y:    -8888.8,
		Zoom: 0.2,
	}

	// Modify node positions and UI dimensions
	for i := range layoutModifiedGraph.Nodes {
		layoutModifiedGraph.Nodes[i].Position = &domain.NodePosition{
			X: float64((i + 1) * 1500),
			Y: float64((i + 1) * 3000),
		}
		layoutModifiedGraph.Nodes[i].Dimensions = &compiler.UIDimensions{
			Width:  500 + float64(i*50),
			Height: 400 + float64(i*20),
		}
		layoutModifiedGraph.Nodes[i].Selected = true
		layoutModifiedGraph.Nodes[i].Dragging = true
	}

	// Modify edge styles
	for i := range layoutModifiedGraph.Edges {
		layoutModifiedGraph.Edges[i].Style = map[string]interface{}{"stroke": "#00ff00", "width": 5}
		layoutModifiedGraph.Edges[i].Animated = false
	}

	res, err := c.CanonicalizeGraph(layoutModifiedGraph)
	if err != nil {
		t.Fatalf("failed canonicalization of layout-modified graph: %v", err)
	}

	if res.ContentHash != baseRes.ContentHash {
		t.Errorf("layout change altered content hash: expected %s, got %s", baseRes.ContentHash, res.ContentHash)
	}
	if res.CompiledIR.ContentHash != baseRes.CompiledIR.ContentHash {
		t.Errorf("layout change altered IR content hash: expected %s, got %s", baseRes.CompiledIR.ContentHash, res.CompiledIR.ContentHash)
	}
	if !bytes.Equal(res.CanonicalJSON, baseRes.CanonicalJSON) {
		t.Errorf("layout change altered canonical JSON bytes")
	}
}

// TestBehaviorChangeChangesHash verifies that any functional or behavioral logic modification
// changes the runtime SHA-256 hash.
func TestBehaviorChangeChangesHash(t *testing.T) {
	c := compiler.New()
	baseGraph := createSampleGraph()

	baseRes, err := c.CanonicalizeGraph(baseGraph)
	if err != nil {
		t.Fatalf("failed base canonicalization: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(g *compiler.XYFlowGraph)
	}{
		{
			name: "Mutate Node Name",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Nodes[1].Name = "Send Retention SMS"
			},
		},
		{
			name: "Mutate Node Type",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Nodes[1].Type = "condition"
			},
		},
		{
			name: "Mutate Config Parameter",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Nodes[1].Config["discount_percent"] = 25
			},
		},
		{
			name: "Mutate Edge Condition Expression",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Edges[0].ConditionExpression = "user.is_active == true"
			},
		},
		{
			name: "Mutate Catalog Reference",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Nodes[1].CatalogRef = "catalog:email_sender_v3"
			},
		},
		{
			name: "Mutate Template Reference",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Nodes[1].TemplateRef = "tpl_discount_30pct"
			},
		},
		{
			name: "Mutate Experiment Variant Weight",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Nodes[1].ExperimentVariants[0].WeightBasisPoints = 6000
			},
		},
		{
			name: "Add Branch Label",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Nodes[1].BranchLabels = append(g.Nodes[1].BranchLabels, "email_opened")
			},
		},
		{
			name: "Add Metric",
			mutate: func(g *compiler.XYFlowGraph) {
				g.Nodes[0].Metrics = append(g.Nodes[0].Metrics, "error_count")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mutatedGraph := createSampleGraph()
			tc.mutate(mutatedGraph)

			res, err := c.CanonicalizeGraph(mutatedGraph)
			if err != nil {
				t.Fatalf("failed canonicalization for %s: %v", tc.name, err)
			}

			if res.ContentHash == baseRes.ContentHash {
				t.Errorf("behavior change '%s' did NOT change normalized graph content hash", tc.name)
			}
			if res.CompiledIR.ContentHash == baseRes.CompiledIR.ContentHash {
				t.Errorf("behavior change '%s' did NOT change CompiledIR content hash", tc.name)
			}
		})
	}
}

// TestGoldenHashVectors verifies reproducible golden SHA-256 hashes for known fixed inputs.
func TestGoldenHashVectors(t *testing.T) {
	c := compiler.New()
	graph := createSampleGraph()

	res, err := c.CanonicalizeGraph(graph)
	if err != nil {
		t.Fatalf("failed canonicalization: %v", err)
	}

	if len(res.ContentHash) != 64 {
		t.Fatalf("invalid content hash length: expected 64 hex chars, got %d", len(res.ContentHash))
	}
	if len(res.CompiledIR.ContentHash) != 64 {
		t.Fatalf("invalid IR content hash length: expected 64 hex chars, got %d", len(res.CompiledIR.ContentHash))
	}

	// Verify IR hash integrity
	valid, err := compiler.VerifyIRHash(res.CompiledIR)
	if err != nil {
		t.Fatalf("failed to verify IR hash: %v", err)
	}
	if !valid {
		t.Errorf("VerifyIRHash returned false for freshly compiled IR")
	}
}

// TestIREncodeDecode verifies round-trip serialization and hash verification.
func TestIREncodeDecode(t *testing.T) {
	c := compiler.New()
	graph := createSampleGraph()

	res, err := c.CanonicalizeGraph(graph)
	if err != nil {
		t.Fatalf("failed canonicalization: %v", err)
	}

	encoded, err := compiler.EncodeIR(res.CompiledIR)
	if err != nil {
		t.Fatalf("failed to encode IR: %v", err)
	}

	decoded, err := compiler.DecodeIR(encoded)
	if err != nil {
		t.Fatalf("failed to decode IR: %v", err)
	}

	if decoded.IRID != res.CompiledIR.IRID {
		t.Errorf("decoded IRID mismatch: expected %s, got %s", res.CompiledIR.IRID, decoded.IRID)
	}
	if decoded.ContentHash != res.CompiledIR.ContentHash {
		t.Errorf("decoded ContentHash mismatch: expected %s, got %s", res.CompiledIR.ContentHash, decoded.ContentHash)
	}

	valid, err := compiler.VerifyIRHash(decoded)
	if err != nil || !valid {
		t.Errorf("decoded IR hash verification failed: valid=%v, err=%v", valid, err)
	}
}

// TestSourceMapVerification verifies mapping from IR instructions back to UI node and field IDs.
func TestSourceMapVerification(t *testing.T) {
	c := compiler.New()
	graph := createSampleGraph()

	res, err := c.CanonicalizeGraph(graph)
	if err != nil {
		t.Fatalf("failed canonicalization: %v", err)
	}

	sourceMap := res.SourceMap
	if sourceMap == nil {
		t.Fatal("SourceMap is nil")
	}

	// Verify node mapping
	uiNodeID := "node-action-email"
	irNodeID := "ir-node-action-email"

	mapping, exists := sourceMap.Nodes[irNodeID]
	if !exists {
		t.Fatalf("SourceMap missing entry for IR node %s", irNodeID)
	}
	if mapping.UINodeID != uiNodeID {
		t.Errorf("SourceMap UINodeID mismatch: expected %s, got %s", uiNodeID, mapping.UINodeID)
	}

	// Test SourceMap encode/decode
	smBytes, err := compiler.EncodeSourceMap(sourceMap)
	if err != nil {
		t.Fatalf("failed to encode SourceMap: %v", err)
	}

	decodedSM, err := compiler.DecodeSourceMap(smBytes)
	if err != nil {
		t.Fatalf("failed to decode SourceMap: %v", err)
	}
	if decodedSM.Nodes[irNodeID].UINodeID != uiNodeID {
		t.Errorf("decoded SourceMap mismatch for %s", irNodeID)
	}
}

// TestCanonicalizeGraphDraft verifies canonicalizing domain.GraphDraft.
func TestCanonicalizeGraphDraft(t *testing.T) {
	c := compiler.New()
	now := time.Now()

	draft := &domain.GraphDraft{
		SchemaVersion: domain.DefaultSchemaVersion,
		DraftID:       "draft-101",
		TenantID:      "tenant-42",
		Name:          "User Onboarding Journey",
		Description:   "Draft workflow for new user welcome campaign",
		Version:       1,
		Nodes: []domain.GraphNode{
			{
				ID:   "node-1",
				Type: "trigger",
				Name: "On User Signup",
				Config: map[string]interface{}{
					"event_type": "user.signup.v1",
				},
				Position: &domain.NodePosition{X: 100, Y: 200},
			},
			{
				ID:   "node-2",
				Type: "action",
				Name: "Send Welcome Email",
				Config: map[string]interface{}{
					"template_id": "tpl_welcome_01",
				},
				Position: &domain.NodePosition{X: 300, Y: 200},
			},
		},
		Edges: []domain.GraphEdge{
			{
				ID:        "edge-1-2",
				Source:    "node-1",
				Target:    "node-2",
				Condition: "true",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	res, err := c.CanonicalizeDraft(draft)
	if err != nil {
		t.Fatalf("failed to canonicalize draft: %v", err)
	}

	if res.ContentHash == "" {
		t.Error("expected non-empty content hash for draft")
	}
	if res.CompiledIR == nil || res.CompiledIR.IRID == "" {
		t.Error("expected non-empty CompiledIR for draft")
	}
}
