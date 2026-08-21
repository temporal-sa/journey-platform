package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/api/middleware"
)

// ActivitySimulationConfig represents response/request payload for activity simulation settings.
type ActivitySimulationConfig struct {
	SimulatedActivityFailure bool `json:"simulated_activity_failure"`
	LatencyMS                int  `json:"latency_ms"`
}

// ActivityLatencyConfig represents response/request payload for activity latency setting.
type ActivityLatencyConfig struct {
	LatencyMS int `json:"latency_ms"`
}

// GetActivityFailureSimulationSetting handles GET /api/v1/simulation/activity-failure
func (h *Handlers) GetActivityFailureSimulationSetting(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}
	resp := ActivitySimulationConfig{
		SimulatedActivityFailure: activities.IsSimulatedActivityFailureEnabled(),
		LatencyMS:                activities.GetSimulatedActivityLatencyMS(),
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// UpdateActivityFailureSimulationSetting handles POST /api/v1/simulation/activity-failure
func (h *Handlers) UpdateActivityFailureSimulationSetting(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}
	var req struct {
		SimulatedActivityFailure *bool `json:"simulated_activity_failure,omitempty"`
		LatencyMS                *int  `json:"latency_ms,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid JSON payload: %v", err))
		return
	}

	h.simMu.Lock()
	if req.SimulatedActivityFailure != nil {
		h.simActivityFailure = *req.SimulatedActivityFailure
		activities.SetSimulatedActivityFailure(*req.SimulatedActivityFailure)
	}
	if req.LatencyMS != nil {
		activities.SetSimulatedActivityLatencyMS(*req.LatencyMS)
	}
	h.simMu.Unlock()

	resp := ActivitySimulationConfig{
		SimulatedActivityFailure: activities.IsSimulatedActivityFailureEnabled(),
		LatencyMS:                activities.GetSimulatedActivityLatencyMS(),
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// GetActivityLatencySetting handles GET /api/v1/simulation/activity-latency
func (h *Handlers) GetActivityLatencySetting(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}
	resp := ActivityLatencyConfig{
		LatencyMS: activities.GetSimulatedActivityLatencyMS(),
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// UpdateActivityLatencySetting handles POST /api/v1/simulation/activity-latency
func (h *Handlers) UpdateActivityLatencySetting(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}
	var req ActivityLatencyConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid JSON payload: %v", err))
		return
	}

	activities.SetSimulatedActivityLatencyMS(req.LatencyMS)

	resp := ActivityLatencyConfig{
		LatencyMS: activities.GetSimulatedActivityLatencyMS(),
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}
