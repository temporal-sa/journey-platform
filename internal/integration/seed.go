package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

// SeedOptions specifies parameters for data seeding.
type SeedOptions struct {
	TenantID string
}

// SeedResult summarizes the records seeded into the stores.
type SeedResult struct {
	TenantID            string
	CatalogsSeeded      int
	StaticListsSeeded   int
	ExperimentsSeeded   int
	JourneyDraftsSeeded int
	JourneyVersionsSeeded int
}

// Seed populates PostgreSQL and local stores with golden path deterministic fixture data idempotently.
func Seed(ctx context.Context, repo postgres.Repository, opts ...SeedOptions) (*SeedResult, error) {
	if repo == nil {
		repo = postgres.NewMemoryRepository()
	}

	tenantID := "default"
	if len(opts) > 0 && opts[0].TenantID != "" {
		tenantID = opts[0].TenantID
	}

	res := &SeedResult{TenantID: tenantID}

	// 1. Seed Catalogs (events, actions, attributes, parameters, metrics, templates)
	catalogs := getSeedCatalogs(tenantID)
	for _, c := range catalogs {
		existing, err := repo.GetCatalog(ctx, tenantID, c.RecordID)
		if err == nil && existing != nil {
			// Record exists; update idempotently
			if _, updateErr := repo.UpdateCatalog(ctx, &c); updateErr != nil {
				return nil, fmt.Errorf("failed to update existing catalog %s: %w", c.RecordID, updateErr)
			}
		} else {
			if _, createErr := repo.CreateCatalog(ctx, &c); createErr != nil {
				return nil, fmt.Errorf("failed to create catalog %s: %w", c.RecordID, createErr)
			}
		}
		res.CatalogsSeeded++
	}

	// 2. VIP Customers Segment List (5 contacts with tier & ltv attributes)
	vipList := getVipCustomersList(tenantID)
	existingVip, err := repo.GetStaticList(ctx, tenantID, vipList.ListID)
	if err == nil && existingVip != nil {
		if _, updateErr := repo.UpdateStaticList(ctx, &vipList); updateErr != nil {
			return nil, fmt.Errorf("failed to update VIP static list: %w", updateErr)
		}
	} else {
		if _, createErr := repo.CreateStaticList(ctx, &vipList); createErr != nil {
			return nil, fmt.Errorf("failed to create VIP static list: %w", createErr)
		}
	}
	res.StaticListsSeeded++

	// 3. Verified Contacts Fixture List (5 contacts: user_1 to user_5)
	verifiedContactsList := getVerifiedContactsList(tenantID)
	existingList, err := repo.GetStaticList(ctx, tenantID, verifiedContactsList.ListID)
	if err == nil && existingList != nil {
		if _, updateErr := repo.UpdateStaticList(ctx, &verifiedContactsList); updateErr != nil {
			return nil, fmt.Errorf("failed to update verified contacts static list: %w", updateErr)
		}
	} else {
		if _, createErr := repo.CreateStaticList(ctx, &verifiedContactsList); createErr != nil {
			return nil, fmt.Errorf("failed to create verified contacts static list: %w", createErr)
		}
	}
	res.StaticListsSeeded++

	// 3. Communication Policy Default States & Suppression List
	suppressionList := getCommunicationSuppressionList(tenantID)
	existingSupp, err := repo.GetStaticList(ctx, tenantID, suppressionList.ListID)
	if err == nil && existingSupp != nil {
		if _, updateErr := repo.UpdateStaticList(ctx, &suppressionList); updateErr != nil {
			return nil, fmt.Errorf("failed to update suppression static list: %w", updateErr)
		}
	} else {
		if _, createErr := repo.CreateStaticList(ctx, &suppressionList); createErr != nil {
			return nil, fmt.Errorf("failed to create suppression static list: %w", createErr)
		}
	}
	res.StaticListsSeeded++

	// 4. Onboarding Experiment Definition (70/30 split)
	onboardingExp := getOnboardingExperiment(tenantID)
	existingExp, err := repo.GetExperimentDefinition(ctx, tenantID, onboardingExp.ExperimentID)
	if err == nil && existingExp != nil {
		if _, updateErr := repo.UpdateExperimentDefinition(ctx, &onboardingExp); updateErr != nil {
			return nil, fmt.Errorf("failed to update experiment definition: %w", updateErr)
		}
	} else {
		if _, createErr := repo.CreateExperimentDefinition(ctx, &onboardingExp); createErr != nil {
			return nil, fmt.Errorf("failed to create experiment definition: %w", createErr)
		}
	}
	res.ExperimentsSeeded++
	// Seed exp-101 alias for frontend experiment report view
	exp101 := onboardingExp
	exp101.ExperimentID = "exp-101"
	exp101.Name = "Onboarding A/B Test"
	existing101Exp, err := repo.GetExperimentDefinition(ctx, tenantID, "exp-101")
	if err == nil && existing101Exp != nil {
		_, _ = repo.UpdateExperimentDefinition(ctx, &exp101)
	} else {
		_, _ = repo.CreateExperimentDefinition(ctx, &exp101)
	}
	onboardingDraft, onboardingVer := getOnboardingJourney(tenantID)
	existingDraft, err := repo.GetJourneyDraft(ctx, tenantID, onboardingDraft.DraftID)
	if err == nil && existingDraft != nil {
		if _, updateErr := repo.UpdateJourneyDraft(ctx, &onboardingDraft); updateErr != nil {
			return nil, fmt.Errorf("failed to update onboarding draft: %w", updateErr)
		}
	} else {
		if _, createErr := repo.CreateJourneyDraft(ctx, &onboardingDraft); createErr != nil {
			return nil, fmt.Errorf("failed to create onboarding draft: %w", createErr)
		}
	}
	res.JourneyDraftsSeeded++

	// Also seed draft-101 alias for frontend default journey
	draft101 := onboardingDraft
	draft101.DraftID = "draft-101"
	draft101.Name = "Welcome Journey Draft"
	existing101, err := repo.GetJourneyDraft(ctx, tenantID, "draft-101")
	if err == nil && existing101 != nil {
		if _, updateErr := repo.UpdateJourneyDraft(ctx, &draft101); updateErr != nil {
			return nil, fmt.Errorf("failed to update draft-101: %w", updateErr)
		}
	} else {
		if _, createErr := repo.CreateJourneyDraft(ctx, &draft101); createErr != nil {
			return nil, fmt.Errorf("failed to create draft-101: %w", createErr)
		}
	}
	res.JourneyDraftsSeeded++

	existingVer, err := repo.GetJourneyVersion(ctx, tenantID, onboardingVer.VersionID)
	if err == nil && existingVer != nil {
		// Version exists; skip
	} else {
		if _, createErr := repo.CreateJourneyVersion(ctx, &onboardingVer); createErr != nil {
			return nil, fmt.Errorf("failed to create onboarding version: %w", createErr)
		}
	}
	res.JourneyVersionsSeeded++

	// 6. Long-running Wait Journey: order_placed -> delay node (10s) -> WaitForEvent (order_delivered) -> exit
	waitDraft, waitVer := getLongRunningWaitJourney(tenantID)
	existingWaitDraft, err := repo.GetJourneyDraft(ctx, tenantID, waitDraft.DraftID)
	if err == nil && existingWaitDraft != nil {
		if _, updateErr := repo.UpdateJourneyDraft(ctx, &waitDraft); updateErr != nil {
			return nil, fmt.Errorf("failed to update wait journey draft: %w", updateErr)
		}
	} else {
		if _, createErr := repo.CreateJourneyDraft(ctx, &waitDraft); createErr != nil {
			return nil, fmt.Errorf("failed to create wait journey draft: %w", createErr)
		}
	}
	res.JourneyDraftsSeeded++

	existingWaitVer, err := repo.GetJourneyVersion(ctx, tenantID, waitVer.VersionID)
	if err == nil && existingWaitVer != nil {
		// Version exists; skip
	} else {
		if _, createErr := repo.CreateJourneyVersion(ctx, &waitVer); createErr != nil {
			return nil, fmt.Errorf("failed to create wait journey version: %w", createErr)
		}
	}
	res.JourneyVersionsSeeded++

	// 7. Seed Sample Enrollments & Lifecycle Events for Execution Runs (run-601, run-602, run-603, run-604)
	sampleRuns := []struct {
		runID       string
		workflowID  string
		status      string
		currentNode string
		execMode    string
		events      []struct {
			eventID   string
			eventName string
			nodeID    string
			status    string
			payload   map[string]interface{}
		}
	}{
		{
			runID:       "run-601",
			workflowID:  "wf-welcome-series",
			status:      "completed",
			currentNode: "node-email-2",
			execMode:    "production",
			events: []struct {
				eventID   string
				eventName string
				nodeID    string
				status    string
				payload   map[string]interface{}
			}{
				{eventID: "evt-101", eventName: "node_entered", nodeID: "node-signup-trigger", status: "completed", payload: map[string]interface{}{"user_id": "usr-8812", "email": "user@example.com"}},
				{eventID: "evt-102", eventName: "condition_evaluated", nodeID: "node-tier-check", status: "completed", payload: map[string]interface{}{"evaluated_tier": "gold", "match": true}},
				{eventID: "evt-103", eventName: "action_executed", nodeID: "node-send-welcome-email", status: "completed", payload: map[string]interface{}{"email_template": "tmpl_welcome_v2", "delivered": true}},
			},
		},
		{
			runID:       "run-602",
			workflowID:  "wf-cart-recovery",
			status:      "running",
			currentNode: "node-delay-check",
			execMode:    "production",
			events: []struct {
				eventID   string
				eventName string
				nodeID    string
				status    string
				payload   map[string]interface{}
			}{
				{eventID: "evt-201", eventName: "node_entered", nodeID: "node-cart-abandoned", status: "completed", payload: map[string]interface{}{"cart_id": "cart-9921", "items_count": 3}},
				{eventID: "evt-202", eventName: "delay_started", nodeID: "node-delay-check", status: "running", payload: map[string]interface{}{"wait_duration": "1h"}},
			},
		},
		{
			runID:       "run-603",
			workflowID:  "wf-kyc-nudge",
			status:      "failed",
			currentNode: "node-sms-action",
			execMode:    "test",
			events: []struct {
				eventID   string
				eventName string
				nodeID    string
				status    string
				payload   map[string]interface{}
			}{
				{eventID: "evt-301", eventName: "node_entered", nodeID: "node-kyc-trigger", status: "completed", payload: map[string]interface{}{"user_id": "usr-fintech-01", "kyc_status": "pending"}},
				{eventID: "evt-302", eventName: "action_executed", nodeID: "node-sms-action", status: "failed", payload: map[string]interface{}{"phone": "+15550199", "error": "Provider SMS gateway rate-limited: HTTP 429"}},
			},
		},
		{
			runID:       "run-604",
			workflowID:  "wf-promo-campaign",
			status:      "terminated",
			currentNode: "node-cancel-trigger",
			execMode:    "production",
			events: []struct {
				eventID   string
				eventName string
				nodeID    string
				status    string
				payload   map[string]interface{}
			}{
				{eventID: "evt-401", eventName: "node_entered", nodeID: "node-promo-trigger", status: "completed", payload: map[string]interface{}{"campaign_id": "cmp-summer"}},
				{eventID: "evt-402", eventName: "workflow_cancelled", nodeID: "node-cancel-trigger", status: "terminated", payload: map[string]interface{}{"reason": "User opted out"}},
			},
		},
	}

	for _, r := range sampleRuns {
		now := time.Now().UTC()
		stateBytes, _ := json.Marshal(map[string]interface{}{
			"execution_mode": r.execMode,
			"user_id":        "usr-fintech-01",
			"kyc_status":     "pending",
		})
		enr := &postgres.Enrollment{
			TenantID:         tenantID,
			EnrollmentID:     r.runID,
			JourneyVersionID: r.workflowID,
			SubjectID:        "taylor.khan@temporal.io",
			Status:           r.status,
			CurrentNodeID:    r.currentNode,
			StateData:        stateBytes,
			EnrolledAt:       now.Add(-10 * time.Minute),
			UpdatedAt:        now,
			CompletedAt:      &now,
		}
		_, _ = repo.CreateEnrollment(ctx, enr)

		for i, e := range r.events {
			pMap := map[string]interface{}{
				"node_id":     e.nodeID,
				"status":      e.status,
				"workflow_id": r.workflowID,
			}
			for k, v := range e.payload {
				pMap[k] = v
			}
			pBytes, _ := json.Marshal(pMap)
			_, _ = repo.RecordLifecycleEvent(ctx, &postgres.LifecycleEvent{
				TenantID:   tenantID,
				EventID:    e.eventID,
				EntityType: "workflow_run",
				EntityID:   r.runID,
				EventName:  e.eventName,
				Payload:    pBytes,
				CreatedAt:  now.Add(time.Duration(i-len(r.events)) * time.Minute),
			})
		}
	}

	return res, nil
}

