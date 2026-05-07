package ingest

import (
"crypto/rsa"
"fmt"
"sync"
"sync/atomic"

"github.com/jacksonopp/openwaves/internal/hls"
)

// SegmentIngestor accepts pre-made .ts segments, signs them, and stores them in the HLS store.
// FFmpeg runs on the broadcaster's machine; this server only handles signing and serving.
type SegmentIngestor struct {
store    *hls.Store
privKeys map[string]*rsa.PrivateKey
seqNums  sync.Map // username → *int64
}

// NewSegmentIngestor creates a SegmentIngestor backed by the given store and key map.
func NewSegmentIngestor(store *hls.Store, privKeys map[string]*rsa.PrivateKey) *SegmentIngestor {
return &SegmentIngestor{store: store, privKeys: privKeys}
}

// AcceptSegment signs data and adds it to the store under the given filename.
func (s *SegmentIngestor) AcceptSegment(username, filename string, data []byte) error {
privKey, ok := s.privKeys[username]
if !ok {
return fmt.Errorf("no private key for station %s", username)
}

sig, err := hls.Sign(privKey, data)
if err != nil {
return fmt.Errorf("sign segment: %w", err)
}

seqNum := s.nextSeqNum(username)
s.store.Add(username, hls.Segment{
Filename:  filename,
Data:      data,
Signature: sig,
SeqNum:    int(seqNum),
})
return nil
}

// Stop clears all segments for the given station.
func (s *SegmentIngestor) Stop(username string) error {
s.store.Clear(username)
return nil
}

func (s *SegmentIngestor) nextSeqNum(username string) int64 {
val, _ := s.seqNums.LoadOrStore(username, new(int64))
return atomic.AddInt64(val.(*int64), 1) - 1
}
