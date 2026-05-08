# OpenWaves: Core Protocol Features
## 1. Federated Audio Relay (The "Bucket Brigade")
Unlike current decentralized streaming (where every listener connects to the source), OpenWaves allows servers to "help" each other.

Bandwidth Scaling: Server B can follow Server A and act as a "repeater," pulling a single stream and fanning it out to its own local users.

Reduced Source Load: The original broadcaster only needs enough bandwidth to reach a few federated peers, rather than thousands of individual listeners.

License Territory Enforcement: Each station actor includes a `licenseTerritory` field containing an array of ISO 3166-1 alpha-2 country codes (e.g. `["US", "CA"]`); the special value `["*"]` means worldwide. Before a relay server accepts and re-hosts a stream, it MUST check this field against its own declared jurisdiction. If the relay's jurisdiction is not in the list, it MUST refuse to relay that stream.

## 2. Ephemeral-by-Default Architecture
Designed for live moments, not permanent archives.

No Native Storage: The protocol is optimized for live HLS (HTTP Live Streaming) chunks that are purged from the relay servers as soon as they are played.

TTL (Time-to-Live): Metadata packets include an expiration timestamp, instructing the network to "forget" the broadcast once it ends.

## 3. ActivityPub Control Plane
Leverages the existing Fediverse social graph for discovery and signaling.

Actor Model: Each radio station is a Service actor that can be followed by any Mastodon, Pleroma, or PixelFed user.

Live Signals: Uses the Announce activity to notify followers the moment a stream goes live, including the HLS manifest URL in the payload.

## 4. Public-by-Default Discovery
Optimized for the "Digital Public Square."

Open Access: Handshakes are unencrypted by default (though the transport layer uses TLS) to allow for "tuning in" without complex key exchanges.
 
openwaves:// URI Scheme: A custom deep-linking format (openwaves://station@domain) that allows mobile and desktop apps to instantly launch the radio tuner.

## 5. Admin & Moderation Hooks
Built-in tools to handle the legal and social risks of decentralized broadcasting.

Defederation Signals: Admins can "mute" or "block" specific federated streams at the relay level if they contain infringing or illegal content.

Source Attribution: Every audio chunk is cryptographically signed by the originating server to prevent "stream spoofing" or impersonation.

Broadcast Termination Signal: The source server admin can send a `TerminateStream` signal at any time via the admin API (`POST /admin/stations/{username}/stream/stop`). This clears the source's segment store and propagates a `TerminateStream` ActivityPub activity to all relay followers, triggering a cascading shutdown across the relay graph. **TerminateStream is an admin-only action on the source server** — relay servers and external actors cannot trigger it via the ActivityPub inbox. A short grace window of 5 seconds is allowed to complete any in-flight segment requests before purging.

## 6. Passive Device Compliance
OpenWaves relay servers are designed to function as passive transmission devices — analogous to how the FCC classifies cable retransmission equipment — rather than as active content retransmitters.

Relays MUST NOT: re-encode or transcode audio, inject advertisements, alter HLS segment content, or modify stream metadata fields.

Relays MAY: buffer segments temporarily for local fan-out, serve those segments to local listeners, and enforce defederation by dropping a stream entirely.

The cryptographic signatures applied to each audio chunk by the originating server serve as the technical enforcement mechanism: any relay that modifies segment content will produce chunks that fail signature verification at the listener's client, breaking the stream.

## 7. Proof-of-Listen Telemetry
Each relay server is required to send a cryptographically signed heartbeat back to the source server at a regular interval (every 30 seconds) while actively relaying a stream.

The heartbeat payload includes: `relayId` (the relay's ActivityPub actor URL), `streamId` (the stream being relayed), `listenerCount` (aggregate number of active listeners on that relay), `timestamp`, and a `signature` generated using the relay's ActivityPub HTTP Signature private key.

Heartbeats report aggregate listener counts only and never include individual listener identities or connection metadata. Listener counts include **both direct listeners** (clients fetching the manifest directly from a server) **and relay-reported listeners** (counts from downstream relays via heartbeat). The source server uses these heartbeats to build a real-time aggregate listener count across the relay graph. A relay that stops sending heartbeats for more than 60 seconds is considered offline by the source.