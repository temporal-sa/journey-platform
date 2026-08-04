package compiler

import (
	"errors"
	"strings"
	"testing"

	"github.com/validated-pattern/journey-platform/internal/compiler/expression"
	"github.com/validated-pattern/journey-platform/internal/domain"
)

// -----------------------------------------------------------------------------
// 1. Template Compiler & Escaping Tests
// -----------------------------------------------------------------------------

func TestTemplateCompiler_HTMLEscaping_Email(t *testing.T) {
	tc := NewTemplateCompiler()

	raw := RawTemplate{
		Channel: ChannelEmail,
		Subject: "Welcome {{subject.name}}",
		Body:    "<p>Hello {{subject.name}}, your code is {{parameter.code}}</p>",
	}

	plan, err := tc.CompileRaw(raw)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	ctx := expression.NewEvalContext()
	ctx.Subject["name"] = "<script>alert('xss')</script>"
	ctx.Parameter["code"] = "<b>123456</b>"

	msg, resolved, err := plan.Render(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	expectedBody := "<p>Hello &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;, your code is &lt;b&gt;123456&lt;/b&gt;</p>"
	if msg.Body != expectedBody {
		t.Errorf("email HTML escaping failed.\nExpected: %s\nGot:      %s", expectedBody, msg.Body)
	}

	if resolved["subject.name"] != "<script>alert('xss')</script>" {
		t.Errorf("unexpected resolved param: %v", resolved["subject.name"])
	}
}

func TestTemplateCompiler_PlainText_SMSAndPush(t *testing.T) {
	tc := NewTemplateCompiler()

	raw := RawTemplate{
		Channel: ChannelSMS,
		Body:    "Hi {{subject.name}} & welcome to {{parameter.app_name}}!",
	}

	plan, err := tc.CompileRaw(raw)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	ctx := expression.NewEvalContext()
	ctx.Subject["name"] = "Alice & Bob"
	ctx.Parameter["app_name"] = "Journey Platform <v1>"

	msg, _, err := plan.Render(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	// SMS plain text should NOT HTML-escape & or < >
	expectedBody := "Hi Alice & Bob & welcome to Journey Platform <v1>!"
	if msg.Body != expectedBody {
		t.Errorf("SMS plain text rendering failed.\nExpected: %s\nGot:      %s", expectedBody, msg.Body)
	}
}

func TestTemplateCompiler_JSONEscaping_Webhook(t *testing.T) {
	tc := NewTemplateCompiler()

	raw := RawTemplate{
		Channel: ChannelWebhook,
		Body:    `{"user": "{{subject.name}}", "msg": "{{parameter.note}}"}`,
	}

	plan, err := tc.CompileRaw(raw)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	ctx := expression.NewEvalContext()
	ctx.Subject["name"] = `John "Johnny" Doe`
	ctx.Parameter["note"] = "Line1\nLine2"

	msg, _, err := plan.Render(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	// JSON escaping should properly handle double quotes and newlines inside webhook parameters
	if !strings.Contains(msg.Body, `John \"Johnny\" Doe`) {
		t.Errorf("Webhook JSON quote escaping failed. Got: %s", msg.Body)
	}
}

func TestTemplateCompiler_HeaderInjectionPrevention(t *testing.T) {
	tc := NewTemplateCompiler()

	raw := RawTemplate{
		Channel: ChannelEmail,
		Subject: "Order Update: {{event.order_id}}\r\nX-Injected-Header: EvilValue",
		Headers: map[string]string{
			"X-Custom-Header": "HeaderVal\nInjectedVal",
		},
		Body: "Your order is ready.",
	}

	plan, err := tc.CompileRaw(raw)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}

	ctx := expression.NewEvalContext()
	ctx.Event["order_id"] = "ORD-1001\r\nBcc: attacker@evil.com"

	msg, _, err := plan.Render(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if strings.Contains(msg.Subject, "\r") || strings.Contains(msg.Subject, "\n") {
		t.Errorf("Subject header contains unstripped linebreaks: %q", msg.Subject)
	}

	expectedSub := "Order Update: ORD-1001Bcc: attacker@evil.comX-Injected-Header: EvilValue"
	if msg.Subject != expectedSub {
		t.Errorf("Header injection stripping failed.\nExpected: %s\nGot:      %s", expectedSub, msg.Subject)
	}

	hVal := msg.Headers["X-Custom-Header"]
	if strings.Contains(hVal, "\n") || strings.Contains(hVal, "\r") {
		t.Errorf("Custom header contains unstripped linebreaks: %q", hVal)
	}
}

func TestTemplateCompiler_URLAllowlistAndSchemes(t *testing.T) {
	tc := NewTemplateCompiler(WithURLAllowlist([]string{"https://example.com", "https://api.acme.com"}))

	// Safe URL matching allowlist
	rawSafe := RawTemplate{
		Channel: ChannelEmail,
		Body:    "Click link",
		URL:     "https://example.com/confirm?id={{subject.id}}",
	}

	planSafe, err := tc.CompileRaw(rawSafe)
	if err != nil {
		t.Fatalf("compile error for safe URL: %v", err)
	}

	ctx := expression.NewEvalContext()
	ctx.Subject["id"] = "123"

	msgSafe, _, err := planSafe.Render(ctx, nil)
	if err != nil {
		t.Fatalf("render error for safe URL: %v", err)
	}
	if msgSafe.URL != "https://example.com/confirm?id=123" {
		t.Errorf("unexpected rendered URL: %s", msgSafe.URL)
	}

	// Disallowed URL domain
	rawDisallowed := RawTemplate{
		Channel: ChannelEmail,
		Body:    "Click link",
		URL:     "https://malicious-phishing.com/steal",
	}
	planDisallowed, _ := tc.CompileRaw(rawDisallowed)
	_, _, err = planDisallowed.Render(ctx, nil)
	if err == nil || !errors.Is(err, ErrDisallowedURL) {
		t.Errorf("expected ErrDisallowedURL for unallowed domain, got: %v", err)
	}

	// Unsafe javascript: scheme
	rawJS := RawTemplate{
		Channel: ChannelEmail,
		Body:    "Click link",
		URL:     "javascript:alert(1)",
	}
	planJS, _ := tc.CompileRaw(rawJS)
	_, _, err = planJS.Render(ctx, nil)
	if err == nil || !errors.Is(err, ErrDisallowedURL) {
		t.Errorf("expected ErrDisallowedURL for javascript: scheme, got: %v", err)
	}
}

func TestTemplateCompiler_OutputLengthBounds(t *testing.T) {
	tc := NewTemplateCompiler(WithMaxOutputLength(ChannelSMS, 50))

	raw := RawTemplate{
		Channel: ChannelSMS,
		Body:    "Hello {{subject.name}}, this message is intentionally made very long to exceed max bound",
	}

	plan, err := tc.CompileRaw(raw)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	ctx := expression.NewEvalContext()
	ctx.Subject["name"] = "Alexander"

	_, _, err = plan.Render(ctx, nil)
	if err == nil || !errors.Is(err, ErrOutputLengthExceeded) {
		t.Errorf("expected ErrOutputLengthExceeded, got: %v", err)
	}
}

func TestTemplateCompiler_UnicodeHandling(t *testing.T) {
	tc := NewTemplateCompiler()

	raw := RawTemplate{
		Channel: ChannelPush,
		Subject: "🎉 Welcome {{subject.name}}! 🚀",
		Body:    "こんにちは {{subject.name}}様! 特典: 1000円",
	}

	plan, err := tc.CompileRaw(raw)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	ctx := expression.NewEvalContext()
	ctx.Subject["name"] = "太郎"

	msg, _, err := plan.Render(ctx, nil)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	expectedSubject := "🎉 Welcome 太郎! 🚀"
	expectedBody := "こんにちは 太郎様! 特典: 1000円"

	if msg.Subject != expectedSubject {
		t.Errorf("Unicode subject mismatch.\nExpected: %s\nGot:      %s", expectedSubject, msg.Subject)
	}
	if msg.Body != expectedBody {
		t.Errorf("Unicode body mismatch.\nExpected: %s\nGot:      %s", expectedBody, msg.Body)
	}
}

func TestTemplateCompiler_ProhibitedTokenDetection(t *testing.T) {
	tc := NewTemplateCompiler()

	raw := RawTemplate{
		Channel: ChannelEmail,
		Body:    "Hello <script>alert('hack')</script>",
	}

	_, err := tc.CompileRaw(raw)
	if err == nil || !errors.Is(err, ErrProhibitedToken) {
		t.Errorf("expected ErrProhibitedToken, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// 2. Simulator Execution Trace Tests
// -----------------------------------------------------------------------------

func TestSimulator_TraceMatchingInterpreterDecisionPaths(t *testing.T) {
	sim := NewSimulator(nil)

	// Create CompiledIR representing a workflow with:
	// Start -> Experiment Node -> Condition Node -> HighValue Email or LowValue SMS -> Exit
	ir := &domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-test-journey",
		DraftID:       "draft-123",
		TenantID:      "tenant-456",
		Version:       1,
		EntryNodeID:   "node-start",
		Nodes: []domain.IRNode{
			{
				ID:           "node-start",
				Type:         "trigger",
				ActivityName: "StartTrigger",
			},
			{
				ID:           "node-exp",
				Type:         "experiment",
				ActivityName: "ABTestSplit",
				Params: map[string]interface{}{
					"experiment_id": "exp-promo-2026",
					"experiment_variants": []interface{}{
						map[string]interface{}{"variant_id": "var-discount-20"},
						map[string]interface{}{"variant_id": "var-discount-10"},
					},
				},
			},
			{
				ID:           "node-condition",
				Type:         "condition",
				ActivityName: "CheckOrderAmount",
			},
			{
				ID:           "node-high-value",
				Type:         "email",
				ActivityName: "SendHighValueEmail",
				Params: map[string]interface{}{
					"subject": "VIP Offer for {{subject.name}}",
					"body":    "Hi {{subject.name}}, since your order of {{event.amount}} is over 100, enjoy code {{parameter.promo_code}}",
				},
			},
			{
				ID:           "node-standard",
				Type:         "sms",
				ActivityName: "SendStandardSMS",
				Params: map[string]interface{}{
					"body": "Hi {{subject.name}}, thanks for your order of {{event.amount}}.",
				},
			},
			{
				ID:           "node-exit",
				Type:         "exit",
				ActivityName: "ExitNode",
			},
		},
		Edges: []domain.IREdge{
			{
				ID:       "edge-1",
				SourceID: "node-start",
				TargetID: "node-exp",
			},
			{
				ID:       "edge-2",
				SourceID: "node-exp",
				TargetID: "node-condition",
			},
			{
				ID:                  "edge-cond-high",
				SourceID:            "node-condition",
				TargetID:            "node-high-value",
				ConditionExpression: "event.amount > 100",
			},
			{
				ID:                  "edge-cond-default",
				SourceID:            "node-condition",
				TargetID:            "node-standard",
				ConditionExpression: "default",
			},
			{
				ID:       "edge-high-exit",
				SourceID: "node-high-value",
				TargetID: "node-exit",
			},
			{
				ID:       "edge-standard-exit",
				SourceID: "node-standard",
				TargetID: "node-exit",
			},
		},
	}

	opts := &SimulationOptions{
		EventPayload: map[string]interface{}{
			"amount": float64(250),
		},
		SubjectProfile: map[string]interface{}{
			"name":  "Charlie",
			"email": "charlie@example.com",
		},
		Parameters: map[string]interface{}{
			"promo_code": "VIP2026",
		},
		ExperimentBucketFixtures: map[string]string{
			"exp-promo-2026": "var-discount-20",
		},
	}

	result, err := sim.Simulate(ir, opts)
	if err != nil {
		t.Fatalf("simulation failed: %v", err)
	}

	// Verify Visited Nodes Sequence
	expectedVisited := []string{
		"node-start",
		"node-exp",
		"node-condition",
		"node-high-value",
		"node-exit",
	}

	if len(result.VisitedNodes) != len(expectedVisited) {
		t.Fatalf("unexpected visited nodes length.\nExpected: %v\nGot:      %v", expectedVisited, result.VisitedNodes)
	}
	for i, nodeID := range expectedVisited {
		if result.VisitedNodes[i] != nodeID {
			t.Errorf("visited node mismatch at index %d. Expected: %s, Got: %s", i, nodeID, result.VisitedNodes[i])
		}
	}

	// Verify Experiment Selection
	if variant, ok := result.ChosenExperimentVariants["exp-promo-2026"]; !ok || variant != "var-discount-20" {
		t.Errorf("experiment variant mismatch: got %v", variant)
	}

	// Verify Decision Tracing
	if len(result.Decisions) < 2 {
		t.Fatalf("expected at least 2 decisions, got %d", len(result.Decisions))
	}
	condDecision := result.Decisions[1]
	if condDecision.NodeID != "node-condition" || !condDecision.Result || condDecision.TargetNodeID != "node-high-value" {
		t.Errorf("condition decision mismatch: %+v", condDecision)
	}

	// Verify Resolved Parameters
	if result.ResolvedParameters["subject.name"] != "Charlie" {
		t.Errorf("resolved param subject.name mismatch: %v", result.ResolvedParameters["subject.name"])
	}
	if result.ResolvedParameters["event.amount"] != float64(250) {
		t.Errorf("resolved param event.amount mismatch: %v", result.ResolvedParameters["event.amount"])
	}

	// Verify WouldSendPreviews
	if len(result.WouldSendPreviews) != 1 {
		t.Fatalf("expected 1 would send preview, got %d", len(result.WouldSendPreviews))
	}

	preview := result.WouldSendPreviews[0]
	if preview.NodeID != "node-high-value" {
		t.Errorf("preview node ID mismatch: %s", preview.NodeID)
	}
	if preview.Channel != ChannelEmail {
		t.Errorf("preview channel mismatch: %s", preview.Channel)
	}
	if !strings.Contains(preview.Body, "Charlie") || !strings.Contains(preview.Body, "250") {
		t.Errorf("preview body missing rendered values: %s", preview.Body)
	}
}

func TestSimulator_SuppressionHandling_OptOut(t *testing.T) {
	sim := NewSimulator(nil)

	ir := &domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-opt-out",
		DraftID:       "draft-opt",
		TenantID:      "tenant-opt",
		Version:       1,
		EntryNodeID:   "start",
		Nodes: []domain.IRNode{
			{ID: "start", Type: "trigger"},
			{
				ID:   "sms-node",
				Type: "sms",
				Params: map[string]interface{}{
					"body": "Daily update for {{subject.name}}",
				},
			},
			{ID: "exit", Type: "exit"},
		},
		Edges: []domain.IREdge{
			{ID: "e1", SourceID: "start", TargetID: "sms-node"},
			{ID: "e2", SourceID: "sms-node", TargetID: "exit"},
		},
	}

	opts := &SimulationOptions{
		SubjectProfile: map[string]interface{}{"name": "Bob"},
		OptOutChannels: map[Channel]bool{
			ChannelSMS: true,
		},
	}

	result, err := sim.Simulate(ir, opts)
	if err != nil {
		t.Fatalf("simulation error: %v", err)
	}

	if len(result.Suppressions) != 1 {
		t.Fatalf("expected 1 suppression, got %d", len(result.Suppressions))
	}

	supp := result.Suppressions[0]
	if supp.NodeID != "sms-node" || supp.Reason != "CHANNEL_OPT_OUT" {
		t.Errorf("unexpected suppression: %+v", supp)
	}

	if len(result.WouldSendPreviews) != 0 {
		t.Errorf("expected 0 previews due to suppression, got %d", len(result.WouldSendPreviews))
	}
}

// -----------------------------------------------------------------------------
// 3. Redacted Preview Snapshot Tests
// -----------------------------------------------------------------------------

func TestSimulator_SnapshotRedactedPreviews(t *testing.T) {
	sim := NewSimulator(nil)

	ir := &domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-snapshot",
		DraftID:       "draft-snap",
		TenantID:      "tenant-snap",
		Version:       1,
		EntryNodeID:   "start",
		Nodes: []domain.IRNode{
			{ID: "start", Type: "trigger"},
			{
				ID:   "email-node",
				Type: "email",
				Params: map[string]interface{}{
					"subject": "Password reset for {{subject.email}}",
					"body":    "Hello {{subject.name}}, your temp password is password={{parameter.password}}. Credit Card: 4111-2222-3333-4444. Auth token: Bearer abc123xyz789",
					"headers": map[string]interface{}{
						"X-Auth-Token": "Bearer token_secret_999",
					},
					"url": "https://example.com/reset?token={{parameter.auth_token}}",
				},
			},
		},
		Edges: []domain.IREdge{
			{ID: "e1", SourceID: "start", TargetID: "email-node"},
		},
	}

	opts := &SimulationOptions{
		SubjectProfile: map[string]interface{}{
			"name":  "Daniel",
			"email": "daniel@example.com",
		},
		Parameters: map[string]interface{}{
			"password":   "SuperSecretPass123",
			"auth_token": "secret_tok_456",
		},
	}

	result, err := sim.Simulate(ir, opts)
	if err != nil {
		t.Fatalf("simulation failed: %v", err)
	}

	if len(result.WouldSendPreviews) != 1 {
		t.Fatalf("expected 1 preview, got %d", len(result.WouldSendPreviews))
	}

	preview := result.WouldSendPreviews[0]

	// Verify Redaction in Body
	if strings.Contains(preview.Body, "SuperSecretPass123") {
		t.Errorf("password leak in preview body: %s", preview.Body)
	}
	if strings.Contains(preview.Body, "4111-2222-3333-4444") {
		t.Errorf("credit card leak in preview body: %s", preview.Body)
	}
	if strings.Contains(preview.Body, "abc123xyz789") {
		t.Errorf("bearer token leak in preview body: %s", preview.Body)
	}

	// Verify Redaction in Headers
	if strings.Contains(preview.Headers["X-Auth-Token"], "token_secret_999") {
		t.Errorf("token leak in header: %s", preview.Headers["X-Auth-Token"])
	}

	// Verify RedactedFields List
	if len(preview.RedactedFields) == 0 {
		t.Errorf("expected non-empty RedactedFields list")
	}

	// Snapshot Assertion on Redacted Body
	expectedBodySnapshot := "Hello Daniel, your temp password is password=[REDACTED]. Credit Card: [REDACTED:CREDIT_CARD]. Auth token: Bearer [REDACTED]"
	if preview.Body != expectedBodySnapshot {
		t.Errorf("Snapshot verification failed for preview body.\nExpected: %s\nGot:      %s", expectedBodySnapshot, preview.Body)
	}
}
