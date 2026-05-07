package inbox

import "sync"

// FollowerStore is a thread-safe in-memory store of followers per station.
type FollowerStore struct {
	mu        sync.RWMutex
	followers map[string][]Follower // key: station username
}

func NewFollowerStore() *FollowerStore {
	return &FollowerStore{
		followers: make(map[string][]Follower),
	}
}

// Add appends a follower for the given station. Duplicate ActorURLs are not deduplicated.
func (s *FollowerStore) Add(username string, f Follower) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.followers[username] = append(s.followers[username], f)
}

// List returns a copy of the follower list for the given station.
func (s *FollowerStore) List(username string) []Follower {
	s.mu.RLock()
	defer s.mu.RUnlock()

	src := s.followers[username]
	if len(src) == 0 {
		return nil
	}
	out := make([]Follower, len(src))
	copy(out, src)
	return out
}

// Remove deletes the follower with the given actorURL from the given station's list.
func (s *FollowerStore) Remove(username, actorURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.followers[username]
	filtered := existing[:0]
	for _, f := range existing {
		if f.ActorURL != actorURL {
			filtered = append(filtered, f)
		}
	}
	s.followers[username] = filtered
}
