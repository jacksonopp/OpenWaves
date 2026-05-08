package main

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/actor"
	"github.com/jacksonopp/openwaves/internal/admin"
	"github.com/jacksonopp/openwaves/internal/adminui"
	"github.com/jacksonopp/openwaves/internal/broadcaster"
	"github.com/jacksonopp/openwaves/internal/config"
	"github.com/jacksonopp/openwaves/internal/hls"
	"github.com/jacksonopp/openwaves/internal/inbox"
	"github.com/jacksonopp/openwaves/internal/ingest"
	"github.com/jacksonopp/openwaves/internal/keystore"
	"github.com/jacksonopp/openwaves/internal/logstream"
	"github.com/jacksonopp/openwaves/internal/relay"
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

	logStream := logstream.New()
	log.SetOutput(io.MultiWriter(os.Stderr, logStream))

	followerStore := inbox.NewFollowerStore()
	relayMgr := relay.NewManager(store, privKeys)
	bcMgr := broadcaster.NewManager()

	router := mux.NewRouter()

	// Admin web UI — must be registered before the /admin API prefix.
	router.PathPrefix("/admin/ui").Handler(adminui.Handler())

	router.HandleFunc("/ns/openwaves", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/ld+json")
		w.Write(static.OpenWavesContext)
	}).Methods(http.MethodGet)

	router.HandleFunc("/.well-known/webfinger", webfinger.Handler(cfg)).Methods(http.MethodGet)

	router.HandleFunc("/stations", publicStationsListHandler(cfg, store)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}", stationHandler(cfg, store, pubKeyPEMs)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}/ingest/{filename}", ingest.Handler(cfg, store, privKeys)).Methods(http.MethodPost)

	router.HandleFunc("/stations/{username}/hls/stream.m3u8", hls.ManifestHandler(cfg, store, 6)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}/hls/{segment:[^/]+\\.ts}", hls.SegmentHandler(cfg, store)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}/hls/{segment:[^/]+\\.ts}.sig", hls.SigHandler(cfg, store)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}/inbox", inbox.Handler(cfg, followerStore, nil)).Methods(http.MethodPost)

	router.PathPrefix("/admin").Handler(admin.Handler(cfg, store, followerStore, relayMgr, logStream, bcMgr))

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

func publicStationsListHandler(cfg *config.Config, store *hls.Store) http.HandlerFunc {
	type publicStation struct {
		Username      string `json:"username"`
		Name          string `json:"name"`
		Summary       string `json:"summary"`
		IsLive        bool   `json:"isLive"`
		SegmentCount  int    `json:"segmentCount"`
		ListenerCount int    `json:"listenerCount"`
		HLSURL        string `json:"hlsUrl"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		registry := cfg.Registry()
		stations := make([]publicStation, 0, len(registry))
		for username, sc := range registry {
			segs := store.Segments(username)
			stations = append(stations, publicStation{
				Username:      username,
				Name:          sc.Name,
				Summary:       sc.Summary,
				IsLive:        store.IsLive(username),
				SegmentCount:  len(segs),
				ListenerCount: store.ListenerCount(username),
				HLSURL:        cfg.BaseURL() + "/stations/" + username + "/hls/stream.m3u8",
			})
		}
		sort.Slice(stations, func(i, j int) bool {
			return stations[i].Username < stations[j].Username
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stations)
	}
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

		if store.IsLive(username) {
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
