package ingest

import (
"bytes"
"crypto/rand"
"crypto/rsa"
"net/http"
"net/http/httptest"
"testing"

"github.com/gorilla/mux"
"github.com/jacksonopp/openwaves/internal/config"
"github.com/jacksonopp/openwaves/internal/hls"
)

func newTestRouter(cfg *config.Config, store *hls.Store, privKeys map[string]*rsa.PrivateKey) *mux.Router {
r := mux.NewRouter()
r.Handle("/stations/{username}/ingest/{filename}", Handler(cfg, store, privKeys)).Methods(http.MethodPost)
return r
}

func generateKey(t *testing.T) *rsa.PrivateKey {
t.Helper()
key, err := rsa.GenerateKey(rand.Reader, 2048)
if err != nil {
t.Fatalf("rsa.GenerateKey: %v", err)
}
return key
}

func TestHandler_MethodNotAllowed(t *testing.T) {
cfg := &config.Config{
Domain:       "example.com",
Scheme:       "https",
Registration: config.AdminOnly,
}
store := hls.NewStore(10)
privKeys := map[string]*rsa.PrivateKey{}

r := newTestRouter(cfg, store, privKeys)
req := httptest.NewRequest(http.MethodGet, "/stations/alice/ingest/seg0001.ts", nil)
rr := httptest.NewRecorder()
r.ServeHTTP(rr, req)

if rr.Code != http.StatusMethodNotAllowed {
t.Errorf("expected 405, got %d", rr.Code)
}
}

func TestHandler_UnknownStation_AdminOnly(t *testing.T) {
cfg := &config.Config{
Domain:       "example.com",
Scheme:       "https",
Registration: config.AdminOnly,
Stations:     []config.StationConfig{{Username: "alice"}},
}
store := hls.NewStore(10)
privKeys := map[string]*rsa.PrivateKey{}

r := newTestRouter(cfg, store, privKeys)
req := httptest.NewRequest(http.MethodPost, "/stations/unknown/ingest/seg0001.ts", bytes.NewReader([]byte("data")))
rr := httptest.NewRecorder()
r.ServeHTTP(rr, req)

if rr.Code != http.StatusNotFound {
t.Errorf("expected 404, got %d", rr.Code)
}
}

func TestHandler_UnknownStation_Open(t *testing.T) {
cfg := &config.Config{
Domain:       "example.com",
Scheme:       "https",
Registration: config.Open,
}
store := hls.NewStore(10)
privKeys := map[string]*rsa.PrivateKey{} // no key for "unknown" → 500

r := newTestRouter(cfg, store, privKeys)
req := httptest.NewRequest(http.MethodPost, "/stations/unknown/ingest/seg0001.ts", bytes.NewReader([]byte("data")))
rr := httptest.NewRecorder()
r.ServeHTTP(rr, req)

if rr.Code != http.StatusInternalServerError {
t.Errorf("expected 500, got %d", rr.Code)
}
}

func TestHandler_InvalidFilename(t *testing.T) {
cfg := &config.Config{
Domain:       "example.com",
Scheme:       "https",
Registration: config.AdminOnly,
Stations:     []config.StationConfig{{Username: "alice"}},
}
store := hls.NewStore(10)
privKeys := map[string]*rsa.PrivateKey{"alice": generateKey(t)}

r := newTestRouter(cfg, store, privKeys)

for _, bad := range []string{"seg0001.mp4", "seg0001.ts.gz", "seg0001"} {
req := httptest.NewRequest(http.MethodPost, "/stations/alice/ingest/"+bad, bytes.NewReader([]byte("data")))
rr := httptest.NewRecorder()
r.ServeHTTP(rr, req)
if rr.Code != http.StatusBadRequest {
t.Errorf("filename %q: expected 400, got %d", bad, rr.Code)
}
}
}

func TestHandler_AcceptsSegment(t *testing.T) {
cfg := &config.Config{
Domain:       "example.com",
Scheme:       "https",
Registration: config.AdminOnly,
Stations:     []config.StationConfig{{Username: "alice"}},
}
store := hls.NewStore(10)
privKeys := map[string]*rsa.PrivateKey{"alice": generateKey(t)}

r := newTestRouter(cfg, store, privKeys)
body := bytes.NewReader([]byte("fake-ts-data"))
req := httptest.NewRequest(http.MethodPost, "/stations/alice/ingest/seg0001.ts", body)
rr := httptest.NewRecorder()
r.ServeHTTP(rr, req)

if rr.Code != http.StatusCreated {
t.Errorf("expected 201, got %d", rr.Code)
}

segs := store.Segments("alice")
if len(segs) != 1 {
t.Fatalf("expected 1 segment in store, got %d", len(segs))
}
if segs[0].Filename != "seg0001.ts" {
t.Errorf("expected filename seg0001.ts, got %s", segs[0].Filename)
}
if len(segs[0].Signature) == 0 {
t.Error("expected non-empty signature")
}
}

func TestHandler_IngestKey_Valid(t *testing.T) {
	cfg := &config.Config{
		Domain:       "example.com",
		Scheme:       "https",
		Registration: config.AdminOnly,
		Stations:     []config.StationConfig{{Username: "alice", IngestKey: "secret"}},
	}
	store := hls.NewStore(10)
	privKeys := map[string]*rsa.PrivateKey{"alice": generateKey(t)}

	r := newTestRouter(cfg, store, privKeys)
	body := bytes.NewReader([]byte("fake-ts-data"))
	req := httptest.NewRequest(http.MethodPost, "/stations/alice/ingest/seg0001.ts", body)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestHandler_IngestKey_Missing(t *testing.T) {
	cfg := &config.Config{
		Domain:       "example.com",
		Scheme:       "https",
		Registration: config.AdminOnly,
		Stations:     []config.StationConfig{{Username: "alice", IngestKey: "secret"}},
	}
	store := hls.NewStore(10)
	privKeys := map[string]*rsa.PrivateKey{"alice": generateKey(t)}

	r := newTestRouter(cfg, store, privKeys)
	body := bytes.NewReader([]byte("fake-ts-data"))
	req := httptest.NewRequest(http.MethodPost, "/stations/alice/ingest/seg0001.ts", body)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestSegmentIngestor_Stop(t *testing.T) {
store := hls.NewStore(10)
privKeys := map[string]*rsa.PrivateKey{}
ingestor := NewSegmentIngestor(store, privKeys)

store.Add("alice", hls.Segment{Filename: "seg0000.ts", Data: []byte("data"), SeqNum: 0})
if segs := store.Segments("alice"); len(segs) != 1 {
t.Fatalf("expected 1 segment before Stop, got %d", len(segs))
}

if err := ingestor.Stop("alice"); err != nil {
t.Fatalf("Stop returned error: %v", err)
}

if segs := store.Segments("alice"); len(segs) != 0 {
t.Errorf("expected 0 segments after Stop, got %d", len(segs))
}
}
