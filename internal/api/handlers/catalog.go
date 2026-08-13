package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

// ListEventCatalog handles GET /api/v1/catalogs/events
func (h *Handlers) ListEventCatalog(w http.ResponseWriter, r *http.Request) {
	h.listCatalogByTypes(w, r, []string{"trigger", "event"})
}

// ListActionCatalog handles GET /api/v1/catalogs/actions
func (h *Handlers) ListActionCatalog(w http.ResponseWriter, r *http.Request) {
	h.listCatalogByTypes(w, r, []string{"action", "activity"})
}

// ListAttributeCatalog handles GET /api/v1/catalogs/attributes
func (h *Handlers) ListAttributeCatalog(w http.ResponseWriter, r *http.Request) {
	h.listCatalogByTypes(w, r, []string{"attribute", "condition"})
}

// ListParameterCatalog handles GET /api/v1/catalogs/parameters
func (h *Handlers) ListParameterCatalog(w http.ResponseWriter, r *http.Request) {
	h.listCatalogByTypes(w, r, []string{"parameter"})
}

// ListMetricCatalog handles GET /api/v1/catalogs/metrics
func (h *Handlers) ListMetricCatalog(w http.ResponseWriter, r *http.Request) {
	h.listCatalogByTypes(w, r, []string{"metric"})
}

// ListTemplateCatalog handles GET /api/v1/catalogs/templates
func (h *Handlers) ListTemplateCatalog(w http.ResponseWriter, r *http.Request) {
	h.listCatalogByTypes(w, r, []string{"template"})
}

// CreateCatalog handles POST /api/v1/catalogs/{type}
func (h *Handlers) CreateCatalog(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}

	rawType := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "type")))
	if rawType == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "catalog component type is required in path")
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	var req domain.CatalogRecord
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid JSON payload: %v", err))
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "catalog item name is required")
		return
	}

	compType := string(req.ComponentType)
	if compType == "" {
		compType = normalizeComponentType(rawType)
	}

	recordID := strings.TrimSpace(req.RecordID)
	if recordID == "" {
		recordID = fmt.Sprintf("cat-%s-%s", compType, uuid.New().String()[:8])
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = "1.0.0"
	}

	schemaDefBytes, _ := json.Marshal(req.SchemaDefinition)
	if req.SchemaDefinition == nil {
		schemaDefBytes = []byte("{}")
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{compType}
	}

	now := time.Now().UTC()
	dbCat := &postgres.Catalog{
		TenantID:         tenantID,
		RecordID:         recordID,
		Name:             req.Name,
		ComponentType:    compType,
		Version:          version,
		Description:      req.Description,
		SchemaDefinition: schemaDefBytes,
		ContentHash:      "",
		Tags:             tags,
		IsDeprecated:     req.IsDeprecated,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	domainRec := domain.CatalogRecord{
		SchemaVersion:    domain.DefaultSchemaVersion,
		RecordID:         recordID,
		Name:             req.Name,
		ComponentType:    domain.ComponentType(compType),
		Version:          version,
		Description:      req.Description,
		SchemaDefinition: req.SchemaDefinition,
		Tags:             tags,
		IsDeprecated:     req.IsDeprecated,
	}
	if hash, err := domainRec.CalculateSHA256(); err == nil {
		dbCat.ContentHash = hash
		domainRec.ContentHash = hash
	}

	saved, err := h.repo.CreateCatalog(r.Context(), dbCat)
	if err != nil {
		if errors.Is(err, postgres.ErrAlreadyExists) {
			updated, updateErr := h.repo.UpdateCatalog(r.Context(), dbCat)
			if updateErr != nil {
				middleware.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save catalog record: %v", updateErr))
				return
			}
			saved = updated
		} else {
			middleware.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to create catalog record: %v", err))
			return
		}
	}

	var savedSchemaDef map[string]interface{}
	if len(saved.SchemaDefinition) > 0 {
		_ = json.Unmarshal(saved.SchemaDefinition, &savedSchemaDef)
	}

	resp := domain.CatalogRecord{
		SchemaVersion:    domain.DefaultSchemaVersion,
		RecordID:         saved.RecordID,
		Name:             saved.Name,
		ComponentType:    domain.ComponentType(saved.ComponentType),
		Version:          saved.Version,
		Description:      saved.Description,
		SchemaDefinition: savedSchemaDef,
		ContentHash:      saved.ContentHash,
		Tags:             saved.Tags,
		IsDeprecated:     saved.IsDeprecated,
	}

	middleware.WriteJSON(w, http.StatusCreated, resp)
}

func normalizeComponentType(rawType string) string {
	switch strings.ToLower(rawType) {
	case "events", "event", "trigger", "triggers":
		return "trigger"
	case "actions", "action", "activity", "activities":
		return "action"
	case "attributes", "attribute", "condition", "conditions":
		return "attribute"
	case "parameters", "parameter", "params", "param":
		return "parameter"
	case "metrics", "metric":
		return "metric"
	case "templates", "template":
		return "template"
	default:
		return rawType
	}
}

