package main

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/actor"
	"github.com/jacksonopp/openwaves/internal/config"
	"github.com/jacksonopp/openwaves/internal/hls"
	"github.com/jacksonopp/openwaves/internal/ingest"
	"github.com/jacksonopp/openwaves/internal/keystore"
	"github.com/jacksonopp/openwaves/internal/webfinger"
	"github.com/jacksonopp/openwaves/static"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	privKeys, pubKeyPEMs := loadKeys(cfg)

	store := hls.NewStore(10)

	router := mux.NewRouter()

	router.HandleFunc("/ns/openwaves", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/ld+json")
		w.Write(static.OpenWavesContext)
	}).Methods(http.MethodGet)

	router.HandleFunc("/.well-known/webfinger", webfinger.Handler(cfg)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}", stationHandler(cfg, store, pubKeyPEMs)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}/ingest/{filename}", ingest.Handler(cfg, store, privKeys)).Methods(http.MethodPost)

	router.HandleFunc("/stations/{username}/hls/stream.m3u8", hls.ManifestHandler(cfg, store, 6)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}/hls/{segment}", hls.SegmentHandler(cfg, store)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}/hls/{segment}.sig", hls.SigHandler(cfg, store)).Methods(http.MethodGet)

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	log.Printf("OpenWaves server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func loadKeys(cfg *config.Config) (map[string]*rsa.PrivateKey, map[string]string) {
	privKeys := make(map[string]*rsa.PrivateKey)
	pubKeyPEMs := make(map[string]string)
	for _, station := range cfg.Stations {
		priv, pubPEM, err := keystore.LoadOrGenerate(station.Username, cfg.KeysDir)
		if err != nil {
			log.Fatalf("failed to load key for station %s: %v", station.Username, err)
		}
		privKeys[station.Username] = priv
		pubKeyPEMs[station.Username] = pubPEM
	}
	return privKeys, pubKeyPEMs
}

func stationHandler(cfg *config.Config, store *hls.Store, pubKeyPEMs map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		if username == "" {
			http.NotFound(w, r)
			return
		}

		base := cfg.BaseURL() + "/stations/" + username

		registry := cfg.Registry()
		stationCfg, found := registry[username]

		if !found && cfg.Registration == config.AdminOnly {
			http.NotFound(w, r)
			return
		}

		var name string
		if found {
			name = stationCfg.Name
		} else {
			name = username
		}

		s := actor.NewStation(base, name, username)
		s.URL = base
		s.Inbox = base + "/inbox"
		s.Outbox = base + "/outbox"
		s.Followers = base + "/followers"
		s.Following = base + "/following"
		s.PublicKey = actor.PublicKey{
			ID:           base + "#main-key",
			Owner:        base,
			PublicKeyPem: pubKeyPEMs[username],
		}
		s.StationURI = fmt.Sprintf("openwaves://%s@%s", username, cfg.Domain)

		if len(store.Segments(username)) > 0 {
			s.IsLive = true
			s.BroadcastStatus = actor.LIVE
		}

		if found {
			s.Summary = stationCfg.Summary
			s.LicenseTerritory = stationCfg.LicenseTerritory
			s.RelayPolicy = actor.RelayPolicy(stationCfg.RelayPolicy)
		}

		data, err := json.Marshal(s)
		if err != nil {
			log.Printf("error marshaling station: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/activity+json")
		w.Write(data)
	}
}
