package identity_test

import (
	"testing"
	"time"

	"github.com/arseniisemenow/ttbot-core/pkg/identity"
)

func TestS21NickCacheHitWithinTTL(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := identity.NewS21NickCache(7*24*time.Hour, func() time.Time { return now })
	c.Put(100, identity.User{TelegramID: 100, Nickname: "alice_s21", Found: true})
	now = now.Add(6 * 24 * time.Hour) // still inside the 7-day window
	u, ok := c.Get(100)
	if !ok || u.Nickname != "alice_s21" {
		t.Errorf("expected fresh hit; got u=%+v ok=%v", u, ok)
	}
}

func TestS21NickCacheMissAfterTTL(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := identity.NewS21NickCache(7*24*time.Hour, func() time.Time { return now })
	c.Put(100, identity.User{TelegramID: 100, Nickname: "alice_s21", Found: true})
	now = now.Add(8 * 24 * time.Hour) // expired
	if _, ok := c.Get(100); ok {
		t.Errorf("expected expiry miss")
	}
}

func TestS21NickCacheMissOnUnknown(t *testing.T) {
	c := identity.NewS21NickCache(time.Hour, func() time.Time { return time.Now() })
	if _, ok := c.Get(999); ok {
		t.Errorf("expected miss on unknown tid")
	}
}

func TestS21NickCacheCachesNotFound(t *testing.T) {
	// A cached "no S21 nickname registered" still counts as a fresh hit.
	c := identity.NewS21NickCache(time.Hour, func() time.Time { return time.Now() })
	c.Put(100, identity.User{TelegramID: 100, Found: false})
	u, ok := c.Get(100)
	if !ok {
		t.Errorf("expected cached not-found to be a hit")
	}
	if u.Found {
		t.Errorf("expected Found=false, got %+v", u)
	}
}

func TestS21NickCacheInvalidate(t *testing.T) {
	c := identity.NewS21NickCache(time.Hour, func() time.Time { return time.Now() })
	c.Put(100, identity.User{TelegramID: 100, Nickname: "x", Found: true})
	c.Invalidate(100)
	if _, ok := c.Get(100); ok {
		t.Errorf("expected miss after invalidate")
	}
}

// TestS21NickCacheRespectsMaxSize: spraying N+1 distinct tids must not
// grow the cache past N. Direct coverage of the S3 unbounded-growth vuln.
func TestS21NickCacheRespectsMaxSize(t *testing.T) {
	c := identity.NewS21NickCacheWithMax(time.Hour, 3, func() time.Time { return time.Now() })
	c.Put(1, identity.User{TelegramID: 1})
	c.Put(2, identity.User{TelegramID: 2})
	c.Put(3, identity.User{TelegramID: 3})
	c.Put(4, identity.User{TelegramID: 4})
	if c.Size() != 3 {
		t.Errorf("expected size capped at 3, got %d", c.Size())
	}
}

// TestS21NickCacheLRUEvictsOldest: under cap pressure, the least-
// recently-accessed entry is evicted, not just the oldest insertion.
func TestS21NickCacheLRUEvictsOldest(t *testing.T) {
	c := identity.NewS21NickCacheWithMax(time.Hour, 3, func() time.Time { return time.Now() })
	c.Put(1, identity.User{TelegramID: 1})
	c.Put(2, identity.User{TelegramID: 2})
	c.Put(3, identity.User{TelegramID: 3})
	// Promote tid=1 to most-recent; tid=2 is now LRU.
	if _, ok := c.Get(1); !ok {
		t.Fatal("tid=1 should be present")
	}
	c.Put(4, identity.User{TelegramID: 4}) // evicts tid=2
	if _, ok := c.Get(2); ok {
		t.Errorf("tid=2 should have been evicted as LRU")
	}
	for _, tid := range []int64{1, 3, 4} {
		if _, ok := c.Get(tid); !ok {
			t.Errorf("tid=%d should have survived eviction", tid)
		}
	}
}

// TestS21NickCacheRepeatedPutDoesNotGrow: repeat Puts of the same tid
// update in place rather than piling new entries on.
func TestS21NickCacheRepeatedPutDoesNotGrow(t *testing.T) {
	c := identity.NewS21NickCacheWithMax(time.Hour, 5, func() time.Time { return time.Now() })
	for i := 0; i < 100; i++ {
		c.Put(42, identity.User{TelegramID: 42})
	}
	if c.Size() != 1 {
		t.Errorf("repeated Puts should update in place; got size %d", c.Size())
	}
}
