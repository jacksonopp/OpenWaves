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

## Protocol goals

- Support federated live audio at scale
- Keep broadcasts ephemeral unless explicitly extended
- Reuse ActivityPub for discovery and stream signaling
- Make tuning in simple and public
- Provide moderation controls for operators
