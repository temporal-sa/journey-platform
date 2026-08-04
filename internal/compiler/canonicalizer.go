package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/validated-pattern/journey-platform/internal/domain"
)

// Viewport represents UI presentation viewport state (excluded from runtime hash).
type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

// UIDimensions represents node presentation dimensions (excluded from runtime hash).
type UIDimensions struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// XYFlowNode represents a node in the xyflow presentation graph.
type XYFlowNode struct {
	ID                 string                     `json:"id"`
	Type               string                     `json:"type"`
	Name               string                     `json:"name"`
	Config             map[string]interface{}     `json:"config,omitempty"`
	Params             map[string]interface{}     `json:"params,omitempty"`
	Position           *domain.NodePosition       `json:"position,omitempty"`
	Dimensions         *UIDimensions              `json:"dimensions,omitempty"`
	BranchLabels       []string                   `json:"branch_labels,omitempty"`
	CatalogRef         string                     `json:"catalog_ref,omitempty"`
	ExperimentVariants []domain.ExperimentVariant `json:"experiment_variants,omitempty"`
	Metrics            []string                   `json:"metrics,omitempty"`
	TemplateRef        string                     `json:"template_ref,omitempty"`
	TimeoutSeconds     int                        `json:"timeout_seconds,omitempty"`
	RetryPolicy        *domain.RetryPolicy        `json:"retry_policy,omitempty"`
	Selected           bool                       `json:"selected,omitempty"`
	Dragging           bool                       `json:"dragging,omitempty"`
}

// XYFlowEdge represents an edge in the xyflow presentation graph.
type XYFlowEdge struct {
	ID                  string                 `json:"id"`
	Source              string                 `json:"source"`
	Target              string                 `json:"target"`
	SourceHandle        string                 `json:"source_handle,omitempty"`
	TargetHandle        string                 `json:"target_handle,omitempty"`
	Condition           string                 `json:"condition,omitempty"`
	ConditionExpression string                 `json:"condition_expression,omitempty"`
	BranchLabel         string                 `json:"branch_label,omitempty"`
	Style               map[string]interface{} `json:"style,omitempty"`
	Animated            bool                   `json:"animated,omitempty"`
}

// XYFlowGraph represents the raw xyflow presentation graph.
type XYFlowGraph struct {
	DraftID     string       `json:"draft_id"`
	TenantID    string       `json:"tenant_id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Version     int          `json:"version"`
	Viewport    *Viewport    `json:"viewport,omitempty"`
	Nodes       []XYFlowNode `json:"nodes"`
	Edges       []XYFlowEdge `json:"edges"`
}

// NormalizedNode is the layout-free presentation node in canonical representation.
type NormalizedNode struct {
	ID                 string                     `json:"id"`
	Type               string                     `json:"type"`
	Name               string                     `json:"name"`
	Config             map[string]interface{}     `json:"config,omitempty"`
	Params             map[string]interface{}     `json:"params,omitempty"`
	BranchLabels       []string                   `json:"branch_labels,omitempty"`
	CatalogRef         string                     `json:"catalog_ref,omitempty"`
	ExperimentVariants []domain.ExperimentVariant `json:"experiment_variants,omitempty"`
	Metrics            []string                   `json:"metrics,omitempty"`
	TemplateRef        string                     `json:"template_ref,omitempty"`
	TimeoutSeconds     int                        `json:"timeout_seconds,omitempty"`
	RetryPolicy        *domain.RetryPolicy        `json:"retry_policy,omitempty"`
}

// NormalizedEdge is the layout-free presentation edge in canonical representation.
type NormalizedEdge struct {
	ID                  string `json:"id"`
	Source              string `json:"source"`
	Target              string `json:"target"`
	SourceHandle        string `json:"source_handle,omitempty"`
	TargetHandle        string `json:"target_handle,omitempty"`
	ConditionExpression string `json:"condition_expression,omitempty"`
	BranchLabel         string `json:"branch_label,omitempty"`
}

// NormalizedGraph represents the layout-free, deterministically-sorted presentation graph.
type NormalizedGraph struct {
	DraftID     string           `json:"draft_id"`
	TenantID    string           `json:"tenant_id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Version     int              `json:"version"`
	Nodes       []NormalizedNode `json:"nodes"`
	Edges       []NormalizedEdge `json:"edges"`
	ContentHash string           `json:"content_hash"`
}

