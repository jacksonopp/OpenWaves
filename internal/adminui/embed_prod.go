//go:build !admindev

package adminui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distEmbed embed.FS

var dist fs.FS = distEmbed
