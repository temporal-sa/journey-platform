package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
)

type WorkerStatusResponse struct {
	Status        string `json:"status"`
	PID           int    `json:"pid"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

func (h *Handlers) getWorkerStatusResponse() WorkerStatusResponse {
	h.workerMu.Lock()
	defer h.workerMu.Unlock()

	if h.workerCmd != nil && h.workerCmd.Process != nil {
		uptime := int64(time.Since(h.workerStartTime).Seconds())
		return WorkerStatusResponse{
			Status:        "running",
			PID:           h.workerCmd.Process.Pid,
			UptimeSeconds: uptime,
		}
	}

	return WorkerStatusResponse{
		Status:        "stopped",
		PID:           0,
		UptimeSeconds: 0,
	}
}

// StartWorker spawns the journey-worker process if not running.
func (h *Handlers) StartWorker(w http.ResponseWriter, r *http.Request) {
	h.workerMu.Lock()
	defer h.workerMu.Unlock()

	if h.workerCmd != nil && h.workerCmd.Process != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "journey-worker process is already running",
		})
		return
	}

	cmd := exec.Command("go", "run", "./cmd/journey-worker")
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		logging.Debug().Err(err).Msg("Failed to start journey-worker process")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Failed to start journey-worker: %v", err),
		})
		return
	}

	doneCh := make(chan struct{})
	h.workerCmd = cmd
	h.workerDoneCh = doneCh
	h.workerStartTime = time.Now()
	pid := cmd.Process.Pid

	go func(c *exec.Cmd, done chan struct{}) {
		_ = c.Wait()
		close(done)
		h.workerMu.Lock()
		if h.workerCmd == c {
			h.workerCmd = nil
			h.workerDoneCh = nil
		}
		h.workerMu.Unlock()
	}(cmd, doneCh)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WorkerStatusResponse{
		Status:        "running",
		PID:           pid,
		UptimeSeconds: 0,
	})
}

// StopWorker stops the journey-worker process cleanly.
func (h *Handlers) StopWorker(w http.ResponseWriter, r *http.Request) {
	h.stopWorkerProcess()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WorkerStatusResponse{
		Status:        "stopped",
		PID:           0,
		UptimeSeconds: 0,
	})
}

func (h *Handlers) stopWorkerProcess() {
	h.workerMu.Lock()
	cmd := h.workerCmd
	doneCh := h.workerDoneCh
	h.workerCmd = nil
	h.workerDoneCh = nil
	h.workerMu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return
	}

	pid := cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGTERM)

	if doneCh != nil {
		select {
		case <-doneCh:
		case <-time.After(3 * time.Second):
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			<-doneCh
		}
	}
}

// GetWorkerStatus returns worker status JSON.
func (h *Handlers) GetWorkerStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.getWorkerStatusResponse())
}

// StreamWorkerStatus broadcasts worker status JSON over SSE every 2 seconds.
func (h *Handlers) StreamWorkerStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	sendStatus := func() bool {
		status := h.getWorkerStatusResponse()
		data, err := json.Marshal(status)
		if err != nil {
			return false
		}
		_, err = fmt.Fprintf(w, "data: %s\n\n", data)
		if err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !sendStatus() {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if !sendStatus() {
				return
			}
		}
	}
}
