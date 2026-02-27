package telemetry

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Middleware returns a chi-compatible HTTP middleware that creates spans
// for incoming requests using OpenTelemetry.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewMiddleware("http.request",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				// Use the chi route pattern for a cleaner span name
				rctx := chi.RouteContext(r.Context())
				if rctx != nil && rctx.RoutePattern() != "" {
					return r.Method + " " + rctx.RoutePattern()
				}
				return r.Method + " " + r.URL.Path
			}),
		)(next)
	}
}
