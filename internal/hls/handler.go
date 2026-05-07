package hls

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/config"
)

// ManifestHandler serves GET /stations/{username}/hls/stream.m3u8
func ManifestHandler(cfg *config.Config, store *Store, targetDuration int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		registry := cfg.Registry()

		if _, ok := registry[username]; !ok && cfg.Registration == config.AdminOnly {
			http.NotFound(w, r)
			return
		}

		// Return 503 until at least one segment is available so HLS clients
		// retry instead of treating an empty manifest as a permanent failure.
		if len(store.Segments(username)) == 0 {
			http.Error(w, "stream not yet available", http.StatusServiceUnavailable)
			return
		}

		baseURL := cfg.BaseURL() + "/stations/" + username + "/hls"
		playlist := Manifest(store, username, baseURL, targetDuration)

		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(playlist))
	}
}

// SegmentHandler serves GET /stations/{username}/hls/{segment}
// {segment} must end in .ts
func SegmentHandler(cfg *config.Config, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		filename := mux.Vars(r)["segment"]
		registry := cfg.Registry()

		if _, ok := registry[username]; !ok && cfg.Registration == config.AdminOnly {
			http.NotFound(w, r)
			return
		}

		if !strings.HasSuffix(filename, ".ts") {
			http.NotFound(w, r)
			return
		}

		seg, ok := store.Get(username, filename)
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "video/mp2t")
		w.WriteHeader(http.StatusOK)
		w.Write(seg.Data)
	}
}

// SigHandler serves GET /stations/{username}/hls/{segment}.sig
// {segment} here is the .ts filename without .sig
func SigHandler(cfg *config.Config, store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		// mux captures the segment name without the .sig extension
		filename := mux.Vars(r)["segment"]
		registry := cfg.Registry()

		if _, ok := registry[username]; !ok && cfg.Registration == config.AdminOnly {
			http.NotFound(w, r)
			return
		}

		seg, ok := store.Get(username, filename)
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(seg.Signature)
	}
}
