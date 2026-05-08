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

	// Read index.html once at startup for the SPA fallback. We serve it
	// directly (not via fileServer) because http.FileServer redirects any
	// path ending in "index.html" away, which would cause an infinite loop.
	indexHTML, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic("adminui: dist/index.html not found in embedded FS")
	}

	serveIndex := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the /admin/ui prefix so the file server sees paths relative
		// to the dist/ root (e.g. "/assets/main.js", not "/admin/ui/assets/…").
		fsPath := strings.TrimPrefix(r.URL.Path, "/admin/ui")
		if fsPath == "" {
			fsPath = "/"
		}

		// Directories and missing files both fall back to index.html so that
		// React Router can handle client-side paths.
		openPath := strings.TrimPrefix(fsPath, "/")
		if openPath == "" {
			openPath = "."
		}
		f, err := sub.Open(openPath)
		if err != nil {
			serveIndex(w)
			return
		}
		stat, err := f.Stat()
		f.Close()
		if err != nil || stat.IsDir() {
			serveIndex(w)
			return
		}

		// Serve the static asset with the stripped path.
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + openPath
		fileServer.ServeHTTP(w, r2)
	})
}