// CanonicalizationResult holds the output of canonicalizing a presentation graph.
type CanonicalizationResult struct {
	NormalizedGraph *NormalizedGraph   `json:"normalized_graph"`
	CompiledIR      *domain.CompiledIR `json:"compiled_ir"`
	SourceMap       *SourceMap         `json:"source_map"`
	CanonicalJSON   []byte             `json:"canonical_json"`
	ContentHash     string             `json:"content_hash"`
}

// Canonicalizer converts presentation graphs into normalized layout-free graphs and CompiledIR with SourceMaps.
type Canonicalizer struct{}

// NewCanonicalizer creates a new Canonicalizer.
func NewCanonicalizer() *Canonicalizer {
	return &Canonicalizer{}
}

// Canonicalize converts an XYFlowGraph presentation graph into a layout-free NormalizedGraph, CompiledIR, and SourceMap.
func (c *Canonicalizer) Canonicalize(graph *XYFlowGraph) (*CanonicalizationResult, error) {
	if graph == nil {
		return nil, fmt.Errorf("input presentation graph is nil")
	}

	// 1. Normalize layout-free presentation graph
	normGraph, err := c.normalizePresentationGraph(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize presentation graph: %w", err)
	}

	// 2. Compute canonical JSON and hash for normalized graph
	normJSON, normHash, err := c.computeCanonicalJSONAndHash(normGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to compute normalized graph hash: %w", err)
	}
	normGraph.ContentHash = normHash

	// 3. Transform normalized presentation graph into CompiledIR and SourceMap
	ir, sourceMap, err := c.compileToIR(graph, normGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to compile normalized graph to IR: %w", err)
	}

	// 4. Compute hash for CompiledIR
	irHash, err := ComputeIRHash(ir)
	if err != nil {
		return nil, fmt.Errorf("failed to compute CompiledIR hash: %w", err)
	}
	ir.ContentHash = irHash

	return &CanonicalizationResult{
		NormalizedGraph: normGraph,
		CompiledIR:      ir,
		SourceMap:       sourceMap,
		CanonicalJSON:   normJSON,
		ContentHash:     normHash,
	}, nil
}

// CanonicalizeDraft converts a domain.GraphDraft into a CanonicalizationResult.
func (c *Canonicalizer) CanonicalizeDraft(draft *domain.GraphDraft) (*CanonicalizationResult, error) {
	if draft == nil {
		return nil, fmt.Errorf("graph draft is nil")
	}

	xyGraph := &XYFlowGraph{
		DraftID:     draft.DraftID,
		TenantID:    draft.TenantID,
		Name:        draft.Name,
		Description: draft.Description,
		Version:     draft.Version,
		Nodes:       make([]XYFlowNode, len(draft.Nodes)),
		Edges:       make([]XYFlowEdge, len(draft.Edges)),
	}

	for i, n := range draft.Nodes {
		xyGraph.Nodes[i] = XYFlowNode{
			ID:       n.ID,
			Type:     n.Type,
			Name:     n.Name,
			Config:   n.Config,
			Position: n.Position,
		}
	}

	for i, e := range draft.Edges {
		xyGraph.Edges[i] = XYFlowEdge{
			ID:                  e.ID,
			Source:              e.Source,
			Target:              e.Target,
			ConditionExpression: e.Condition,
		}
	}

	return c.Canonicalize(xyGraph)
}

