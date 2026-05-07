package main

import (
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/config"
	"github.com/jacksonopp/openwaves/internal/hls"
	"github.com/jacksonopp/openwaves/internal/inbox"
	"github.com/jacksonopp/openwaves/internal/keystore"
	"github.com/jacksonopp/openwaves/internal/relay"
)

// Configuration is entirely via environment variables — no config.yaml needed.
//
//	SOURCE_URL      required  Full base URL of the remote station to relay
//	                          e.g. http://localhost:8080/stations/morning-vibes
//	LOCAL_USERNAME  required  Username for this relay station, e.g. morning-vibes
//	PORT            optional  Port to listen on (default: 8081)
//	DOMAIN          optional  Public hostname:port (default: localhost:<PORT>)
//	SCHEME          optional  http or https (default: http)
//	RELAY_POLICY    optional  open | allowlist | closed (default: open)
//	KEYS_DIR        optional  Directory for RSA key files (default: keys-relay)
func main() {
	sourceURL := mustEnv("SOURCE_URL")
	username := mustEnv("LOCAL_USERNAME")

	port := envOr("PORT", "8081")
	scheme := envOr("SCHEME", "http")
	domain := envOr("DOMAIN", "localhost:"+port)
	relayPolicy := envOr("RELAY_POLICY", "open")
	keysDir := envOr("KEYS_DIR", "keys-relay")

	// Build a minimal in-memory config — no config.yaml required.
	cfg := &config.Config{
		Domain:       domain,
		Scheme:       scheme,
		Registration: config.AdminOnly,
		KeysDir:      keysDir,
		Stations: []config.StationConfig{
			{
				Username:    username,
				Name:        username,
				RelayPolicy: relayPolicy,
			},
		},
	}

	privKey, pubKeyPEM, err := keystore.LoadOrGenerate(username, keysDir)
	if err != nil {
		log.Fatalf("failed to load/generate key: %v", err)
	}
	privKeys := map[string]*rsa.PrivateKey{username: privKey}
	pubKeyPEMs := map[string]string{username: pubKeyPEM}

	store := hls.NewStore(10)
	followerStore := inbox.NewFollowerStore()
	relayMgr := relay.NewManager(store, privKeys)

	selfURL := cfg.BaseURL() + "/stations/" + username
	if err := relayMgr.StartRelay(username, sourceURL, selfURL); err != nil {
		log.Fatalf("failed to start relay: %v", err)
	}
	log.Printf("Relaying %s → local station %q", sourceURL, username)
	log.Printf("Self URL: %s", selfURL)

	router := mux.NewRouter()

	// Station actor — so the source can fetch our public key for verification
	router.HandleFunc("/stations/{username}", stationActorHandler(cfg, store, pubKeyPEMs)).Methods(http.MethodGet)

	// Inbox — receives TerminateStream from source, ProofOfListen ACKs, etc.
	router.HandleFunc("/stations/{username}/inbox", inbox.Handler(cfg, store, followerStore, relayMgr.StopRelay)).Methods(http.MethodPost)

	// HLS — re-serve downloaded+verified segments to local listeners
	router.HandleFunc("/stations/{username}/hls/stream.m3u8", relayMgr.ListenerMiddleware(username, hls.ManifestHandler(cfg, store, 6))).Methods(http.MethodGet)
	router.HandleFunc("/stations/{username}/hls/{segment:[^/]+\\.ts}", hls.SegmentHandler(cfg, store)).Methods(http.MethodGet)
	router.HandleFunc("/stations/{username}/hls/{segment:[^/]+\\.ts}.sig", hls.SigHandler(cfg, store)).Methods(http.MethodGet)

	log.Printf("OpenWaves relay listening on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func stationActorHandler(cfg *config.Config, store *hls.Store, pubKeyPEMs map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		registry := cfg.Registry()
		if _, ok := registry[username]; !ok {
			http.NotFound(w, r)
			return
		}

		base := cfg.BaseURL() + "/stations/" + username
		actor := map[string]interface{}{
			"@context":          []string{"https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1"},
			"type":              "Service",
			"id":                base,
			"name":              username,
			"inbox":             base + "/inbox",
			"isLive":            len(store.Segments(username)) > 0,
			"publicKey": map[string]string{
				"id":           base + "#main-key",
				"owner":        base,
				"publicKeyPem": pubKeyPEMs[username],
			},
		}

		w.Header().Set("Content-Type", "application/activity+json")
		if err := json.NewEncoder(w).Encode(actor); err != nil {
			log.Printf("relay: encode actor: %v", err)
		}
	}
}
