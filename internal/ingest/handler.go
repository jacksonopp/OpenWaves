package ingest

import (
"crypto/rsa"
"io"
"log"
"net/http"
"path/filepath"
"strings"

"github.com/gorilla/mux"
"github.com/jacksonopp/openwaves/internal/config"
"github.com/jacksonopp/openwaves/internal/hls"
)

// Handler returns an http.HandlerFunc for POST /stations/{username}/ingest/{filename}.
// The request body must be the raw bytes of a single .ts HLS segment.
// The server signs the segment and adds it to the store.
func Handler(cfg *config.Config, store *hls.Store, privKeys map[string]*rsa.PrivateKey) http.HandlerFunc {
ingestor := NewSegmentIngestor(store, privKeys)
return func(w http.ResponseWriter, r *http.Request) {
username := mux.Vars(r)["username"]
filename := mux.Vars(r)["filename"]
registry := cfg.Registry()

if _, ok := registry[username]; !ok && cfg.Registration == config.AdminOnly {
http.NotFound(w, r)
return
}
if _, ok := privKeys[username]; !ok {
http.Error(w, "no key for station", http.StatusInternalServerError)
return
}

if stationCfg, ok := registry[username]; ok && stationCfg.IngestKey != "" {
authHeader := r.Header.Get("Authorization")
expected := "Bearer " + stationCfg.IngestKey
if authHeader != expected {
http.Error(w, "unauthorized", http.StatusUnauthorized)
return
}
}

if !strings.HasSuffix(filename, ".ts") {
http.Error(w, "filename must end in .ts", http.StatusBadRequest)
return
}
if filename != filepath.Base(filename) || strings.Contains(filename, "..") {
http.Error(w, "invalid filename", http.StatusBadRequest)
return
}

data, err := io.ReadAll(r.Body)
if err != nil {
http.Error(w, "failed to read body", http.StatusInternalServerError)
return
}

if err := ingestor.AcceptSegment(username, filename, data); err != nil {
log.Printf("ingest: %s/%s: %v", username, filename, err)
http.Error(w, "ingest error", http.StatusInternalServerError)
return
}

w.WriteHeader(http.StatusCreated)
}
}