func (c *Canonicalizer) normalizePresentationGraph(graph *XYFlowGraph) (*NormalizedGraph, error) {
	norm := &NormalizedGraph{
		DraftID:     strings.TrimSpace(graph.DraftID),
		TenantID:    strings.TrimSpace(graph.TenantID),
		Name:        strings.TrimSpace(graph.Name),
		Description: strings.TrimSpace(graph.Description),
		Version:     graph.Version,
		Nodes:       make([]NormalizedNode, 0, len(graph.Nodes)),
		Edges:       make([]NormalizedEdge, 0, len(graph.Edges)),
	}

	// 1. Normalize Nodes (exclude position, dimensions, selected, dragging)
	for _, n := range graph.Nodes {
		normNode := NormalizedNode{
			ID:             strings.TrimSpace(n.ID),
			Type:           strings.TrimSpace(n.Type),
			Name:           strings.TrimSpace(n.Name),
			CatalogRef:     strings.TrimSpace(n.CatalogRef),
			TemplateRef:    strings.TrimSpace(n.TemplateRef),
			TimeoutSeconds: n.TimeoutSeconds,
			RetryPolicy:    n.RetryPolicy,
		}

		if n.Config != nil {
			normNode.Config = c.normalizeMap(n.Config)
		}
		if n.Params != nil {
			normNode.Params = c.normalizeMap(n.Params)
		}

		// Sort branch labels
		if len(n.BranchLabels) > 0 {
			labels := make([]string, len(n.BranchLabels))
			copy(labels, n.BranchLabels)
			sort.Strings(labels)
			normNode.BranchLabels = labels
		}

		// Sort metrics
		if len(n.Metrics) > 0 {
			metrics := make([]string, len(n.Metrics))
			copy(metrics, n.Metrics)
			sort.Strings(metrics)
			normNode.Metrics = metrics
		}

		// Sort experiment variants
		if len(n.ExperimentVariants) > 0 {
			variants := make([]domain.ExperimentVariant, len(n.ExperimentVariants))
			for i, v := range n.ExperimentVariants {
				normVar := domain.ExperimentVariant{
					VariantID:         strings.TrimSpace(v.VariantID),
					Name:              strings.TrimSpace(v.Name),
					WeightBasisPoints: v.WeightBasisPoints,
				}
				if v.Config != nil {
					normVar.Config = c.normalizeMap(v.Config)
				}
				variants[i] = normVar
			}
			sort.Slice(variants, func(i, j int) bool {
				if variants[i].VariantID != variants[j].VariantID {
					return variants[i].VariantID < variants[j].VariantID
				}
				return variants[i].Name < variants[j].Name
			})
			normNode.ExperimentVariants = variants
		}

		norm.Nodes = append(norm.Nodes, normNode)
	}

	// Sort nodes deterministically by ID
	sort.Slice(norm.Nodes, func(i, j int) bool {
		if norm.Nodes[i].ID != norm.Nodes[j].ID {
			return norm.Nodes[i].ID < norm.Nodes[j].ID
		}
		if norm.Nodes[i].Type != norm.Nodes[j].Type {
			return norm.Nodes[i].Type < norm.Nodes[j].Type
		}
		return norm.Nodes[i].Name < norm.Nodes[j].Name
	})

	// 2. Normalize Edges (exclude style, animated)
	for _, e := range graph.Edges {
		condExpr := strings.TrimSpace(e.ConditionExpression)
		if condExpr == "" {
			condExpr = strings.TrimSpace(e.Condition)
		}
		if condExpr == "" {
			condExpr = strings.TrimSpace(e.BranchLabel)
		}

		normEdge := NormalizedEdge{
			ID:                  strings.TrimSpace(e.ID),
			Source:              strings.TrimSpace(e.Source),
			Target:              strings.TrimSpace(e.Target),
			SourceHandle:        strings.TrimSpace(e.SourceHandle),
			TargetHandle:        strings.TrimSpace(e.TargetHandle),
			ConditionExpression: condExpr,
			BranchLabel:         strings.TrimSpace(e.BranchLabel),
		}

		norm.Edges = append(norm.Edges, normEdge)
	}

	// Sort edges deterministically by ID, Source, Target
	sort.Slice(norm.Edges, func(i, j int) bool {
		if norm.Edges[i].ID != norm.Edges[j].ID {
			return norm.Edges[i].ID < norm.Edges[j].ID
		}
		if norm.Edges[i].Source != norm.Edges[j].Source {
			return norm.Edges[i].Source < norm.Edges[j].Source
		}
		if norm.Edges[i].Target != norm.Edges[j].Target {
			return norm.Edges[i].Target < norm.Edges[j].Target
		}
		return norm.Edges[i].ConditionExpression < norm.Edges[j].ConditionExpression
	})

	return norm, nil
}

