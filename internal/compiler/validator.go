package compiler

import (
	"fmt"
	"sort"
	"strings"

	"github.com/validated-pattern/journey-platform/internal/compiler/expression"
	"github.com/validated-pattern/journey-platform/internal/domain"
)

// Machine code constants for validation issues.
const (
	CodeNoStartNode              = "ERR_NO_START_NODE"
	CodeMultipleStartNodes       = "ERR_MULTIPLE_START_NODES"
	CodeNoReachableExit          = "ERR_NO_REACHABLE_EXIT"
	CodeCycleDetected            = "ERR_CYCLE_DETECTED"
	CodeUnreachableNode          = "ERR_UNREACHABLE_NODE"
	CodeInvalidNodeType          = "ERR_INVALID_NODE_TYPE"
	CodeStartNodeIncomingEdge    = "ERR_START_NODE_INCOMING_EDGE"
	CodeExitNodeOutgoingEdge     = "ERR_EXIT_NODE_OUTGOING_EDGE"
	CodeMaxOutgoingEdgesExceeded = "ERR_MAX_OUTGOING_EDGES_EXCEEDED"
	CodeDuplicateEdge            = "ERR_DUPLICATE_EDGE"
	CodeMissingDefaultBranch     = "ERR_MISSING_DEFAULT_BRANCH"
	CodeDuplicateBranchLabel     = "ERR_DUPLICATE_BRANCH_LABEL"
	CodeInvalidBranchLabel       = "ERR_INVALID_BRANCH_LABEL"
	CodeGraphSizeExceeded        = "ERR_GRAPH_SIZE_EXCEEDED"
	CodeMissingNodeID            = "ERR_MISSING_NODE_ID"
	CodeMissingNodeType          = "ERR_MISSING_NODE_TYPE"
	CodeDuplicateNodeID          = "ERR_DUPLICATE_NODE_ID"
	CodeInvalidEdgeNodeRef       = "ERR_INVALID_EDGE_NODE_REF"
	CodeInvalidDelay             = "ERR_INVALID_DELAY"
	CodeTotalDurationExceeded    = "ERR_TOTAL_DURATION_EXCEEDED"
	CodeInvalidCatalogRef        = "ERR_INVALID_CATALOG_REF"
	CodeDeprecatedCatalogRef     = "WARN_DEPRECATED_CATALOG_REF"
	CodeExpressionSyntax         = "ERR_EXPRESSION_SYNTAX"
	CodeForwardReference         = "ERR_FORWARD_REFERENCE"
	CodeTypeMismatch             = "ERR_TYPE_MISMATCH"
)

// ValidationResult represents the output of graph validation.
type ValidationResult struct {
	DraftID string                   `json:"draft_id"`
	IsValid bool                     `json:"is_valid"`
	Issues  []domain.ValidationIssue `json:"issues"`
}

// ValidationNode represents a normalized graph node for validation.
type ValidationNode struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Params       map[string]interface{} `json:"params,omitempty"`
	BranchLabels []string               `json:"branch_labels,omitempty"`
	CatalogRef   string                 `json:"catalog_ref,omitempty"`
}

// ValidationEdge represents a normalized graph edge for validation.
type ValidationEdge struct {
	ID                  string `json:"id"`
	Source              string `json:"source"`
	Target              string `json:"target"`
	ConditionExpression string `json:"condition_expression,omitempty"`
	BranchLabel         string `json:"branch_label,omitempty"`
}

// ValidationGraph represents a normalized graph structure for validation.
type ValidationGraph struct {
	DraftID string           `json:"draft_id"`
	Nodes   []ValidationNode `json:"nodes"`
	Edges   []ValidationEdge `json:"edges"`
}

// ValidationConfig configures validator constraints and options.
type ValidationConfig struct {
	MaxNodes                int
	MaxEdges                int
	MaxSingleDelaySeconds   int64
	MaxTotalDurationSeconds int64
	CatalogRegistry         map[string]domain.CatalogRecord
	RequireCatalogRef       bool
	EnforceStrictTypes      bool
}

