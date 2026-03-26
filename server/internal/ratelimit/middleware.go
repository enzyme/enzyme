package ratelimit

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
)

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Middleware returns chi middleware that applies rate limiting using the given Limiter.
// If limiter is nil, requests pass through untouched.
func Middleware(limiter *Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if limiter == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := stripPort(r.RemoteAddr)
			result, allowed := limiter.Allow(ip, r.Method, r.URL.Path)

			// Unmatched path — no rate limit headers, just pass through
			if allowed && result.Limit == 0 {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !allowed {
				retryAfter := int(math.Ceil(result.RetryIn.Seconds()))
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(errorResponse{
					Error: errorDetail{
						Code:    "RATE_LIMITED",
						Message: fmt.Sprintf("Too many requests. Try again in %d seconds.", retryAfter),
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// stripPort removes the port from an address (handles both IPv4 and IPv6).
func stripPort(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
