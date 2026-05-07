# OpenWaves Copilot Instructions

OpenWaves is a decentralized live audio protocol built on top of ActivityPub/Fediverse infrastructure. It enables federated radio-style broadcasting where relay servers can re-host HLS streams from source servers, forming a "bucket brigade" for bandwidth distribution.

## Build & Test

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/webfinger/...

# Run a single test
go test ./internal/webfinger/... -run TestWebFingerValidStation

# Run the server (uses config.yaml by default)
go run ./cmd/server

# Override config path or port
CONFIG_PATH=./config.yaml PORT=8080 go run ./cmd/server
```

## Architecture

The server is a single binary (`cmd/server/main.go`) that wires together three internal packages and serves HTTP via Gorilla Mux:

```
cmd/server/main.go          — entry point, route registration, config loading
internal/actor/             — Station struct + JSON-LD context (the ActivityPub actor model)
internal/config/            — YAML config loading, station registry, registration policy
internal/webfinger/         — /.well-known/webfinger handler
static/                     — embeds static/ns/openwaves.jsonld into the binary
```

**Request flow**: All actor URLs are canonical and built from `cfg.BaseURL()` (derived from `config.yaml`'s `domain` + `scheme`), never from `r.Host`. This ensures URLs survive reverse proxy setups.

**Station registry**: Stations are defined in `config.yaml`. At runtime, `cfg.Registry()` returns a `map[string]StationConfig`. The server's `registration` field controls behavior for unknown usernames: `admin_only` → 404, `open` → generate a stub actor.

**HTTP routes**:
- `GET /stations/{username}` — ActivityPub Station actor (`application/activity+json`)
- `GET /.well-known/webfinger?resource=acct:username@domain` — WebFinger JRD (`application/jrd+json`)
- `GET /ns/openwaves` — JSON-LD context document (`application/ld+json`)

## Key Conventions

**JSON-LD context**: The `Context` variable in `internal/actor/station.go` is `[]interface{}` — not a struct or `[]string`. This is intentional: ActivityPub contexts are a mixed array of strings and objects, which cannot be expressed as a typed slice with `encoding/json`.

**Typed string constants**: `BroadcastStatus` and `RelayPolicy` are typed `string` aliases with defined constants in `internal/actor/constants.go`. Use these constants (`actor.OFFLINE`, `actor.LIVE`, `actor.OPEN`, etc.) rather than raw strings.

**`//go:embed` constraint**: Embedded files must be direct children of the package directory containing `embed.go`. The `static/` package exists solely to satisfy this constraint — `static/ns/openwaves.jsonld` is embedded and exported as `static.OpenWavesContext []byte`.

**Config is the source of truth**: `PublicKeyPem` is currently an empty string placeholder in the actor handler. RSA key generation/storage is not yet implemented. Do not build features that assume a populated public key without first implementing key management.

**`licenseTerritory`**: An array of ISO 3166-1 alpha-2 country codes. The special value `["*"]` means worldwide. Relay servers MUST check this before accepting a stream. This is protocol-level enforcement, not optional.

## Protocol Rules (from `docs/core.md`)

These are hard protocol requirements, not preferences:

- **Passive device**: Relays MUST NOT re-encode, transcode, inject ads, or alter segment content. Violations break cryptographic signature verification at the client.
- **Termination signal**: A `TerminateStream` ActivityPub activity from the source requires relays to purge buffered segments within 5 seconds and propagate the signal downstream.
- **Proof-of-listen**: Relays MUST send a signed heartbeat every 30 seconds containing `relayId`, `streamId`, `listenerCount`, `timestamp`, and `signature`. Aggregate counts only — no individual listener data.
- **License territory**: Relays MUST check `licenseTerritory` before accepting any stream.

## Implementation Roadmap (`docs/get-started.md`)

- ✅ Step 1: Station actor JSON-LD schema
- ✅ Step 2: WebFinger discovery
- ⬜ Step 3: HLS implementation (FFmpeg segmentation + per-chunk RSA signing)
- ⬜ Step 4: Relay logic (Follow-based subscription, territory check, heartbeat, TerminateStream)

**Prerequisites for Steps 3/4**: RSA key pair generation + storage, ActivityPub inbox handler (`POST /stations/{username}/inbox`).
