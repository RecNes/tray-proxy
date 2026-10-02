package core

import "testing"

func TestProxyAddress(t *testing.T) {
	p := Proxy{IP: "1.2.3.4", Port: "8080"}
	if got := p.Address(); got != "1.2.3.4:8080" {
		t.Fatalf("Address() = %q", got)
	}
}

func TestFiltersDefaultAllEnabled(t *testing.T) {
	f := DefaultFilters()
	if !f.HTTPS || !f.Elite || !f.Anonymous || !f.Transparent {
		t.Fatalf("DefaultFilters should enable all: %+v", f)
	}
}
