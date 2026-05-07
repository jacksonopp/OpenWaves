package actor

type BroadcastStatus string

const (
	OFFLINE   BroadcastStatus = "offline"
	LIVE      BroadcastStatus = "live"
	SCHEDULED BroadcastStatus = "scheduled"
)

type RelayPolicy string

const (
	OPEN      RelayPolicy = "open"
	ALLOWLIST RelayPolicy = "allowlist"
	CLOSED    RelayPolicy = "closed"
)
