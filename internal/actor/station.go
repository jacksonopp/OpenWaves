package actor

import "time"

// Context is the JSON-LD @context array for all Station actors.
var Context = []interface{}{
	"https://www.w3.org/ns/activitystreams",
	"https://w3id.org/security/v1",
	map[string]interface{}{
		"ow":              "https://example.com/ns/openwaves#",
		"stationUri":      map[string]string{"@id": "ow:stationUri", "@type": "@id"},
		"isLive":          map[string]string{"@id": "ow:isLive", "@type": "xsd:boolean"},
		"hlsManifest":     map[string]string{"@id": "ow:hlsManifest", "@type": "@id"},
		"streamExpiry":    map[string]string{"@id": "ow:streamExpiry", "@type": "xsd:dateTime"},
		"broadcastStatus": map[string]string{"@id": "ow:broadcastStatus"},
		"audioCodec":      map[string]string{"@id": "ow:audioCodec"},
		"bitrate":         map[string]string{"@id": "ow:bitrate", "@type": "xsd:integer"},
		"relayPolicy":      map[string]string{"@id": "ow:relayPolicy"},
		"licenseTerritory": map[string]string{"@id": "ow:licenseTerritory"},
	},
}

type PublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

// Station is an ActivityPub Service actor with OpenWaves broadcasting extensions.
type Station struct {
	Context           interface{} `json:"@context"`
	Type              string      `json:"type"`
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	PreferredUsername string      `json:"preferredUsername"`
	Summary           string      `json:"summary,omitempty"`
	URL               string      `json:"url"`
	Inbox             string      `json:"inbox"`
	Outbox            string      `json:"outbox"`
	Followers         string      `json:"followers"`
	Following         string      `json:"following"`
	PublicKey         PublicKey   `json:"publicKey"`

	// OpenWaves extensions
	StationURI      string          `json:"stationUri"`
	IsLive          bool            `json:"isLive"`
	HLSManifest     *string         `json:"hlsManifest,omitempty"`
	StreamExpiry    *time.Time      `json:"streamExpiry,omitempty"`
	BroadcastStatus BroadcastStatus `json:"broadcastStatus"` // "offline" | "live" | "scheduled"
	AudioCodec      string          `json:"audioCodec,omitempty"`
	Bitrate         int             `json:"bitrate,omitempty"`
	RelayPolicy     RelayPolicy     `json:"relayPolicy,omitempty"` // "open" | "allowlist" | "closed"
	// ISO 3166-1 alpha-2 country codes; ["*"] means worldwide
	LicenseTerritory []string `json:"licenseTerritory,omitempty"`
}

// NewStation creates a Station pre-populated with the standard JSON-LD context and type.
func NewStation(id, name, preferredUsername string) Station {
	return Station{
		Context:           Context,
		Type:              "Service",
		ID:                id,
		Name:              name,
		PreferredUsername: preferredUsername,
		BroadcastStatus:   OFFLINE,
	}
}
