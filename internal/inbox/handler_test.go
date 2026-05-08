package inbox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/activity"
	"github.com/jacksonopp/openwaves/internal/config"
)

// makeRouter wraps the Handler in a gorilla/mux router so {username} is populated.
func makeRouter(cfg *config.Config, fs *FollowerStore) http.Handler {
	r := mux.NewRouter()
	r.Handle("/stations/{username}/inbox", Handler(cfg, fs, nil)).Methods(http.MethodPost)
	return r
}

func testConfig(policy string) *config.Config {
	return &config.Config{
		Domain:       "example.com",
		Scheme:       "http",
		Registration: config.AdminOnly,
		Stations: []config.StationConfig{
			{Username: "teststation", RelayPolicy: policy},
		},
	}
}

func postInbox(router http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/stations/teststation/inbox", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/activity+json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestHandler_Follow_Open(t *testing.T) {
	var (
		acceptReceived bool
		mu             sync.Mutex
	)

	// Mock remote server: serves actor JSON and records Accept POSTs.
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Return actor JSON pointing inbox at /inbox on this same server.
			json.NewEncoder(w).Encode(map[string]string{
				"inbox": "http://" + r.Host + "/inbox",
			})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/inbox" {
			mu.Lock()
			acceptReceived = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer remote.Close()

	cfg := testConfig("open")
	fs := NewFollowerStore()
	router := makeRouter(cfg, fs)

	followBody, _ := json.Marshal(activity.Activity{
		Type:  "Follow",
		Actor: remote.URL, // remote actor URL
	})

	rr := postInbox(router, string(followBody))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Accept is sent in a goroutine; give it a moment.
	time.Sleep(50 * time.Millisecond)

	followers := fs.List("teststation")
	if len(followers) != 1 {
		t.Fatalf("expected 1 follower, got %d", len(followers))
	}
	if followers[0].ActorURL != remote.URL {
		t.Errorf("unexpected ActorURL: %s", followers[0].ActorURL)
	}

	mu.Lock()
	defer mu.Unlock()
	if !acceptReceived {
		t.Error("expected Accept to be posted to remote inbox")
	}
}

func TestHandler_Follow_Closed(t *testing.T) {
	var (
		rejectReceived bool
		mu             sync.Mutex
	)

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]string{
				"inbox": "http://" + r.Host + "/inbox",
			})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/inbox" {
			mu.Lock()
			rejectReceived = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer remote.Close()

	cfg := testConfig("closed")
	fs := NewFollowerStore()
	router := makeRouter(cfg, fs)

	followBody, _ := json.Marshal(activity.Activity{
		Type:  "Follow",
		Actor: remote.URL,
	})

	rr := postInbox(router, string(followBody))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	time.Sleep(50 * time.Millisecond)

	followers := fs.List("teststation")
	if len(followers) != 0 {
		t.Errorf("expected 0 followers for closed station, got %d", len(followers))
	}

	mu.Lock()
	defer mu.Unlock()
	if !rejectReceived {
		t.Error("expected Reject to be posted to remote inbox")
	}
}

// TestHandler_TerminateStream verifies that TerminateStream sent to the inbox
// is silently ignored (returns 202) — it is an admin-only action, not an
// activity external servers can trigger.
func TestHandler_TerminateStream(t *testing.T) {
	cfg := testConfig("open")
	fs := NewFollowerStore()
	router := makeRouter(cfg, fs)

	body, _ := json.Marshal(activity.TerminateStream{
		Type:   "TerminateStream",
		Actor:  "http://remote.example.com/stations/relay",
		Object: "http://example.com/stations/teststation",
	})

	rr := postInbox(router, string(body))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 (TerminateStream ignored via inbox), got %d", rr.Code)
	}
}

func TestHandler_ProofOfListen(t *testing.T) {
	cfg := testConfig("open")
	fs := NewFollowerStore()
	router := makeRouter(cfg, fs)

	body, _ := json.Marshal(activity.ProofOfListen{
		Type:          "ProofOfListen",
		Actor:         "http://remote.example.com/stations/relay",
		Object:        "http://example.com/stations/teststation",
		ListenerCount: 5,
		Timestamp:     "2024-01-01T00:00:00Z",
	})

	rr := postInbox(router, string(body))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestHandler_TerminateStream_PropagatestoFollowers(t *testing.T) {
	// TerminateStream sent to inbox should be ignored (202), not propagated.
	// Propagation only happens when triggered by the admin API.
	followerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer followerSrv.Close()

	cfg := testConfig("open")
	fs := NewFollowerStore()
	fs.Add("teststation", Follower{ActorURL: followerSrv.URL, InboxURL: followerSrv.URL + "/inbox"})
	router := makeRouter(cfg, fs)

	body, _ := json.Marshal(activity.TerminateStream{
		Type:   "TerminateStream",
		Actor:  "http://source.example.com/stations/src",
		Object: "http://example.com/stations/teststation",
	})

	rr := postInbox(router, string(body))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 (TerminateStream via inbox ignored), got %d", rr.Code)
	}
}

func TestHandler_UnknownType(t *testing.T) {
	cfg := testConfig("open")
	fs := NewFollowerStore()
	router := makeRouter(cfg, fs)

	body, _ := json.Marshal(activity.Activity{
		Type:  "Create",
		Actor: "http://remote.example.com/actor",
	})

	rr := postInbox(router, string(body))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for unknown type, got %d", rr.Code)
	}
}

func TestHandler_BadJSON(t *testing.T) {
	cfg := testConfig("open")
	fs := NewFollowerStore()
	router := makeRouter(cfg, fs)

	rr := postInbox(router, `{not valid json`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad JSON, got %d", rr.Code)
	}
}
