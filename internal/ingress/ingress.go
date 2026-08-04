package ingress

// Ingress manages event ingestion.
type Ingress struct{}

// New creates a new Ingress.
func New() *Ingress {
	return &Ingress{}
}

// Ingest accepts raw event payloads.
func (i *Ingress) Ingest(payload []byte) error {
	return nil
}
