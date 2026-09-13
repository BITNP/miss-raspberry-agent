package tagging

import "sync"

// Store is a concurrency-safe in-memory collection of tag sets keyed by name.
type Store struct {
	mu   sync.RWMutex
	sets map[string]TagSet
}

// NewStore creates an empty tag-set store.
func NewStore() *Store {
	return &Store{sets: make(map[string]TagSet)}
}

// Replace stores set, overwriting any existing set with the same name.
func (s *Store) Replace(set TagSet) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sets := make([]Tag, len(set.Tags))
	copy(sets, set.Tags)
	set.Tags = sets
	s.sets[set.Name] = set
}

// Get returns the set with the given name and whether it exists.
func (s *Store) Get(name string) (TagSet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set, ok := s.sets[name]
	return set, ok
}
