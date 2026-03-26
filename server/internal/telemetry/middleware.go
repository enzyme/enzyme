package telemetry

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Middleware returns a chi-compatible HTTP middleware that creates spans
// for incoming requests using OpenTelemetry.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Use a static fallback name to avoid high-cardinality span names from
		// unmatched routes (bots, scanners, etc). The SpanRenameMiddleware below
		// updates the name after chi resolves the route pattern.
		return otelhttp.NewMiddleware("http.request")(next)
	}
}

// SpanRenameMiddleware runs after chi has matched the route and updates the
// span name to use the route pattern (e.g. "GET /api/workspaces/{wid}/channels")
// instead of the raw URL path, keeping span cardinality low.
func SpanRenameMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			rctx := chi.RouteContext(r.Context())
			if rctx != nil && rctx.RoutePattern() != "" {
				span := trace.SpanFromContext(r.Context())
				span.SetName(r.Method + " " + rctx.RoutePattern())
			}
		})
	}
}