// DefaultValidationConfig returns default validation bounds and settings.
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxNodes:                500,
		MaxEdges:                1000,
		MaxSingleDelaySeconds:   30 * 24 * 3600, // 30 days = 2,592,000s
		MaxTotalDurationSeconds: 90 * 24 * 3600, // 90 days = 7,776,000s
		RequireCatalogRef:       false,
		EnforceStrictTypes:      false,
	}
}

// ValidationOption defines a functional option for configuring Validator.
type ValidationOption func(*ValidationConfig)

func WithMaxNodes(maxNodes int) ValidationOption {
	return func(c *ValidationConfig) { c.MaxNodes = maxNodes }
}

func WithMaxEdges(maxEdges int) ValidationOption {
	return func(c *ValidationConfig) { c.MaxEdges = maxEdges }
}

func WithMaxSingleDelay(seconds int64) ValidationOption {
	return func(c *ValidationConfig) { c.MaxSingleDelaySeconds = seconds }
}

func WithMaxTotalDuration(seconds int64) ValidationOption {
	return func(c *ValidationConfig) { c.MaxTotalDurationSeconds = seconds }
}

func WithCatalogRegistry(catalog map[string]domain.CatalogRecord) ValidationOption {
	return func(c *ValidationConfig) { c.CatalogRegistry = catalog }
}

func WithRequireCatalogRef(require bool) ValidationOption {
	return func(c *ValidationConfig) { c.RequireCatalogRef = require }
}

func WithEnforceStrictTypes(enforce bool) ValidationOption {
	return func(c *ValidationConfig) { c.EnforceStrictTypes = enforce }
}

// Validator performs structural, type, and reference validation on workflow graphs.
type Validator struct {
	config ValidationConfig
}

// NewValidator creates a new Validator with optional functional options.
func NewValidator(opts ...ValidationOption) *Validator {
	cfg := DefaultValidationConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &Validator{config: cfg}
}

// ValidateDraft validates a domain.GraphDraft.
func (v *Validator) ValidateDraft(draft *domain.GraphDraft) (*ValidationResult, error) {
	if draft == nil {
		return &ValidationResult{
			IsValid: false,
			Issues: []domain.ValidationIssue{
				createIssue(1, CodeGraphSizeExceeded, domain.ValidationSeverityError, "", "graph", "Graph draft is nil. Suggested fix: Provide a non-nil graph draft."),
			},
		}, nil
	}
	vg := convertDraftToValidationGraph(draft)
	return v.ValidateValidationGraph(vg)
}

// ValidateXYFlowGraph validates an XYFlowGraph presentation graph.
func (v *Validator) ValidateXYFlowGraph(graph *XYFlowGraph) (*ValidationResult, error) {
	if graph == nil {
		return &ValidationResult{
			IsValid: false,
			Issues: []domain.ValidationIssue{
				createIssue(1, CodeGraphSizeExceeded, domain.ValidationSeverityError, "", "graph", "Presentation graph is nil. Suggested fix: Provide a non-nil graph."),
			},
		}, nil
	}
	vg := convertXYFlowToValidationGraph(graph)
	return v.ValidateValidationGraph(vg)
}

// ValidateGraph accepts *domain.GraphDraft, *XYFlowGraph, or *ValidationGraph.
func ValidateGraph(input interface{}, opts ...ValidationOption) (*ValidationResult, error) {
	val := NewValidator(opts...)
	if input == nil {
		return &ValidationResult{
			IsValid: false,
			Issues: []domain.ValidationIssue{
				createIssue(1, CodeGraphSizeExceeded, domain.ValidationSeverityError, "", "graph", "Input graph is nil. Suggested fix: Provide a valid graph instance."),
			},
		}, nil
	}
	switch g := input.(type) {
	case *domain.GraphDraft:
		return val.ValidateDraft(g)
	case *XYFlowGraph:
		return val.ValidateXYFlowGraph(g)
	case *ValidationGraph:
		return val.ValidateValidationGraph(g)
	default:
		return nil, fmt.Errorf("unsupported graph input type %T", input)
	}
}

