package relay

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jacksonopp/openwaves/internal/activity"
)

const heartbeatInterval = 30 * time.Second

func runHeartbeat(ctx context.Context, s *Session) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendHeartbeat(ctx, s)
		}
	}
}

func sendHeartbeat(ctx context.Context, s *Session) {
	pol := activity.ProofOfListen{
		Type:          "ProofOfListen",
		Actor:         s.selfURL,
		Object:        s.sourceURL,
		ListenerCount: s.listenerCount(),
		Timestamp:     time.Now().Format(time.RFC3339),
	}

	if s.privKey != nil {
		h := sha256.Sum256([]byte(pol.SignableString()))
		sig, err := s.privKey.Sign(rand.Reader, h[:], crypto.SHA256)
		if err != nil {
			log.Printf("relay: heartbeat sign error for %s: %v", s.username, err)
			return
		}
		pol.Signature = base64.StdEncoding.EncodeToString(sig)
	}

	body, err := json.Marshal(pol)
	if err != nil {
		log.Printf("relay: heartbeat marshal error for %s: %v", s.username, err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.sourceURL+"/inbox", bytes.NewReader(body))
	if err != nil {
		log.Printf("relay: heartbeat request error for %s: %v", s.username, err)
		return
	}
	req.Header.Set("Content-Type", "application/activity+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("relay: heartbeat POST error for %s: %v", s.username, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("relay: heartbeat POST returned status %d for %s", resp.StatusCode, s.username)
	}
}
