package catalog

// Service provides catalog capabilities.
type Service struct{}

// New creates a new Catalog Service instance.
func New() *Service {
	return &Service{}
}

// Version returns the catalog version.
func (s *Service) Version() string {
	return "0.1.0"
}
