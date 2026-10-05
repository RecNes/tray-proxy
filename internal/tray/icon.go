package tray

import (
	trayassets "tray-proxy/assets"
)

// iconData is the app icon shared with the exe/installer icon
// (assets/trayproxy.ico, generated from assets/logo.png via
// `go run ./tools/icongen`).
var iconData = trayassets.TrayIcon
