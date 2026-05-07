package hls

import "sync"

// Segment is a single HLS transport-stream chunk with metadata.
type Segment struct {
	Filename  string
	Data      []byte
	Signature []byte
	SeqNum    int
}

// Store is a thread-safe in-memory ring buffer of HLS segments per station.
type Store struct {
	mu          sync.RWMutex
	maxSegments int
	segments    map[string][]Segment
}

// NewStore creates a Store that retains at most maxSegments per station.
func NewStore(maxSegments int) *Store {
	return &Store{
		maxSegments: maxSegments,
		segments:    make(map[string][]Segment),
	}
}

// Add appends a segment for the given station, evicting the oldest if at capacity.
func (s *Store) Add(username string, seg Segment) {
	s.mu.Lock()
	defer s.mu.Unlock()

	segs := s.segments[username]
	if len(segs) == s.maxSegments {
		segs = segs[1:]
	}
	s.segments[username] = append(segs, seg)
}

// Segments returns a copy of the current segment window for a station (oldest first).
func (s *Store) Segments(username string) []Segment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	src := s.segments[username]
	if len(src) == 0 {
		return nil
	}
	out := make([]Segment, len(src))
	copy(out, src)
	return out
}

// Get returns a single segment by filename for a station.
func (s *Store) Get(username, filename string) (Segment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, seg := range s.segments[username] {
		if seg.Filename == filename {
			return seg, true
		}
	}
	return Segment{}, false
}

// Clear removes all segments for a station.
func (s *Store) Clear(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.segments, username)
}
