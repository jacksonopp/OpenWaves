//go:build admindev

package adminui

import "io/fs"

// dist is nil in dev mode; Handler() will proxy to ADMINUI_DEV_PROXY instead.
var dist fs.FS