func (h *Handlers) listCatalogByTypes(w http.ResponseWriter, r *http.Request, targetTypes []string) {
	if checkRateLimit(w, r) {
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	searchQuery := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if searchQuery == "" {
		searchQuery = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))
	}

	var results []domain.CatalogRecord
	seen := make(map[string]bool)

	// Fetch catalog items from DB repo
	dbRecords, err := h.repo.ListCatalogs(r.Context(), tenantID)
	if err == nil && len(dbRecords) > 0 {
		for _, dbRec := range dbRecords {
			if matchesType(dbRec.ComponentType, targetTypes) {
				var schemaDef map[string]interface{}
				if len(dbRec.SchemaDefinition) > 0 {
					_ = json.Unmarshal(dbRec.SchemaDefinition, &schemaDef)
				}
				tags := dbRec.Tags
				if tags == nil {
					tags = []string{}
				}
				rec := domain.CatalogRecord{
					SchemaVersion:    domain.DefaultSchemaVersion,
					RecordID:         dbRec.RecordID,
					Name:             dbRec.Name,
					ComponentType:    domain.ComponentType(dbRec.ComponentType),
					Version:          dbRec.Version,
					Description:      dbRec.Description,
					SchemaDefinition: schemaDef,
					ContentHash:      dbRec.ContentHash,
					Tags:             tags,
					IsDeprecated:     dbRec.IsDeprecated,
				}
				results = append(results, rec)
				seen[dbRec.RecordID] = true
			}
		}
	}

	// Merge built-in default catalog records (not overridden by DB)
	for _, rec := range defaultCatalogRecords {
		if matchesType(string(rec.ComponentType), targetTypes) && !seen[rec.RecordID] {
			results = append(results, rec)
			seen[rec.RecordID] = true
		}
	}

	// Apply search filter if query parameter provided
	if searchQuery != "" {
		var filtered []domain.CatalogRecord
		for _, rec := range results {
			if matchesSearch(rec, searchQuery) {
				filtered = append(filtered, rec)
			}
		}
		results = filtered
	}

	if results == nil {
		results = []domain.CatalogRecord{}
	}

	middleware.WriteJSON(w, http.StatusOK, results)
}

func matchesType(recType string, targetTypes []string) bool {
	recTypeLower := strings.ToLower(recType)
	for _, t := range targetTypes {
		if strings.ToLower(t) == recTypeLower {
			return true
		}
	}
	return false
}

