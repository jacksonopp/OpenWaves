# OpenWaves — Next Steps

This document outlines the planned work after completing the core protocol implementation (Steps 1–6).

---

## ✅ 6. Admin Web UI

**Done.** A React + Vite SPA is embedded in the Go binary and served at `/admin/ui/`. The UI was subsequently redesigned to match the Figma spec (see [`docs/admin-ui-design.md`](./admin-ui-design.md)):

- Two-panel layout: fixed 256px sidebar with nav (Overview, Streams, Moderation, Federation) + scrollable main content
- **TanStack Router** for client-side routing; **TanStack Query** for data polling (`refetchInterval: 3000`); **CSS Modules** for scoped styles
- `StreamsPage` — live station list with per-card controls (Monitor inline HLS player, relay/ingest settings, stop)
- `OverviewPage` — station stats + live log feed
- Light theme with indigo accent (`#6366f1`) per Figma design

**Dynamic channels + audio input** were also added (see `docs/get-started.md § 5` for the full API):

- Channels can be created/deleted at runtime via `POST/DELETE /admin/channels` — no `config.yaml` edit required; persisted in `channels.json`
- `stream/start` now auto-starts ingest (defaulting to silence); `stream/stop` auto-stops ingest before terminating
- `StationStatus` includes `audioInput` and `isStatic` fields
- `CreateChannelModal` in the UI replaces `StartStreamModal`; `StreamCard` has an audio input selector (Silence / Test Tone / File) and a delete button for dynamic channels
- `keystore.Store` (thread-safe) replaces the raw `map[string]*rsa.PrivateKey` throughout the server

See `internal/adminui/`, `internal/broadcaster/`, `internal/config/`, `internal/keystore/`, and `ui/` for the implementation.

---

## ✅ 7. Docker Packaging

**Done.** A single image (`ghcr.io/jacksonopp/openwaves`) contains both the `server` and `relay` binaries, the embedded admin UI, and `ffmpeg` for the Start Ingest feature. Keys are persisted via named Docker volumes; config is bind-mounted. The GitHub Actions workflow (`.github/workflows/docker.yml`) publishes `latest` on pushes to `main` and semver tags on `v*.*.*` git tags; PRs build but do not push. See `docs/get-started.md § 7` for the full quick-start, standalone run commands, Compose structure, and publishing details. New files: `Dockerfile`, `docker-compose.yml`, `.github/workflows/docker.yml`.

---

## 8. Client/User UI

A separate application for listeners (not admins).

- Discovers stations via WebFinger
- Renders the HLS stream in-browser
- Displays station metadata (`isLive`, listener count, station name)
- This is a separate deliverable from the admin UI

---

## 9. Announce Activity (Go-Live Notifications)

When a station transitions from offline to live (first segment received after a period of inactivity), the server should publish an `Announce` activity to all followers' inboxes, containing the HLS manifest URL.

- Detect transition: `!wasLive && store.IsLive(username)` — can be checked in the ingest handler after a successful `store.Add()`
- Activity payload: standard ActivityPub `Announce` with `object` pointing to the station actor URL and an `attachment` or `url` field with the manifest URL
- Send to all followers in `inbox.FollowerStore` for that station

This enables Mastodon/Pleroma users who follow a station to receive a toot-style notification the moment a broadcast starts.

---

## 10. HTTP Signature Verification on Inbox

The inbox currently accepts any POST without verifying the sender's identity. A malicious actor could send arbitrary `Follow` or `ProofOfListen` activities.

- Implement ActivityPub HTTP Signature verification (`Signature` header) on all inbox requests
- Fetch the sender's actor document to retrieve their public key, then verify the request signature
- Cache fetched public keys with a short TTL to avoid per-request fetches
- Return `401 Unauthorized` for requests with missing or invalid signatures
- This is required for production federation with the broader Fediverse

---

## 11. Persistent Follower Store

The follower store is currently in-memory. If the server restarts, all relay subscriptions are lost — relays would need to re-send `Follow` activities to reconnect.

- Replace `inbox.FollowerStore`'s in-memory map with a simple file-backed or SQLite-backed store
- Persist on every `Add`/`Remove` operation
- Load on startup
- Minimal schema: `(station_username, actor_url, inbox_url)`

---

## 12. Allowlist Relay Policy

The `relay_policy: allowlist` config value is defined but not enforced — it currently behaves the same as `open`. 

- Add an `allowlist` section per station in `config.yaml` (array of allowed actor URLs or domains)
- In the inbox `Follow` handler, when `relay_policy == "allowlist"`, check the actor URL against the allowlist before accepting
- Return a `Reject` activity if the actor is not on the list

---

## 13. Defederation Controls

The protocol spec (`docs/core.md`) describes defederation — admins muting or blocking specific federated streams. Not yet implemented.

- Admin API endpoints: `POST /admin/stations/{username}/block` and `/unblock` (body: `{"actor_url":"..."}`)
- Block list stored alongside the follower store
- Relay poller: before storing a segment, check if the source is blocked
- Inbox: reject Follow activities from blocked actors

---

## Notes

- **FFmpeg dependency**: FFmpeg is bundled in the Docker image (`alpine:3.21` runtime stage) so the "Start Ingest" button in the admin UI can spawn `bin/broadcast.sh` inside the container. For deployments outside Docker, FFmpeg must still be installed on whatever machine runs `bin/broadcast.sh`. The `ffplay` command (used for dev-only local stream monitoring) is a separate concern and is not included in the image.
- **openwaves:// URI scheme**: Mentioned in `docs/core.md` as a deep-link format for client apps. Relevant once the client/user UI is being built.
- **HTTPS**: Production deployments should run behind a reverse proxy (nginx, Caddy) that terminates TLS. The server itself uses `scheme: https` in config to generate correct actor URLs without needing to handle TLS directly.
