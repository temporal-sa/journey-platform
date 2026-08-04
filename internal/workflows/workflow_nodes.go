package workflows

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/experiments"
	"github.com/validated-pattern/journey-platform/internal/outcomes"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var (
	ErrWorkflowCancelled = errors.New("workflow cancelled before branch entry")
)

// ExecuteExperimentNode handles Experiment Node execution:
// 1. Resolve randomization unit (user_id, subject_id, etc.)
// 2. Call GetOrAssign activity to get sticky assignment
// 3. Record exposure via RecordExposure activity ONLY if cancellation does not occur before branch entry
// 4. Move to immutable assigned variant branch
// 5. If randomization unit is missing or ineligible, take compiled fallback/ineligible branch.
func ExecuteExperimentNode(
	ctx workflow.Context,
	act *activities.Activities,
	node domain.IRNode,
	ir *domain.CompiledIR,
	state *CompiledJourneyState,
	input CompiledJourneyInput,
	cancelSignalChan workflow.ReceiveChannel,
) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing experiment node", "nodeID", node.ID, "mode", state.ExecutionMode)

	// Check eligibility override
	isIneligible := false
	if v, ok := node.Params["is_ineligible"].(bool); ok && v {
		isIneligible = true
	}
	if v, ok := state.Variables["is_ineligible"].(bool); ok && v {
		isIneligible = true
	}
	if v, ok := state.Variables["ineligible"].(bool); ok && v {
		isIneligible = true
	}

	// 1. Resolve Randomization Unit Key and Value
	unitKey := resolveRandomizationUnitKey(node)
	unitVal := resolveRandomizationUnitValue(unitKey, node, state, input)

	outgoing := findOutgoingEdges(ir.Edges, node.ID)

	if isIneligible || unitVal == "" {
		logger.Info("Randomization unit missing or ineligible, taking fallback/ineligible branch", "nodeID", node.ID, "unitKey", unitKey)
		state.NodeOutputs[node.ID] = "ineligible"
		edge := selectExperimentBranch(outgoing, "", true)
		if edge != nil {
			return edge.TargetID, nil
		}
		if len(outgoing) > 0 {
			return outgoing[len(outgoing)-1].TargetID, nil
		}
		return "", fmt.Errorf("no outgoing edges found for experiment node %s", node.ID)
	}

	// 2. Call Activity GetOrAssign to get sticky assignment
	expID := fmt.Sprintf("%v", node.Params["experiment_id"])
	if expID == "" || expID == "<nil>" {
		expID = fmt.Sprintf("%v", node.Params["experiment_key"])
	}
	if expID == "" || expID == "<nil>" {
		expID = node.ID
	}

	expDef := buildExperimentDefinition(expID, input.TenantID, node)

	asgMode := experiments.ModeProduction
	if state.ExecutionMode == ExecutionModeTest {
		asgMode = experiments.ModeTest
	}

	asgReq := experiments.AssignmentRequest{
		TenantID:   input.TenantID,
		Experiment: expDef,
		SubjectID:  unitVal,
		Mode:       asgMode,
	}

	var asgRecord *experiments.AssignmentRecord
	asgCtx := WithActivitySummary(ctx, fmt.Sprintf("Assign A/B experiment variant for node %s (%s)", node.ID, expID))
	err := workflow.ExecuteActivity(asgCtx, act.GetOrAssign, asgReq).Get(asgCtx, &asgRecord)
	if err != nil {
		logger.Error("GetOrAssign activity failed", "nodeID", node.ID, "error", err)
		edge := selectExperimentBranch(outgoing, "", true)
		if edge != nil {
			state.NodeOutputs[node.ID] = "ineligible"
			return edge.TargetID, nil
		}
		return "", fmt.Errorf("experiment assignment failed: %w", err)
	}

	// 3. Check for cancellation BEFORE exposure recording and branch entry!
	var sig domain.WorkflowSignal
	if cancelSignalChan != nil && cancelSignalChan.ReceiveAsync(&sig) {
		logger.Info("Cancellation signal received after assignment but before exposure", "nodeID", node.ID)
		return "", ErrWorkflowCancelled
	}

	// 4. Record exposure via RecordExposure activity
	expReq := outcomes.ExposureRequest{
		TenantID:          input.TenantID,
		ExperimentID:      expID,
		AssignmentID:      asgRecord.AssignmentID,
		SubjectID:         unitVal,
		VariantID:         asgRecord.VariantID,
		WeightBasisPoints: asgRecord.WeightBasisPoints,
		ExecutionMode:     string(state.ExecutionMode),
		Context: map[string]interface{}{
			"workflow_id": input.WorkflowID,
			"run_id":      input.RunID,
			"node_id":     node.ID,
			"node_visit":  state.VisitCounts[node.ID],
		},
	}

	var expRecord *outcomes.ExposureRecord
	expCtx := WithActivitySummary(ctx, fmt.Sprintf("Record A/B experiment exposure for variant %s on node %s", asgRecord.VariantID, node.ID))
	err = workflow.ExecuteActivity(expCtx, act.RecordExposure, expReq).Get(expCtx, &expRecord)
	if err != nil {
		logger.Warn("RecordExposure activity failed or non-fatal warning", "nodeID", node.ID, "error", err)
	}

	// 5. Update state & move to assigned variant branch
	state.ExperimentContext[node.ID] = asgRecord.VariantID
	state.NodeOutputs[node.ID] = asgRecord.VariantID

	edge := selectExperimentBranch(outgoing, asgRecord.VariantID, false)
	if edge != nil {
		return edge.TargetID, nil
	}
	if len(outgoing) > 0 {
		return outgoing[0].TargetID, nil
	}

	return "", nil
}

