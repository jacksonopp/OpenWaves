package adminui

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// Handler returns an http.Handler that serves the embedded admin SPA.
//
// In dev mode (built with -tags admindev), if ADMINUI_DEV_PROXY is set, all
// requests are reverse-proxied to that URL (e.g. http://localhost:5173).
//
// In production mode (default build), the compiled React app embedded in
// dist/ is served directly. Any path that doesn't match a real asset falls
// back to dist/index.html so that React Router can handle client-side routing.
func Handler() http.Handler {
	if dist == nil {
		// Dev mode: proxy to Vite dev server.
		devProxy := os.Getenv("ADMINUI_DEV_PROXY")
		if devProxy == "" {
			devProxy = "http://localhost:5173"
		}
		target, err := url.Parse(devProxy)
		if err != nil {
			panic("adminui: invalid ADMINUI_DEV_PROXY: " + err.Error())
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Host = target.Host
			proxy.ServeHTTP(w, r)
		})
	}

	// Serve the embedded dist/ subtree. Strip the /admin/ui prefix so that the
	// file system paths (e.g. dist/index.html) are resolved correctly.
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic("adminui: failed to sub embedded FS: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the /admin/ui prefix.
		path := strings.TrimPrefix(r.URL.Path, "/admin/ui")
		if path == "" {
			path = "/"
		}

		// Try to serve the file directly; if it doesn't exist, fall back to index.html.
		f, err := sub.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			// Serve the SPA root so React Router can handle the path.
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/admin/ui/index.html"
			fileServer.ServeHTTP(w, r2)
			return
		}
		f.Close()

		r2 := r.Clone(r.Context())
		r2.URL.Path = "/admin/ui" + path
		fileServer.ServeHTTP(w, r2)
	})
}
