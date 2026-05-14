package identity

import (
	"container/list"
	"sync"
	"time"
)

// S21NickCache is a process-local cache of S21 nickname records keyed by
// Telegram ID. Entries are valid for `ttl` from the moment they were
// inserted; older entries are treated as cache misses (the next lookup
// fetches fresh from the identity service).
//
// The "S21" prefix in every name in this file is intentional: ttbot deals
// with two unrelated kinds of "nickname" — S21 nicknames (the school's
// user identifier, looked up via the identity service) and Telegram
// @usernames (cached in the per-group participants table). This cache
// only covers the former.
//
// Memory safety: capped at `maxEntries`. When the cap is hit, the least-
// recently-used entry is evicted. Without this bound, a long-lived
// container serving lookups for many distinct telegram_ids could grow
// the cache without limit.
type S21NickCache struct {
	mu         sync.Mutex // protects entries + lru
	ttl        time.Duration
	maxEntries int
	now        func() time.Time
	entries    map[int64]*list.Element // tid → list element
	lru        *list.List              // values are *s21NickEntry; front = most recently used
}

type s21NickEntry struct {
	tid       int64
	user      User
	fetchedAt time.Time
}

// defaultS21NickCacheMax caps the cache at 20000 distinct Telegram IDs.
// School 21 has thousands of users; the cap leaves comfortable room for
// every active player while preventing an unbounded leak.
const defaultS21NickCacheMax = 20000

// NewS21NickCache constructs a cache with the default size cap.
func NewS21NickCache(ttl time.Duration, now func() time.Time) *S21NickCache {
	return NewS21NickCacheWithMax(ttl, defaultS21NickCacheMax, now)
}

// NewS21NickCacheWithMax constructs a cache with a caller-specified size
// cap. Used by tests to exercise the LRU eviction path without inserting
// 20000 entries.
func NewS21NickCacheWithMax(ttl time.Duration, maxEntries int, now func() time.Time) *S21NickCache {
	if now == nil {
		now = time.Now
	}
	if maxEntries < 1 {
		maxEntries = defaultS21NickCacheMax
	}
	return &S21NickCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		now:        now,
		entries:    map[int64]*list.Element{},
		lru:        list.New(),
	}
}

// Get returns the cached identity.User for tid if present and still
// fresh. The bool is false on miss or on expiry — callers should then
// fetch + Put. A cached "not found" record (User.Found == false) is
// treated as a fresh hit too: identity service has already told us this
// telegram_id has no S21 nickname, no reason to ask again until the TTL
// passes. A hit promotes the entry to the front of the LRU list.
func (c *S21NickCache) Get(tid int64) (User, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[tid]
	if !ok {
		return User{}, false
	}
	e := el.Value.(*s21NickEntry)
	if c.now().Sub(e.fetchedAt) > c.ttl {
		c.lru.Remove(el)
		delete(c.entries, tid)
		return User{}, false
	}
	c.lru.MoveToFront(el)
	return e.user, true
}

// Put stores u in the cache, stamping fetchedAt to now. Promotes to the
// front of the LRU list. If the cache is at capacity, the LRU entry is
// evicted.
func (c *S21NickCache) Put(tid int64, u User) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[tid]; ok {
		e := el.Value.(*s21NickEntry)
		e.user = u
		e.fetchedAt = c.now()
		c.lru.MoveToFront(el)
		return
	}
	e := &s21NickEntry{
		tid:       tid,
		user:      u,
		fetchedAt: c.now(),
	}
	el := c.lru.PushFront(e)
	c.entries[tid] = el
	c.evictLocked()
}

// Invalidate removes a single entry. Reserved for the future case where
// the bot gets a strong signal a record changed (e.g. a webhook from the
// identity bot announcing a /provide_nickname). Not currently invoked.
func (c *S21NickCache) Invalidate(tid int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[tid]; ok {
		c.lru.Remove(el)
		delete(c.entries, tid)
	}
}

// Size returns the number of entries currently held. Used by tests; not
// part of the production hot path.
func (c *S21NickCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// evictLocked drops least-recently-used entries until the cache size is
// within the cap. Caller must hold c.mu.
func (c *S21NickCache) evictLocked() {
	for len(c.entries) > c.maxEntries {
		back := c.lru.Back()
		if back == nil {
			return
		}
		c.lru.Remove(back)
		delete(c.entries, back.Value.(*s21NickEntry).tid)
	}
}
