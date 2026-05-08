package logstream

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const bufSize = 100

// Stream is an io.Writer that fans log output out to SSE subscribers.
type Stream struct {
	mu          sync.Mutex
	ring        [bufSize]string
	ringStart   int
	ringLen     int
	subscribers map[chan string]struct{}
}

// New creates and returns a new Stream.
func New() *Stream {
	return &Stream{
		subscribers: make(map[chan string]struct{}),
	}
}

// Write implements io.Writer. Each newline-delimited line is stored in the
// ring buffer and sent to all current subscribers (non-blocking).
func (s *Stream) Write(p []byte) (int, error) {
	lines := strings.Split(string(p), "\n")
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, line := range lines {
		if line == "" {
			continue
		}
		idx := (s.ringStart + s.ringLen) % bufSize
		if s.ringLen < bufSize {
			s.ring[idx] = line
			s.ringLen++
		} else {
			s.ring[s.ringStart] = line
			s.ringStart = (s.ringStart + 1) % bufSize
		}
		for ch := range s.subscribers {
			select {
			case ch <- line:
			default:
			}
		}
	}
	return len(p), nil
}

// Subscribe returns a receive-only channel of log lines and an unsubscribe function.
func (s *Stream) Subscribe() (<-chan string, func()) {
	ch := make(chan string, 64)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	unsub := func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		s.mu.Unlock()
	}
	return ch, unsub
}

// Handler returns an SSE http.HandlerFunc that streams log lines to the client.
func (s *Stream) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Snapshot the ring buffer before subscribing so we don't miss lines
		// written between the snapshot and subscribe.
		s.mu.Lock()
		history := make([]string, s.ringLen)
		for i := 0; i < s.ringLen; i++ {
			history[i] = s.ring[(s.ringStart+i)%bufSize]
		}
		ch := make(chan string, 64)
		s.subscribers[ch] = struct{}{}
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			delete(s.subscribers, ch)
			s.mu.Unlock()
		}()

		for _, line := range history {
			fmt.Fprintf(w, "data: %s\n\n", line)
		}
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case line := <-ch:
				fmt.Fprintf(w, "data: %s\n\n", line)
				flusher.Flush()
			}
		}
	}
}