// ValidateValidationGraph performs comprehensive validation on a ValidationGraph.
func (v *Validator) ValidateValidationGraph(graph *ValidationGraph) (*ValidationResult, error) {
	if graph == nil {
		return &ValidationResult{
			IsValid: false,
			Issues: []domain.ValidationIssue{
				createIssue(1, CodeGraphSizeExceeded, domain.ValidationSeverityError, "", "graph", "Validation graph is nil. Suggested fix: Provide a valid non-nil graph."),
			},
		}, nil
	}

	var issues []domain.ValidationIssue
	issueCounter := 0
	addIssue := func(code string, severity domain.ValidationSeverity, nodeID, fieldPath, message string) {
		issueCounter++
		issues = append(issues, createIssue(issueCounter, code, severity, nodeID, fieldPath, message))
	}

	// 1. Graph Size Bounds
	if len(graph.Nodes) > v.config.MaxNodes || len(graph.Edges) > v.config.MaxEdges {
		addIssue(CodeGraphSizeExceeded, domain.ValidationSeverityError, "", "nodes",
			fmt.Sprintf("Graph size (%d nodes, %d edges) exceeds allowable limits (%d nodes, %d edges). Suggested fix: Reduce graph size.",
				len(graph.Nodes), len(graph.Edges), v.config.MaxNodes, v.config.MaxEdges))
	}

	// 2. Node ID & Type Validity, Duplicates
	nodeMap := make(map[string]*ValidationNode)
	nodeList := make([]ValidationNode, 0, len(graph.Nodes))
	var startNodes []*ValidationNode
	var exitNodes []*ValidationNode

	for i, n := range graph.Nodes {
		fieldPathPrefix := fmt.Sprintf("nodes[%d]", i)
		if strings.TrimSpace(n.ID) == "" {
			addIssue(CodeMissingNodeID, domain.ValidationSeverityError, "", fieldPathPrefix+".id",
				"Node ID is missing or blank. Suggested fix: Provide a non-empty unique string ID for every node.")
			continue
		}
		if _, exists := nodeMap[n.ID]; exists {
			addIssue(CodeDuplicateNodeID, domain.ValidationSeverityError, n.ID, fieldPathPrefix+".id",
				fmt.Sprintf("Duplicate node ID '%s' found. Suggested fix: Ensure node IDs are unique across the graph.", n.ID))
			continue
		}
		if strings.TrimSpace(n.Type) == "" {
			addIssue(CodeMissingNodeType, domain.ValidationSeverityError, n.ID, fieldPathPrefix+".type",
				fmt.Sprintf("Node '%s' is missing a node type. Suggested fix: Set a valid node type (e.g., trigger, action, condition, delay, exit).", n.ID))
		} else if !isAllowedNodeType(n.Type) {
			addIssue(CodeInvalidNodeType, domain.ValidationSeverityError, n.ID, fieldPathPrefix+".type",
				fmt.Sprintf("Node '%s' has unsupported node type '%s'. Allowed types: trigger, action, condition, delay, experiment, exit. Suggested fix: Use a supported node type.", n.ID, n.Type))
		}

		nodeCopy := n
		nodeMap[n.ID] = &nodeCopy
		nodeList = append(nodeList, n)

		if isStartKind(n.Type) {
			startNodes = append(startNodes, &nodeCopy)
		}
		if isExitKind(n.Type) {
			exitNodes = append(exitNodes, &nodeCopy)
		}
	}

	// 3. Start & Exit Node Counts
	if len(startNodes) == 0 {
		addIssue(CodeNoStartNode, domain.ValidationSeverityError, "", "nodes",
			"Graph has no start node. Exactly one start node (type: trigger, start, entry) is required. Suggested fix: Add a trigger node to serve as workflow entry point.")
	} else if len(startNodes) > 1 {
		for _, sn := range startNodes[1:] {
			addIssue(CodeMultipleStartNodes, domain.ValidationSeverityError, sn.ID, "nodes",
				fmt.Sprintf("Graph contains multiple start nodes ('%s'). Exactly one start node is allowed. Suggested fix: Retain only one trigger/start node.", sn.ID))
		}
	}

	// 4. Edge Validity & Node References
	incomingEdges := make(map[string][]ValidationEdge)
	outgoingEdges := make(map[string][]ValidationEdge)
	seenEdges := make(map[string]bool)

	for i, e := range graph.Edges {
		fieldPath := fmt.Sprintf("edges[%d]", i)
		if e.Source == "" || e.Target == "" {
			addIssue(CodeInvalidEdgeNodeRef, domain.ValidationSeverityError, e.Source, fieldPath,
				"Edge has empty source or target node ID. Suggested fix: Set valid source and target node IDs.")
			continue
		}

		_, srcOk := nodeMap[e.Source]
		_, tgtOk := nodeMap[e.Target]
		if !srcOk {
			addIssue(CodeInvalidEdgeNodeRef, domain.ValidationSeverityError, e.Source, fieldPath+".source",
				fmt.Sprintf("Edge '%s' references non-existent source node '%s'. Suggested fix: Update edge source to an existing node ID.", e.ID, e.Source))
		}
		if !tgtOk {
			addIssue(CodeInvalidEdgeNodeRef, domain.ValidationSeverityError, e.Target, fieldPath+".target",
				fmt.Sprintf("Edge '%s' references non-existent target node '%s'. Suggested fix: Update edge target to an existing node ID.", e.ID, e.Target))
		}

		if !srcOk || !tgtOk {
			continue
		}

		// Duplicate Edge Check
		edgeKey := fmt.Sprintf("%s->%s:%s", e.Source, e.Target, e.BranchLabel)
		if seenEdges[edgeKey] {
			addIssue(CodeDuplicateEdge, domain.ValidationSeverityError, e.Source, fieldPath,
				fmt.Sprintf("Duplicate edge from '%s' to '%s' (label: '%s'). Suggested fix: Remove duplicate edge.", e.Source, e.Target, e.BranchLabel))
		}
		seenEdges[edgeKey] = true

		outgoingEdges[e.Source] = append(outgoingEdges[e.Source], e)
		incomingEdges[e.Target] = append(incomingEdges[e.Target], e)
	}

	// Dynamic Exit Node identification: nodes with 0 outgoing edges act as exits if no explicit exit nodes
	if len(exitNodes) == 0 {
		for _, n := range nodeList {
			if len(outgoingEdges[n.ID]) == 0 && !isStartKind(n.Type) {
				exitCopy := n
				exitNodes = append(exitNodes, &exitCopy)
			}
		}
	}

	// 5. Source / Target Edge Restrictions & Outgoing Edge Limits
	for _, n := range nodeList {
		outCount := len(outgoingEdges[n.ID])
		inCount := len(incomingEdges[n.ID])

		if isStartKind(n.Type) && inCount > 0 {
			addIssue(CodeStartNodeIncomingEdge, domain.ValidationSeverityError, n.ID, "edges",
				fmt.Sprintf("Start node '%s' cannot have incoming edges (found %d). Suggested fix: Remove incoming edges to the start node.", n.ID, inCount))
		}
		if isExitKind(n.Type) && outCount > 0 {
			addIssue(CodeExitNodeOutgoingEdge, domain.ValidationSeverityError, n.ID, "edges",
				fmt.Sprintf("Exit node '%s' cannot have outgoing edges (found %d). Suggested fix: Remove outgoing edges from the exit node.", n.ID, outCount))
		}

		maxOut := getMaxOutgoingEdges(n)
		if outCount > maxOut {
			addIssue(CodeMaxOutgoingEdgesExceeded, domain.ValidationSeverityError, n.ID, "edges",
				fmt.Sprintf("Node '%s' of type '%s' has %d outgoing edges, exceeding maximum allowed (%d). Suggested fix: Remove excess outgoing edges.", n.ID, n.Type, outCount, maxOut))
		}
	}

	// 6. Branch Labels & Missing Default Branch Validation
	for _, n := range nodeList {
		outList := outgoingEdges[n.ID]
		if len(outList) == 0 {
			continue
		}

		declaredLabels := extractBranchLabels(n)
		if len(declaredLabels) > 0 {
			labelSet := make(map[string]bool)
			for _, l := range declaredLabels {
				labelSet[l] = true
			}
			seenOutLabels := make(map[string]bool)
			for _, e := range outList {
				if e.BranchLabel != "" {
					if !labelSet[e.BranchLabel] {
						addIssue(CodeInvalidBranchLabel, domain.ValidationSeverityError, n.ID, "edges",
							fmt.Sprintf("Outgoing edge from node '%s' uses undeclared branch label '%s'. Declared: %v. Suggested fix: Use a declared branch label or update node branch labels.", n.ID, e.BranchLabel, declaredLabels))
					}
					if seenOutLabels[e.BranchLabel] {
						addIssue(CodeDuplicateBranchLabel, domain.ValidationSeverityError, n.ID, "edges",
							fmt.Sprintf("Node '%s' has duplicate outgoing branch label '%s'. Suggested fix: Ensure each outgoing branch label is unique.", n.ID, e.BranchLabel))
					}
					seenOutLabels[e.BranchLabel] = true
				}
			}
		}

		// Missing default branch check for branching nodes
		if isBranchingKind(n.Type) || len(outList) > 1 {
			hasDefault := false
			for _, e := range outList {
				lbl := strings.ToLower(strings.TrimSpace(e.BranchLabel))
				cond := strings.TrimSpace(e.ConditionExpression)
				if lbl == "default" || lbl == "else" || cond == "" || cond == "true" {
					hasDefault = true
					break
				}
			}
			if !hasDefault {
				addIssue(CodeMissingDefaultBranch, domain.ValidationSeverityError, n.ID, "edges",
					fmt.Sprintf("Branching node '%s' is missing a default or fallthrough branch. Suggested fix: Add a default branch edge with empty condition or label 'default'.", n.ID))
			}
		}
	}

	// 7. Graph Reachability & Cycle Detection
	var primaryStart *ValidationNode
	if len(startNodes) > 0 {
		primaryStart = startNodes[0]
	}

	reachable := make(map[string]bool)
	if primaryStart != nil {
		queue := []string{primaryStart.ID}
		reachable[primaryStart.ID] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			for _, e := range outgoingEdges[curr] {
				if !reachable[e.Target] {
					reachable[e.Target] = true
					queue = append(queue, e.Target)
				}
			}
		}
	}

	// Reachable Exit Check
	if primaryStart != nil {
		hasReachableExit := false
		for _, ex := range exitNodes {
			if reachable[ex.ID] {
				hasReachableExit = true
				break
			}
		}
		if !hasReachableExit {
			addIssue(CodeNoReachableExit, domain.ValidationSeverityError, primaryStart.ID, "nodes",
				fmt.Sprintf("No exit node is reachable from start node '%s'. Suggested fix: Connect workflow path from start node to an exit node.", primaryStart.ID))
		}
	}

	// Unreachable Nodes Check
	if primaryStart != nil {
		for i, n := range nodeList {
			if !reachable[n.ID] {
				addIssue(CodeUnreachableNode, domain.ValidationSeverityError, n.ID, fmt.Sprintf("nodes[%d]", i),
					fmt.Sprintf("Node '%s' is unreachable from the start node '%s'. Suggested fix: Connect node '%s' to the workflow graph or remove it.", n.ID, primaryStart.ID, n.ID))
			}
		}
	}

	// Cycle Detection (Color DFS)
	color := make(map[string]int) // 0: White, 1: Gray (Visiting), 2: Black (Visited)
	var cycleFound bool
	var dfsCycle func(curr string, stack []string)
	dfsCycle = func(curr string, stack []string) {
		color[curr] = 1
		stack = append(stack, curr)
		for _, e := range outgoingEdges[curr] {
			nxt := e.Target
			if color[nxt] == 1 {
				cycleFound = true
				addIssue(CodeCycleDetected, domain.ValidationSeverityError, curr, "edges",
					fmt.Sprintf("Graph contains a directed cycle involving node '%s' -> '%s'. Workflows must be acyclic DAGs. Suggested fix: Break loop by removing edge targeting '%s'.", curr, nxt, nxt))
			} else if color[nxt] == 0 {
				dfsCycle(nxt, stack)
			}
		}
		color[curr] = 2
	}

	for _, n := range nodeList {
		if color[n.ID] == 0 {
			dfsCycle(n.ID, nil)
		}
	}

	// 8. Node Config & Delay Bounds & Catalog References
	for i, n := range nodeList {
		fieldPrefix := fmt.Sprintf("nodes[%d]", i)

		// Delay Node Validation
		if isDelayKind(n.Type) || hasDelayConfig(n) {
			delaySec, ok := extractValidationDelaySeconds(n)
			if !ok || delaySec <= 0 {
				addIssue(CodeInvalidDelay, domain.ValidationSeverityError, n.ID, fieldPrefix+".config",
					fmt.Sprintf("Node '%s' has invalid delay duration (%d seconds). Delay must be > 0. Suggested fix: Specify positive delay seconds in node config.", n.ID, delaySec))
			} else if delaySec > v.config.MaxSingleDelaySeconds {
				addIssue(CodeInvalidDelay, domain.ValidationSeverityError, n.ID, fieldPrefix+".config",
					fmt.Sprintf("Delay duration for node '%s' (%d seconds) exceeds maximum single delay bound (%d seconds / 30 days). Suggested fix: Reduce delay duration.", n.ID, delaySec, v.config.MaxSingleDelaySeconds))
			}
		}

		// Catalog Reference Validation
		catRef := extractCatalogRef(n)
		if isActionKind(n.Type) && catRef == "" && v.config.RequireCatalogRef {
			addIssue(CodeInvalidCatalogRef, domain.ValidationSeverityError, n.ID, fieldPrefix+".catalog_ref",
				fmt.Sprintf("Action node '%s' is missing a required catalog reference. Suggested fix: Provide a valid catalog_ref.", n.ID))
		} else if catRef != "" {
			if v.config.CatalogRegistry != nil && len(v.config.CatalogRegistry) > 0 {
				record, exists := v.config.CatalogRegistry[catRef]
				if !exists {
					addIssue(CodeInvalidCatalogRef, domain.ValidationSeverityError, n.ID, fieldPrefix+".catalog_ref",
						fmt.Sprintf("Catalog reference '%s' on node '%s' not found in catalog. Suggested fix: Reference an existing catalog item ID.", catRef, n.ID))
				} else if record.IsDeprecated {
					addIssue(CodeDeprecatedCatalogRef, domain.ValidationSeverityWarning, n.ID, fieldPrefix+".catalog_ref",
						fmt.Sprintf("Catalog reference '%s' on node '%s' is deprecated. Suggested fix: Upgrade catalog_ref to a non-deprecated version.", catRef, n.ID))
				}
			}
		}
	}

	// 9. Total Journey Duration Bounds Along Paths
	if primaryStart != nil && !cycleFound {
		maxTotalDelay := computeMaxPathDelay(primaryStart.ID, nodeList, outgoingEdges)
		if maxTotalDelay > v.config.MaxTotalDurationSeconds {
			addIssue(CodeTotalDurationExceeded, domain.ValidationSeverityError, primaryStart.ID, "nodes",
				fmt.Sprintf("Total journey duration along path (%d seconds) exceeds maximum journey bound (%d seconds / 90 days). Suggested fix: Reduce delay values along the workflow execution path.", maxTotalDelay, v.config.MaxTotalDurationSeconds))
		}
	}

	// 10. Ancestor Computation & Expression Forward-Reference & Type Checks
	if !cycleFound {
		ancestorsMap := computeAncestorsMap(nodeList, incomingEdges)
		for _, e := range graph.Edges {
			cond := strings.TrimSpace(e.ConditionExpression)
			if cond == "" {
				continue
			}

			ast, err := expression.Parse(cond)
			if err != nil {
				addIssue(CodeExpressionSyntax, domain.ValidationSeverityError, e.Source, "edges.condition",
					fmt.Sprintf("Invalid condition expression syntax '%s' on edge from '%s': %v. Suggested fix: Fix expression syntax.", cond, e.Source, err))
				continue
			}

			availableNodes := append(ancestorsMap[e.Source], e.Source)

			opts := expression.ValidationOptions{
				AvailableNodeIDs:   availableNodes,
				EnforceStrictTypes: v.config.EnforceStrictTypes,
			}

			if err := expression.Validate(ast, opts); err != nil {
				errMsg := err.Error()
				if strings.Contains(errMsg, "forward reference") {
					addIssue(CodeForwardReference, domain.ValidationSeverityError, e.Source, "edges.condition",
						fmt.Sprintf("Forward reference in expression '%s' on edge from '%s': %v. Suggested fix: Only reference outputs from preceding upstream nodes.", cond, e.Source, err))
				} else {
					addIssue(CodeTypeMismatch, domain.ValidationSeverityError, e.Source, "edges.condition",
						fmt.Sprintf("Type mismatch in expression '%s' on edge from '%s': %v. Suggested fix: Ensure expression types are compatible.", cond, e.Source, err))
				}
			}
		}
	}

	isValid := true
	for _, iss := range issues {
		if iss.Severity == domain.ValidationSeverityError {
			isValid = false
			break
		}
	}

	return &ValidationResult{
		DraftID: graph.DraftID,
		IsValid: isValid,
		Issues:  issues,
	}, nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

