// Package static embeds the OpenWaves JSON-LD context file for use in the server binary.
package static

import _ "embed"

//go:embed ns/openwaves.jsonld
var OpenWavesContext []byte
