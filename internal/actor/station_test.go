package actor

import (
	"encoding/json"
	"testing"
)

func TestStationMarshalRoundTrip(t *testing.T) {
	original := NewStation(
		"https://example.com/users/kexp",
		"KEXP",
		"kexp",
	)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Station
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Type != "Service" {
		t.Errorf("Type: got %q, want %q", decoded.Type, "Service")
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.PreferredUsername != original.PreferredUsername {
		t.Errorf("PreferredUsername: got %q, want %q", decoded.PreferredUsername, original.PreferredUsername)
	}
	if decoded.BroadcastStatus != "offline" {
		t.Errorf("BroadcastStatus: got %q, want %q", decoded.BroadcastStatus, "offline")
	}
	if decoded.IsLive != false {
		t.Errorf("IsLive: got %v, want false", decoded.IsLive)
	}
}

func TestStationLicenseTerritoryRoundTrip(t *testing.T) {
	s := NewStation("https://example.com/users/kexp", "KEXP", "kexp")
	s.LicenseTerritory = []string{"US", "CA", "GB"}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Station
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(decoded.LicenseTerritory) != 3 {
		t.Fatalf("LicenseTerritory length: got %d, want 3", len(decoded.LicenseTerritory))
	}
	want := map[string]bool{"US": true, "CA": true, "GB": true}
	for _, v := range decoded.LicenseTerritory {
		if !want[v] {
			t.Errorf("unexpected territory %q", v)
		}
		delete(want, v)
	}
	for k := range want {
		t.Errorf("missing territory %q", k)
	}
}

func TestStationLicenseTerritoryOmitempty(t *testing.T) {
	s := NewStation("https://example.com/users/kexp", "KEXP", "kexp")

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if _, ok := raw["licenseTerritory"]; ok {
		t.Error("licenseTerritory should be omitted when nil, but it was present")
	}
}

func TestStationLicenseTerritoryWorldwide(t *testing.T) {
	s := NewStation("https://example.com/users/kexp", "KEXP", "kexp")
	s.LicenseTerritory = []string{"*"}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Station
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(decoded.LicenseTerritory) != 1 || decoded.LicenseTerritory[0] != "*" {
		t.Errorf("LicenseTerritory: got %v, want [\"*\"]", decoded.LicenseTerritory)
	}
}

func TestStationContextField(t *testing.T) {
	s := NewStation("https://example.com/users/wfmu", "WFMU", "wfmu")

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if _, ok := raw["@context"]; !ok {
		t.Error("@context field missing from marshaled JSON")
	}
}
