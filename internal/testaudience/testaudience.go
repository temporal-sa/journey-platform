package testaudience

import (
	"sync"
	"time"
)

// Evaluator evaluates audience inclusion rules across registered static lists.
type Evaluator struct {
	mu    sync.RWMutex
	lists map[string]*StaticList
}

// New creates a new Evaluator.
func New() *Evaluator {
	return &Evaluator{
		lists: make(map[string]*StaticList),
	}
}

// RegisterList registers an immutable static list into the evaluator.
func (e *Evaluator) RegisterList(list *StaticList) {
	if list == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lists[list.ListID] = list
}

// RemoveList unregisters a static list from the evaluator.
func (e *Evaluator) RemoveList(listID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.lists, listID)
}

// GetList retrieves a registered static list by ID.
func (e *Evaluator) GetList(listID string) (*StaticList, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	list, ok := e.lists[listID]
	return list, ok
}

// Matches checks if a user matches the test audience.
func (e *Evaluator) Matches(userID string) bool {
	if userID == "" {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.lists) == 0 {
		return true
	}
	now := time.Now()
	for _, l := range e.lists {
		if l.Contains(userID, now) {
			return true
		}
	}
	return false
}

// MatchesList checks if a user/member matches a specific static list ID.
func (e *Evaluator) MatchesList(listID string, memberKeyOrID string, now time.Time) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	l, ok := e.lists[listID]
	if !ok {
		return false
	}
	return l.Contains(memberKeyOrID, now)
}
