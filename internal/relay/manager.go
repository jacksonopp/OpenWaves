package relay

import (
	"crypto/rsa"
	"sync"

	"github.com/jacksonopp/openwaves/internal/hls"
)

// Manager manages relay sessions for all local stations.
// A relay session fetches segments from a remote source and re-serves them locally.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session // key: local station username
	store    *hls.Store
	privKeys map[string]*rsa.PrivateKey
}

func NewManager(store *hls.Store, privKeys map[string]*rsa.PrivateKey) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		store:    store,
		privKeys: privKeys,
	}
}

// StartRelay begins relaying from sourceURL for the local station username.
// sourceURL is the base URL of the remote station (e.g. "https://remote.example.com/stations/alice").
// selfURL is the full URL of this relay station (for ProofOfListen actor field).
// If a session already exists for username, it is stopped first.
func (m *Manager) StartRelay(username, sourceURL, selfURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.sessions[username]; ok {
		existing.stop()
		delete(m.sessions, username)
	}

	s := newSession(username, sourceURL, selfURL, m.store, m.privKeys[username])
	s.start()
	m.sessions[username] = s
	return nil
}

// StopRelay stops the relay session for a local station. No-op if not relaying.
func (m *Manager) StopRelay(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[username]; ok {
		s.stop()
		delete(m.sessions, username)
	}
}

// IsRelaying returns true if a relay session is active for username.
func (m *Manager) IsRelaying(username string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.sessions[username]
	return ok
}

// SourceURL returns the source URL for the active relay session of username,
// or empty string if no session exists.
func (m *Manager) SourceURL(username string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[username]; ok {
		return s.sourceURL
	}
	return ""
}
