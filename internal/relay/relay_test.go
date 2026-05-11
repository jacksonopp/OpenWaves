package relay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jacksonopp/openwaves/internal/hls"
	"github.com/jacksonopp/openwaves/internal/keystore"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return key
}

func pubKeyPEM(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func signData(t *testing.T, key *rsa.PrivateKey, data []byte) []byte {
	t.Helper()
	h := sha256.Sum256(data)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		t.Fatalf("failed to sign data: %v", err)
	}
	return sig
}

// newTestStore creates a keystore.Store with keys generated for each username.
func newTestStore(t *testing.T, usernames ...string) *keystore.Store {
	t.Helper()
	ks := keystore.NewStore(t.TempDir())
	for _, u := range usernames {
		if err := ks.Load(u); err != nil {
			t.Fatalf("keystore.Load(%q): %v", u, err)
		}
	}
	return ks
}

func TestManager_StartStop(t *testing.T) {
	store := hls.NewStore(10)
	mgr := NewManager(store, newTestStore(t, "alice"))

	// Use a no-op server so the session doesn't error on network calls
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := mgr.StartRelay("alice", srv.URL, srv.URL+"/stations/alice"); err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	if !mgr.IsRelaying("alice") {
		t.Fatal("expected IsRelaying=true after StartRelay")
	}

	mgr.StopRelay("alice")
	if mgr.IsRelaying("alice") {
		t.Fatal("expected IsRelaying=false after StopRelay")
	}
}

func TestPoller_FetchesSegments(t *testing.T) {
	key := generateTestKey(t)
	segData := []byte("fake-segment-data")
	sigBytes := signData(t, key, segData)
	pubPEM := pubKeyPEM(t, key)

	actorJSON, _ := json.Marshal(map[string]interface{}{
		"publicKey": map[string]string{
			"publicKeyPem": pubPEM,
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hls/stream.m3u8":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-VERSION:3\nseg0.ts\n"))
		case "/hls/seg0.ts":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(segData)
		case "/hls/seg0.ts.sig":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(sigBytes)
		case "/":
			w.Header().Set("Content-Type", "application/activity+json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(actorJSON)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	store := hls.NewStore(10)
	mgr := NewManager(store, newTestStore(t, "bob"))

	if err := mgr.StartRelay("bob", srv.URL, srv.URL+"/stations/bob"); err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	defer mgr.StopRelay("bob")

	// Wait up to 6 seconds for a poll cycle to store the segment.
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		segs := store.Segments("bob")
		if len(segs) > 0 {
			if segs[0].Filename != "seg0.ts" {
				t.Errorf("expected filename seg0.ts, got %s", segs[0].Filename)
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("timed out waiting for segment to be stored")
}

func TestManager_StartRelay_ReplacesExisting(t *testing.T) {
	store := hls.NewStore(10)
	mgr := NewManager(store, newTestStore(t, "carol"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := mgr.StartRelay("carol", srv.URL, srv.URL+"/stations/carol"); err != nil {
		t.Fatalf("first StartRelay: %v", err)
	}
	first := mgr.sessions["carol"]

	if err := mgr.StartRelay("carol", srv.URL, srv.URL+"/stations/carol"); err != nil {
		t.Fatalf("second StartRelay: %v", err)
	}
	second := mgr.sessions["carol"]

	if first == second {
		t.Fatal("expected a new session to replace the old one")
	}

	mgr.mu.Lock()
	count := len(mgr.sessions)
	mgr.mu.Unlock()

	if count != 1 {
		t.Fatalf("expected 1 active session, got %d", count)
	}

	mgr.StopRelay("carol")
}