func createIssue(id int, code string, severity domain.ValidationSeverity, nodeID, fieldPath, message string) domain.ValidationIssue {
	return domain.ValidationIssue{
		SchemaVersion: domain.DefaultSchemaVersion,
		IssueID:       fmt.Sprintf("val-issue-%d", id),
		NodeID:        nodeID,
		Severity:      severity,
		Code:          code,
		Message:       message,
		FieldPath:     fieldPath,
	}
}

func isStartKind(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return t == "trigger" || t == "start" || t == "entry"
}

func isExitKind(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return t == "exit" || t == "end" || t == "outcome"
}

func isDelayKind(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return t == "delay" || t == "wait"
}

func isActionKind(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return t == "action" || t == "activity"
}

func isBranchingKind(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	return t == "condition" || t == "split" || t == "branch" || t == "experiment" || t == "ab_test"
}

func isAllowedNodeType(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	allowed := map[string]bool{
		"trigger":    true,
		"start":      true,
		"entry":      true,
		"action":     true,
		"activity":   true,
		"condition":  true,
		"split":      true,
		"branch":     true,
		"delay":      true,
		"wait":       true,
		"experiment": true,
		"ab_test":    true,
		"exit":       true,
		"end":        true,
		"outcome":    true,
	}
	return allowed[t]
}

