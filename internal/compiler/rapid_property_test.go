package compiler_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
)

// Rapid Property Tests for Canonicalization & State Machine Transitions.
func TestRapidPropertyCanonicalizationLayoutInvariance(t *testing.T) {
	c := compiler.New()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Run 100 property test iterations with randomized graph layouts
	for iteration := 0; iteration < 100; iteration++ {
		numNodes := rng.Intn(10) + 1
		numEdges := rng.Intn(10)

		nodes := make([]compiler.XYFlowNode, numNodes)
		for i := 0; i < numNodes; i++ {
			nodes[i] = compiler.XYFlowNode{
				ID:   fmt.Sprintf("node-%d", i),
				Type: "action",
				Name: fmt.Sprintf("Node %d", i),
				Position: &domain.NodePosition{
					X: rng.Float64() * 1000,
					Y: rng.Float64() * 1000,
				},
				Dimensions: &compiler.UIDimensions{
					Width:  rng.Float64() * 500,
					Height: rng.Float64() * 300,
				},
				Selected: rng.Float64() > 0.5,
				Dragging: rng.Float64() > 0.5,
			}
		}

		edges := make([]compiler.XYFlowEdge, numEdges)
		for i := 0; i < numEdges; i++ {
			src := rng.Intn(numNodes)
			tgt := rng.Intn(numNodes)
			edges[i] = compiler.XYFlowEdge{
				ID:       fmt.Sprintf("edge-%d", i),
				Source:   fmt.Sprintf("node-%d", src),
				Target:   fmt.Sprintf("node-%d", tgt),
				Animated: rng.Float64() > 0.5,
				Style: map[string]interface{}{
					"color": fmt.Sprintf("#%06x", rng.Intn(0xFFFFFF)),
				},
			}
		}

		g1 := &compiler.XYFlowGraph{
			DraftID:  "draft-rapid",
			TenantID: "tenant-rapid",
			Name:     "Rapid Layout Test",
			Version:  1,
			Viewport: &compiler.Viewport{
				X:    rng.Float64() * 100,
				Y:    rng.Float64() * 100,
				Zoom: rng.Float64() * 2,
			},
			Nodes: nodes,
			Edges: edges,
		}

		res1, err := c.CanonicalizeGraph(g1)
		require.NoError(t, err)

		// Create g2 with perturbed UI presentation layout properties (X, Y, Dimensions, Viewport, Selected, Dragging)
		g2 := &compiler.XYFlowGraph{
			DraftID:  g1.DraftID,
			TenantID: g1.TenantID,
			Name:     g1.Name,
			Version:  g1.Version,
			Viewport: &compiler.Viewport{
				X:    g1.Viewport.X + 999.9,
				Y:    g1.Viewport.Y - 444.4,
				Zoom: g1.Viewport.Zoom + 0.5,
			},
			Nodes: make([]compiler.XYFlowNode, len(g1.Nodes)),
			Edges: make([]compiler.XYFlowEdge, len(g1.Edges)),
		}

		for i, n := range g1.Nodes {
			g2.Nodes[i] = n
			g2.Nodes[i].Position = &domain.NodePosition{X: n.Position.X + 1234.5, Y: n.Position.Y + 6789.0}
			g2.Nodes[i].Dimensions = &compiler.UIDimensions{Width: n.Dimensions.Width + 50, Height: n.Dimensions.Height + 50}
			g2.Nodes[i].Selected = !n.Selected
			g2.Nodes[i].Dragging = !n.Dragging
		}
		for i, e := range g1.Edges {
			g2.Edges[i] = e
			g2.Edges[i].Animated = !e.Animated
			g2.Edges[i].Style = map[string]interface{}{"stroke": "#000000"}
		}

		res2, err := c.CanonicalizeGraph(g2)
		require.NoError(t, err)

		assert.Equal(t, res1.ContentHash, res2.ContentHash, "ContentHash must be layout invariant")
		assert.Equal(t, res1.CompiledIR.ContentHash, res2.CompiledIR.ContentHash, "CompiledIR ContentHash must be layout invariant")
		assert.True(t, bytes.Equal(res1.CanonicalJSON, res2.CanonicalJSON), "CanonicalJSON bytes must be layout invariant")
	}
}

