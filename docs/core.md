# OpenWaves: Core Protocol Features
## 1. Federated Audio Relay (The "Bucket Brigade")
Unlike current decentralized streaming (where every listener connects to the source), OpenWaves allows servers to "help" each other.

Bandwidth Scaling: Server B can follow Server A and act as a "repeater," pulling a single stream and fanning it out to its own local users.

Reduced Source Load: The original broadcaster only needs enough bandwidth to reach a few federated peers, rather than thousands of individual listeners.

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