package ratelimit

import (
	"sync"
	"time"
)

// Clock abstracts time for testing.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Rule defines a rate limit for a specific method+path combination.
type Rule struct {
	Method string
	Path   string
	Limit  int
	Window time.Duration
}

// Result contains rate limit status for a request.
type Result struct {
	Limit     int
	Remaining int
	ResetAt   time.Time
	RetryIn   time.Duration
}

type entry struct {
	ruleKey  string
	count    int
	windowAt time.Time
}

// Limiter implements fixed-window rate limiting per IP+method+path.
type Limiter struct {
	mu      sync.Mutex
	rules   map[string]Rule // key: "METHOD:PATH"
	entries map[string]*entry
	clock   Clock
}

// NewLimiter creates a Limiter with the given rules.
func NewLimiter(rules []Rule) *Limiter {
	ruleMap := make(map[string]Rule, len(rules))
	for _, r := range rules {
		ruleMap[r.Method+":"+r.Path] = r
	}
	return &Limiter{
		rules:   ruleMap,
		entries: make(map[string]*entry),
		clock:   realClock{},
	}
}

// Allow checks whether a request from ip to method+path is allowed.
// If no rule matches the method+path, it returns (Result{}, true).
func (l *Limiter) Allow(ip, method, path string) (Result, bool) {
	ruleKey := method + ":" + path
	rule, ok := l.rules[ruleKey]
	if !ok {
		return Result{}, true
	}

	now := l.clock.Now()
	key := ip + ":" + ruleKey

	l.mu.Lock()
	defer l.mu.Unlock()

	e, exists := l.entries[key]
	if !exists || now.Sub(e.windowAt) >= rule.Window {
		// New window
		l.entries[key] = &entry{ruleKey: ruleKey, count: 1, windowAt: now}
		return Result{Limit: rule.Limit, Remaining: rule.Limit - 1, ResetAt: now.Add(rule.Window)}, true
	}

	resetAt := e.windowAt.Add(rule.Window)

	if e.count >= rule.Limit {
		retryIn := rule.Window - now.Sub(e.windowAt)
		return Result{Limit: rule.Limit, Remaining: 0, ResetAt: resetAt, RetryIn: retryIn}, false
	}

	e.count++
	return Result{Limit: rule.Limit, Remaining: rule.Limit - e.count, ResetAt: resetAt}, true
}

// Cleanup removes expired entries. Call periodically to prevent unbounded growth.
func (l *Limiter) Cleanup() {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, e := range l.entries {
		rule, ok := l.rules[e.ruleKey]
		if !ok || now.Sub(e.windowAt) >= rule.Window {
			delete(l.entries, key)
		}
	}
}
