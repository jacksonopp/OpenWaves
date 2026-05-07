package inbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jacksonopp/openwaves/internal/activity"
	"github.com/jacksonopp/openwaves/internal/config"
	"github.com/jacksonopp/openwaves/internal/hls"
)

// remoteActor holds the fields we need from a fetched ActivityPub actor.
type remoteActor struct {
	Inbox string `json:"inbox"`
}

// Handler returns an http.HandlerFunc for POST /stations/{username}/inbox.
// onTerminate is an optional callback invoked when a TerminateStream is received;
// relay servers pass relay.Manager.StopRelay; source servers pass nil.
func Handler(cfg *config.Config, hlsStore *hls.Store, followerStore *FollowerStore, onTerminate func(string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		username := vars["username"]

		registry := cfg.Registry()
		stationCfg, ok := registry[username]
		if !ok {
			http.Error(w, "station not found", http.StatusNotFound)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		var act activity.Activity
		if err := json.Unmarshal(body, &act); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		stationURL := fmt.Sprintf("%s/stations/%s", cfg.BaseURL(), username)

		switch act.Type {
		case "Follow":
			handleFollow(w, act, body, stationCfg, stationURL, followerStore)
		case "ProofOfListen":
			var pol activity.ProofOfListen
			if err := json.Unmarshal(body, &pol); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			log.Printf("inbox: ProofOfListen from %s — listeners=%d timestamp=%s", pol.Actor, pol.ListenerCount, pol.Timestamp)
			w.WriteHeader(http.StatusOK)
		case "TerminateStream":
			var ts activity.TerminateStream
			if err := json.Unmarshal(body, &ts); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			log.Printf("inbox: TerminateStream received from %s — clearing segments for %s", ts.Actor, username)
			TerminateStation(username, hlsStore, followerStore, onTerminate)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

func handleFollow(w http.ResponseWriter, act activity.Activity, rawBody []byte, stationCfg config.StationConfig, stationURL string, followerStore *FollowerStore) {
	remote, err := fetchRemoteActor(act.Actor)
	if err != nil {
		log.Printf("inbox: failed to fetch remote actor %s: %v", act.Actor, err)
		// Still return 200; the follow request was received even if we can't respond.
		w.WriteHeader(http.StatusOK)
		return
	}

	if stationCfg.RelayPolicy == "closed" {
		reject := activity.Reject{
			Type:   "Reject",
			Actor:  stationURL,
			Object: act,
		}
		sendActivity(remote.Inbox, reject)
		w.WriteHeader(http.StatusOK)
		return
	}

	// "open" or "allowlist" (allowlist treated as open for now)
	followerStore.Add(stationCfg.Username, Follower{
		ActorURL: act.Actor,
		InboxURL: remote.Inbox,
	})

	accept := activity.Accept{
		Type:   "Accept",
		Actor:  stationURL,
		Object: act,
	}
	sendActivity(remote.Inbox, accept)
	w.WriteHeader(http.StatusOK)
}

// fetchRemoteActor GETs the actor URL and parses the inbox field.
func fetchRemoteActor(actorURL string) (*remoteActor, error) {
	req, err := http.NewRequest(http.MethodGet, actorURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("remote actor fetch returned %d", resp.StatusCode)
	}

	var ra remoteActor
	if err := json.NewDecoder(resp.Body).Decode(&ra); err != nil {
		return nil, err
	}
	return &ra, nil
}

// sendActivity marshals payload and POSTs it to inboxURL in a fire-and-forget goroutine.
func sendActivity(inboxURL string, payload interface{}) {
	go func() {
		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("inbox: marshal error sending to %s: %v", inboxURL, err)
			return
		}
		resp, err := http.Post(inboxURL, "application/activity+json", bytes.NewReader(data)) //nolint:noctx
		if err != nil {
			log.Printf("inbox: HTTP error sending to %s: %v", inboxURL, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			log.Printf("inbox: remote %s returned %d", inboxURL, resp.StatusCode)
		}
	}()
}
