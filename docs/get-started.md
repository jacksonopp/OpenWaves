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
- **`internal/hls/`** — Thread-safe in-memory segment ring buffer (`Store`, last 10 segments), live `.m3u8` manifest builder, RSA-PKCS1v15/SHA-256 segment signer, and three HTTP handler factories (`ManifestHandler`, `SegmentHandler`, `SigHandler`). The `Store` also tracks:
  - **Liveness** (`IsLive`): a station is considered live if a segment arrived within the last 20 seconds. Stale ring-buffer segments don't falsely indicate liveness.
  - **Listeners** (`TrackListener` / `ListenerCount`): every `GET stream.m3u8` request records the client IP. `ListenerCount` returns unique IPs active in the last 35 seconds.
  - **Ingest suspension** (`Suspend` / `Resume` / `IsSuspended`): `stream/stop` suspends the station; new ingest POSTs return `503` until `stream/start` resumes it.
- **`internal/ingest/`** — `SegmentIngestor` accepts individual `.ts` segments POSTed by the broadcaster, signs each one, and stores it in the ring buffer. **FFmpeg runs on the broadcaster's machine, not the server.**
- **`bin/broadcast.sh`** — broadcaster-side client script. Runs FFmpeg locally to produce `.ts` segments and POSTs each new segment to the server as it appears.

New routes:
```
POST /stations/{username}/ingest/{filename}   — broadcaster POSTs a single .ts segment
GET  /stations/{username}/hls/stream.m3u8     — live HLS playlist
GET  /stations/{username}/hls/{segment}       — .ts segment bytes
GET  /stations/{username}/hls/{segment}.sig   — RSA signature for the segment
```

The Station actor's `isLive` and `broadcastStatus` fields are updated dynamically: `isLive` is `true` only if a segment arrived within the last 20 seconds.

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
- Broadcast termination signal handling (admin-only via admin API, cascading shutdown to relays)

**Done.** The relay system is implemented across three new packages and a standalone binary:

- **`internal/activity/`** — ActivityPub activity structs: `Activity`, `Accept`, `Reject`, `ProofOfListen` (with `SignableString()` for canonical signing), `TerminateStream`.
- **`internal/inbox/`** — `POST /stations/{username}/inbox` handler. Dispatches on activity type:
  - `Follow` — fetches the remote actor, checks `relay_policy` (`closed` → send Reject; `open`/`allowlist` → store follower, send Accept)
  - `ProofOfListen` — logs aggregate listener count and timestamp
  - `TerminateStream` — on **relay servers**, if the actor URL matches the active source, stops the relay session. On **source servers**, silently accepted with 202 (termination is admin-only there).
  - Unknown types → 202 Accepted
- **`internal/relay/`** — Active relay session manager:
  - `Manager` — mutex-guarded session map; `StartRelay`, `StopRelay`, `IsRelaying`
  - `Session` — wraps a context + done channel; starts two goroutines on `start()`
  - `poller.go` — polls `{sourceURL}/hls/stream.m3u8` every 3s, downloads new segments + `.sig` sidecars, RSA-verifies each segment against the source's public key, stores in local `hls.Store`
  - `heartbeat.go` — sends a signed `ProofOfListen` POST to `{sourceURL}/inbox` every 30s with the real-time listener count (sourced from `hls.Store.ListenerCount`)
  - Listener tracking: every manifest fetch (`GET stream.m3u8`) is tracked by client IP in `hls.Store`. Both direct listeners and relay listeners are counted. Heartbeats report the number of unique IPs active in the last 35 seconds.
- **`cmd/relay/main.go`** — standalone relay server binary configured entirely via environment variables (no `config.yaml` required)

New routes (on both source and relay servers):
```
POST /stations/{username}/inbox   — ActivityPub inbox (Follow, ProofOfListen)
```

Relay server routes:
```
GET /stations/{username}                      — relay station actor (publicKey for verification)
POST /stations/{username}/inbox               — inbox (Follow, ProofOfListen, TerminateStream from source)
GET /stations/{username}/hls/stream.m3u8      — relay serves live manifest to local listeners
GET /stations/{username}/hls/{segment}        — relay serves verified segment bytes
GET /stations/{username}/hls/{segment}.sig    — relay serves signature sidecar
```

Config additions in `config.yaml`:
- `relay_policy: open | allowlist | closed` — controls whether Follow requests are accepted
- `ingest_key: <secret>` — if set, requires `Authorization: Bearer <secret>` header on all ingest POSTs
- `territory: <ISO 3166-1 alpha-2>` — this server's territory (e.g. `"US"` or `"*"` for worldwide). Relays check the source station's `licenseTerritory` against this value before starting a relay session.

