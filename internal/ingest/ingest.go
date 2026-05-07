package ingest

// Ingestor accepts pre-made HLS segments for a station and stores them.
//
// TODO: Implement RtmpIngestor for RTMP ingest (e.g. from OBS, Liquidsoap).
type Ingestor interface {
	// AcceptSegment signs and stores a single .ts segment for the given station.
	AcceptSegment(username, filename string, data []byte) error
	// Stop clears the segment store for the given station.
	Stop(username string) error
}
