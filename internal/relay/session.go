package relay

import (
	"context"
	"crypto/rsa"
	"sync"
	"time"

	"github.com/jacksonopp/openwaves/internal/hls"
)

const listenerWindow = 35 * time.Second

// Session is a single active relay — poller + heartbeat goroutines.
type Session struct {
	username  string
	sourceURL string // remote station base URL
	selfURL   string // this relay's station URL
	store     *hls.Store
	privKey   *rsa.PrivateKey
	cancel    context.CancelFunc
	done      chan struct{}

	listenerMu sync.Mutex
	listeners  map[string]time.Time // IP → last manifest fetch time
}

func newSession(username, sourceURL, selfURL string, store *hls.Store, privKey *rsa.PrivateKey) *Session {
	return &Session{
		username:  username,
		sourceURL: sourceURL,
		selfURL:   selfURL,
		store:     store,
		privKey:   privKey,
		done:      make(chan struct{}),
		listeners: make(map[string]time.Time),
	}
}

// noteListener records a manifest fetch from the given IP address.
func (s *Session) noteListener(ip string) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	s.listeners[ip] = time.Now()
}

// listenerCount returns the number of unique IPs that fetched the manifest
// within the last listenerWindow.
func (s *Session) listenerCount() int {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	cutoff := time.Now().Add(-listenerWindow)
	count := 0
	for ip, t := range s.listeners {
		if t.After(cutoff) {
			count++
		} else {
			delete(s.listeners, ip) // prune stale entries
		}
	}
	return count
}

// start launches the poller and heartbeat goroutines.
func (s *Session) start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// Two goroutines share the done channel; close it when both finish.
	finCh := make(chan struct{}, 2)

	go func() {
		runPoller(ctx, s)
		finCh <- struct{}{}
	}()

	go func() {
		runHeartbeat(ctx, s)
		finCh <- struct{}{}
	}()

	go func() {
		<-finCh
		<-finCh
		close(s.done)
	}()
}

// stop cancels the context and waits for goroutines to finish.
func (s *Session) stop() {
	if s.cancel != nil {
		s.cancel()
	}
	<-s.done
}
