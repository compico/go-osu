// Package assets embeds static binary assets bundled with the application.
package assets

import _ "embed"

//go:embed icon.ico
var IconIco []byte
