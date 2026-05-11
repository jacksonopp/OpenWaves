package channelstore

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sync"
)

// Station is the persisted representation of a dynamic channel.
type Station struct {
	ID               uint        `gorm:"primarykey;autoIncrement"`
	Username         string      `gorm:"uniqueIndex;not null"`
	Name             string
	Summary          string
	LicenseTerritory StringSlice `gorm:"type:text;serializer:json"`
	RelayPolicy      string
	IngestType       string
	IngestKey        string
}

// StringSlice is a []string that serializes to/from JSON for GORM.
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringSlice) Scan(v interface{}) error {
	var data []byte
	switch val := v.(type) {
	case string:
		data = []byte(val)
	case []byte:
		data = val
	default:
		return fmt.Errorf("channelstore: cannot scan %T into StringSlice", v)
	}
	return json.Unmarshal(data, s)
}

// Store defines the persistence contract for dynamic channels.
type Store interface {
	Create(s Station) error
	Delete(username string) error
	List() ([]Station, error)
}

// MemoryStore is a thread-safe in-memory Store for use in tests.
type MemoryStore struct {
	mu       sync.RWMutex
	stations []Station
}

func NewMemory() *MemoryStore { return &MemoryStore{} }

func (m *MemoryStore) Create(s Station) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.stations {
		if existing.Username == s.Username {
			return fmt.Errorf("channelstore: %q already exists", s.Username)
		}
	}
	m.stations = append(m.stations, s)
	return nil
}

func (m *MemoryStore) Delete(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, s := range m.stations {
		if s.Username == username {
			m.stations = append(m.stations[:i], m.stations[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("channelstore: %q not found", username)
}

func (m *MemoryStore) List() ([]Station, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Station, len(m.stations))
	copy(out, m.stations)
	return out, nil
}
