package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/config"
	"github.com/jacksonopp/openwaves/internal/hls"
	"github.com/jacksonopp/openwaves/internal/inbox"
	"github.com/jacksonopp/openwaves/internal/logstream"
	"github.com/jacksonopp/openwaves/internal/relay"
)

// StationStatus is the JSON response type for admin station info.
type StationStatus struct {
	Username      string `json:"username"`
	IsLive        bool   `json:"isLive"`
	SegmentCount  int    `json:"segmentCount"`
	ListenerCount int    `json:"listenerCount"`
	IsRelaying    bool   `json:"isRelaying"`
}

// Handler returns an http.Handler (gorilla/mux sub-router) for all /admin/* routes.
// Mount it at /admin in the main router.
func Handler(cfg *config.Config, store *hls.Store, followerStore *inbox.FollowerStore, relayMgr *relay.Manager, stream *logstream.Stream) http.Handler {
	r := mux.NewRouter()
	r.Use(authMiddleware(cfg.AdminKey))
	r.HandleFunc("/admin/stations", listStationsHandler(cfg, store, relayMgr)).Methods(http.MethodGet)
	r.HandleFunc("/admin/stations/{username}", getStationHandler(cfg, store, relayMgr)).Methods(http.MethodGet)
	r.HandleFunc("/admin/stations/{username}/stream/stop", stopStreamHandler(cfg, store, followerStore, relayMgr)).Methods(http.MethodPost)
	r.HandleFunc("/admin/stations/{username}/stream/start", startStreamHandler(cfg, store)).Methods(http.MethodPost)
	r.HandleFunc("/admin/stations/{username}/relay/start", startRelayHandler(cfg, relayMgr)).Methods(http.MethodPost)
	r.HandleFunc("/admin/stations/{username}/relay/stop", stopRelayHandler(cfg, relayMgr)).Methods(http.MethodPost)
	r.HandleFunc("/admin/logs", stream.Handler()).Methods(http.MethodGet)
	return r
}

func authMiddleware(adminKey string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if adminKey == "" {
				http.Error(w, "admin API disabled", http.StatusForbidden)
				return
			}
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+adminKey {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func stationStatus(username string, store *hls.Store, relayMgr *relay.Manager) StationStatus {
	segs := store.Segments(username)
	return StationStatus{
		Username:      username,
		IsLive:        store.IsLive(username),
		SegmentCount:  len(segs),
		ListenerCount: store.ListenerCount(username),
		IsRelaying:    relayMgr.IsRelaying(username),
	}
}

func listStationsHandler(cfg *config.Config, store *hls.Store, relayMgr *relay.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := cfg.Registry()
		statuses := make([]StationStatus, 0, len(registry))
		for username := range registry {
			statuses = append(statuses, stationStatus(username, store, relayMgr))
		}
		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].Username < statuses[j].Username
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statuses)
	}
}

func getStationHandler(cfg *config.Config, store *hls.Store, relayMgr *relay.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		registry := cfg.Registry()
		if _, ok := registry[username]; !ok {
			http.Error(w, "station not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stationStatus(username, store, relayMgr))
	}
}

func stopStreamHandler(cfg *config.Config, store *hls.Store, followerStore *inbox.FollowerStore, relayMgr *relay.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		registry := cfg.Registry()
		if _, ok := registry[username]; !ok {
			http.Error(w, "station not found", http.StatusNotFound)
			return
		}
		inbox.TerminateStation(username, store, followerStore, nil)
		relayMgr.StopRelay(username)
		log.Printf("admin: stopped stream for station %s", username)
		w.WriteHeader(http.StatusOK)
	}
}

func startStreamHandler(cfg *config.Config, store *hls.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		registry := cfg.Registry()
		if _, ok := registry[username]; !ok {
			http.Error(w, "station not found", http.StatusNotFound)
			return
		}
		store.Resume(username)
		store.Clear(username)
		log.Printf("admin: cleared store for station %s (ready for ingest)", username)
		w.WriteHeader(http.StatusOK)
	}
}

func startRelayHandler(cfg *config.Config, relayMgr *relay.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		registry := cfg.Registry()
		if _, ok := registry[username]; !ok {
			http.Error(w, "station not found", http.StatusNotFound)
			return
		}

		var body struct {
			SourceURL string `json:"source_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.SourceURL) == "" {
			http.Error(w, "missing source_url", http.StatusBadRequest)
			return
		}

		// Accept either a station base URL or an HLS manifest URL — normalize to base.
		sourceURL := strings.TrimSuffix(strings.TrimSpace(body.SourceURL), "/hls/stream.m3u8")

		// Territory check: fetch source actor and verify licenseTerritory.
		if err := checkTerritory(cfg.Territory, sourceURL); err != nil {
			log.Printf("admin: relay denied for %s: %v", username, err)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		selfURL := cfg.BaseURL() + "/stations/" + username
		if err := relayMgr.StartRelay(username, sourceURL, selfURL); err != nil {
			log.Printf("admin: failed to start relay for %s: %v", username, err)
			http.Error(w, "failed to start relay", http.StatusInternalServerError)
			return
		}
		log.Printf("admin: started relay for station %s from %s", username, sourceURL)
		w.WriteHeader(http.StatusOK)
	}
}

// checkTerritory fetches the source station actor and verifies the relay's
// territory is in the source's licenseTerritory list. Returns an error if
// the territory is not allowed.
func checkTerritory(relayTerritory, sourceURL string) error {
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build request to source: %w", err)
	}
	req.Header.Set("Accept", "application/activity+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch source actor: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read source actor: %w", err)
	}

	var actor struct {
		LicenseTerritory []string `json:"licenseTerritory"`
	}
	if err := json.Unmarshal(body, &actor); err != nil {
		return fmt.Errorf("failed to parse source actor: %w", err)
	}

	// No restriction if empty or worldwide.
	if len(actor.LicenseTerritory) == 0 {
		return nil
	}
	for _, t := range actor.LicenseTerritory {
		if t == "*" || strings.EqualFold(t, relayTerritory) {
			return nil
		}
	}
	return fmt.Errorf("stream not licensed for relay in this territory (%s)", relayTerritory)
}

func stopRelayHandler(cfg *config.Config, relayMgr *relay.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		registry := cfg.Registry()
		if _, ok := registry[username]; !ok {
			http.Error(w, "station not found", http.StatusNotFound)
			return
		}
		relayMgr.StopRelay(username)
		log.Printf("admin: stopped relay for station %s", username)
		w.WriteHeader(http.StatusOK)
	}
}