// ExecuteChannelActionNode handles Email, SMS, Push, InApp, and Webhook nodes:
// 1. Derives channel & parameters from node definition
// 2. Executes ActionGateway activity idempotently
// 3. Carries execution mode (production vs test) and test_run_id for test mode
// 4. Routes cleanly on success, suppression, failure, or fallback
func ExecuteChannelActionNode(
	ctx workflow.Context,
	act *activities.Activities,
	node domain.IRNode,
	ir *domain.CompiledIR,
	state *CompiledJourneyState,
	input CompiledJourneyInput,
) (string, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing channel action node", "nodeID", node.ID, "nodeType", node.Type, "mode", state.ExecutionMode)

	// 1. Determine channel
	nodeType := strings.ToLower(strings.TrimSpace(node.Type))
	channel := nodeType
	switch nodeType {
	case "email":
		channel = "email"
	case "sms":
		channel = "sms"
	case "push":
		channel = "push"
	case "inapp", "in_app":
		channel = "inapp"
	case "webhook":
		channel = "webhook"
	default:
		if ch, ok := node.Params["channel"].(string); ok && ch != "" {
			channel = ch
		} else {
			channel = "email"
		}
	}

	// 2. Resolve Subject Ref, Recipient Address, Templates
	subjectRef := resolveRandomizationUnitValue("subject_id", node, state, input)
	if subjectRef == "" {
		subjectRef = resolveRandomizationUnitValue("user_id", node, state, input)
	}
	if subjectRef == "" {
		subjectRef = "subject-default"
	}

	recipientAddress := resolveRecipientAddress(channel, node, state, input)
	templateVersion := getParamString(node.Params, "template_version", "1.0")
	templateBody := getParamString(node.Params, "template_body", getParamString(node.Params, "body", ""))
	actionVersion := getParamString(node.Params, "action_version", "1.0")

	// 3. Mode & TestRunID
	execMode := activities.ExecutionMode(state.ExecutionMode)
	if execMode == "" {
		execMode = activities.ExecutionModeProduction
	}

	testRunID := ""
	if execMode == activities.ExecutionModeTest {
		testRunID = getParamString(node.Params, "test_run_id", "")
		if testRunID == "" {
			if trID, ok := state.Variables["test_run_id"].(string); ok {
				testRunID = trID
			}
		}
		if testRunID == "" {
			if trID, ok := input.InputPayload["test_run_id"].(string); ok {
				testRunID = trID
			}
		}
		if testRunID == "" {
			testRunID = input.RunID
		}
	}

	// 4. Stable Node Visit
	nodeVisit := state.VisitCounts[node.ID]

	// 5. Construct ActionRequest and execute ExecuteActionGateway activity
	expContexts := make(map[string]interface{})
	for k, v := range state.ExperimentContext {
		expContexts[k] = v
	}

	paramRefs := make(map[string]interface{})
	for k, v := range state.Variables {
		paramRefs[k] = v
	}
	if node.Params != nil {
		for k, v := range node.Params {
			paramRefs[k] = v
		}
	}

	actionReq := activities.ActionRequest{
		TenantID:           input.TenantID,
		WorkflowID:         input.WorkflowID,
		JourneyVersion:     fmt.Sprintf("%d", state.Generation),
		NodeID:             node.ID,
		NodeVisit:          nodeVisit,
		ActionVersion:      actionVersion,
		TemplateVersion:    templateVersion,
		SubjectRef:         subjectRef,
		ParameterRefs:      paramRefs,
		ExecutionMode:      execMode,
		TestRunID:          testRunID,
		ExperimentContexts: expContexts,
		Channel:            channel,
		RecipientAddress:   recipientAddress,
		TemplateBody:       templateBody,
	}

	var gwResult *activities.GatewayResult
	gwAo := workflow.ActivityOptions{
		Summary:             fmt.Sprintf("Execute %s channel action for node %s (%s)", channel, node.ID, node.ActivityName),
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    100 * time.Millisecond,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Second,
			MaximumAttempts:    3,
		},
	}
	gwCtx := workflow.WithActivityOptions(ctx, gwAo)
	err := workflow.ExecuteActivity(gwCtx, act.ExecuteActionGateway, actionReq).Get(gwCtx, &gwResult)
	if err != nil {
		logger.Error("ExecuteActionGateway activity failed", "nodeID", node.ID, "error", err)
		outgoing := findOutgoingEdges(ir.Edges, node.ID)
		edge := selectActionBranch(outgoing, string(activities.LedgerStatusPermanentFailure))
		if edge != nil {
			state.NodeOutputs[node.ID] = map[string]interface{}{
				"status":        string(activities.LedgerStatusPermanentFailure),
				"error_message": err.Error(),
			}
			return edge.TargetID, nil
		}
		return "", fmt.Errorf("action gateway activity failed: %w", err)
	}

	statusStr := string(gwResult.Status)
	state.NodeOutputs[node.ID] = map[string]interface{}{
		"status":           statusStr,
		"idempotency_key":  gwResult.IdempotencyKey,
		"reason_code":      gwResult.ReasonCode,
		"error_message":    gwResult.ErrorMessage,
		"rendered_content": gwResult.RenderedContent,
	}

	// 6. Branch routing based on GatewayResult
	outgoing := findOutgoingEdges(ir.Edges, node.ID)
	edge := selectActionBranch(outgoing, statusStr)
	if edge != nil {
		return edge.TargetID, nil
	}
	if len(outgoing) > 0 {
		return outgoing[0].TargetID, nil
	}

	return "", nil
}

