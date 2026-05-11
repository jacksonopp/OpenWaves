package hls

import (
	"fmt"
	"strings"
)

// liveWindowSegments is the number of segments exposed in the live manifest.
// Clients always join at the live edge (tail), not at the beginning of the buffer.
const liveWindowSegments = 3

// Manifest generates a live HLS playlist for the given station.
// baseURL is the prefix for segment URLs (e.g. "https://example.com/stations/bob/hls").
// targetDuration is the segment duration in seconds (typically 6).
func Manifest(store *Store, username, baseURL string, targetDuration int) string {
	segs := store.Segments(username)

	// Trim to the live window so clients join near the live edge.
	if len(segs) > liveWindowSegments {
		segs = segs[len(segs)-liveWindowSegments:]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "#EXTM3U\n")
	fmt.Fprintf(&b, "#EXT-X-VERSION:3\n")
	fmt.Fprintf(&b, "#EXT-X-TARGETDURATION:%d\n", targetDuration)

	mediaSeq := 0
	if len(segs) > 0 {
		mediaSeq = segs[0].SeqNum
	}
	fmt.Fprintf(&b, "#EXT-X-MEDIA-SEQUENCE:%d\n", mediaSeq)

	for i, seg := range segs {
		if i > 0 && seg.DiscontinuitySeq != segs[i-1].DiscontinuitySeq {
			fmt.Fprintf(&b, "#EXT-X-DISCONTINUITY\n")
		}
		fmt.Fprintf(&b, "#EXTINF:%.3f,\n", float64(targetDuration))
		fmt.Fprintf(&b, "%s/%s\n", baseURL, seg.Filename)
	}

	return b.String()
}
