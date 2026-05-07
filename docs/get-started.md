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

## 3. HLS Implementation
Build the core logic that takes an audio input (likely via FFmpeg) and segments it into the HLS `.m3u8` format.

Each segment should be cryptographically signed by the originating server (as per the passive device compliance requirement) so that listeners and relays can verify content integrity.

---

## 4. The Relay Logic
This is the most unique feature. Build the code that allows Server B to subscribe to and re-host HLS segments from Server A, implementing:

- License territory check before accepting a stream
- Proof-of-listen heartbeat (signed, every 30s) back to the source
- Broadcast termination signal handling (`TerminateStream` ActivityPub activity) with cascading shutdown