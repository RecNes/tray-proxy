package tray

import (
	"encoding/binary"
	"testing"
)

// The systray icon must be a real multi-entry .ico (the app logo shared
// with the exe/installer icon), not the old 16x16 placeholder.
func TestIconDataIsValidICO(t *testing.T) {
	if len(iconData) < 6 {
		t.Fatalf("iconData too short: %d bytes", len(iconData))
	}
	if reserved := binary.LittleEndian.Uint16(iconData[0:2]); reserved != 0 {
		t.Errorf("ICO reserved = %d, want 0", reserved)
	}
	if typ := binary.LittleEndian.Uint16(iconData[2:4]); typ != 1 {
		t.Errorf("ICO type = %d, want 1 (icon)", typ)
	}
	count := binary.LittleEndian.Uint16(iconData[4:6])
	if count < 2 {
		t.Errorf("ICO entries = %d, want >= 2 (systray picks from sizes)", count)
	}
}