func (c *Canonicalizer) compileToIR(raw *XYFlowGraph, norm *NormalizedGraph) (*domain.CompiledIR, *SourceMap, error) {
	irID := fmt.Sprintf("ir-%s", norm.DraftID)

	irNodes := make([]domain.IRNode, 0, len(norm.Nodes))
	nodeMappings := make(map[string]NodeFieldMapping, len(norm.Nodes))

	entryNodeID := ""

	for _, n := range norm.Nodes {
		irNodeID := n.ID
		if !strings.HasPrefix(irNodeID, "ir-") {
			irNodeID = "ir-" + n.ID
		}

		// Identify entry node if trigger
		if entryNodeID == "" && (n.Type == "trigger" || strings.Contains(strings.ToLower(n.Type), "start")) {
			entryNodeID = irNodeID
		}

		// Consolidate parameters
		params := make(map[string]interface{})
		for k, v := range n.Config {
			params[k] = v
		}
		for k, v := range n.Params {
			params[k] = v
		}
		if n.CatalogRef != "" {
			params["catalog_ref"] = n.CatalogRef
		}
		if n.TemplateRef != "" {
			params["template_ref"] = n.TemplateRef
		}
		if len(n.BranchLabels) > 0 {
			params["branch_labels"] = n.BranchLabels
		}
		if len(n.Metrics) > 0 {
			params["metrics"] = n.Metrics
		}
		if len(n.ExperimentVariants) > 0 {
			vars := make([]interface{}, len(n.ExperimentVariants))
			for i, v := range n.ExperimentVariants {
				varMap := map[string]interface{}{
					"variant_id":          v.VariantID,
					"name":                v.Name,
					"weight_basis_points": float64(v.WeightBasisPoints),
				}
				if len(v.Config) > 0 {
					varMap["config"] = c.normalizeMap(v.Config)
				}
				vars[i] = varMap
			}
			params["experiment_variants"] = vars
		}

		actName := n.Name
		if actName == "" {
			actName = n.Type
		}

		irNode := domain.IRNode{
			ID:             irNodeID,
			Type:           n.Type,
			ActivityName:   actName,
			Params:         c.normalizeMap(params),
			TimeoutSeconds: n.TimeoutSeconds,
			RetryPolicy:    n.RetryPolicy,
		}
		irNodes = append(irNodes, irNode)

		nodeMappings[irNodeID] = NodeFieldMapping{
			UINodeID: n.ID,
			Fields: map[string]string{
				"activity_name": "name",
				"type":          "type",
				"params":        "config",
			},
		}
	}

	// Fallback entry node ID if no trigger was explicitly identified
	if entryNodeID == "" && len(irNodes) > 0 {
		entryNodeID = irNodes[0].ID
	}

	// Sort IR nodes deterministically by ID
	sort.Slice(irNodes, func(i, j int) bool {
		return irNodes[i].ID < irNodes[j].ID
	})

	irEdges := make([]domain.IREdge, 0, len(norm.Edges))
	edgeMappings := make(map[string]string, len(norm.Edges))

	for _, e := range norm.Edges {
		irEdgeID := e.ID
		if !strings.HasPrefix(irEdgeID, "ir-") {
			irEdgeID = "ir-" + e.ID
		}

		srcID := e.Source
		if !strings.HasPrefix(srcID, "ir-") {
			srcID = "ir-" + e.Source
		}
		tgtID := e.Target
		if !strings.HasPrefix(tgtID, "ir-") {
			tgtID = "ir-" + e.Target
		}

		irEdge := domain.IREdge{
			ID:                  irEdgeID,
			SourceID:            srcID,
			TargetID:            tgtID,
			ConditionExpression: e.ConditionExpression,
		}
		irEdges = append(irEdges, irEdge)
		edgeMappings[irEdgeID] = e.ID
	}

	// Sort IR edges deterministically by ID
	sort.Slice(irEdges, func(i, j int) bool {
		if irEdges[i].ID != irEdges[j].ID {
			return irEdges[i].ID < irEdges[j].ID
		}
		if irEdges[i].SourceID != irEdges[j].SourceID {
			return irEdges[i].SourceID < irEdges[j].SourceID
		}
		return irEdges[i].TargetID < irEdges[j].TargetID
	})

	compiledIR := &domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          irID,
		DraftID:       norm.DraftID,
		TenantID:      norm.TenantID,
		Version:       norm.Version,
		EntryNodeID:   entryNodeID,
		Nodes:         irNodes,
		Edges:         irEdges,
	}

	sourceMap := &SourceMap{
		Version: domain.DefaultSchemaVersion,
		Nodes:   nodeMappings,
		Edges:   edgeMappings,
	}

	return compiledIR, sourceMap, nil
}

func (c *Canonicalizer) normalizeMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		normKey := strings.TrimSpace(k)
		result[normKey] = c.normalizeValue(v)
	}
	return result
}

func (c *Canonicalizer) normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return c.normalizeMap(val)
	case []interface{}:
		items := make([]interface{}, len(val))
		for i, elem := range val {
			items[i] = c.normalizeValue(elem)
		}
		return items
	case string:
		return strings.TrimSpace(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return val
	}
}

func (c *Canonicalizer) computeCanonicalJSONAndHash(v interface{}) ([]byte, string, error) {
	bytes, err := EncodeCanonicalJSON(v)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal struct for hash calculation: %w", err)
	}
	hash := sha256.Sum256(bytes)
	return bytes, hex.EncodeToString(hash[:]), nil
}
