package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
)

// ActivitySimulationConfig represents response/request payload for activity failure simulation settings.
type ActivitySimulationConfig struct {
	SimulatedActivityFailure bool `json:"simulated_activity_failure"`
	MaxFailureAttempts       int  `json:"max_failure_attempts"`
}

// GetActivityFailureSimulationSetting handles GET /api/v1/simulation/activity-failure
func (h *Handlers) GetActivityFailureSimulationSetting(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}
	h.simMu.Lock()
	maxAtt := h.simMaxFailureAttempts
	if maxAtt <= 0 {
		maxAtt = 99
	}
	resp := ActivitySimulationConfig{
		SimulatedActivityFailure: h.simActivityFailure,
		MaxFailureAttempts:       maxAtt,
	}
	h.simMu.Unlock()

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// UpdateActivityFailureSimulationSetting handles POST /api/v1/simulation/activity-failure
func (h *Handlers) UpdateActivityFailureSimulationSetting(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}
	var req ActivitySimulationConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid JSON payload: %v", err))
		return
	}

	h.simMu.Lock()
	h.simActivityFailure = req.SimulatedActivityFailure
	if req.MaxFailureAttempts > 0 {
		h.simMaxFailureAttempts = req.MaxFailureAttempts
	}
	maxAtt := h.simMaxFailureAttempts
	if maxAtt <= 0 {
		maxAtt = 99
	}
	resp := ActivitySimulationConfig{
		SimulatedActivityFailure: h.simActivityFailure,
		MaxFailureAttempts:       maxAtt,
	}
	h.simMu.Unlock()

	middleware.WriteJSON(w, http.StatusOK, resp)
}
