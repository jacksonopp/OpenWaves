# Where to Get Started (Development Order)

## ✅ 1. Define the Actor
Create the JSON-LD schema for an OpenWaves Station.

**Done.** The Station actor is implemented as an ActivityPub `Service` type in `internal/actor/station.go`, with a custom `ow:` JSON-LD namespace served at `/ns/openwaves`. The HTTP server (`cmd/server/main.go`) serves actor documents at `GET /stations/{username}` and the context at `GET /ns/openwaves`.

Core protocol features are also documented in `docs/core.md` and `README.md`, including: passive device compliance, broadcast termination signal, proof-of-listen telemetry, and license territory enforcement.

---

## ✅ 2. WebFinger Discovery
Implement the `.well-known/webfinger` logic so your Station is discoverable from other Fediverse apps (Mastodon, Pleroma, etc.).

**Done.** The WebFinger endpoint is implemented in `internal/webfinger/webfinger.go` and served at `GET /.well-known/webfinger?resource=acct:username@domain`. It returns a JRD (JSON Resource Descriptor) with `self` and `profile-page` links pointing to the Station actor URL.

Station registration is controlled by `config.yaml` at the repo root:
- `registration: admin_only` — only stations listed in the config are resolvable (unknown usernames → 404)
- `registration: open` — unknown usernames receive a generated stub actor

The server loads config from `config.yaml` by default; override with the `CONFIG_PATH` env var.

---

## ✅ 3. HLS Implementation
Build the core logic that segments audio into HLS `.m3u8` format and serves it with cryptographic signatures.

**Done.** The HLS pipeline is fully implemented across four packages:

- **`internal/keystore/`** — RSA-2048 key pairs generated per station on first run, persisted to `keys/<username>.pem` and `keys/<username>.pub.pem`. The public key is populated in the Station actor document at `publicKey.publicKeyPem`.
- **`internal/hls/`** — Thread-safe in-memory segment ring buffer (`Store`, last 10 segments), live `.m3u8` manifest builder, RSA-PKCS1v15/SHA-256 segment signer, and three HTTP handler factories (`ManifestHandler`, `SegmentHandler`, `SigHandler`).
- **`internal/ingest/`** — `SegmentIngestor` accepts individual `.ts` segments POSTed by the broadcaster, signs each one, and stores it in the ring buffer. **FFmpeg runs on the broadcaster's machine, not the server.**
- **`bin/broadcast.sh`** — broadcaster-side client script. Runs FFmpeg locally to produce `.ts` segments and POSTs each new segment to the server as it appears.

New routes:
```
POST /stations/{username}/ingest/{filename}   — broadcaster POSTs a single .ts segment
GET  /stations/{username}/hls/stream.m3u8     — live HLS playlist
GET  /stations/{username}/hls/{segment}       — .ts segment bytes
GET  /stations/{username}/hls/{segment}.sig   — RSA signature for the segment
```

The Station actor's `isLive` and `broadcastStatus` fields are updated dynamically based on whether the store has active segments.

Config additions in `config.yaml`:
- `keys_dir: keys` — where key pairs are stored (gitignored)

### Testing

1. **Start the server:**
   ```bash
   go run ./cmd/server
   ```

2. **Broadcast a test tone** (built-in FFmpeg test signal, 60-second duration):
   ```bash
   ./bin/broadcast.sh morning-vibes http://localhost:8080 60
   ```

3. **Broadcast real audio on macOS:**

   List available audio input devices:
   ```bash
   ffmpeg -f avfoundation -list_devices true -i "" 2>&1 | grep -A20 "audio devices"
   ```

   Start broadcasting from a device (replace `N` with the device index):
   ```bash
   AUDIO_INPUT="-f avfoundation -i none:N" ./bin/broadcast.sh morning-vibes
   ```
   > **Note:** Do NOT quote the device specifier inside `AUDIO_INPUT`. The variable is word-split intentionally when passed to FFmpeg.

4. **Connect a listener** (wait ~12 seconds for the first segments to appear):
   ```bash
   ffplay http://localhost:8080/stations/morning-vibes/hls/stream.m3u8
   ```

---

