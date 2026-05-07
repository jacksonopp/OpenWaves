package webfinger

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jacksonopp/openwaves/internal/config"
)

type JRDLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href"`
}

type JRD struct {
	Subject string    `json:"subject"`
	Aliases []string  `json:"aliases,omitempty"`
	Links   []JRDLink `json:"links"`
}

func Handler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resource := r.URL.Query().Get("resource")
		if resource == "" {
			http.Error(w, "missing resource parameter", http.StatusBadRequest)
			return
		}

		if !strings.HasPrefix(resource, "acct:") {
			http.Error(w, "unsupported resource format", http.StatusBadRequest)
			return
		}

		acct := strings.TrimPrefix(resource, "acct:")
		parts := strings.Split(acct, "@")
		if len(parts) != 2 {
			http.Error(w, "invalid acct format", http.StatusBadRequest)
			return
		}

		username, domain := parts[0], parts[1]

		if domain != cfg.Domain {
			http.Error(w, "domain mismatch", http.StatusBadRequest)
			return
		}

		registry := cfg.Registry()
		station, found := registry[username]
		if !found {
			if cfg.Registration == config.AdminOnly {
				http.NotFound(w, r)
				return
			}
			station = config.StationConfig{
				Username: username,
				Name:     username,
			}
		}
		_ = station

		actorURL := cfg.BaseURL() + "/stations/" + username

		jrd := JRD{
			Subject: resource,
			Aliases: []string{actorURL},
			Links: []JRDLink{
				{Rel: "self", Type: "application/activity+json", Href: actorURL},
				{Rel: "http://webfinger.net/rel/profile-page", Type: "text/html", Href: actorURL},
			},
		}

		data, err := json.Marshal(jrd)
		if err != nil {
			log.Printf("webfinger: marshal error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/jrd+json")
		w.Write(data)
	}
}
