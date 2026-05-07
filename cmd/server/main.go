package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/actor"
	"github.com/jacksonopp/openwaves/internal/config"
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

	router := mux.NewRouter()

	router.HandleFunc("/ns/openwave", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/ld+json")
		w.Write(static.OpenwaveContext)
	}).Methods(http.MethodGet)

	router.HandleFunc("/.well-known/webfinger", webfinger.Handler(cfg)).Methods(http.MethodGet)

	router.HandleFunc("/stations/{username}", func(w http.ResponseWriter, r *http.Request) {
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
			PublicKeyPem: "",
		}
		s.StationURI = fmt.Sprintf("openwaves://%s@%s", username, cfg.Domain)

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
	}).Methods(http.MethodGet)

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	log.Printf("OpenWave server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
