package hls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/config"
)

// --- Store tests ---

func TestStore_AddAndGet(t *testing.T) {
	store := NewStore(5)
	seg := Segment{Filename: "seg0001.ts", Data: []byte("data"), SeqNum: 1}
	store.Add("alice", seg)

	got, ok := store.Get("alice", "seg0001.ts")
	if !ok {
		t.Fatal("expected segment to be found")
	}
	if got.SeqNum != 1 {
		t.Errorf("got SeqNum %d, want 1", got.SeqNum)
	}

	_, ok = store.Get("alice", "seg0099.ts")
	if ok {
		t.Error("expected missing segment to return false")
	}
}

func TestStore_Eviction(t *testing.T) {
	store := NewStore(2)
	store.Add("bob", Segment{Filename: "seg0001.ts", SeqNum: 1})
	store.Add("bob", Segment{Filename: "seg0002.ts", SeqNum: 2})
	store.Add("bob", Segment{Filename: "seg0003.ts", SeqNum: 3})

	segs := store.Segments("bob")
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].SeqNum != 2 {
		t.Errorf("expected oldest to be evicted; got SeqNum %d", segs[0].SeqNum)
	}
	if segs[1].SeqNum != 3 {
		t.Errorf("expected newest at index 1; got SeqNum %d", segs[1].SeqNum)
	}

	_, ok := store.Get("bob", "seg0001.ts")
	if ok {
		t.Error("evicted segment should not be retrievable")
	}
}

func TestStore_Clear(t *testing.T) {
	store := NewStore(5)
	store.Add("carol", Segment{Filename: "seg0001.ts", SeqNum: 1})
	store.Clear("carol")

	segs := store.Segments("carol")
	if len(segs) != 0 {
		t.Errorf("expected 0 segments after Clear, got %d", len(segs))
	}
}

// --- Signer tests ---

func generateTestKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return key, pubPEM
}

func TestSigner_RoundTrip(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	data := []byte("hello, world")

	sig, err := Sign(key, data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if err := Verify(pubPEM, data, sig); err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestSigner_TamperedData(t *testing.T) {
	key, pubPEM := generateTestKey(t)
	data := []byte("hello, world")

	sig, err := Sign(key, data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	tampered := []byte("hello, WORLD")
	if err := Verify(pubPEM, tampered, sig); err == nil {
		t.Error("expected Verify to fail on tampered data, but it succeeded")
	}
}

// --- Manifest tests ---

func TestManifest_Empty(t *testing.T) {
	store := NewStore(5)
	out := Manifest(store, "dave", "https://example.com/stations/dave/hls", 6)

	if !strings.Contains(out, "#EXTM3U") {
		t.Error("missing #EXTM3U")
	}
	if !strings.Contains(out, "#EXT-X-MEDIA-SEQUENCE:0") {
		t.Error("expected MEDIA-SEQUENCE:0 for empty store")
	}
	if strings.Contains(out, "#EXTINF") {
		t.Error("expected no #EXTINF lines for empty store")
	}
}

func TestManifest_WithSegments(t *testing.T) {
	store := NewStore(5)
	store.Add("eve", Segment{Filename: "seg0042.ts", SeqNum: 42})
	store.Add("eve", Segment{Filename: "seg0043.ts", SeqNum: 43})

	baseURL := "https://example.com/stations/eve/hls"
	out := Manifest(store, "eve", baseURL, 6)

	if !strings.Contains(out, "#EXT-X-MEDIA-SEQUENCE:42") {
		t.Errorf("expected MEDIA-SEQUENCE:42\ngot:\n%s", out)
	}
	if !strings.Contains(out, baseURL+"/seg0042.ts") {
		t.Errorf("expected seg0042.ts URL\ngot:\n%s", out)
	}
	if !strings.Contains(out, baseURL+"/seg0043.ts") {
		t.Errorf("expected seg0043.ts URL\ngot:\n%s", out)
	}
}

func TestManifest_LiveWindow(t *testing.T) {
	store := NewStore(10)
	// Add more segments than the live window (3).
	for i := 1; i <= 6; i++ {
		store.Add("frank", Segment{
			Filename: fmt.Sprintf("seg%010d.ts", i),
			SeqNum:   i,
		})
	}

	baseURL := "https://example.com/stations/frank/hls"
	out := Manifest(store, "frank", baseURL, 6)

	// Only the last 3 segments should appear.
	if !strings.Contains(out, "#EXT-X-MEDIA-SEQUENCE:4") {
		t.Errorf("expected MEDIA-SEQUENCE:4 (live edge start)\ngot:\n%s", out)
	}
	for _, wantSeg := range []int{4, 5, 6} {
		url := fmt.Sprintf("%s/seg%010d.ts", baseURL, wantSeg)
		if !strings.Contains(out, url) {
			t.Errorf("expected %s in manifest\ngot:\n%s", url, out)
		}
	}
	for _, oldSeg := range []int{1, 2, 3} {
		url := fmt.Sprintf("%s/seg%010d.ts", baseURL, oldSeg)
		if strings.Contains(out, url) {
			t.Errorf("old segment %s should not appear in live window manifest\ngot:\n%s", url, out)
		}
	}
	if strings.Contains(out, "#EXT-X-ENDLIST") {
		t.Error("live manifest must not contain #EXT-X-ENDLIST")
	}
}

// --- Handler tests ---

func newTestConfig() *config.Config {
	return &config.Config{
		Domain:       "localhost",
		Scheme:       "http",
		Registration: config.AdminOnly,
		Stations: []config.StationConfig{
			{Username: "alice"},
		},
	}
}

// routeRequest wires a handler into a mux so that mux.Vars work correctly.
func routeRequest(pattern string, handler http.HandlerFunc, method, url string) *httptest.ResponseRecorder {
	router := mux.NewRouter()
	router.HandleFunc(pattern, handler).Methods(method)
	req := httptest.NewRequest(method, url, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestManifestHandler_AdminOnly_Unknown(t *testing.T) {
	cfg := newTestConfig()
	store := NewStore(5)
	handler := ManifestHandler(cfg, store, 6)

	rr := routeRequest(
		"/stations/{username}/hls/stream.m3u8",
		handler,
		http.MethodGet,
		"/stations/unknown/hls/stream.m3u8",
	)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown username in admin_only mode, got %d", rr.Code)
	}
}

func TestSegmentHandler_NotFound(t *testing.T) {
	cfg := newTestConfig()
	store := NewStore(5)
	handler := SegmentHandler(cfg, store)

	rr := routeRequest(
		"/stations/{username}/hls/{segment}",
		handler,
		http.MethodGet,
		"/stations/alice/hls/seg9999.ts",
	)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing segment, got %d", rr.Code)
	}
}
