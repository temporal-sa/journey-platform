package handlers

import (
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
)
// GetTemporalClient returns the active Temporal SDK client or lazily connects to Temporal server.
func (h *Handlers) GetTemporalClient() client.Client {
	h.serviceMutex.Lock()
	defer h.serviceMutex.Unlock()

	if h.temporalClient != nil {
		return h.temporalClient
	}

	temporalHost := os.Getenv("TEMPORAL_HOST_PORT")
	if temporalHost == "" {
		temporalHost = "127.0.0.1:7233"
	}
	temporalNamespace := os.Getenv("TEMPORAL_NAMESPACE")
	if temporalNamespace == "" {
		temporalNamespace = "default"
	}

	tracingInterceptor, _ := workflows.CreateTemporalTracingInterceptor()
	var clientInterceptors []interceptor.ClientInterceptor
	if tracingInterceptor != nil {
		clientInterceptors = append(clientInterceptors, tracingInterceptor)
	}

	tc, err := client.Dial(client.Options{
		HostPort:     temporalHost,
		Namespace:    temporalNamespace,
		Interceptors: clientInterceptors,
	})
	if err == nil && tc != nil {
		h.temporalClient = tc
		logging.Debug().Str("host", temporalHost).Msg("Lazily connected to Temporal server")
		return tc
	} else if err != nil {
		logging.Debug().Err(err).Str("host", temporalHost).Msg("Failed lazy dial to Temporal server")
	}
	return nil
}

// Handlers encapsulates dependencies for API handlers.
type Handlers struct {
	repo           postgres.Repository
	compiler       *compiler.Compiler
	simulator      *compiler.Simulator
	logger         any
	temporalClient client.Client
	logBuffer      *logging.LogBuffer
	serviceMutex   sync.Mutex

	workerMu        sync.Mutex
	workerCmd       *exec.Cmd
	workerDoneCh    chan struct{}
	workerStartTime time.Time
}

// SetLogBuffer attaches a custom LogBuffer instance.
func (h *Handlers) SetLogBuffer(buf *logging.LogBuffer) {
	h.serviceMutex.Lock()
	defer h.serviceMutex.Unlock()
	h.logBuffer = buf
}

// GetLogBuffer returns the active LogBuffer instance or default buffer.
func (h *Handlers) GetLogBuffer() *logging.LogBuffer {
	h.serviceMutex.Lock()
	defer h.serviceMutex.Unlock()
	if h.logBuffer == nil {
		h.logBuffer = logging.DefaultBuffer()
	}
	return h.logBuffer
}
func (h *Handlers) SetTemporalClient(c client.Client) {
	h.serviceMutex.Lock()
	defer h.serviceMutex.Unlock()
	h.temporalClient = c
}

// New creates a new Handlers instance.
func New(repo postgres.Repository, comp *compiler.Compiler, logger any) *Handlers {
	if repo == nil {
		repo = postgres.NewMemoryRepository()
	}
	if comp == nil {
		comp = compiler.New()
	}
	return &Handlers{
		repo:      repo,
		compiler:  comp,
		simulator: compiler.NewSimulator(nil),
		logger:    logger,
		logBuffer: logging.DefaultBuffer(),
	}
}