## ✅ 4. The Relay Logic
This is the most unique feature. Build the code that allows Server B to subscribe to and re-host HLS segments from Server A, implementing:

- License territory check before accepting a stream
- Proof-of-listen heartbeat (signed, every 30s) back to the source
- Broadcast termination signal handling (`TerminateStream` ActivityPub activity) with cascading shutdown

**Done.** The relay system is implemented across three new packages and a standalone binary:

- **`internal/activity/`** — ActivityPub activity structs: `Activity`, `Accept`, `Reject`, `ProofOfListen` (with `SignableString()` for canonical signing), `TerminateStream`.
- **`internal/inbox/`** — `POST /stations/{username}/inbox` handler. Dispatches on activity type:
  - `Follow` — fetches the remote actor, checks `relay_policy` (`closed` → send Reject; `open`/`allowlist` → store follower, send Accept)
  - `TerminateStream` — clears the local segment store, calls `onTerminate` callback (relay uses this to stop its poller), and **propagates the signal to all followers** for cascading shutdown
  - `ProofOfListen` — logs aggregate listener count and timestamp
  - Unknown types → 202 Accepted
- **`internal/relay/`** — Active relay session manager:
  - `Manager` — mutex-guarded session map; `StartRelay`, `StopRelay`, `IsRelaying`
  - `Session` — wraps a context + done channel; starts two goroutines on `start()`
  - `poller.go` — polls `{sourceURL}/hls/stream.m3u8` every 3s, downloads new segments + `.sig` sidecars, RSA-verifies each segment against the source's public key, stores in local `hls.Store`
  - `heartbeat.go` — sends a signed `ProofOfListen` POST to `{sourceURL}/inbox` every 30s with the real-time listener count
  - Listener tracking: each manifest poll from a unique client IP is counted; heartbeats report the number of unique IPs active in the last 35 seconds
- **`cmd/relay/main.go`** — standalone relay server binary configured entirely via environment variables (no `config.yaml` required)

New routes (on both source and relay servers):
```
POST /stations/{username}/inbox   — ActivityPub inbox (Follow, TerminateStream, ProofOfListen)
```

Relay server routes:
```
GET /stations/{username}                      — relay station actor (publicKey for verification)
POST /stations/{username}/inbox               — inbox with onTerminate → StopRelay
GET /stations/{username}/hls/stream.m3u8      — relay serves live manifest to local listeners
GET /stations/{username}/hls/{segment}        — relay serves verified segment bytes
GET /stations/{username}/hls/{segment}.sig    — relay serves signature sidecar
```

Config additions in `config.yaml` per station:
- `relay_policy: open | allowlist | closed` — controls whether Follow requests are accepted
- `ingest_key: <secret>` — if set, requires `Authorization: Bearer <secret>` header on all ingest POSTs

### Testing Two Servers

**Terminal 1 — source server:**
```bash
go run ./cmd/server
```

**Terminal 2 — relay server:**
```bash
SOURCE_URL=http://localhost:8080/stations/morning-vibes \
LOCAL_USERNAME=morning-vibes \
PORT=8081 \
go run ./cmd/relay
```

**Terminal 3 — broadcaster (test tone for 60s):**
```bash
./bin/broadcast.sh morning-vibes http://localhost:8080 60
```

**Terminal 3 — broadcaster (loop an MP3 indefinitely):**
```bash
AUDIO_INPUT="-stream_loop -1 -i /path/to/file.mp3" ./bin/broadcast.sh morning-vibes http://localhost:8080 3600
```

**Terminal 4 — listener via relay:**
```bash
ffplay http://localhost:8081/stations/morning-vibes/hls/stream.m3u8
```

The relay polls the source every 3 seconds, verifies each segment cryptographically, and re-serves it. The listener count appears in the source server logs (`ProofOfListen listenerCount=N`) every 30 seconds.

**To terminate the stream:**
```bash
curl -X POST http://localhost:8080/stations/morning-vibes/inbox \
  -H "Content-Type: application/activity+json" \
  -d '{"type":"TerminateStream","actor":"http://localhost:8080/stations/morning-vibes","object":"http://localhost:8080/stations/morning-vibes"}'
```

This cascades: source clears its store → sends TerminateStream to relay's inbox → relay stops its poller and clears its store → ffplay disconnects.

