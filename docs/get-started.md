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

## 4. The Relay Logic
This is the most unique feature. Build the code that allows Server B to subscribe to and re-host HLS segments from Server A, implementing:

- License territory check before accepting a stream
- Proof-of-listen heartbeat (signed, every 30s) back to the source
- Broadcast termination signal handling (`TerminateStream` ActivityPub activity) with cascading shutdown