func getSeedCatalogs(tenantID string) []postgres.Catalog {
	now := time.Now().UTC()
	return []postgres.Catalog{
		// Events
		{
			TenantID:         tenantID,
			RecordID:         "rec_evt_signup",
			Name:             "User Signup Event",
			ComponentType:    "trigger",
			Version:          "1.0.0",
			Description:      "Triggered when a new user signs up",
			SchemaDefinition: mustJSON(map[string]interface{}{"user_id": "string", "email": "string", "locale": "string"}),
			ContentHash:      calcHash("rec_evt_signup"),
			Tags:             []string{"user", "onboarding"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			TenantID:         tenantID,
			RecordID:         "rec_evt_order_placed",
			Name:             "Order Placed Event",
			ComponentType:    "trigger",
			Version:          "1.0.0",
			Description:      "Triggered when a customer completes an order",
			SchemaDefinition: mustJSON(map[string]interface{}{"order_id": "string", "amount": "number"}),
			ContentHash:      calcHash("rec_evt_order_placed"),
			Tags:             []string{"ecommerce", "order"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			TenantID:         tenantID,
			RecordID:         "rec_evt_order_delivered",
			Name:             "Order Delivered Event",
			ComponentType:    "trigger",
			Version:          "1.0.0",
			Description:      "Triggered when an order is delivered to customer",
			SchemaDefinition: mustJSON(map[string]interface{}{"order_id": "string", "delivered_at": "string"}),
			ContentHash:      calcHash("rec_evt_order_delivered"),
			Tags:             []string{"ecommerce", "delivery"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			TenantID:         tenantID,
			RecordID:         "rec_evt_cart_abandoned",
			Name:             "Cart Abandoned Event",
			ComponentType:    "trigger",
			Version:          "1.0.0",
			Description:      "Triggered when user abandons shopping cart",
			SchemaDefinition: mustJSON(map[string]interface{}{"cart_id": "string"}),
			ContentHash:      calcHash("rec_evt_cart_abandoned"),
			Tags:             []string{"ecommerce", "cart"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		// Actions
		{
			TenantID:         tenantID,
			RecordID:         "rec_act_send_email",
			Name:             "Send Email Action",
			ComponentType:    "action",
			Version:          "1.0.0",
			Description:      "Send personalized email to user",
			SchemaDefinition: mustJSON(map[string]interface{}{"template_id": "string", "recipient": "string"}),
			ContentHash:      calcHash("rec_act_send_email"),
			Tags:             []string{"messaging", "email"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			TenantID:         tenantID,
			RecordID:         "rec_act_send_push",
			Name:             "Send Push Notification",
			ComponentType:    "action",
			Version:          "1.0.0",
			Description:      "Send push notification to mobile device",
			SchemaDefinition: mustJSON(map[string]interface{}{"message": "string", "device_token": "string"}),
			ContentHash:      calcHash("rec_act_send_push"),
			Tags:             []string{"messaging", "push"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			TenantID:         tenantID,
			RecordID:         "rec_act_send_sms",
			Name:             "Send SMS Action",
			ComponentType:    "action",
			Version:          "1.0.0",
			Description:      "Send SMS text message to subject mobile number",
			SchemaDefinition: mustJSON(map[string]interface{}{"phone_number": "string"}),
			ContentHash:      calcHash("rec_act_send_sms"),
			Tags:             []string{"messaging", "sms"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		// Attributes / Conditions
		{
			TenantID:         tenantID,
			RecordID:         "rec_attr_user_locale",
			Name:             "User Locale Attribute",
			ComponentType:    "condition",
			Version:          "1.0.0",
			Description:      "Evaluates user geographic locale or preferred language",
			SchemaDefinition: mustJSON(map[string]interface{}{"locale": "string"}),
			ContentHash:      calcHash("rec_attr_user_locale"),
			Tags:             []string{"user", "attribute"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			TenantID:         tenantID,
			RecordID:         "rec_attr_user_tier",
			Name:             "User Tier Attribute",
			ComponentType:    "condition",
			Version:          "1.0.0",
			Description:      "Evaluates customer membership classification tier",
			SchemaDefinition: mustJSON(map[string]interface{}{"tier": "string"}),
			ContentHash:      calcHash("rec_attr_user_tier"),
			Tags:             []string{"user", "tier"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		// Parameters
		{
			TenantID:         tenantID,
			RecordID:         "rec_param_timeout",
			Name:             "Execution Timeout Parameter",
			ComponentType:    "parameter",
			Version:          "1.0.0",
			Description:      "Step execution timeout limit in seconds",
			SchemaDefinition: mustJSON(map[string]interface{}{"timeout_seconds": "number"}),
			ContentHash:      calcHash("rec_param_timeout"),
			Tags:             []string{"config", "parameter"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		// Metrics
		{
			TenantID:         tenantID,
			RecordID:         "rec_metric_conversion_rate",
			Name:             "Journey Conversion Rate Metric",
			ComponentType:    "metric",
			Version:          "1.0.0",
			Description:      "Ratio of subjects reaching goal exit node",
			SchemaDefinition: mustJSON(map[string]interface{}{"rate": "number"}),
			ContentHash:      calcHash("rec_metric_conversion_rate"),
			Tags:             []string{"analytics", "metric"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		// Templates
		{
			TenantID:         tenantID,
			RecordID:         "rec_tmpl_welcome_onboarding",
			Name:             "Welcome Onboarding Template",
			ComponentType:    "template",
			Version:          "1.0.0",
			Description:      "Pre-built welcome onboarding sequence for new signups",
			SchemaDefinition: mustJSON(map[string]interface{}{"template_name": "string"}),
			ContentHash:      calcHash("rec_tmpl_welcome_onboarding"),
			Tags:             []string{"template", "onboarding"},
			IsDeprecated:     false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
}

func getVipCustomersList(tenantID string) postgres.StaticList {
	now := time.Now().UTC()
	vipRecords := []map[string]interface{}{
		{"id": "usr_101", "email": "alexander.smith@example.com", "name": "Alexander Smith", "tier": "VIP Platinum", "ltv": "$12,450", "status": "Active"},
		{"id": "usr_102", "email": "sophia.martinez@enterprise.org", "name": "Sophia Martinez", "tier": "VIP Gold", "ltv": "$8,920", "status": "Active"},
		{"id": "usr_103", "email": "liam.johnson@corporate.co", "name": "Liam Johnson", "tier": "VIP Platinum", "ltv": "$15,100", "status": "Active"},
		{"id": "usr_104", "email": "emma.williams@techsolutions.io", "name": "Emma Williams", "tier": "VIP Gold", "ltv": "$9,400", "status": "Active"},
		{"id": "usr_105", "email": "noah.brown@cloudops.net", "name": "Noah Brown", "tier": "VIP Silver", "ltv": "$6,800", "status": "Active"},
	}
	return postgres.StaticList{
		TenantID:           tenantID,
		ListID:             "list-vip-users",
		Name:               "VIP Customers Segment",
		Description:        "High LTV accounts eligible for priority loyalty rewards",
		ItemCount:          int32(len(vipRecords)),
		DataClassification: string(domain.DataClassificationPII),
		Items:              mustJSON(vipRecords),
		ContentHash:        calcHash("list-vip-users"),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func getVerifiedContactsList(tenantID string) postgres.StaticList {
	now := time.Now().UTC()
	contacts := []string{"user_1", "user_2", "user_3", "user_4", "user_5"}
	return postgres.StaticList{
		TenantID:           tenantID,
		ListID:             "list_verified_contacts",
		Name:               "Verified Contacts Fixture List",
		Description:        "Verified test contacts list containing user_1 to user_5",
		ItemCount:          int32(len(contacts)),
		DataClassification: string(domain.DataClassificationNonPII),
		Items:              mustJSON(contacts),
		ContentHash:        calcHash("list_verified_contacts"),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func getCommunicationSuppressionList(tenantID string) postgres.StaticList {
	now := time.Now().UTC()
	suppressed := []string{"suppressed_user@example.com", "user_suppressed"}
	return postgres.StaticList{
		TenantID:           tenantID,
		ListID:             "communication_suppression_list",
		Name:               "Communication Policy Global Suppression List",
		Description:        "Opted-out and suppressed recipient addresses for communication policy enforcement",
		ItemCount:          int32(len(suppressed)),
		DataClassification: string(domain.DataClassificationNonPII),
		Items:              mustJSON(suppressed),
		ContentHash:        calcHash("communication_suppression_list"),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func getOnboardingExperiment(tenantID string) postgres.ExperimentDefinition {
	now := time.Now().UTC()
	variants := []domain.ExperimentVariant{
		{
			VariantID:         "control",
			Name:              "Control Variant (Email)",
			WeightBasisPoints: 7000,
			Config:            map[string]interface{}{"channel": "email"},
		},
		{
			VariantID:         "treatment",
			Name:              "Treatment Variant (Push)",
			WeightBasisPoints: 3000,
			Config:            map[string]interface{}{"channel": "push"},
		},
	}

	return postgres.ExperimentDefinition{
		TenantID:       tenantID,
		ExperimentID:   "exp_onboarding_split",
		Name:           "Onboarding Communication Split Experiment (70/30)",
		Description:    "70% Control (Email) / 30% Treatment (Push) experiment node",
		Status:         string(domain.ExperimentStatusActive),
		Variants:       mustJSON(variants),
		TargetAudience: "new_signups",
		ContentHash:    calcHash("exp_onboarding_split"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func getOnboardingJourney(tenantID string) (postgres.JourneyDraft, postgres.JourneyVersion) {
	now := time.Now().UTC()

	nodes := []domain.GraphNode{
		{ID: "node_signup", Type: "trigger", Name: "Signup Event", Position: &domain.NodePosition{X: 100, Y: 150}, Config: map[string]interface{}{"event_type": "signup"}},
		{ID: "node_condition", Type: "condition", Name: "Check Locale & Search", Position: &domain.NodePosition{X: 460, Y: 150}, Config: map[string]interface{}{"expression": "locale == 'en-US'"}},
		{ID: "node_experiment", Type: "experiment", Name: "70/30 Experiment Node", Position: &domain.NodePosition{X: 820, Y: 150}, Config: map[string]interface{}{"experiment_id": "exp_onboarding_split"}},
		{ID: "node_email_action", Type: "action", Name: "Send Personalized Email", Position: &domain.NodePosition{X: 460, Y: 370}, Config: map[string]interface{}{"action_type": "send_email", "variant": "control"}},
		{ID: "node_push_action", Type: "action", Name: "Send Push Notification", Position: &domain.NodePosition{X: 820, Y: 370}, Config: map[string]interface{}{"action_type": "send_push", "variant": "treatment"}},
		{ID: "node_exit", Type: "exit", Name: "Exit Journey", Position: &domain.NodePosition{X: 640, Y: 570}},
	}

	edges := []domain.GraphEdge{
		{ID: "edge_1", Source: "node_signup", Target: "node_condition"},
		{ID: "edge_2", Source: "node_condition", Target: "node_experiment", Condition: "true"},
		{ID: "edge_3", Source: "node_experiment", Target: "node_email_action", Condition: "variant == 'control'"},
		{ID: "edge_4", Source: "node_experiment", Target: "node_push_action", Condition: "variant == 'treatment'"},
		{ID: "edge_5", Source: "node_email_action", Target: "node_exit"},
		{ID: "edge_6", Source: "node_push_action", Target: "node_exit"},
	}

	nodesJSON := mustJSON(nodes)
	edgesJSON := mustJSON(edges)

	draft := postgres.JourneyDraft{
		TenantID:    tenantID,
		DraftID:     "draft_onboarding_journey",
		Name:        "Onboarding Journey Flow",
		Description: "signup event -> condition check (locale/search) -> experiment node (70/30 control/treatment) -> personalized email or push action -> exit",
		Version:     1,
		Nodes:       nodesJSON,
		Edges:       edgesJSON,
		ContentHash: calcHash("draft_onboarding_journey"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	version := postgres.JourneyVersion{
		TenantID:    tenantID,
		VersionID:   "ver_onboarding_journey_v1",
		DraftID:     "draft_onboarding_journey",
		Version:     1,
		EntryNodeID: "node_signup",
		Nodes:       nodesJSON,
		Edges:       edgesJSON,
		ContentHash: calcHash("ver_onboarding_journey_v1"),
		CompiledAt:  now,
		CreatedAt:   now,
	}

	return draft, version
}

func getLongRunningWaitJourney(tenantID string) (postgres.JourneyDraft, postgres.JourneyVersion) {
	now := time.Now().UTC()

	nodes := []domain.GraphNode{
		{ID: "node_order_placed", Type: "trigger", Name: "Order Placed Event", Config: map[string]interface{}{"event_type": "order_placed"}},
		{ID: "node_delay_10s", Type: "delay", Name: "Delay Node 10s", Config: map[string]interface{}{"duration": "10s"}},
		{ID: "node_wait_delivery", Type: "wait_for_event", Name: "WaitForEvent Order Delivered", Config: map[string]interface{}{"event_type": "order_delivered", "timeout": "24h"}},
		{ID: "node_exit", Type: "exit", Name: "Exit Journey"},
	}

	edges := []domain.GraphEdge{
		{ID: "edge_w1", Source: "node_order_placed", Target: "node_delay_10s"},
		{ID: "edge_w2", Source: "node_delay_10s", Target: "node_wait_delivery"},
		{ID: "edge_w3", Source: "node_wait_delivery", Target: "node_exit"},
	}

	nodesJSON := mustJSON(nodes)
	edgesJSON := mustJSON(edges)

	draft := postgres.JourneyDraft{
		TenantID:    tenantID,
		DraftID:     "draft_order_wait_journey",
		Name:        "Long-Running Order Wait Journey",
		Description: "order_placed event -> delay node (10s) -> WaitForEvent (order_delivered) -> exit",
		Version:     1,
		Nodes:       nodesJSON,
		Edges:       edgesJSON,
		ContentHash: calcHash("draft_order_wait_journey"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	version := postgres.JourneyVersion{
		TenantID:    tenantID,
		VersionID:   "ver_order_wait_journey_v1",
		DraftID:     "draft_order_wait_journey",
		Version:     1,
		EntryNodeID: "node_order_placed",
		Nodes:       nodesJSON,
		Edges:       edgesJSON,
		ContentHash: calcHash("ver_order_wait_journey_v1"),
		CompiledAt:  now,
		CreatedAt:   now,
	}

	return draft, version
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func calcHash(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
