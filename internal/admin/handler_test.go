package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jacksonopp/openwaves/internal/broadcaster"
	"github.com/jacksonopp/openwaves/internal/config"
	"github.com/jacksonopp/openwaves/internal/hls"
	"github.com/jacksonopp/openwaves/internal/inbox"
	"github.com/jacksonopp/openwaves/internal/logstream"
	"github.com/jacksonopp/openwaves/internal/relay"
)

func testSetup() (*config.Config, *hls.Store, *inbox.FollowerStore, *relay.Manager, *logstream.Stream, *broadcaster.Manager) {
	cfg := &config.Config{
		Domain:   "example.com",
		Scheme:   "http",
		AdminKey: "test-key",
		Stations: []config.StationConfig{
			{Username: "alice", Name: "Alice Radio", RelayPolicy: "open"},
		},
	}
	store := hls.NewStore(10)
	followerStore := inbox.NewFollowerStore()
	relayMgr := relay.NewManager(store, nil)
	stream := logstream.New()
	bcMgr := broadcaster.NewManager()
	return cfg, store, followerStore, relayMgr, stream, bcMgr
}

func doRequest(h http.Handler, method, path string, body []byte, key string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// 1. Admin disabled (empty AdminKey) → GET /admin/stations → 403
func TestAdminDisabled(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	cfg.AdminKey = ""
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	rr := doRequest(h, http.MethodGet, "/admin/stations", nil, "")
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// 2. Wrong key → 401
func TestWrongKey(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	rr := doRequest(h, http.MethodGet, "/admin/stations", nil, "wrong-key")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// 3. Correct key → GET /admin/stations → 200, JSON with station list
func TestListStations(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	rr := doRequest(h, http.MethodGet, "/admin/stations", nil, "test-key")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var statuses []StationStatus
	if err := json.NewDecoder(rr.Body).Decode(&statuses); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 station, got %d", len(statuses))
	}
	if statuses[0].Username != "alice" {
		t.Errorf("unexpected username: %s", statuses[0].Username)
	}
}

// 4. GET /admin/stations/{username} → correct status
func TestGetStation(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	store.Add("alice", hls.Segment{Filename: "seg0.ts", SeqNum: 0})
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	rr := doRequest(h, http.MethodGet, "/admin/stations/alice", nil, "test-key")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var status StationStatus
	if err := json.NewDecoder(rr.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if status.Username != "alice" {
		t.Errorf("unexpected username: %s", status.Username)
	}
	if !status.IsLive {
		t.Error("expected IsLive=true")
	}
	if status.SegmentCount != 1 {
		t.Errorf("expected SegmentCount=1, got %d", status.SegmentCount)
	}
}

// 5. GET /admin/stations/nonexistent → 404
func TestGetStationNotFound(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	rr := doRequest(h, http.MethodGet, "/admin/stations/nonexistent", nil, "test-key")
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// 6. POST /admin/stations/{username}/stream/stop → 200, store cleared
func TestStopStream(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	store.Add("alice", hls.Segment{Filename: "seg0.ts", SeqNum: 0})
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	rr := doRequest(h, http.MethodPost, "/admin/stations/alice/stream/stop", nil, "test-key")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if segs := store.Segments("alice"); len(segs) != 0 {
		t.Errorf("expected store cleared, got %d segments", len(segs))
	}
}

// 7. POST /admin/stations/{username}/stream/start → 200
func TestStartStream(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	store.Add("alice", hls.Segment{Filename: "seg0.ts", SeqNum: 0})
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	rr := doRequest(h, http.MethodPost, "/admin/stations/alice/stream/start", nil, "test-key")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if segs := store.Segments("alice"); len(segs) != 0 {
		t.Errorf("expected store cleared on start, got %d segments", len(segs))
	}
}

// 8. POST /admin/stations/{username}/relay/start valid body → 200
func TestStartRelay(t *testing.T) {
	// Mock source server that returns an actor with licenseTerritory: ["*"]
	sourceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"licenseTerritory": []string{"*"},
		})
	}))
	defer sourceSrv.Close()

	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	body, _ := json.Marshal(map[string]string{"source_url": sourceSrv.URL})
	rr := doRequest(h, http.MethodPost, "/admin/stations/alice/relay/start", body, "test-key")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

// 9. POST /admin/stations/{username}/relay/start missing source_url → 400
func TestStartRelayMissingSourceURL(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	body, _ := json.Marshal(map[string]string{})
	rr := doRequest(h, http.MethodPost, "/admin/stations/alice/relay/start", body, "test-key")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// 10. POST /admin/stations/{username}/relay/stop → 200
func TestStopRelay(t *testing.T) {
	cfg, store, followerStore, relayMgr, stream, bcMgr := testSetup()
	h := Handler(cfg, store, followerStore, relayMgr, stream, bcMgr)

	rr := doRequest(h, http.MethodPost, "/admin/stations/alice/relay/stop", nil, "test-key")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
