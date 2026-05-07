package activity

import "fmt"

// Activity is a generic ActivityPub activity wrapper.
type Activity struct {
	Context string `json:"@context,omitempty"`
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Actor   string `json:"actor"`
	Object  string `json:"object,omitempty"`
}

// Accept is sent in response to a Follow to confirm the relay subscription.
type Accept struct {
	Context string   `json:"@context,omitempty"`
	Type    string   `json:"type"`   // always "Accept"
	Actor   string   `json:"actor"`  // the station URL accepting the follow
	Object  Activity `json:"object"` // the original Follow activity
}

// Reject is sent in response to a Follow to deny the relay subscription.
type Reject struct {
	Context string   `json:"@context,omitempty"`
	Type    string   `json:"type"`   // always "Reject"
	Actor   string   `json:"actor"`
	Object  Activity `json:"object"` // the original Follow activity
}

// ProofOfListen is a signed heartbeat POSTed by a relay to the source station's
// inbox every 30 seconds to prove the relay is active and report listener count.
type ProofOfListen struct {
	Context       string `json:"@context,omitempty"`
	Type          string `json:"type"`          // always "ProofOfListen"
	Actor         string `json:"actor"`         // relay station URL
	Object        string `json:"object"`        // source station URL
	ListenerCount int    `json:"listenerCount"`
	Timestamp     string `json:"timestamp"`  // RFC3339
	Signature     string `json:"signature"`  // base64(RSA-PKCS1v15-SHA256 over SignableString)
}

// SignableString returns the canonical string that is signed for a ProofOfListen.
// Format: "{actor}\n{object}\n{listenerCount}\n{timestamp}"
func (p ProofOfListen) SignableString() string {
	return fmt.Sprintf("%s\n%s\n%d\n%s", p.Actor, p.Object, p.ListenerCount, p.Timestamp)
}

// TerminateStream is sent by the source station to instruct all relays to
// immediately drop the stream and clear their segment stores.
type TerminateStream struct {
	Context string `json:"@context,omitempty"`
	Type    string `json:"type"`   // always "TerminateStream"
	Actor   string `json:"actor"`  // source station URL
	Object  string `json:"object"` // source station URL (the stream being terminated)
}