func getMaxOutgoingEdges(n ValidationNode) int {
	if isExitKind(n.Type) {
		return 0
	}
	labels := extractBranchLabels(n)
	if len(labels) > 0 {
		return len(labels)
	}
	if isStartKind(n.Type) || isDelayKind(n.Type) {
		return 1
	}
	if isActionKind(n.Type) {
		return 1
	}
	if isBranchingKind(n.Type) {
		return 100
	}
	return 1
}

func extractBranchLabels(n ValidationNode) []string {
	if len(n.BranchLabels) > 0 {
		return n.BranchLabels
	}
	return extractBranchLabelsFromConfig(n.Config)
}

func extractCatalogRef(n ValidationNode) string {
	if n.CatalogRef != "" {
		return n.CatalogRef
	}
	return extractCatalogRefFromConfig(n.Config)
}

func hasDelayConfig(n ValidationNode) bool {
	if n.Config == nil {
		return false
	}
	keys := []string{"delay_seconds", "duration_seconds", "delay", "duration", "delay_ms"}
	for _, k := range keys {
		if _, ok := n.Config[k]; ok {
			return true
		}
	}
	return false
}

func extractValidationDelaySeconds(n ValidationNode) (int64, bool) {
	if n.Config == nil {
		return 0, false
	}
	sec := extractDelaySeconds(n.Config)
	if sec > 0 {
		return sec, true
	}
	return 0, false
}

