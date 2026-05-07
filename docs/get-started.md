# Where to Get Started (Development Order)

## ✅ 1. Define the Actor
Create the JSON-LD schema for an OpenWaves Station.

**Done.** The Station actor is implemented as an ActivityPub `Service` type in `internal/actor/station.go`, with a custom `ow:` JSON-LD namespace served at `/ns/openwave`. The HTTP server (`cmd/server/main.go`) serves actor documents at `GET /stations/{username}` and the context at `GET /ns/openwave`.

Core protocol features are also documented in `docs/core.md` and `README.md`, including: passive device compliance, broadcast termination signal, proof-of-listen telemetry, and license territory enforcement.

---

## 2. WebFinger Discovery
Implement the `.well-known/webfinger` logic so your Station is discoverable from other Fediverse apps (Mastodon, Pleroma, etc.).

The WebFinger endpoint (`GET /.well-known/webfinger?resource=acct:username@domain`) should return a JRD (JSON Resource Descriptor) linking to the Station actor URL. This is required for Fediverse interoperability.

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