### Territory Enforcement

Before a relay starts, the admin API fetches the source station's ActivityPub actor and checks the `licenseTerritory` field:
- If `licenseTerritory` is empty or `["*"]`, the relay is allowed.
- Otherwise, the relay server's `territory` config value must be in the list.
- If not, `POST /admin/stations/{username}/relay/start` returns `403 Forbidden`.

Example in `config.relay.yaml`:
```yaml
territory: "US"
```

### Terminating a Stream

Stream termination is **admin-only** on the source server. Use the admin API:
```bash
curl -X POST http://localhost:8080/admin/stations/morning-vibes/stream/stop \
  -H "Authorization: Bearer secret"
```

This does the following:
1. **Suspends ingest** — the server rejects any further segment POSTs from the broadcaster with `503`. The broadcaster process can keep running but its uploads are blocked until `stream/start` is called.
2. **Propagates TerminateStream** — sends a `TerminateStream` ActivityPub activity to all known relay followers.
3. **Relay cascade** — each relay server's inbox verifies the actor URL matches its active source and stops the relay poller, purging buffered segments.

To restart the stream (re-enable ingest for a fresh broadcast):
```bash
curl -X POST http://localhost:8080/admin/stations/morning-vibes/stream/start \
  -H "Authorization: Bearer secret"
```

Then run `bin/broadcast.sh` again to begin a new broadcast.

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
| `GET` | `/admin/stations` | List all stations with live/relay/ingest status |
| `GET` | `/admin/stations/{username}` | Single station status |
| `POST` | `/admin/stations/{username}/stream/stop` | Stop ingest + propagate TerminateStream to relay followers |
| `POST` | `/admin/stations/{username}/stream/start` | Start stream: resumes ingest + auto-starts broadcast (silence by default) |
| `POST` | `/admin/stations/{username}/relay/start` | Start relay (body: `{"source_url":"..."}`) |
| `POST` | `/admin/stations/{username}/relay/stop` | Stop relay |
| `POST` | `/admin/stations/{username}/ingest/start` | Spawn `broadcast.sh` subprocess (body: `{"audio_type":"silence\|test_tone\|file","audio_file":""}`) |
| `POST` | `/admin/stations/{username}/ingest/stop` | Kill the managed broadcast subprocess |
| `POST` | `/admin/stations/{username}/ingest/input` | Hot-swap audio input while live (body: `{"type":"silence\|test_tone\|file","file":""}`) |
| `POST` | `/admin/channels` | Create a new dynamic channel (body: `StationConfig`) |
| `DELETE` | `/admin/channels/{username}` | Delete a dynamic channel (error if static) |
| `GET` | `/admin/logs` | SSE stream of server log lines (`text/event-stream`) |

Station status response now includes `audioInput` and `isStatic`:
```json
{
  "username": "morning-vibes",
  "isLive": true,
  "segmentCount": 8,
  "listenerCount": 3,
  "isRelaying": false,
  "isIngesting": true,
  "audioInput": {"type": "silence"},
  "isStatic": true
}
```

### Database Configuration

OpenWaves uses SQL to persist dynamic channels and (in future releases) all other server data — user accounts, federation metadata, etc. The database is configured in `config.yaml`:

```yaml
database:
  driver: sqlite       # sqlite | postgres | mysql | mssql
  dsn: openwaves.db    # see driver-specific DSN formats below
```

**Supported drivers:**

| Driver | `driver` value | `dsn` format |
|---|---|---|
| SQLite (default) | `sqlite` | File path, e.g. `openwaves.db` or `:memory:` for tests |
| PostgreSQL | `postgres` | `host=localhost user=openwaves password=secret dbname=openwaves port=5432 sslmode=disable` |
| MySQL / MariaDB | `mysql` | `openwaves:secret@tcp(localhost:3306)/openwaves?charset=utf8mb4&parseTime=True&loc=Local` |
| SQL Server | `mssql` | `sqlserver://openwaves:secret@localhost:1433?database=openwaves` |

The default configuration uses SQLite with a single file (`openwaves.db` in the working directory), which requires no external database server. For production deployments, PostgreSQL is recommended.

> **Upgrade note:** `channels.json` (used in prior versions for dynamic channel persistence) is no longer read. If you have existing dynamic channels in `channels.json`, recreate them via the admin API (`POST /admin/channels`) after upgrading.

