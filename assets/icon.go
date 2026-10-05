package assets

import (
	_ "embed"
)

// TrayIcon is the app icon (.ico with 16–256px entries, generated from
// logo.png via `go run ./tools/icongen`). Embedded here so the systray
// icon (internal/tray), the exe file icon (via rsrc at link time) and the
// installer all share the same source.
//
//go:embed trayproxy.ico
var TrayIcon []byte
