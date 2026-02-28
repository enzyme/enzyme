package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

// Config holds frontend-relevant settings injected into index.html at runtime.
type Config struct {
	TelemetryEnabled  bool
	TelemetryEndpoint string // default "/v1/traces"
}

// Handler returns an http.Handler that serves the embedded SPA.
// Static files are served directly; all other paths fall back to index.html
// so that React Router can handle client-side routing.
//
// If cfg enables any runtime features, a <script> tag setting
// window.__ENZYME_CONFIG__ is injected into index.html before </head>.
func Handler(cfg Config) http.Handler {
	// Strip the "dist" prefix from the embedded filesystem
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic("web: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	// Build a patched index.html with runtime config injected.
	indexHTML := buildIndexHTML(sub, cfg)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the exact file
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Check if the file exists in the embedded FS
		f, err := sub.Open(path)
		if err == nil {
			_ = f.Close()

			// Set cache headers based on path
			if strings.HasPrefix(path, "assets/") {
				// Vite content-hashed assets: cache forever
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else if path == "index.html" {
				// Serve the patched index.html with runtime config
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(indexHTML) //nolint:errcheck
				return
			}

			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found: serve index.html for SPA routing
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML) //nolint:errcheck
	})
}

// runtimeConfig is the JSON shape of window.__ENZYME_CONFIG__.
type runtimeConfig struct {
	Telemetry *runtimeTelemetry `json:"telemetry,omitempty"`
}

type runtimeTelemetry struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint,omitempty"`
}

// buildIndexHTML reads index.html from the embedded FS and optionally injects
// a <script> tag with runtime config before </head>.
func buildIndexHTML(fsys fs.FS, cfg Config) []byte {
	raw, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		panic("web: reading index.html: " + err.Error())
	}

	if !cfg.TelemetryEnabled {
		return raw
	}

	rc := runtimeConfig{
		Telemetry: &runtimeTelemetry{
			Enabled:  cfg.TelemetryEnabled,
			Endpoint: cfg.TelemetryEndpoint,
		},
	}
	jsonBytes, err := json.Marshal(rc)
	if err != nil {
		panic("web: marshaling runtime config: " + err.Error())
	}

	tag := "<script>window.__ENZYME_CONFIG__=" + string(jsonBytes) + "</script>"
	html := strings.Replace(string(raw), "</head>", tag+"</head>", 1)
	return []byte(html)
}
