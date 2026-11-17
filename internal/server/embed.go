package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var embeddedWebFS embed.FS

// WebHandler returns an http.Handler that serves the embedded web studio assets,
// falling back to index.html for client-side routing.
func WebHandler() http.Handler {
	distFS, err := fs.Sub(embeddedWebFS, "dist")
	if err != nil {
		// Fallback to minimal handler if dist directory not found
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><h1>gridpipe Studio</h1><p>API is active. Build web assets with 'npm run build' inside web/.</p></body></html>`))
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" && !strings.Contains(path, ".") {
			// SPA fallback: rewrite to index.html
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