### Dynamic channels

Channels can be created and deleted at runtime without editing `config.yaml`. Dynamic channels are persisted to the configured SQL database and survive server restarts.

```bash
# Create a new channel
curl -X POST http://localhost:8080/admin/channels \
  -H "Authorization: Bearer secret" \
  -H "Content-Type: application/json" \
  -d '{"username":"jazz-hour","name":"Jazz Hour","relay_policy":"open","license_territory":["*"]}'

# Delete a dynamic channel
curl -X DELETE http://localhost:8080/admin/channels/jazz-hour \
  -H "Authorization: Bearer secret"
```

- Static channels (defined in `config.yaml`) cannot be deleted via the API
- RSA keys are generated automatically for new dynamic channels
- `GET /admin/stations` includes both static and dynamic channels

### Audio input

Each station has a configurable audio source, hot-swappable while live:

| Type | FFmpeg source | Notes |
|---|---|---|
| `silence` | `anullsrc` stereo/44100 | Default for all new streams |
| `test_tone` | `sine=frequency=440` | 440 Hz test tone |
| `file` | `stream_loop -1 -i <path>` | Loops a local audio file indefinitely |

```bash
# Switch to a looping file while broadcasting
curl -X POST http://localhost:8080/admin/stations/morning-vibes/ingest/input \
  -H "Authorization: Bearer secret" \
  -H "Content-Type: application/json" \
  -d '{"type":"file","file":"/path/to/audio.mp3"}'

# Switch back to silence
curl -X POST http://localhost:8080/admin/stations/morning-vibes/ingest/input \
  -H "Authorization: Bearer secret" \
  -H "Content-Type: application/json" \
  -d '{"type":"silence"}'
```

Config: add `admin_key` and `territory` to `config.yaml`:
```yaml
admin_key: "your-secret-key"
territory: "*"   # or "US", "CA", etc.
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
# Stop the current stream (suspends ingest + propagates TerminateStream to all relays):
curl -X POST http://localhost:8080/admin/stations/morning-vibes/stream/stop \
  -H "Authorization: Bearer your-secret-key"

# Re-enable ingest for a fresh broadcast:
curl -X POST http://localhost:8080/admin/stations/morning-vibes/stream/start \
  -H "Authorization: Bearer your-secret-key"

# Then run broadcast.sh again to start a new stream:
./bin/broadcast.sh morning-vibes http://localhost:8080 60
```

---

## ✅ 6. Admin Web UI

A browser-based admin UI embedded directly in the Go binary, providing a full station management interface without requiring a separate service.

**Done.** A React + Vite + TypeScript SPA is built separately and embedded into the Go binary at `/admin/ui/` via `//go:embed`. Two new packages were added alongside updates to the admin handler and main server entrypoint:

### New packages

