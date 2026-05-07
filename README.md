# OpenWaves

OpenWaves is a decentralized live audio protocol for federated radio-style broadcasting. It is designed for live moments, not long-term archives, and uses the Fediverse for discovery, signaling, and control.

## Core protocol features

### Federated audio relay
Servers can help each other relay live streams. Instead of every listener connecting to the source, one server can follow another, pull a single stream, and fan it out locally. This reduces load on the original broadcaster and makes scaling easier.

### Ephemeral by default
OpenWaves is built around live HLS chunks that are purged after playback. Metadata includes expiration timestamps so the network can forget broadcasts when they end.

### ActivityPub control plane
Each station is modeled as an ActivityPub Service actor. Followers can receive Announce activities when a stream goes live, including the HLS manifest URL.

### Public-by-default discovery
The protocol favors open access and simple tuning-in. It uses the `openwaves://station@domain` URI scheme for deep linking into compatible apps.

### Admin and moderation hooks
Relays can mute or block federated streams, and source audio chunks are cryptographically signed to prevent spoofing or impersonation.

### Passive device compliance
Relay servers operate as passive transmission devices. They must not re-encode audio, inject advertisements, or alter segment content. Cryptographic chunk signatures ensure any modification is detectable by listeners.

### Broadcast termination
The source server can send a signed `TerminateStream` activity to immediately shut down all relaying servers. Relays cascade the signal to their own downstream peers, enabling a network-wide shutdown from a single signal.

### Proof-of-listen telemetry
Relay servers send cryptographically signed heartbeats back to the source every 30 seconds with an aggregate listener count. This gives broadcasters real-time visibility across the relay graph without tracking individual listeners.

### License territory enforcement
Stations declare a `licenseTerritory` field (ISO 3166-1 alpha-2 country codes) in their actor metadata. Relay servers must check this field against their own jurisdiction before accepting a stream, enabling broadcasters to respect geographic licensing agreements.

## Protocol goals

- Support federated live audio at scale
- Keep broadcasts ephemeral unless explicitly extended
- Reuse ActivityPub for discovery and stream signaling
- Make tuning in simple and public
- Provide moderation controls for operators
- Enforce relay passivity to protect content integrity
- Enable broadcasters to terminate streams across the relay graph instantly
- Provide aggregate listener telemetry without individual tracking
- Respect geographic licensing through territory-aware relay enforcement
