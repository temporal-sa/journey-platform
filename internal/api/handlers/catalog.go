package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/domain"
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
			}
		}
	}

	// Fallback to default catalog records if none in DB for these types
	if len(results) == 0 {
		for _, rec := range defaultCatalogRecords {
			if matchesType(string(rec.ComponentType), targetTypes) {
				results = append(results, rec)
			}
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
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_act_send_sms",
		Name:          "Send SMS",
		ComponentType: domain.ComponentTypeAction,
		Version:       "1.0.0",
		Description:   "Dispatch an SMS text message to subject mobile number",
		Tags:          []string{"messaging", "sms"},
	},
	{
		SchemaVersion: domain.DefaultSchemaVersion,
		RecordID:      "rec_act_webhook_post",
		Name:          "Webhook HTTP Post",
		ComponentType: domain.ComponentTypeAction,
		Version:       "1.0.0",
		Description:   "Execute HTTP POST request to external system endpoint",
		Tags:          []string{"integration", "webhook"},
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