- **`internal/adminui/`** — Serves the embedded SPA at `/admin/ui/`
  - `embed_prod.go` (build tag `!admindev`) — `//go:embed all:dist`, exports `var dist fs.FS`
  - `embed_dev.go` (build tag `admindev`) — sets `var dist fs.FS = nil`; dev mode proxies to Vite dev server
  - `handler.go` — `Handler() http.Handler`: in production, strips the `/admin/ui` prefix and serves from the embedded `dist/`, with an SPA fallback (returns `index.html` for directories and missing files to avoid `http.FileServer`'s redirect behavior); in dev mode, reverse-proxies to the URL in `ADMINUI_DEV_PROXY`

- **`internal/broadcaster/manager.go`** — Per-station subprocess manager for `bin/broadcast.sh`
  - `NewManager()`, `Start(username, serverURL string) error`, `Stop(username string) error`, `IsRunning(username string) bool`
  - `AudioInput{Type, File}` struct — `type` is `"silence"`, `"test_tone"`, or `"file"`. Default is `silence`.
  - `SetInput(username, AudioInput)` — stores without restarting; `GetInput(username) AudioInput`; `ChangeInput(username, AudioInput) error` — hot-swaps while live
  - `AUDIO_INPUT` env var passed to `bin/broadcast.sh` with the resolved FFmpeg flags
  - Process group management with SIGINT + SIGKILL fallback on stop

### Modified files

- **`internal/admin/handler.go`** — accepts `*broadcaster.Manager` and `*keystore.Store`; implements all admin endpoints; `StationStatus` now includes `"isIngesting": bool`, `"audioInput": AudioInput`, `"isStatic": bool`; `stream/start` auto-starts ingest (silence); `stream/stop` stops ingest before terminating
- **`cmd/server/main.go`** — creates `ks := keystore.NewStore(cfg.KeysDir)`, loads static channel keys at startup, passes `ks` to `relay.NewManager`, `ingest.Handler`, `admin.Handler`, and `stationHandler`; removes the old `loadKeys` raw-map approach

### Frontend (ui/)

The SPA lives in `ui/` and is built independently before embedding.

**Libraries:** React 19, TypeScript, Vite, [TanStack Router](https://tanstack.com/router) (client-side routing), [TanStack Query](https://tanstack.com/query) (data fetching + polling), CSS Modules (scoped per-component styles).

**Visual design:** Two-panel layout (fixed 256px sidebar + scrollable main area) matching the Figma spec. See [`docs/admin-ui-design.md`](./admin-ui-design.md) for color tokens, typography, and component patterns.

| File | Purpose |
|---|---|
| `ui/src/api/client.ts` | Typed `AdminClient` class wrapping all admin API endpoints |
| `ui/src/context/AuthContext.tsx` | Admin key stored in `localStorage`; injects `Authorization` header |
| `ui/src/App.tsx` | TanStack Router route tree; wraps app with `QueryClientProvider` + `AuthProvider` |
| `ui/src/pages/Login.tsx` | Admin key entry form (light theme) |
| `ui/src/pages/admin/StreamsPage.tsx` | Active streams list with TanStack Query polling (`refetchInterval: 3000`) |
| `ui/src/pages/admin/OverviewPage.tsx` | Station stats (total / live / relaying) + live log feed |
| `ui/src/pages/admin/ModerationPage.tsx` | Placeholder — moderation tools |
| `ui/src/pages/admin/FederationPage.tsx` | Placeholder — ActivityPub federation management |
| `ui/src/components/admin/AdminLayout.tsx` | Outer shell: top bar + sidebar + `<Outlet />` |
| `ui/src/components/admin/TopBar.tsx` | Brand, Client/Admin toggle tabs, "Federated via ActivityPub" |
| `ui/src/components/admin/Sidebar.tsx` | Nav items (Overview, Streams, Moderation, Federation) + user profile footer |
| `ui/src/components/admin/StreamCard.tsx` | Per-station card: LIVE/OFFLINE badge, listener count, inline HLS player (Monitor), relay/ingest controls (Settings) |
| `ui/src/components/admin/StreamCard.tsx` | Per-station card: LIVE/OFFLINE badge, listener count, inline HLS player (Monitor), relay/ingest settings, audio input selector, delete button for dynamic channels |
| `ui/src/components/admin/CreateChannelModal.tsx` | Modal for creating a new dynamic channel (username, display name, summary, relay policy, territory) |
| `ui/src/components/HLSPlayer.tsx` | hls.js audio player configured for live streaming |
| `ui/src/components/LogFeed.tsx` | Connects to `GET /admin/logs` SSE stream and displays log lines |

### HLS live playback configuration

`HLSPlayer.tsx` configures hls.js for minimum latency:

```ts
{
  liveSyncDurationCount: 1,       // start ~6 s from live edge (1 segment)
  liveMaxLatencyDurationCount: 3, // re-sync if more than 3 segments behind
  liveBackBufferLength: 0,        // no rewind buffer
}
```

### CORS headers on HLS endpoints

`ManifestHandler` and `SegmentHandler` now set `Access-Control-Allow-Origin: *`. This is required because hls.js resolves segment URLs from the absolute `cfg.BaseURL()`, which is a different origin from the Vite dev server (or any other cross-origin listener).

### Dev workflow

```bash
# Terminal 1 — Go server with Vite proxy enabled
go run -tags admindev ./cmd/server

# Terminal 2 — Vite dev server on :5173
cd ui && npm run dev
```

Vite proxy rules in `ui/vite.config.ts`:
- `/stations` → `http://localhost:8080`
- `^/admin(?!/ui)` → `http://localhost:8080` (regex skips the SPA path itself to avoid a proxy loop)

The UI is then available at `http://localhost:5173/` with hot-module reload.

### Production workflow

```bash
# Step 1 — build the SPA (outputs to ui/dist/, embedded by embed_prod.go)
cd ui && npm run build

# Step 2 — compile Go binary with embedded assets
go build ./...
```

The resulting binary serves the SPA at `/admin/ui/` with no external dependencies.

### VS Code integration

- `.vscode/launch.json` — **"Run Server (dev)"** launch config uses `-tags admindev`; **"Run Server (prod)"** builds and runs without the tag
- `.vscode/tasks.json` — **"Build UI"** shell task runs `npm run build` inside `ui/`

### Admin Web UI environment variables

| Variable | Default | Description |
|---|---|---|
| `ADMINUI_DEV_PROXY` | `http://localhost:5173` | Vite dev server URL (used in `admindev` build only) |
| `BROADCAST_SCRIPT` | `./bin/broadcast.sh` | Path to the broadcast script spawned by `ingest/start` |

---

## ✅ 7. Docker Packaging

A single Docker image containing both the `server` and `relay` binaries, the embedded admin UI, and all runtime dependencies (including ffmpeg for the Start Ingest feature).

**Done.** Three new files implement the full packaging and publishing pipeline:

- **`Dockerfile`** — multi-stage build: `node:22-alpine` builds the React SPA → `golang:1.26-alpine` compiles both Go binaries with the embedded UI → `alpine:3.21` runtime image with `ffmpeg`, `curl`, and `bash` installed. Both `./server` and `./relay` land at `/app/`.
- **`docker-compose.yml`** — example composition with a `source` service (running `./server`) and a `relay` service (overriding the command to `["./relay"]`).
- **`.github/workflows/docker.yml`** — GitHub Actions workflow that builds the image and pushes to `ghcr.io/jacksonopp/openwaves` on every push to `main` (tagged `latest`) and on `v*.*.*` git tags (tagged with the full semver, e.g. `1.2.3`, and the minor prefix `1.2`). PRs build the image but do not push.

### Quick start (pre-built image)

```bash
# Copy the example config and edit it for your environment:
cp config.yaml config.my.yaml

# Then bring up both services with Compose:
docker compose up
```

The `docker-compose.yml` in the repo contains a ready-to-use `source` + `relay` pair. Edit the bind-mounted `config.yaml` paths and environment variables to suit your deployment before starting.

### Building the image locally

The Dockerfile runs the UI build internally, so a plain `docker build` is self-contained:

```bash
docker build -t openwaves .
```

Alternatively, build via Compose (same result, uses the project name as the image tag):

```bash
docker compose build
```

If you want to build the Go binary outside Docker (e.g. for faster iteration), build the UI first so the embed step succeeds:

```bash
cd ui && npm run build   # outputs ui/dist/ — required for embed_prod.go
go build ./...
```

### Running the source server standalone

```bash
docker run -p 8080:8080 \
  -v ./config.yaml:/app/config.yaml:ro \
  -v openwaves-keys:/app/keys \
  ghcr.io/jacksonopp/openwaves:latest
```

- Config is bind-mounted read-only; override the path with the `CONFIG_PATH` env var.
- RSA key pairs are persisted in the named volume `openwaves-keys` at `/app/keys`.

### Running the relay standalone

```bash
docker run -p 8081:8081 \
  -e SOURCE_URL=http://source:8080/stations/morning-vibes \
  -e LOCAL_USERNAME=morning-vibes \
  -e PORT=8081 \
  -e KEYS_DIR=/app/keys-relay \
  -v relay-keys:/app/keys-relay \
  ghcr.io/jacksonopp/openwaves:latest ./relay
```

The relay binary is selected by passing `./relay` as the container command. All relay configuration is supplied via environment variables (see the relay env-var table in § 4 above).

### `docker-compose.yml` structure

The bundled Compose file defines two services:

| Service | Binary | Port | Config |
|---|---|---|---|
| `source` | `./server` (default entrypoint) | `8080` | bind-mounts `./config.yaml` → `/app/config.yaml`; named volume for `/app/keys` |
| `relay` | `./relay` (command override) | `8081` | all relay config via environment variables; named volume for `/app/keys-relay` |

Both services pull `ghcr.io/jacksonopp/openwaves:latest` by default; replace with a local build tag after `docker compose build`.

### GitHub Actions publishing

The workflow at `.github/workflows/docker.yml` runs on every `push` event:

| Trigger | Image tag(s) pushed |
|---|---|
| Push to `main` | `ghcr.io/jacksonopp/openwaves:latest` |
| Push of `v*.*.*` tag | `ghcr.io/jacksonopp/openwaves:1.2.3` **and** `ghcr.io/jacksonopp/openwaves:1.2` |
| Pull request | Image is built (to catch breakage) but **not** pushed |

Authentication uses the automatic `GITHUB_TOKEN`; no additional secrets are required for publishing to `ghcr.io` within the same repository.