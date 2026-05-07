package activity

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestActivity_MarshalRoundtrip(t *testing.T) {
	orig := Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Follow",
		ID:      "https://relay.example/follow/1",
		Actor:   "https://relay.example/actor",
		Object:  "https://station.example/actor",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Activity
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Context != orig.Context {
		t.Errorf("Context: got %q, want %q", got.Context, orig.Context)
	}
	if got.Type != orig.Type {
		t.Errorf("Type: got %q, want %q", got.Type, orig.Type)
	}
	if got.ID != orig.ID {
		t.Errorf("ID: got %q, want %q", got.ID, orig.ID)
	}
	if got.Actor != orig.Actor {
		t.Errorf("Actor: got %q, want %q", got.Actor, orig.Actor)
	}
	if got.Object != orig.Object {
		t.Errorf("Object: got %q, want %q", got.Object, orig.Object)
	}
}

func TestAccept_MarshalRoundtrip(t *testing.T) {
	orig := Accept{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "Accept",
		Actor:   "https://station.example/actor",
		Object: Activity{
			Type:  "Follow",
			Actor: "https://relay.example/actor",
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Accept
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != orig.Type {
		t.Errorf("Type: got %q, want %q", got.Type, orig.Type)
	}
	if got.Actor != orig.Actor {
		t.Errorf("Actor: got %q, want %q", got.Actor, orig.Actor)
	}
	if got.Object.Type != orig.Object.Type {
		t.Errorf("Object.Type: got %q, want %q", got.Object.Type, orig.Object.Type)
	}
	if got.Object.Actor != orig.Object.Actor {
		t.Errorf("Object.Actor: got %q, want %q", got.Object.Actor, orig.Object.Actor)
	}
}

func TestProofOfListen_MarshalRoundtrip(t *testing.T) {
	orig := ProofOfListen{
		Context:       "https://www.w3.org/ns/activitystreams",
		Type:          "ProofOfListen",
		Actor:         "https://relay.example/actor",
		Object:        "https://station.example/actor",
		ListenerCount: 42,
		Timestamp:     "2024-01-01T00:00:00Z",
		Signature:     "base64sighere==",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ProofOfListen
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != orig.Type {
		t.Errorf("Type: got %q, want %q", got.Type, orig.Type)
	}
	if got.Actor != orig.Actor {
		t.Errorf("Actor: got %q, want %q", got.Actor, orig.Actor)
	}
	if got.Object != orig.Object {
		t.Errorf("Object: got %q, want %q", got.Object, orig.Object)
	}
	if got.ListenerCount != orig.ListenerCount {
		t.Errorf("ListenerCount: got %d, want %d", got.ListenerCount, orig.ListenerCount)
	}
	if got.Timestamp != orig.Timestamp {
		t.Errorf("Timestamp: got %q, want %q", got.Timestamp, orig.Timestamp)
	}
	if got.Signature != orig.Signature {
		t.Errorf("Signature: got %q, want %q", got.Signature, orig.Signature)
	}
}

func TestProofOfListen_SignableString(t *testing.T) {
	p := ProofOfListen{
		Actor:         "https://relay.example/actor",
		Object:        "https://station.example/actor",
		ListenerCount: 7,
		Timestamp:     "2024-06-15T12:00:00Z",
	}

	want := fmt.Sprintf("%s\n%s\n%d\n%s", p.Actor, p.Object, p.ListenerCount, p.Timestamp)
	got := p.SignableString()
	if got != want {
		t.Errorf("SignableString:\n got  %q\n want %q", got, want)
	}
}

func TestTerminateStream_MarshalRoundtrip(t *testing.T) {
	orig := TerminateStream{
		Context: "https://www.w3.org/ns/activitystreams",
		Type:    "TerminateStream",
		Actor:   "https://station.example/actor",
		Object:  "https://station.example/actor",
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got TerminateStream
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != orig.Type {
		t.Errorf("Type: got %q, want %q", got.Type, orig.Type)
	}
	if got.Actor != orig.Actor {
		t.Errorf("Actor: got %q, want %q", got.Actor, orig.Actor)
	}
	if got.Object != orig.Object {
		t.Errorf("Object: got %q, want %q", got.Object, orig.Object)
	}
}
