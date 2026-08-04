package store

// Store handles data persistence.
type Store struct{}

// New creates a new Store instance.
func New() *Store {
	return &Store{}
}

// Ping checks store availability.
func (s *Store) Ping() error {
	return nil
}