func convertDraftToValidationGraph(draft *domain.GraphDraft) *ValidationGraph {
	vg := &ValidationGraph{
		DraftID: draft.DraftID,
		Nodes:   make([]ValidationNode, len(draft.Nodes)),
		Edges:   make([]ValidationEdge, len(draft.Edges)),
	}
	for i, n := range draft.Nodes {
		vg.Nodes[i] = ValidationNode{
			ID:           n.ID,
			Type:         n.Type,
			Name:         n.Name,
			Config:       n.Config,
			BranchLabels: extractBranchLabelsFromConfig(n.Config),
			CatalogRef:   extractCatalogRefFromConfig(n.Config),
		}
	}
	for i, e := range draft.Edges {
		vg.Edges[i] = ValidationEdge{
			ID:                  e.ID,
			Source:              e.Source,
			Target:              e.Target,
			ConditionExpression: e.Condition,
		}
	}
	return vg
}

func convertXYFlowToValidationGraph(graph *XYFlowGraph) *ValidationGraph {
	vg := &ValidationGraph{
		DraftID: graph.DraftID,
		Nodes:   make([]ValidationNode, len(graph.Nodes)),
		Edges:   make([]ValidationEdge, len(graph.Edges)),
	}
	for i, n := range graph.Nodes {
		vg.Nodes[i] = ValidationNode{
			ID:           n.ID,
			Type:         n.Type,
			Name:         n.Name,
			Config:       n.Config,
			Params:       n.Params,
			BranchLabels: n.BranchLabels,
			CatalogRef:   n.CatalogRef,
		}
	}
	for i, e := range graph.Edges {
		cond := e.ConditionExpression
		if cond == "" {
			cond = e.Condition
		}
		vg.Edges[i] = ValidationEdge{
			ID:                  e.ID,
			Source:              e.Source,
			Target:              e.Target,
			ConditionExpression: cond,
			BranchLabel:         e.BranchLabel,
		}
	}
	return vg
}

