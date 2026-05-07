// Package static embeds the OpenWave JSON-LD context file for use in the server binary.
package static

import _ "embed"

//go:embed ns/openwave.jsonld
var OpenwaveContext []byte
