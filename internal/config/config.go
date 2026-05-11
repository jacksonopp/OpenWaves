package config

import (
"fmt"
"os"
"sync"

"github.com/jacksonopp/openwaves/internal/channelstore"
"github.com/jacksonopp/openwaves/internal/db"
"gopkg.in/yaml.v3"
)

type RegistrationPolicy string

const (
AdminOnly RegistrationPolicy = "admin_only"
Open      RegistrationPolicy = "open"
)

type StationConfig struct {
Username         string   `yaml:"username"          json:"username"`
Name             string   `yaml:"name"              json:"name"`
Summary          string   `yaml:"summary"           json:"summary"`
LicenseTerritory []string `yaml:"license_territory" json:"license_territory"`
RelayPolicy      string   `yaml:"relay_policy"      json:"relay_policy"`  // "open" | "allowlist" | "closed"
IngestType       string   `yaml:"ingest_type"       json:"ingest_type"`   // "http" | "rtmp" | "ffmpeg", default "http"
IngestKey        string   `yaml:"ingest_key"        json:"ingest_key"`
}

type DatabaseConfig struct {
Driver string `yaml:"driver"` // sqlite | postgres | mysql | mssql
DSN    string `yaml:"dsn"`
}

type Config struct {
Domain       string             `yaml:"domain"`
Scheme       string             `yaml:"scheme"`       // "http" | "https"
Registration RegistrationPolicy `yaml:"registration"` // "admin_only" | "open"
KeysDir      string             `yaml:"keys_dir"`     // default "keys"
Stations     []StationConfig    `yaml:"stations"`
AdminKey     string             `yaml:"admin_key"` // if set, all /admin requests require Authorization: Bearer <AdminKey>
Territory    string             `yaml:"territory"` // ISO 3166-1 alpha-2 code for this server's territory, e.g. "US" or "*"
Database     DatabaseConfig     `yaml:"database"`

mu              sync.RWMutex
dynamicStations []StationConfig
channelStore    channelstore.Store
staticSet       map[string]struct{}
}

func LoadConfig(path string) (*Config, error) {
data, err := os.ReadFile(path)
if err != nil {
return nil, err
}

var cfg Config
if err := yaml.Unmarshal(data, &cfg); err != nil {
return nil, err
}

if cfg.Domain == "" {
return nil, fmt.Errorf("config: domain must not be empty")
}
if cfg.Scheme == "" {
cfg.Scheme = "https"
}
if cfg.Registration == "" {
cfg.Registration = AdminOnly
}
if cfg.KeysDir == "" {
cfg.KeysDir = "keys"
}

// Database defaults
if cfg.Database.Driver == "" {
cfg.Database.Driver = "sqlite"
}
if cfg.Database.DSN == "" {
cfg.Database.DSN = "openwaves.db"
}

// Open DB and migrate channel model
gormDB, err := db.Open(cfg.Database.Driver, cfg.Database.DSN)
if err != nil {
return nil, fmt.Errorf("config: failed to open database: %w", err)
}
if err := db.Migrate(gormDB, &channelstore.Station{}); err != nil {
return nil, fmt.Errorf("config: failed to migrate database: %w", err)
}
cfg.channelStore = channelstore.New(gormDB)

cfg.staticSet = make(map[string]struct{}, len(cfg.Stations))
for _, s := range cfg.Stations {
cfg.staticSet[s.Username] = struct{}{}
}

// Load dynamic stations from DB
stations, err := cfg.channelStore.List()
if err != nil {
return nil, fmt.Errorf("config: failed to load channels from database: %w", err)
}
for _, s := range stations {
cfg.dynamicStations = append(cfg.dynamicStations, stationFromStore(s))
}

return &cfg, nil
}

// SetChannelStore replaces the channel persistence store. Intended for tests.
func (c *Config) SetChannelStore(s channelstore.Store) {
c.mu.Lock()
defer c.mu.Unlock()
c.channelStore = s
}

// Registry returns all known stations (static + dynamic).
func (c *Config) Registry() map[string]StationConfig {
c.mu.RLock()
defer c.mu.RUnlock()
m := make(map[string]StationConfig, len(c.Stations)+len(c.dynamicStations))
for _, s := range c.dynamicStations {
m[s.Username] = s
}
for _, s := range c.Stations {
m[s.Username] = s
}
return m
}

// IsStatic reports whether username was defined in the config file (not created via API).
func (c *Config) IsStatic(username string) bool {
c.mu.RLock()
defer c.mu.RUnlock()
_, ok := c.staticSet[username]
return ok
}

// CreateChannel adds a new dynamically-created station. Returns an error if the
// username already exists (static or dynamic).
func (c *Config) CreateChannel(sc StationConfig) error {
c.mu.Lock()
defer c.mu.Unlock()
if _, ok := c.staticSet[sc.Username]; ok {
return fmt.Errorf("channel %q already exists as a static channel", sc.Username)
}
for _, d := range c.dynamicStations {
if d.Username == sc.Username {
return fmt.Errorf("channel %q already exists", sc.Username)
}
}
if c.channelStore != nil {
if err := c.channelStore.Create(stationToStore(sc)); err != nil {
return fmt.Errorf("config: failed to persist channel: %w", err)
}
}
c.dynamicStations = append(c.dynamicStations, sc)
return nil
}

// DeleteChannel removes a dynamically-created station. Returns an error if the
// station does not exist or was defined in the config file (static).
func (c *Config) DeleteChannel(username string) error {
c.mu.Lock()
defer c.mu.Unlock()
if _, ok := c.staticSet[username]; ok {
return fmt.Errorf("channel %q is a static channel and cannot be deleted", username)
}
idx := -1
for i, d := range c.dynamicStations {
if d.Username == username {
idx = i
break
}
}
if idx == -1 {
return fmt.Errorf("channel %q not found", username)
}
if c.channelStore != nil {
if err := c.channelStore.Delete(username); err != nil {
return fmt.Errorf("config: failed to remove channel from database: %w", err)
}
}
c.dynamicStations = append(c.dynamicStations[:idx], c.dynamicStations[idx+1:]...)
return nil
}

func (c *Config) BaseURL() string {
return fmt.Sprintf("%s://%s", c.Scheme, c.Domain)
}

func stationToStore(sc StationConfig) channelstore.Station {
return channelstore.Station{
Username:         sc.Username,
Name:             sc.Name,
Summary:          sc.Summary,
LicenseTerritory: channelstore.StringSlice(sc.LicenseTerritory),
RelayPolicy:      sc.RelayPolicy,
IngestType:       sc.IngestType,
IngestKey:        sc.IngestKey,
}
}

func stationFromStore(s channelstore.Station) StationConfig {
return StationConfig{
Username:         s.Username,
Name:             s.Name,
Summary:          s.Summary,
LicenseTerritory: []string(s.LicenseTerritory),
RelayPolicy:      s.RelayPolicy,
IngestType:       s.IngestType,
IngestKey:        s.IngestKey,
}
}
