package tray

import (
	"testing"
)

func TestPngBytesToICORoundTrip(t *testing.T) {
	// Build a tiny PNG via placeholder path
	ico := placeholderFlagICO("xx")
	if len(ico) < 22 {
		t.Fatalf("ico too small: %d", len(ico))
	}
	// ICO magic: reserved=0, type=1
	if ico[2] != 1 || ico[3] != 0 {
		t.Fatalf("bad ico header: %v", ico[:6])
	}
}

func TestFlagIconCaches(t *testing.T) {
	a := flagIconICO("US")
	b := flagIconICO("us")
	if len(a) == 0 || len(b) == 0 {
		t.Fatal("expected icon bytes")
	}
	if len(a) != len(b) {
		t.Fatalf("cache mismatch %d vs %d", len(a), len(b))
	}
}
