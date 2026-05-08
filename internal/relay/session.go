package relay

import (
	"context"
	"crypto/rsa"

	"github.com/jacksonopp/openwaves/internal/hls"
)

// Session is a single active relay — poller + heartbeat goroutines.
type Session struct {
	username  string
	sourceURL string // remote station base URL
	selfURL   string // this relay's station URL
	store     *hls.Store
	privKey   *rsa.PrivateKey
	cancel    context.CancelFunc
	done      chan struct{}
}

func newSession(username, sourceURL, selfURL string, store *hls.Store, privKey *rsa.PrivateKey) *Session {
	return &Session{
		username:  username,
		sourceURL: sourceURL,
		selfURL:   selfURL,
		store:     store,
		privKey:   privKey,
		done:      make(chan struct{}),
	}
}

// listenerCount returns the number of active listeners for this relay station,
// as tracked by the HLS store (manifest fetches within the last 35s).
func (s *Session) listenerCount() int {
	return s.store.ListenerCount(s.username)
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
