package web

import (
	"io/fs"
	"net/http"
	"strings"
)

// Handler returns an http.Handler that serves the embedded SPA.
// Static files are served directly; all other paths fall back to index.html
// so that React Router can handle client-side routing.
func Handler() http.Handler {
	// Strip the "dist" prefix from the embedded filesystem
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic("web: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

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
				// Always revalidate index.html so users get new deploys
				w.Header().Set("Cache-Control", "no-cache")
			}

			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found: serve index.html for SPA routing
		r.URL.Path = "/"
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})
}
