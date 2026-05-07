package relay

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/jacksonopp/openwaves/internal/hls"
)

const pollInterval = 3 * time.Second

func runPoller(ctx context.Context, s *Session) {
	seen := make(map[string]bool)
	seqCounter := 0
	var cachedPubKey string

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pubKey := fetchPubKey(ctx, s.sourceURL, cachedPubKey)
			if pubKey != "" {
				cachedPubKey = pubKey
			}

			filenames := fetchManifest(ctx, s.sourceURL)
			for _, filename := range filenames {
				if seen[filename] {
					continue
				}

				segData, sigBytes, ok := fetchSegment(ctx, s.sourceURL, filename)
				if !ok {
					continue
				}

				if cachedPubKey != "" {
					if err := hls.Verify(cachedPubKey, segData, sigBytes); err != nil {
						log.Printf("relay: segment %q verification failed for %s: %v", filename, s.username, err)
						continue
					}
				} else {
					log.Printf("relay: no public key cached for %s, skipping verification for %q", s.username, filename)
				}

				s.store.Add(s.username, hls.Segment{
					Filename:  filename,
					Data:      segData,
					Signature: sigBytes,
					SeqNum:    seqCounter,
				})
				seen[filename] = true
				seqCounter++
			}
		}
	}
}

// fetchManifest fetches the m3u8 playlist and returns segment filenames.
func fetchManifest(ctx context.Context, sourceURL string) []string {
	url := sourceURL + "/hls/stream.m3u8"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("relay: manifest fetch error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("relay: manifest returned status %d", resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var filenames []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Manifest lines may be absolute URLs or bare filenames; extract just the filename.
		filenames = append(filenames, path.Base(line))
	}
	return filenames
}

// fetchSegment downloads a segment's data and signature bytes.
func fetchSegment(ctx context.Context, sourceURL, filename string) (data, sig []byte, ok bool) {
	data, ok = fetchBytes(ctx, sourceURL+"/hls/"+filename)
	if !ok {
		return nil, nil, false
	}
	sig, ok = fetchBytes(ctx, sourceURL+"/hls/"+filename+".sig")
	if !ok {
		return nil, nil, false
	}
	return data, sig, true
}

func fetchBytes(ctx context.Context, url string) ([]byte, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("relay: fetch error for %s: %v", url, err)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("relay: fetch %s returned status %d", url, resp.StatusCode)
		return nil, false
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false
	}
	return b, true
}

// fetchPubKey fetches the source station's public key PEM.
// Returns the existing cachedKey unchanged on failure.
func fetchPubKey(ctx context.Context, sourceURL, cachedKey string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return cachedKey
	}
	req.Header.Set("Accept", "application/activity+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("relay: public key fetch error: %v", err)
		return cachedKey
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("relay: public key fetch returned status %d", resp.StatusCode)
		return cachedKey
	}

	var actor struct {
		PublicKey struct {
			PublicKeyPem string `json:"publicKeyPem"`
		} `json:"publicKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&actor); err != nil {
		log.Printf("relay: public key decode error: %v", err)
		return cachedKey
	}
	if actor.PublicKey.PublicKeyPem == "" {
		return cachedKey
	}
	return actor.PublicKey.PublicKeyPem
}
