# Where to Get Started (Development Order)
1. Define the Actor: Create the JSON-LD schema for an OpenWaves Station.

2. WebFinger Discovery: Implement the .well-known/webfinger logic so your "Station" is searchable from other Fediverse apps.

3. HLS Implementation: Build the core logic that takes an audio input (likely via FFmpeg) and segments it into the HLS .m3u8 format.

4. The Relay Logic: This is your most unique feature. Focus on the code that allows Server B to request and re-host those HLS segments from Server A.