// Helpers

func resolveRandomizationUnitKey(node domain.IRNode) string {
	if node.Params == nil {
		return "user_id"
	}
	keys := []string{"randomization_unit", "randomization_unit_key", "unit_key", "unit", "subject_key"}
	for _, k := range keys {
		if val, ok := node.Params[k].(string); ok && val != "" {
			return val
		}
	}
	return "user_id"
}

func resolveRandomizationUnitValue(unitKey string, node domain.IRNode, state *CompiledJourneyState, input CompiledJourneyInput) string {
	if node.Params != nil {
		if val, ok := node.Params[unitKey].(string); ok && val != "" {
			return val
		}
	}
	if state.Variables != nil {
		if val, ok := state.Variables[unitKey].(string); ok && val != "" {
			return val
		}
	}
	if input.InputPayload != nil {
		if val, ok := input.InputPayload[unitKey].(string); ok && val != "" {
			return val
		}
	}

	// Fallback to common unit keys
	commonKeys := []string{"user_id", "subject_id", "subject_ref", "user_ref", "customer_id"}
	for _, k := range commonKeys {
		if state.Variables != nil {
			if val, ok := state.Variables[k].(string); ok && val != "" {
				return val
			}
		}
		if input.InputPayload != nil {
			if val, ok := input.InputPayload[k].(string); ok && val != "" {
				return val
			}
		}
		if node.Params != nil {
			if val, ok := node.Params[k].(string); ok && val != "" {
				return val
			}
		}
	}
	return ""
}

func resolveRecipientAddress(channel string, node domain.IRNode, state *CompiledJourneyState, input CompiledJourneyInput) string {
	var keys []string
	switch channel {
	case "email":
		keys = []string{"recipient_address", "email", "to", "address"}
	case "sms":
		keys = []string{"recipient_address", "phone", "phone_number", "mobile", "to"}
	case "push":
		keys = []string{"recipient_address", "push_token", "device_token", "to"}
	default:
		keys = []string{"recipient_address", "address", "to", "url", "endpoint"}
	}

	for _, k := range keys {
		if node.Params != nil {
			if val, ok := node.Params[k].(string); ok && val != "" {
				return val
			}
		}
		if state.Variables != nil {
			if val, ok := state.Variables[k].(string); ok && val != "" {
				return val
			}
		}
		if input.InputPayload != nil {
			if val, ok := input.InputPayload[k].(string); ok && val != "" {
				return val
			}
		}
	}
	return ""
}

