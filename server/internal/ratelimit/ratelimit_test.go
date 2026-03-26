package ratelimit

import (
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{t: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestAllow_WithinLimit(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 3, Window: time.Minute},
	})

	for i := range 3 {
		result, allowed := l.Allow("1.2.3.4", "POST", "/api/auth/login")
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
		if result.Limit != 3 {
			t.Fatalf("expected limit 3, got %d", result.Limit)
		}
		if result.Remaining != 3-i-1 {
			t.Fatalf("request %d: expected remaining %d, got %d", i+1, 3-i-1, result.Remaining)
		}
		if result.ResetAt.IsZero() {
			t.Fatal("expected non-zero ResetAt")
		}
	}
}

func TestAllow_BlockAfterLimit(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 2, Window: time.Minute},
	})

	l.Allow("1.2.3.4", "POST", "/api/auth/login")
	l.Allow("1.2.3.4", "POST", "/api/auth/login")

	result, allowed := l.Allow("1.2.3.4", "POST", "/api/auth/login")
	if allowed {
		t.Fatal("third request should be blocked")
	}
	if result.Remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", result.Remaining)
	}
	if result.RetryIn <= 0 {
		t.Fatal("expected positive RetryIn")
	}
}

func TestAllow_DifferentIPsIndependent(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})

	_, allowed := l.Allow("1.1.1.1", "POST", "/api/auth/login")
	if !allowed {
		t.Fatal("first IP first request should be allowed")
	}

	_, allowed = l.Allow("2.2.2.2", "POST", "/api/auth/login")
	if !allowed {
		t.Fatal("second IP first request should be allowed")
	}

	_, allowed = l.Allow("1.1.1.1", "POST", "/api/auth/login")
	if allowed {
		t.Fatal("first IP second request should be blocked")
	}
}

func TestAllow_UnmatchedPathPassesThrough(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})

	result, allowed := l.Allow("1.2.3.4", "GET", "/api/workspaces")
	if !allowed {
		t.Fatal("unmatched path should be allowed")
	}
	if result.Limit != 0 {
		t.Fatal("unmatched path should have zero limit (no rule matched)")
	}
}

func TestAllow_WindowReset(t *testing.T) {
	clock := newFakeClock(time.Now())
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})
	l.clock = clock

	l.Allow("1.2.3.4", "POST", "/api/auth/login")
	_, allowed := l.Allow("1.2.3.4", "POST", "/api/auth/login")
	if allowed {
		t.Fatal("should be blocked within window")
	}

	// Advance time past the window
	clock.Advance(time.Minute + time.Second)

	_, allowed = l.Allow("1.2.3.4", "POST", "/api/auth/login")
	if !allowed {
		t.Fatal("should be allowed after window reset")
	}
}

func TestCleanup_RemovesExpiredEntries(t *testing.T) {
	clock := newFakeClock(time.Now())
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 5, Window: time.Minute},
	})
	l.clock = clock

	l.Allow("1.2.3.4", "POST", "/api/auth/login")
	l.Allow("5.6.7.8", "POST", "/api/auth/login")

	if len(l.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(l.entries))
	}

	// Advance past window and cleanup
	clock.Advance(2 * time.Minute)
	l.Cleanup()

	if len(l.entries) != 0 {
		t.Fatalf("expected 0 entries after cleanup, got %d", len(l.entries))
	}
}

func TestAllow_MultipleRules(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 2, Window: time.Minute},
		{Method: "POST", Path: "/api/auth/register", Limit: 1, Window: time.Hour},
	})

	// Use up register limit
	l.Allow("1.2.3.4", "POST", "/api/auth/register")
	_, allowed := l.Allow("1.2.3.4", "POST", "/api/auth/register")
	if allowed {
		t.Fatal("register should be blocked")
	}

	// Login should still work
	_, allowed = l.Allow("1.2.3.4", "POST", "/api/auth/login")
	if !allowed {
		t.Fatal("login should still be allowed")
	}
}

func TestAllow_IPv6(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})

	_, allowed := l.Allow("::1", "POST", "/api/auth/login")
	if !allowed {
		t.Fatal("first IPv6 request should be allowed")
	}

	_, allowed = l.Allow("::1", "POST", "/api/auth/login")
	if allowed {
		t.Fatal("second IPv6 request should be blocked")
	}

	// Different IPv6 address should be independent
	_, allowed = l.Allow("2001:db8::1", "POST", "/api/auth/login")
	if !allowed {
		t.Fatal("different IPv6 address should be allowed")
	}
}

func TestAllow_ConcurrentAccess(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 100, Window: time.Minute},
	})

	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := l.Allow("1.2.3.4", "POST", "/api/auth/login")
			allowed <- ok
		}()
	}

	wg.Wait()
	close(allowed)

	allowedCount := 0
	for ok := range allowed {
		if ok {
			allowedCount++
		}
	}

	if allowedCount != 100 {
		t.Fatalf("expected exactly 100 allowed, got %d", allowedCount)
	}
}
