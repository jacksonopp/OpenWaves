package config_test

import (
	"os"
	"testing"

	"github.com/jacksonopp/openwaves/internal/config"
)

const testYAML = `
domain: "example.com"
scheme: "https"
registration: "open"
keys_dir: "mykeys"
stations:
  - username: "test-station"
    name: "Test Station"
    summary: "A test station."
    license_territory: ["US"]
    relay_policy: "open"
    ingest_type: "rtmp"
`

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoadConfig(t *testing.T) {
	path := writeTempConfig(t, testYAML)
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", cfg.Domain, "example.com")
	}
	if cfg.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", cfg.Scheme, "https")
	}
	if cfg.Registration != config.Open {
		t.Errorf("Registration = %q, want %q", cfg.Registration, config.Open)
	}
	if cfg.KeysDir != "mykeys" {
		t.Errorf("KeysDir = %q, want %q", cfg.KeysDir, "mykeys")
	}
}

func TestLoadConfig_IngestType(t *testing.T) {
	path := writeTempConfig(t, testYAML)
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reg := cfg.Registry()
	station, ok := reg["test-station"]
	if !ok {
		t.Fatal("expected station 'test-station' in registry")
	}
	if station.IngestType != "rtmp" {
		t.Errorf("IngestType = %q, want %q", station.IngestType, "rtmp")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	path := writeTempConfig(t, "domain: \"example.com\"\n")
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Scheme != "https" {
		t.Errorf("Scheme = %q, want default %q", cfg.Scheme, "https")
	}
	if cfg.Registration != config.AdminOnly {
		t.Errorf("Registration = %q, want default %q", cfg.Registration, config.AdminOnly)
	}
	if cfg.KeysDir != "keys" {
		t.Errorf("KeysDir = %q, want default %q", cfg.KeysDir, "keys")
	}
}

func TestRegistry(t *testing.T) {
	path := writeTempConfig(t, testYAML)
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	reg := cfg.Registry()
	station, ok := reg["test-station"]
	if !ok {
		t.Fatal("expected station 'test-station' in registry")
	}
	if station.Name != "Test Station" {
		t.Errorf("Name = %q, want %q", station.Name, "Test Station")
	}
}

func TestLoadConfig_MissingDomain(t *testing.T) {
	path := writeTempConfig(t, "scheme: \"http\"\n")
	_, err := config.LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing domain, got nil")
	}
}