func extractBranchLabelsFromConfig(cfg map[string]interface{}) []string {
	if cfg == nil {
		return nil
	}
	if raw, ok := cfg["branch_labels"]; ok {
		if strSlice, ok := raw.([]string); ok {
			return strSlice
		}
		if intfSlice, ok := raw.([]interface{}); ok {
			var res []string
			for _, item := range intfSlice {
				if s, ok := item.(string); ok {
					res = append(res, s)
				}
			}
			return res
		}
	}
	return nil
}

func extractCatalogRefFromConfig(cfg map[string]interface{}) string {
	if cfg == nil {
		return ""
	}
	if s, ok := cfg["catalog_ref"].(string); ok {
		return s
	}
	if s, ok := cfg["catalog_id"].(string); ok {
		return s
	}
	return ""
}

func computeAncestorsMap(nodes []ValidationNode, incomingEdges map[string][]ValidationEdge) map[string][]string {
	ancestors := make(map[string]map[string]bool)
	for _, n := range nodes {
		ancestors[n.ID] = make(map[string]bool)
	}

	for _, n := range nodes {
		visited := make(map[string]bool)
		queue := []string{n.ID}
		visited[n.ID] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			for _, inc := range incomingEdges[curr] {
				parent := inc.Source
				if !visited[parent] {
					visited[parent] = true
					ancestors[n.ID][parent] = true
					queue = append(queue, parent)
				}
			}
		}
	}

	result := make(map[string][]string)
	for id, ancSet := range ancestors {
		var list []string
		for a := range ancSet {
			list = append(list, a)
		}
		sort.Strings(list)
		result[id] = list
	}
	return result
}

func computeMaxPathDelay(startID string, nodes []ValidationNode, outgoingEdges map[string][]ValidationEdge) int64 {
	nodeMap := make(map[string]*ValidationNode)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	memo := make(map[string]int64)
	var dfs func(curr string) int64
	dfs = func(curr string) int64 {
		if val, ok := memo[curr]; ok {
			return val
		}
		var currDelay int64
		if n, ok := nodeMap[curr]; ok {
			if sec, ok := extractValidationDelaySeconds(*n); ok && sec > 0 {
				currDelay = sec
			}
		}
		var maxChildDelay int64
		for _, e := range outgoingEdges[curr] {
			childDelay := dfs(e.Target)
			if childDelay > maxChildDelay {
				maxChildDelay = childDelay
			}
		}
		total := currDelay + maxChildDelay
		memo[curr] = total
		return total
	}
	return dfs(startID)
}