### Relay environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `SOURCE_URL` | ✅ | — | Full base URL of the remote station to relay |
| `LOCAL_USERNAME` | ✅ | — | Username for this relay station |
| `PORT` | | `8081` | Port to listen on |
| `DOMAIN` | | `localhost:<PORT>` | Public hostname (used in actor URLs) |
| `SCHEME` | | `http` | `http` or `https` |
| `RELAY_POLICY` | | `open` | `open`, `allowlist`, or `closed` |
| `KEYS_DIR` | | `keys-relay` | Directory for RSA key files |

---

## ✅ 5. Admin API + TUI

Stream lifecycle management (start/stop without server restart) and a terminal UI for managing broadcasts and relays.

**Done.** Two new components:

- **`internal/admin/`** — admin REST sub-router mounted at `/admin`. Protected by `admin_key` in `config.yaml` (`Authorization: Bearer <key>` header required). Returns 403 if `admin_key` is empty (admin disabled).
- **`cmd/tui/`** — standalone TUI binary built with Bubble Tea + Lip Gloss. Connects to the running server via the admin API, manages broadcaster subprocess, and controls stream/relay lifecycle.

### Admin API endpoints

All require `Authorization: Bearer <admin_key>`.

| Method | Path | Action |
|---|---|---|
| `GET` | `/admin/stations` | List all stations with live/relay status |
| `GET` | `/admin/stations/{username}` | Single station status |
| `POST` | `/admin/stations/{username}/stream/stop` | Terminate stream (clears store, propagates TerminateStream to followers) |
| `POST` | `/admin/stations/{username}/stream/start` | Reset store for a fresh ingest |
| `POST` | `/admin/stations/{username}/relay/start` | Start relay (body: `{"source_url":"..."}`) |
| `POST` | `/admin/stations/{username}/relay/stop` | Stop relay |

Station status response:
```json
{"username":"morning-vibes","isLive":true,"segmentCount":8,"isRelaying":false}
```

Config: add `admin_key` to `config.yaml`:
```yaml
admin_key: "your-secret-key"
```

### TUI

```bash
SERVER_URL=http://localhost:8080 ADMIN_KEY=your-secret-key go run ./cmd/tui
```

**Layout:**
```
OpenWaves  server: http://localhost:8080
┌─────────────────┬─────────────────────────────────────────────┐
│ ► morning-vibes │ morning-vibes                               │
│   ● LIVE        │ Status:    ● LIVE                           │
│   wfmu          │ Segments:  8                                │
│   ○ offline     │ Relay:     not relaying                     │
│                 │ Broadcast: stopped                          │
│                 │                                             │
│                 │ ─── broadcast log ───                       │
│                 │ seg0041.ts → 200 OK                         │
└─────────────────┴─────────────────────────────────────────────┘
esc: back  b: broadcast  B: stop broadcast  s: stop stream  ...
```

**Key bindings (detail view):**

| Key | Action |
|---|---|
| `b` | Start broadcast (prompts for audio input, empty = script default) |
| `B` | Stop broadcast subprocess |
| `s` | Stop stream (terminate + propagate to relays) |
| `S` | Start stream (clear store, ready for fresh ingest) |
| `r` | Start relay (prompts for source URL) |
| `x` | Stop relay |
| `esc` | Back to station list |
| `q` | Quit |

### TUI environment variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_URL` | `http://localhost:8080` | OpenWaves server to manage |
| `ADMIN_KEY` | `` | Bearer token (must match server's `admin_key`) |
| `BROADCAST_SCRIPT` | `./bin/broadcast.sh` | Path to the broadcast script |

### Example: start/stop a stream without restarting the server

```bash
# Stop the current stream (propagates TerminateStream to all relays):
curl -X POST http://localhost:8080/admin/stations/morning-vibes/stream/stop \
  -H "Authorization: Bearer your-secret-key"

# Clear state and allow a fresh ingest to start:
curl -X POST http://localhost:8080/admin/stations/morning-vibes/stream/start \
  -H "Authorization: Bearer your-secret-key"

# Then run broadcast.sh again to start a new stream
./bin/broadcast.sh morning-vibes http://localhost:8080 60
```