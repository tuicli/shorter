package bot

import (
	"hash/fnv"
	"strconv"
	"sync"
	"time"

	"github.com/tuicli/shorter/internal/app"
)

type stateKind string

const (
	stateAddLinksConfirm stateKind = "add_links_confirm"
	stateSearchInput     stateKind = "search_input"
	stateSearchResults   stateKind = "search_results"
)

type stateEntry struct {
	Kind      stateKind
	Preview   app.BulkPreview
	Query     string
	ExpiresAt time.Time
}

type stateStore struct {
	mu  sync.RWMutex
	ttl time.Duration
	now func() time.Time
	by  map[int64]stateEntry
}

func newStateStore(ttl time.Duration) *stateStore {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &stateStore{
		ttl: ttl,
		now: time.Now,
		by:  map[int64]stateEntry{},
	}
}

func (s *stateStore) set(key int64, entry stateEntry) {
	if key == 0 {
		return
	}
	entry.ExpiresAt = s.now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.by[key] = entry
}

func (s *stateStore) get(key int64) (stateEntry, bool) {
	if key == 0 {
		return stateEntry{}, false
	}

	s.mu.RLock()
	entry, ok := s.by[key]
	s.mu.RUnlock()
	if !ok {
		return stateEntry{}, false
	}
	if !s.now().Before(entry.ExpiresAt) {
		s.clear(key)
		return stateEntry{}, false
	}
	return entry, true
}

func (s *stateStore) take(key int64, kind stateKind) (stateEntry, bool) {
	if key == 0 {
		return stateEntry{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.by[key]
	if !ok {
		return stateEntry{}, false
	}
	if !s.now().Before(entry.ExpiresAt) {
		delete(s.by, key)
		return stateEntry{}, false
	}
	if entry.Kind != kind {
		return stateEntry{}, false
	}

	delete(s.by, key)
	return entry, true
}

func (s *stateStore) clear(key int64) {
	if key == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.by, key)
}

type messageTracker struct {
	mu sync.Mutex
	by map[int64]int
}

func newMessageTracker() *messageTracker {
	return &messageTracker{by: map[int64]int{}}
}

func (t *messageTracker) set(key int64, messageID int) {
	if key == 0 || messageID == 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.by[key] = messageID
}

func (t *messageTracker) get(key int64) (int, bool) {
	if key == 0 {
		return 0, false
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	messageID, ok := t.by[key]
	return messageID, ok
}

func (t *messageTracker) clearIf(key int64, messageID int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.by[key] == messageID {
		delete(t.by, key)
	}
}

func scopeKey(userID int64, chatID int64) int64 {
	if userID == 0 {
		return 0
	}
	if chatID == 0 {
		return userID
	}

	h := fnv.New64a()
	_, _ = h.Write(strconv.AppendInt(nil, userID, 10))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(strconv.AppendInt(nil, chatID, 10))
	key := int64(h.Sum64() & 0x7fffffffffffffff)
	if key == 0 {
		return userID
	}
	return key
}
