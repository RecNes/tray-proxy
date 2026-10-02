package core

import "testing"

func TestScannerTryStartLock(t *testing.T) {
	s := NewScanner(nil, nil)
	if !s.TryStart() {
		t.Fatal("first start")
	}
	if s.TryStart() {
		t.Fatal("second start must fail")
	}
	s.Finish()
	if !s.TryStart() {
		t.Fatal("after finish")
	}
	s.Finish()
}
