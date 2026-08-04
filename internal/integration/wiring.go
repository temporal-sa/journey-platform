package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/store/analytics"
	"github.com/validated-pattern/journey-platform/internal/store/object"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/testaudience"
)

// Config defines local dependency endpoints and service configurations for the golden path application.
type Config struct {
	TenantID string

	// PostgreSQL
	PostgresHost string
	PostgresPort string
	PostgresUser string
	PostgresPass string
	PostgresDB   string

	// ClickHouse Analytics
	ClickHouseHost     string
	ClickHouseHTTPPort string
	ClickHouseUser     string
	ClickHousePass     string
	ClickHouseDB       string

	// Kafka & Schema Registry
	KafkaBootstrapServers string
	SchemaRegistryURL     string

	// Temporal Orchestrator
	TemporalHostPort string
	TemporalNamespace string

	// MinIO Object Storage
	MinIOEndpoint        string
	MinIORootUser        string
	MinIORootPassword    string
	MinIOBucketArtifacts string
	MinIOBucketStatic    string

	// Mailpit & Fake Provider
	MailpitSMTPPort      string
	MailpitUIURL         string
	FakeProviderEndpoint string
}

// LoadConfigFromEnv loads configuration from environment variables with sensible defaults for local development.
func LoadConfigFromEnv() *Config {
	return &Config{
		TenantID: getEnv("TENANT_ID", "default"),

		PostgresHost: getEnv("POSTGRES_HOST", "127.0.0.1"),
		PostgresPort: getEnv("POSTGRES_PORT", "5432"),
		PostgresUser: getEnv("POSTGRES_USER", "journey"),
		PostgresPass: getEnv("POSTGRES_PASSWORD", "journey_pass"),
		PostgresDB:   getEnv("POSTGRES_DB", "journeydb"),

		ClickHouseHost:     getEnv("CLICKHOUSE_HOST", "127.0.0.1"),
		ClickHouseHTTPPort: getEnv("CLICKHOUSE_HTTP_PORT", "8123"),
		ClickHouseUser:     getEnv("CLICKHOUSE_USER", "journey"),
		ClickHousePass:     getEnv("CLICKHOUSE_PASSWORD", "journey_pass"),
		ClickHouseDB:       getEnv("CLICKHOUSE_DB", "journeydb"),

		KafkaBootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "127.0.0.1:9092"),
		SchemaRegistryURL:     getEnv("SCHEMA_REGISTRY_URL", "http://127.0.0.1:8081"),

		TemporalHostPort:  getEnv("TEMPORAL_HOST_PORT", "127.0.0.1:7233"),
		TemporalNamespace: getEnv("TEMPORAL_NAMESPACE", "default"),

		MinIOEndpoint:        getEnv("MINIO_ENDPOINT", "http://127.0.0.1:9000"),
		MinIORootUser:        getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinIORootPassword:    getEnv("MINIO_ROOT_PASSWORD", "minioadmin"),
		MinIOBucketArtifacts: getEnv("MINIO_BUCKET_ARTIFACTS", "journey-artifacts"),
		MinIOBucketStatic:    getEnv("MINIO_BUCKET_STATIC_LISTS", "journey-static-lists"),

		MailpitSMTPPort:      getEnv("MAILPIT_SMTP_PORT", "1025"),
		MailpitUIURL:         getEnv("MAILPIT_UI_URL", "http://127.0.0.1:8025"),
		FakeProviderEndpoint: getEnv("FAKE_PROVIDER_ENDPOINT", "http://127.0.0.1:8082"),
	}
}

// PostgresDSN constructs PostgreSQL connection string.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser, c.PostgresPass, c.PostgresHost, c.PostgresPort, c.PostgresDB)
}

// ServiceStack encapsulates wired Go components and local infrastructure connections.
type ServiceStack struct {
	Config          *Config
	Repo            postgres.Repository
	ObjectStore     object.ObjectStore
	AnalyticsStore  analytics.Repository
	Projector       *Projector
	AudienceEval    *testaudience.Evaluator
	Activities      *activities.Activities
	Logger          *slog.Logger
}

// NewServiceStack initializes a wired Go service stack with given repository or in-memory fallback.
func NewServiceStack(ctx context.Context, cfg *Config, repo postgres.Repository) (*ServiceStack, error) {
	if cfg == nil {
		cfg = LoadConfigFromEnv()
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if repo == nil {
		repo = postgres.NewMemoryRepository()
	}

	objStore := object.NewMemoryStore()
	analyticsStore := analytics.NewMemoryRepository()
	projector := NewProjector(repo)
	audienceEval := testaudience.New()
	acts := activities.NewActivities()

	stack := &ServiceStack{
		Config:         cfg,
		Repo:           repo,
		ObjectStore:    objStore,
		AnalyticsStore: analyticsStore,
		Projector:      projector,
		AudienceEval:   audienceEval,
		Activities:     acts,
		Logger:         logger,
	}

	return stack, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