func matchesSearch(rec domain.CatalogRecord, q string) bool {
	if strings.Contains(strings.ToLower(rec.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(rec.Description), q) {
		return true
	}
	if strings.Contains(strings.ToLower(rec.RecordID), q) {
		return true
	}
	for _, tag := range rec.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}

// Built-in catalog seed records for rich catalog responses out-of-the-box
var defaultCatalogRecords = []domain.CatalogRecord{
	// Events / Triggers
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_evt_user_signup",
		Name:          "User Signup",
		ComponentType: domain.ComponentTypeTrigger,
		Version:       "1.0.0",
		Description:   "Fired when a new user registers an account",
		Tags:          []string{"user", "onboarding"},
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_id": map[string]interface{}{"type": "string"},
				"email":   map[string]interface{}{"type": "string"},
			},
		},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_evt_cart_abandoned",
		Name:          "Cart Abandoned",
		ComponentType: domain.ComponentTypeTrigger,
		Version:       "1.0.0",
		Description:   "Fired when user leaves items in cart without purchasing",
		Tags:          []string{"ecommerce", "cart"},
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"cart_id":    map[string]interface{}{"type": "string"},
				"cart_value": map[string]interface{}{"type": "number"},
			},
		},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_evt_order_placed",
		Name:          "Order Placed",
		ComponentType: domain.ComponentTypeTrigger,
		Version:       "1.0.0",
		Description:   "Fired when a purchase order is completed",
		Tags:          []string{"ecommerce", "checkout"},
	},

	// Actions
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_act_send_email",
		Name:          "Send Email",
		ComponentType: domain.ComponentTypeAction,
		Version:       "1.0.0",
		Description:   "Dispatch an email message via ESP provider",
		Tags:          []string{"messaging", "email"},
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"recipient":   map[string]interface{}{"type": "string", "description": "Recipient email address"},
				"subject":     map[string]interface{}{"type": "string", "description": "Email subject line"},
				"body":        map[string]interface{}{"type": "string", "description": "Email body content or template"},
				"template_id": map[string]interface{}{"type": "string", "description": "Identifier for pre-defined email template"},
			},
			"required": []string{"recipient"},
		},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_act_send_sms",
		Name:          "Send SMS",
		ComponentType: domain.ComponentTypeAction,
		Version:       "1.0.0",
		Description:   "Dispatch an SMS text message to subject mobile number",
		Tags:          []string{"messaging", "sms"},
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"phone_number": map[string]interface{}{"type": "string", "description": "Destination phone number in E.164 format"},
				"message":      map[string]interface{}{"type": "string", "description": "SMS text message content"},
				"sender_id":    map[string]interface{}{"type": "string", "description": "Optional sender ID or shortcode"},
			},
			"required": []string{"phone_number", "message"},
		},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_act_send_push",
		Name:          "Send Push Notification",
		ComponentType: domain.ComponentTypeAction,
		Version:       "1.0.0",
		Description:   "Dispatch push notification to mobile device or browser",
		Tags:          []string{"messaging", "push"},
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"device_token": map[string]interface{}{"type": "string", "description": "Target device push notification token"},
				"title":        map[string]interface{}{"type": "string", "description": "Push notification title"},
				"message":      map[string]interface{}{"type": "string", "description": "Notification message body"},
				"badge":        map[string]interface{}{"type": "integer", "description": "Application icon badge count"},
				"deep_link":    map[string]interface{}{"type": "string", "description": "Deep link URL for app navigation"},
			},
			"required": []string{"device_token", "message"},
		},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_act_send_in_app",
		Name:          "Send In-App Message",
		ComponentType: domain.ComponentTypeAction,
		Version:       "1.0.0",
		Description:   "Display an in-app banner, modal, or toast message to active user",
		Tags:          []string{"messaging", "in_app"},
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_id":    map[string]interface{}{"type": "string", "description": "Target user identifier"},
				"title":      map[string]interface{}{"type": "string", "description": "In-app message header/title"},
				"body":       map[string]interface{}{"type": "string", "description": "In-app message body text"},
				"style":      map[string]interface{}{"type": "string", "description": "Message UI style (banner, modal, toast)"},
				"action_url": map[string]interface{}{"type": "string", "description": "Call to action target URL"},
			},
			"required": []string{"user_id", "body"},
		},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_act_webhook_post",
		Name:          "Webhook HTTP Post",
		ComponentType: domain.ComponentTypeAction,
		Version:       "1.0.0",
		Description:   "Execute HTTP POST request to external system endpoint",
		Tags:          []string{"integration", "webhook"},
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url":        map[string]interface{}{"type": "string", "description": "Target webhook destination URL"},
				"headers":    map[string]interface{}{"type": "object", "description": "Custom HTTP request headers"},
				"payload":    map[string]interface{}{"type": "object", "description": "JSON payload body sent to endpoint"},
				"timeout_ms": map[string]interface{}{"type": "integer", "description": "HTTP request timeout in milliseconds"},
			},
			"required": []string{"url"},
		},
	},

	// Attributes / Conditions
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_attr_cart_value",
		Name:          "Cart Value Attribute",
		ComponentType: domain.ComponentTypeCondition,
		Version:       "1.0.0",
		Description:   "Evaluates subject current cart total value",
		Tags:          []string{"cart", "attribute"},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_attr_user_tier",
		Name:          "User Tier Attribute",
		ComponentType: domain.ComponentTypeCondition,
		Version:       "1.0.0",
		Description:   "Customer loyalty classification tier (bronze, silver, gold)",
		Tags:          []string{"user", "attribute"},
	},

	// Parameters
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_param_timeout",
		Name:          "Execution Timeout",
		ComponentType: domain.ComponentType("parameter"),
		Version:       "1.0.0",
		Description:   "Maximum step execution timeout limit in seconds",
		Tags:          []string{"config", "parameter"},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_param_max_retries",
		Name:          "Max Retries",
		ComponentType: domain.ComponentType("parameter"),
		Version:       "1.0.0",
		Description:   "Maximum retry count before triggering failure branch",
		Tags:          []string{"config", "parameter"},
	},

	// Metrics
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_metric_conversion_rate",
		Name:          "Journey Conversion Rate",
		ComponentType: domain.ComponentType("metric"),
		Version:       "1.0.0",
		Description:   "Ratio of subjects reaching goal exit node",
		Tags:          []string{"analytics", "metric"},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_metric_open_rate",
		Name:          "Email Open Rate",
		ComponentType: domain.ComponentType("metric"),
		Version:       "1.0.0",
		Description:   "Percentage of subjects opening sent email activity",
		Tags:          []string{"analytics", "metric"},
	},

	// Templates
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_tmpl_cart_recovery",
		Name:          "Cart Abandonment Recovery Flow",
		ComponentType: domain.ComponentType("template"),
		Version:       "1.0.0",
		Description:   "Pre-built multi-touch cart recovery journey template",
		Tags:          []string{"template", "ecommerce"},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_tmpl_welcome_onboarding",
		Name:          "New User Welcome Onboarding",
		ComponentType: domain.ComponentType("template"),
		Version:       "1.0.0",
		Description:   "Pre-built 3-day welcome sequence for new signups",
		Tags:          []string{"template", "onboarding"},
	},
}
