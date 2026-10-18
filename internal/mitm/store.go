package mitm

import (
	"sync"
	"sync/atomic"
)

// DefaultMaxFlows is how many flows the store keeps before dropping the
// oldest.
const DefaultMaxFlows = 5000

// DefaultMaxBytes is how many body bytes the store keeps before dropping
// the oldest flows.
const DefaultMaxBytes int64 = 256 << 20

// Store keeps captured flows in memory, oldest first, up to a number of
// flows and a total body size.
type Store struct {
	mu       sync.RWMutex
	max      int
	maxBytes int64
	bytes    int64 // request and response bodies held
	flows    []*Flow
	index    map[uint64]int // flow ID -> position in flows, offset by base
	base     int            // flows dropped from the front so far

	nextID  atomic.Uint64
	version atomic.Uint64

	onChange func()
}

// NewStore returns a store keeping at most max flows; max <= 0 uses
// DefaultMaxFlows.
func NewStore(max int) *Store {
	if max <= 0 {
		max = DefaultMaxFlows
	}
	return &Store{max: max, maxBytes: DefaultMaxBytes, index: map[uint64]int{}}
}

// SetOnChange registers fn to run after each change, from the goroutine
// that made it. It must not block.
func (s *Store) SetOnChange(fn func()) {
	s.mu.Lock()
	s.onChange = fn
	s.mu.Unlock()
}

// SetLimits changes how many flows and how many body bytes the store
// keeps, dropping the oldest flows if needed. Values <= 0 use the
// defaults.
func (s *Store) SetLimits(maxFlows int, maxBytes int64) {
	if maxFlows <= 0 {
		maxFlows = DefaultMaxFlows
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	s.mu.Lock()
	s.max = maxFlows
	s.maxBytes = maxBytes
	s.trimLocked()
	s.mu.Unlock()
	s.changed()
}

// Version increases on every change; compare it to skip rebuilding views.
func (s *Store) Version() uint64 { return s.version.Load() }

// Len is the number of flows held.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.flows)
}

// Bytes is the size of the bodies held.
func (s *Store) Bytes() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bytes
}

// Add stores f, assigning its ID, and returns that ID.
func (s *Store) Add(f *Flow) uint64 {
	f.ID = s.nextID.Add(1)
	s.mu.Lock()
	s.index[f.ID] = s.base + len(s.flows)
	s.flows = append(s.flows, f)
	s.bytes += bodyBytes(f)
	s.trimLocked()
	s.mu.Unlock()
	s.changed()
	return f.ID
}

// Update replaces the flow with ID id by the result of fn, which gets a
// copy to change. It does nothing when the flow has been dropped.
func (s *Store) Update(id uint64, fn func(f *Flow)) {
	s.mu.Lock()
	pos, ok := s.index[id]
	if !ok {
		s.mu.Unlock()
		return
	}
	old := s.flows[pos-s.base]
	cp := *old
	fn(&cp)
	cp.ID = id
	s.flows[pos-s.base] = &cp
	// A finished flow brings its bodies, which can tip the store over its
	// byte limit.
	s.bytes += bodyBytes(&cp) - bodyBytes(old)
	s.trimLocked()
	s.mu.Unlock()
	s.changed()
}

// Get returns the flow with ID id.
func (s *Store) Get(id uint64) (*Flow, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pos, ok := s.index[id]
	if !ok {
		return nil, false
	}
	return s.flows[pos-s.base], true
}

// Snapshot returns the flows held, oldest first. The slice is a copy;
// the flows themselves are shared and must not be modified.
func (s *Store) Snapshot() []*Flow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Flow, len(s.flows))
	copy(out, s.flows)
	return out
}

// Clear drops every flow.
func (s *Store) Clear() {
	s.mu.Lock()
	s.base += len(s.flows)
	s.flows = nil
	s.bytes = 0
	s.index = map[uint64]int{}
	s.mu.Unlock()
	s.changed()
}

// Remove drops the flows whose IDs are in ids.
func (s *Store) Remove(ids map[uint64]bool) {
	s.mu.Lock()
	kept := s.flows[:0:0]
	for _, f := range s.flows {
		if ids[f.ID] {
			s.bytes -= bodyBytes(f)
			continue
		}
		kept = append(kept, f)
	}
	s.base += len(s.flows)
	s.flows = kept
	s.index = make(map[uint64]int, len(kept))
	for i, f := range kept {
		s.index[f.ID] = s.base + i
	}
	s.mu.Unlock()
	s.changed()
}

// trimLocked drops the oldest flows until both limits hold. The newest
// flow always stays, even when its bodies alone are over the byte limit.
func (s *Store) trimLocked() {
	over := max(len(s.flows)-s.max, 0)
	bytes := s.bytes
	for _, f := range s.flows[:over] {
		bytes -= bodyBytes(f)
	}
	for over < len(s.flows)-1 && s.maxBytes > 0 && bytes > s.maxBytes {
		bytes -= bodyBytes(s.flows[over])
		over++
	}
	if over == 0 {
		return
	}
	for _, f := range s.flows[:over] {
		delete(s.index, f.ID)
	}
	s.bytes = bytes
	// Copy rather than reslice so the dropped flows can be collected.
	s.flows = append([]*Flow(nil), s.flows[over:]...)
	s.base += over
}

func (s *Store) changed() {
	s.version.Add(1)
	s.mu.RLock()
	fn := s.onChange
	s.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

func bodyBytes(f *Flow) int64 { return int64(len(f.ReqBody) + len(f.RespBody)) }