func TestRapidPropertyCanonicalizationPermutationInvariance(t *testing.T) {
	c := compiler.New()
	rng := rand.New(rand.NewSource(42))

	for iteration := 0; iteration < 50; iteration++ {
		numNodes := 5 + rng.Intn(5)
		nodes := make([]compiler.XYFlowNode, numNodes)
		for i := 0; i < numNodes; i++ {
			nodes[i] = compiler.XYFlowNode{
				ID:   fmt.Sprintf("node-%02d", i),
				Type: "action",
				Name: fmt.Sprintf("Action Node %d", i),
			}
		}

		edges := make([]compiler.XYFlowEdge, numNodes-1)
		for i := 0; i < numNodes-1; i++ {
			edges[i] = compiler.XYFlowEdge{
				ID:     fmt.Sprintf("edge-%02d", i),
				Source: fmt.Sprintf("node-%02d", i),
				Target: fmt.Sprintf("node-%02d", i+1),
			}
		}

		g1 := &compiler.XYFlowGraph{
			DraftID:  "draft-perm",
			TenantID: "tenant-perm",
			Name:     "Permutation Invariance Test",
			Version:  1,
			Nodes:    nodes,
			Edges:    edges,
		}

		res1, err := c.CanonicalizeGraph(g1)
		require.NoError(t, err)

		// Create g2 with shuffled node and edge slice order
		shuffledNodes := make([]compiler.XYFlowNode, len(nodes))
		copy(shuffledNodes, nodes)
		rng.Shuffle(len(shuffledNodes), func(i, j int) {
			shuffledNodes[i], shuffledNodes[j] = shuffledNodes[j], shuffledNodes[i]
		})

		shuffledEdges := make([]compiler.XYFlowEdge, len(edges))
		copy(shuffledEdges, edges)
		rng.Shuffle(len(shuffledEdges), func(i, j int) {
			shuffledEdges[i], shuffledEdges[j] = shuffledEdges[j], shuffledEdges[i]
		})

		g2 := &compiler.XYFlowGraph{
			DraftID:  g1.DraftID,
			TenantID: g1.TenantID,
			Name:     g1.Name,
			Version:  g1.Version,
			Nodes:    shuffledNodes,
			Edges:    shuffledEdges,
		}

		res2, err := c.CanonicalizeGraph(g2)
		require.NoError(t, err)

		assert.Equal(t, res1.ContentHash, res2.ContentHash)
		assert.Equal(t, res1.CompiledIR.ContentHash, res2.CompiledIR.ContentHash)
		assert.True(t, bytes.Equal(res1.CanonicalJSON, res2.CanonicalJSON))
	}
}

func TestRapidPropertySimulatorStateTransitionsMonotonicity(t *testing.T) {
	tc := compiler.NewTemplateCompiler()
	sim := compiler.NewSimulator(tc)
	rng := rand.New(rand.NewSource(12345))

	for iteration := 0; iteration < 50; iteration++ {
		ir := &domain.CompiledIR{
			SchemaVersion: domain.DefaultSchemaVersion,
			IRID:          fmt.Sprintf("ir-rapid-%d", iteration),
			DraftID:       "draft-rapid",
			TenantID:      "tenant-rapid",
			Version:       1,
			EntryNodeID:   "node-start",
			Nodes: []domain.IRNode{
				{ID: "node-start", Type: "trigger", ActivityName: "StartTrigger"},
				{ID: "node-email", Type: "action", ActivityName: "SendEmail"},
				{ID: "node-end", Type: "exit", ActivityName: "ExitNode"},
			},
			Edges: []domain.IREdge{
				{ID: "e1", SourceID: "node-start", TargetID: "node-email"},
				{ID: "e2", SourceID: "node-email", TargetID: "node-end"},
			},
		}

		opts := &compiler.SimulationOptions{
			EventPayload: map[string]interface{}{
				"user_id": fmt.Sprintf("user-%d", rng.Intn(1000)),
			},
			MaxSteps: 50,
		}

		res1, err := sim.Simulate(ir, opts)
		require.NoError(t, err)
		assert.NotNil(t, res1)
		assert.LessOrEqual(t, len(res1.VisitedNodes), 50, "Visited nodes count must not exceed MaxSteps limit")

		// Verify simulation execution determinism over 5 repeated runs
		for r := 0; r < 5; r++ {
			resN, err := sim.Simulate(ir, opts)
			require.NoError(t, err)
			assert.Equal(t, res1.VisitedNodes, resN.VisitedNodes, "Execution trace must be 100%% deterministic")
		}
	}
}
