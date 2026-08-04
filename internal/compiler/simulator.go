package compiler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/validated-pattern/journey-platform/internal/compiler/expression"
	"github.com/validated-pattern/journey-platform/internal/domain"
)

// SimulationDecision records a branching decision made during graph traversal.
type SimulationDecision struct {
	NodeID       string `json:"node_id"`
	NodeType     string `json:"node_type"`
	EdgeID       string `json:"edge_id,omitempty"`
	TargetNodeID string `json:"target_node_id,omitempty"`
	Expression   string `json:"expression,omitempty"`
	Result       bool   `json:"result"`
	Reason       string `json:"reason,omitempty"`
}

// SimulationSuppression records a step or message suppression during simulation.
type SimulationSuppression struct {
	NodeID string `json:"node_id"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// WouldSendPreview contains rendered message content preview with sensitive values redacted.
type WouldSendPreview struct {
	NodeID         string            `json:"node_id"`
	Channel        Channel           `json:"channel"`
	Subject        string            `json:"subject,omitempty"`
	Body           string            `json:"body"`
	Headers        map[string]string `json:"headers,omitempty"`
	URL            string            `json:"url,omitempty"`
	Recipient      string            `json:"recipient,omitempty"`
	RedactedFields []string          `json:"redacted_fields,omitempty"`
}

// SimulationResult holds complete execution traces and output previews from a simulation run.
type SimulationResult struct {
	VisitedNodes             []string                `json:"visited_nodes"`
	Decisions                []SimulationDecision    `json:"decisions"`
	ResolvedParameters       map[string]interface{}  `json:"resolved_parameters"`
	Suppressions             []SimulationSuppression `json:"suppressions"`
	ChosenExperimentVariants map[string]string       `json:"chosen_experiment_variants"`
	WouldSendPreviews        []WouldSendPreview      `json:"would_send_previews"`
	ExecutionTimeMS          int64                   `json:"execution_time_ms"`
}

// SimulationOptions configures simulator context, fixtures, bounds, and fake values.
type SimulationOptions struct {
	EventPayload             map[string]interface{} `json:"event_payload,omitempty"`
	SubjectProfile           map[string]interface{} `json:"subject_profile,omitempty"`
	Parameters               map[string]interface{} `json:"parameters,omitempty"`
	NodeOutputs              map[string]interface{} `json:"node_outputs,omitempty"`
	ExperimentBucketFixtures map[string]string      `json:"experiment_bucket_fixtures,omitempty"`
	URLAllowlist             []string               `json:"url_allowlist,omitempty"`
	MaxOutputLengths         map[Channel]int        `json:"max_output_lengths,omitempty"`
	SensitiveKeys            []string               `json:"sensitive_keys,omitempty"`
	FakeAttributes           map[string]interface{} `json:"fake_attributes,omitempty"`
	OptOutChannels           map[Channel]bool       `json:"opt_out_channels,omitempty"`
	MaxSteps                 int                    `json:"max_steps,omitempty"`
}

// Simulator interprets compiled IR graph and simulates execution without external providers.
type Simulator struct {
	compiler *TemplateCompiler
}

// NewSimulator initializes a new Simulator.
func NewSimulator(tc *TemplateCompiler) *Simulator {
	if tc == nil {
		tc = NewTemplateCompiler()
	}
	return &Simulator{compiler: tc}
}

// Simulate runs execution simulation on a CompiledIR.
func (s *Simulator) Simulate(ir *domain.CompiledIR, opts *SimulationOptions) (*SimulationResult, error) {
	if ir == nil {
		return nil, fmt.Errorf("compiled IR is nil")
	}

	startTime := time.Now()

	if opts == nil {
		opts = &SimulationOptions{}
	}

	// Prepare TemplateCompiler with options
	tcOpts := []TemplateOption{}
	if len(opts.URLAllowlist) > 0 {
		tcOpts = append(tcOpts, WithURLAllowlist(opts.URLAllowlist))
	}
	for ch, maxLen := range opts.MaxOutputLengths {
		tcOpts = append(tcOpts, WithMaxOutputLength(ch, maxLen))
	}
	compiler := NewTemplateCompiler(tcOpts...)

	// Build EvalContext
	evalCtx := expression.NewEvalContext()
	if opts.EventPayload != nil {
		for k, v := range opts.EventPayload {
			evalCtx.Event[k] = v
		}
	}
	if opts.SubjectProfile != nil {
		for k, v := range opts.SubjectProfile {
			evalCtx.Subject[k] = v
		}
	}
	if opts.Parameters != nil {
		for k, v := range opts.Parameters {
			evalCtx.Parameter[k] = v
		}
	}
	if opts.NodeOutputs != nil {
		for k, v := range opts.NodeOutputs {
			evalCtx.NodeOutput[k] = v
		}
	}

	// Index Nodes & Edges
	nodeMap := make(map[string]*domain.IRNode, len(ir.Nodes))
	for i := range ir.Nodes {
		nodeMap[ir.Nodes[i].ID] = &ir.Nodes[i]
	}

	outgoingEdges := make(map[string][]domain.IREdge)
	for _, e := range ir.Edges {
		outgoingEdges[e.SourceID] = append(outgoingEdges[e.SourceID], e)
	}

	// Sort outgoing edges deterministically by ID
	for k := range outgoingEdges {
		sort.Slice(outgoingEdges[k], func(i, j int) bool {
			return outgoingEdges[k][i].ID < outgoingEdges[k][j].ID
		})
	}

	// Find entry node
	entryNodeID := ir.EntryNodeID
	if entryNodeID == "" || nodeMap[entryNodeID] == nil {
		for _, n := range ir.Nodes {
			t := strings.ToLower(n.Type)
			if t == "trigger" || t == "start" || t == "entry" {
				entryNodeID = n.ID
				break
			}
		}
	}
	if entryNodeID == "" && len(ir.Nodes) > 0 {
		entryNodeID = ir.Nodes[0].ID
	}

	result := &SimulationResult{
		VisitedNodes:             []string{},
		Decisions:                []SimulationDecision{},
		ResolvedParameters:       make(map[string]interface{}),
		Suppressions:             []SimulationSuppression{},
		ChosenExperimentVariants: make(map[string]string),
		WouldSendPreviews:        []WouldSendPreview{},
	}

	maxSteps := opts.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 1000
	}

	currNodeID := entryNodeID
	stepCount := 0

	for currNodeID != "" && stepCount < maxSteps {
		stepCount++
		node, exists := nodeMap[currNodeID]
		if !exists {
			break
		}

		result.VisitedNodes = append(result.VisitedNodes, currNodeID)
		nodeTypeLower := strings.ToLower(node.Type)

		// 1. Handle Experiment / A/B Test Nodes
		if nodeTypeLower == "experiment" || nodeTypeLower == "ab_test" || nodeTypeLower == "split" || hasExperimentVariants(node) {
			expID := getExperimentID(node)
			var chosenVariant string

			if opts.ExperimentBucketFixtures != nil {
				if v, ok := opts.ExperimentBucketFixtures[expID]; ok {
					chosenVariant = v
				} else if v, ok := opts.ExperimentBucketFixtures[node.ID]; ok {
					chosenVariant = v
				}
			}

			if chosenVariant == "" {
				chosenVariant = extractDefaultVariant(node)
			}

			result.ChosenExperimentVariants[expID] = chosenVariant
			evalCtx.Experiment[expID] = chosenVariant
			evalCtx.Experiment[node.ID] = chosenVariant

			result.Decisions = append(result.Decisions, SimulationDecision{
				NodeID:   node.ID,
				NodeType: node.Type,
				Result:   true,
				Reason:   fmt.Sprintf("Assigned experiment '%s' to variant '%s'", expID, chosenVariant),
			})
		}

		// 2. Handle Message / Action / Channel Nodes
		if isMessagingNode(node) {
			channel := extractChannel(node)

			if opts.OptOutChannels != nil && opts.OptOutChannels[channel] {
				result.Suppressions = append(result.Suppressions, SimulationSuppression{
					NodeID: node.ID,
					Reason: "CHANNEL_OPT_OUT",
					Detail: fmt.Sprintf("Subject is opted out of channel '%s'", channel),
				})
			} else {
				rawTmpl := extractRawTemplate(node, channel)
				plan, err := compiler.CompileRaw(rawTmpl)
				if err != nil {
					result.Suppressions = append(result.Suppressions, SimulationSuppression{
						NodeID: node.ID,
						Reason: "TEMPLATE_COMPILE_ERROR",
						Detail: err.Error(),
					})
				} else {
					msg, resParams, renderErr := plan.Render(evalCtx, opts.FakeAttributes)
					if renderErr != nil {
						result.Suppressions = append(result.Suppressions, SimulationSuppression{
							NodeID: node.ID,
							Reason: "TEMPLATE_RENDER_ERROR",
							Detail: renderErr.Error(),
						})
					} else {
						for k, v := range resParams {
							result.ResolvedParameters[k] = v
						}

						// Process sensitive value redaction for preview
						redactedSubject, redSubKeys := RedactSensitiveText(msg.Subject, opts.SensitiveKeys)
						redactedBody, redBodyKeys := RedactSensitiveText(msg.Body, opts.SensitiveKeys)
						redactedURL, redURLKeys := RedactSensitiveText(msg.URL, opts.SensitiveKeys)

						redactedHeaders := make(map[string]string)
						var redHeaderKeys []string
						for hk, hv := range msg.Headers {
							rVal, keys := RedactSensitiveText(hv, opts.SensitiveKeys)
							redactedHeaders[hk] = rVal
							redHeaderKeys = append(redHeaderKeys, keys...)
						}

						var allRedacted []string
						allRedacted = append(allRedacted, redSubKeys...)
						allRedacted = append(allRedacted, redBodyKeys...)
						allRedacted = append(allRedacted, redURLKeys...)
						allRedacted = append(allRedacted, redHeaderKeys...)
						allRedacted = deduplicateStrings(allRedacted)

						recipient := extractRecipient(evalCtx, opts.FakeAttributes)

						result.WouldSendPreviews = append(result.WouldSendPreviews, WouldSendPreview{
							NodeID:         node.ID,
							Channel:        msg.Channel,
							Subject:        redactedSubject,
							Body:           redactedBody,
							Headers:        redactedHeaders,
							URL:            redactedURL,
							Recipient:      recipient,
							RedactedFields: allRedacted,
						})
					}
				}
			}
		}

		// 3. Outgoing Edges & Next Node Navigation
		edges := outgoingEdges[currNodeID]
		if len(edges) == 0 {
			// Terminal node
			break
		}

		nextID := ""
		if len(edges) == 1 && edges[0].ConditionExpression == "" {
			nextID = edges[0].TargetID
		} else {
			var defaultEdge *domain.IREdge
			matchedEdge := false

			for i := range edges {
				e := edges[i]
				condStr := strings.TrimSpace(e.ConditionExpression)
				if condStr == "" || strings.EqualFold(condStr, "default") || strings.EqualFold(condStr, "else") {
					defaultEdge = &e
					continue
				}

				// Evaluate condition expression
				astNode, err := expression.Parse(condStr)
				if err != nil {
					result.Decisions = append(result.Decisions, SimulationDecision{
						NodeID:       node.ID,
						NodeType:     node.Type,
						EdgeID:       e.ID,
						TargetNodeID: e.TargetID,
						Expression:   condStr,
						Result:       false,
						Reason:       fmt.Sprintf("Failed to parse condition expression: %v", err),
					})
					continue
				}

				evalBool, err := expression.EvaluateToBool(astNode, evalCtx)
				if err != nil {
					result.Decisions = append(result.Decisions, SimulationDecision{
						NodeID:       node.ID,
						NodeType:     node.Type,
						EdgeID:       e.ID,
						TargetNodeID: e.TargetID,
						Expression:   condStr,
						Result:       false,
						Reason:       fmt.Sprintf("Failed to evaluate condition expression: %v", err),
					})
					continue
				}

				result.Decisions = append(result.Decisions, SimulationDecision{
					NodeID:       node.ID,
					NodeType:     node.Type,
					EdgeID:       e.ID,
					TargetNodeID: e.TargetID,
					Expression:   condStr,
					Result:       evalBool,
				})

				if evalBool && !matchedEdge {
					nextID = e.TargetID
					matchedEdge = true
				}
			}

			if !matchedEdge && defaultEdge != nil {
				result.Decisions = append(result.Decisions, SimulationDecision{
					NodeID:       node.ID,
					NodeType:     node.Type,
					EdgeID:       defaultEdge.ID,
					TargetNodeID: defaultEdge.TargetID,
					Result:       true,
					Reason:       "Taken default fallthrough branch",
				})
				nextID = defaultEdge.TargetID
			}
		}

		currNodeID = nextID
	}

	result.ExecutionTimeMS = time.Since(startTime).Milliseconds()
	return result, nil
}

func isMessagingNode(node *domain.IRNode) bool {
	t := strings.ToLower(node.Type)
	if t == "message" || t == "email" || t == "sms" || t == "push" || t == "in_app" || t == "webhook" {
		return true
	}
	if node.Params != nil {
		if _, ok := node.Params["channel"]; ok {
			return true
		}
		if _, ok := node.Params["template"]; ok {
			return true
		}
		if _, ok := node.Params["body"]; ok {
			return true
		}
	}
	return false
}

func extractChannel(node *domain.IRNode) Channel {
	if node.Params != nil {
		if chStr, ok := node.Params["channel"].(string); ok && chStr != "" {
			return Channel(strings.ToLower(chStr))
		}
	}
	t := strings.ToLower(node.Type)
	switch t {
	case "email":
		return ChannelEmail
	case "sms":
		return ChannelSMS
	case "push":
		return ChannelPush
	case "in_app":
		return ChannelInApp
	case "webhook":
		return ChannelWebhook
	}
	return ChannelEmail
}

func extractRawTemplate(node *domain.IRNode, ch Channel) RawTemplate {
	raw := RawTemplate{Channel: ch}
	if node.Params == nil {
		return raw
	}

	if s, ok := node.Params["subject"].(string); ok {
		raw.Subject = s
	} else if s, ok := node.Params["title"].(string); ok {
		raw.Subject = s
	}

	if s, ok := node.Params["body"].(string); ok {
		raw.Body = s
	} else if s, ok := node.Params["message"].(string); ok {
		raw.Body = s
	} else if s, ok := node.Params["template"].(string); ok {
		raw.Body = s
	} else if s, ok := node.Params["content"].(string); ok {
		raw.Body = s
	}

	if s, ok := node.Params["url"].(string); ok {
		raw.URL = s
	} else if s, ok := node.Params["endpoint"].(string); ok {
		raw.URL = s
	}

	if hMap, ok := node.Params["headers"].(map[string]interface{}); ok {
		headers := make(map[string]string)
		for k, v := range hMap {
			headers[k] = fmt.Sprintf("%v", v)
		}
		raw.Headers = headers
	}

	return raw
}

func hasExperimentVariants(node *domain.IRNode) bool {
	if node.Params == nil {
		return false
	}
	if vars, ok := node.Params["experiment_variants"].([]interface{}); ok && len(vars) > 0 {
		return true
	}
	return false
}

func getExperimentID(node *domain.IRNode) string {
	if node.Params != nil {
		if id, ok := node.Params["experiment_id"].(string); ok && id != "" {
			return id
		}
	}
	return node.ID
}

func extractDefaultVariant(node *domain.IRNode) string {
	if node.Params != nil {
		if vars, ok := node.Params["experiment_variants"].([]interface{}); ok && len(vars) > 0 {
			if first, ok := vars[0].(map[string]interface{}); ok {
				if vid, ok := first["variant_id"].(string); ok {
					return vid
				}
			}
		}
	}
	return "variant-a"
}

func extractRecipient(ctx *expression.EvalContext, fakeAttrs map[string]interface{}) string {
	if ctx != nil && ctx.Subject != nil {
		if email, ok := ctx.Subject["email"].(string); ok && email != "" {
			return email
		}
		if phone, ok := ctx.Subject["phone"].(string); ok && phone != "" {
			return phone
		}
		if id, ok := ctx.Subject["id"].(string); ok && id != "" {
			return id
		}
	}
	if fakeAttrs != nil {
		if r, ok := fakeAttrs["recipient"].(string); ok {
			return r
		}
		if email, ok := fakeAttrs["subject.email"].(string); ok {
			return email
		}
	}
	return "user@example.com"
}

func deduplicateStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	sort.Strings(items)
	var res []string
	for i, item := range items {
		if item == "" {
			continue
		}
		if i == 0 || item != items[i-1] {
			res = append(res, item)
		}
	}
	return res
}
