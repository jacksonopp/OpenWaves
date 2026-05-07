package webfinger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jacksonopp/openwaves/internal/config"
)

func TestWebFingerValidStation(t *testing.T) {
	cfg := &config.Config{
		Domain:       "example.com",
		Scheme:       "https",
		Registration: config.AdminOnly,
		Stations: []config.StationConfig{
			{Username: "kexp", Name: "KEXP"},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/webfinger?resource=acct:kexp@example.com", nil)
	w := httptest.NewRecorder()
	Handler(cfg)(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/jrd+json" {
		t.Fatalf("expected Content-Type application/jrd+json, got %s", ct)
	}

	var jrd JRD
	if err := json.NewDecoder(res.Body).Decode(&jrd); err != nil {
		t.Fatalf("failed to decode JRD: %v", err)
	}

	if jrd.Subject != "acct:kexp@example.com" {
		t.Errorf("expected subject acct:kexp@example.com, got %s", jrd.Subject)
	}

	wantAlias := "https://example.com/stations/kexp"
	foundAlias := false
	for _, a := range jrd.Aliases {
		if a == wantAlias {
			foundAlias = true
			break
		}
	}
	if !foundAlias {
		t.Errorf("expected alias %s in %v", wantAlias, jrd.Aliases)
	}

	if len(jrd.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(jrd.Links))
	}
	if jrd.Links[0].Rel != "self" || jrd.Links[0].Type != "application/activity+json" || jrd.Links[0].Href != wantAlias {
		t.Errorf("unexpected first link: %+v", jrd.Links[0])
	}
	if jrd.Links[1].Rel != "http://webfinger.net/rel/profile-page" {
		t.Errorf("unexpected second link rel: %s", jrd.Links[1].Rel)
	}
}

func TestWebFingerMissingResource(t *testing.T) {
	cfg := &config.Config{Domain: "example.com", Scheme: "https", Registration: config.AdminOnly}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/webfinger", nil)
	w := httptest.NewRecorder()
	Handler(cfg)(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestWebFingerNonAcctFormat(t *testing.T) {
	cfg := &config.Config{Domain: "example.com", Scheme: "https", Registration: config.AdminOnly}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/webfinger?resource=https://example.com/stations/kexp", nil)
	w := httptest.NewRecorder()
	Handler(cfg)(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestWebFingerInvalidAcctFormat(t *testing.T) {
	cfg := &config.Config{Domain: "example.com", Scheme: "https", Registration: config.AdminOnly}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/webfinger?resource=acct:nodomain", nil)
	w := httptest.NewRecorder()
	Handler(cfg)(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestWebFingerDomainMismatch(t *testing.T) {
	cfg := &config.Config{Domain: "example.com", Scheme: "https", Registration: config.AdminOnly}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/webfinger?resource=acct:kexp@other.com", nil)
	w := httptest.NewRecorder()
	Handler(cfg)(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestWebFingerAdminOnlyUnknown(t *testing.T) {
	cfg := &config.Config{Domain: "example.com", Scheme: "https", Registration: config.AdminOnly}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/webfinger?resource=acct:unknown@example.com", nil)
	w := httptest.NewRecorder()
	Handler(cfg)(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Result().StatusCode)
	}
}

func TestWebFingerOpenUnknown(t *testing.T) {
	cfg := &config.Config{Domain: "example.com", Scheme: "https", Registration: config.Open}
	req := httptest.NewRequest(http.MethodGet, "/.well-known/webfinger?resource=acct:newstation@example.com", nil)
	w := httptest.NewRecorder()
	Handler(cfg)(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var jrd JRD
	if err := json.NewDecoder(res.Body).Decode(&jrd); err != nil {
		t.Fatalf("failed to decode JRD: %v", err)
	}
	if jrd.Subject != "acct:newstation@example.com" {
		t.Errorf("expected subject acct:newstation@example.com, got %s", jrd.Subject)
	}
	wantActorURL := "https://example.com/stations/newstation"
	foundAlias := false
	for _, a := range jrd.Aliases {
		if a == wantActorURL {
			foundAlias = true
			break
		}
	}
	if !foundAlias {
		t.Errorf("expected alias %s in %v", wantActorURL, jrd.Aliases)
	}
}

func TestWebFingerMethodNotAllowed(t *testing.T) {
	cfg := &config.Config{Domain: "example.com", Scheme: "https", Registration: config.AdminOnly}
	req := httptest.NewRequest(http.MethodPost, "/.well-known/webfinger", nil)
	w := httptest.NewRecorder()
	Handler(cfg)(w, req)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Result().StatusCode)
	}
}