// RegisterRoutes registers all catalog and journey handlers on the provided Chi router.
func (h *Handlers) RegisterRoutes(r chi.Router) {
	// Catalog Endpoints
	r.Get("/api/v1/catalogs/events", h.ListEventCatalog)
	r.Get("/api/v1/catalogs/actions", h.ListActionCatalog)
	r.Get("/api/v1/catalogs/attributes", h.ListAttributeCatalog)
	r.Get("/api/v1/catalogs/parameters", h.ListParameterCatalog)
	r.Get("/api/v1/catalogs/metrics", h.ListMetricCatalog)
	r.Get("/api/v1/catalogs/templates", h.ListTemplateCatalog)
	r.Post("/api/v1/catalogs/{type}", h.CreateCatalog)

	// Journey Draft Endpoints - Standard /api/v1/journeys path family
	r.Get("/api/v1/journeys", h.ListDrafts)
	r.Post("/api/v1/journeys", h.CreateDraft)
	r.Get("/api/v1/journeys/{id}", h.GetDraft)
	r.Put("/api/v1/journeys/{id}/draft", h.UpdateDraft)
	r.Put("/api/v1/journeys/{id}", h.UpdateDraft)
	r.Post("/api/v1/journeys/{id}/validate", h.ValidateDraft)
	r.Post("/api/v1/journeys/{id}/simulate", h.SimulateDraft)
	r.Get("/api/v1/journeys/{id}/versions", h.ListVersions)
	r.Post("/api/v1/journeys/{id}/versions/publish", h.PublishDraft)
	r.Post("/api/v1/journeys/{id}/versions/{v}/activate", h.ActivateLocal)
	r.Post("/api/v1/journeys/{id}/versions/{v}/pause", h.PauseLocal)
	r.Post("/api/v1/journeys/{id}/activate-local", h.ActivateLocal)
	r.Post("/api/v1/journeys/{id}/pause-local", h.PauseLocal)

	// Journey Draft Endpoints - Alias /api/v1/journeys/drafts path family (OpenAPI / frontend alignment)
	r.Get("/api/v1/journeys/drafts", h.ListDrafts)
	r.Post("/api/v1/journeys/drafts", h.CreateDraft)
	r.Get("/api/v1/journeys/drafts/{id}", h.GetDraft)
	r.Put("/api/v1/journeys/drafts/{id}", h.UpdateDraft)
	r.Post("/api/v1/journeys/drafts/{id}/validate", h.ValidateDraft)
	r.Post("/api/v1/journeys/drafts/{id}/simulate", h.SimulateDraft)
	r.Get("/api/v1/journeys/drafts/{id}/versions", h.ListVersions)
	r.Post("/api/v1/journeys/drafts/{id}/publish", h.PublishDraft)
	r.Post("/api/v1/journeys/drafts/{id}/activate-local", h.ActivateLocal)
	r.Post("/api/v1/journeys/drafts/{id}/pause-local", h.PauseLocal)
	// Run Endpoints
	r.Get("/api/v1/runs", h.ListRuns)
	r.Get("/api/v1/runs/{id}", h.GetRunTimeline)
	r.Get("/api/v1/runs/{id}/timeline", h.GetRunTimeline)
	r.Get("/api/v1/runs/{id}/sub-runs", h.ListJourneyRunSubRuns)
	r.Get("/api/v1/journeys/runs", h.ListRuns)
	r.Get("/api/v1/journeys/runs/{id}", h.GetRunTimeline)
	r.Get("/api/v1/journeys/runs/{id}/timeline", h.GetRunTimeline)
	r.Get("/api/v1/journeys/runs/{id}/sub-runs", h.ListJourneyRunSubRuns)

	// Experiment Endpoints
	r.Post("/api/v1/experiments", h.CreateExperiment)
	r.Get("/api/v1/experiments", h.ListExperiments)
	r.Get("/api/v1/experiments/{id}", h.GetExperiment)
	r.Get("/api/v1/experiments/{id}/versions/{v}", h.GetExperimentVersion)

	// Report Endpoints
	r.Get("/api/v1/reports/experiments/{id}", h.GetExperimentReport)
	r.Get("/api/v1/reports/experiments/{id}/export", h.ExportExperimentReportCSV)
	r.Get("/api/v1/reports/aggregate", h.GetExperimentReport)
	r.Get("/api/v1/exports/csv", h.ExportExperimentReportCSV)

	// Static-List Endpoints
	r.Get("/api/v1/static-lists", h.ListStaticLists)
	r.Post("/api/v1/static-lists/upload", h.UploadStaticList)
	r.Get("/api/v1/static-lists/{id}/versions", h.ListStaticListVersions)
	r.Post("/api/v1/static-lists/{id}/finalize", h.FinalizeStaticList)
	r.Get("/api/v1/static-lists/{id}", h.GetStaticList)
	r.Delete("/api/v1/static-lists/{id}", h.DeleteStaticList)

	// Test-Run Endpoints
	r.Post("/api/v1/test-runs", h.StartTestRun)
	r.Get("/api/v1/test-runs/{id}", h.GetTestRunStatus)
	r.Post("/api/v1/test-runs/{id}/cancel", h.CancelTestRun)

	// Event & Callback Ingress Endpoints
	r.Post("/api/v1/events/emit", h.EmitKafkaTestEvent)
	r.Post("/api/v1/callbacks/outcomes", h.ProcessOutcomeCallback)

	// Worker Process Control Endpoints
	r.Post("/api/v1/worker/start", h.StartWorker)
	r.Post("/api/v1/worker/stop", h.StopWorker)
	r.Get("/api/v1/worker/status", h.GetWorkerStatus)
	r.Get("/api/v1/worker/status/stream", h.StreamWorkerStatus)

	// WebMCP JSON-RPC 2.0 Endpoint
	r.Post("/mcp", h.HandleWebMCP)

	// Log Stream Endpoint
	r.Get("/api/v1/logs/stream", h.StreamLogs)
}

