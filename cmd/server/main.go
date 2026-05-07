package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwave/internal/actor"
	"github.com/jacksonopp/openwave/static"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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

	router.HandleFunc("/stations/{username}", func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["username"]
		if username == "" {
			http.NotFound(w, r)
			return
		}

		host := r.Host
		if host == "" {
			host = "localhost:8080"
		}
		scheme := "http"
		base := fmt.Sprintf("%s://%s/stations/%s", scheme, host, username)

		s := actor.NewStation(base, username, username)
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
		s.StationURI = fmt.Sprintf("openwaves://%s@%s", username, host)

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