func getParamString(params map[string]interface{}, key string, defaultVal string) string {
	if params == nil {
		return defaultVal
	}
	if val, ok := params[key].(string); ok && val != "" {
		return val
	}
	return defaultVal
}

func buildExperimentDefinition(expID string, tenantID string, node domain.IRNode) *experiments.Experiment {
	expDef := &experiments.Experiment{
		TenantID:                 tenantID,
		ExperimentID:             expID,
		Version:                  1,
		Salt:                     expID,
		Status:                   experiments.ExperimentStatusActive,
		RandomizationUnit:        experiments.UnitSubjectID,
		AttributionWindowSeconds: 86400,
		Variants: []experiments.Variant{
			{Key: "control", Name: "Control", WeightBasisPoints: 5000, IsControl: true},
			{Key: "treatment", Name: "Treatment", WeightBasisPoints: 5000, IsControl: false},
		},
		Metrics: []experiments.MetricDefinition{
			{Key: "primary_conversion", Name: "Primary Conversion Metric", EventType: "conversion", Type: "conversion"},
		},
	}
	return expDef
}

func selectExperimentBranch(edges []domain.IREdge, variantID string, isIneligible bool) *domain.IREdge {
	if len(edges) == 0 {
		return nil
	}
	if isIneligible {
		for i := range edges {
			expr := strings.ToLower(strings.TrimSpace(edges[i].ConditionExpression))
			if expr == "ineligible" || expr == "fallback" || expr == "ineligible_branch" || strings.Contains(expr, "ineligible") || strings.Contains(expr, "fallback") {
				return &edges[i]
			}
		}
		for i := range edges {
			expr := strings.ToLower(strings.TrimSpace(edges[i].ConditionExpression))
			if expr == "default" || expr == "fallback" {
				return &edges[i]
			}
		}
		return &edges[len(edges)-1]
	}

	variantLower := strings.ToLower(variantID)
	for i := range edges {
		expr := strings.ToLower(strings.TrimSpace(edges[i].ConditionExpression))
		if expr == variantLower || expr == "variant_"+variantLower || strings.Contains(expr, variantLower) {
			return &edges[i]
		}
	}

	for i := range edges {
		expr := strings.ToLower(strings.TrimSpace(edges[i].ConditionExpression))
		if expr == "" || expr == "true" || expr == "default" {
			return &edges[i]
		}
	}

	return &edges[0]
}

func selectActionBranch(edges []domain.IREdge, status string) *domain.IREdge {
	if len(edges) == 0 {
		return nil
	}
	statusLower := strings.ToLower(status)

	for i := range edges {
		expr := strings.ToLower(strings.TrimSpace(edges[i].ConditionExpression))
		if expr == "" {
			continue
		}
		switch statusLower {
		case "suppressed":
			if expr == "suppressed" || expr == "suppression" || strings.Contains(expr, "suppress") {
				return &edges[i]
			}
		case "permanent_failure", "retryable_failure", "failed", "failure":
			if expr == "permanent_failure" || expr == "retryable_failure" || expr == "failure" || expr == "failed" || expr == "error" || strings.Contains(expr, "fail") {
				return &edges[i]
			}
		case "unknown":
			if expr == "unknown" || expr == "fallback" || strings.Contains(expr, "unknown") {
				return &edges[i]
			}
		case "accepted", "succeeded", "success":
			if expr == "accepted" || expr == "success" || expr == "succeeded" || expr == "true" {
				return &edges[i]
			}
		}
	}

	if statusLower == "accepted" || statusLower == "succeeded" || statusLower == "success" {
		for i := range edges {
			expr := strings.ToLower(strings.TrimSpace(edges[i].ConditionExpression))
			if expr == "" || expr == "true" || expr == "default" || expr == "accepted" || expr == "success" {
				return &edges[i]
			}
		}
		return &edges[0]
	}

	if statusLower == "permanent_failure" || statusLower == "retryable_failure" || statusLower == "failed" || statusLower == "failure" {
		for i := range edges {
			expr := strings.ToLower(strings.TrimSpace(edges[i].ConditionExpression))
			if expr == "failure" || expr == "failed" || expr == "error" || strings.Contains(expr, "fail") {
				return &edges[i]
			}
		}
	}

	return &edges[0]
}
