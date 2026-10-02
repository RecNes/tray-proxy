package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProxiesLimitAndFields(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "sample_table.html"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseProxies(string(b), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Address() != "1.1.1.1:8080" || !got[0].HTTPS || got[0].Anonymity != AnonymityElite || got[0].Code != "US" {
		t.Fatalf("row0=%+v", got[0])
	}
	if got[1].HTTPS || got[1].Anonymity != AnonymityAnonymous {
		t.Fatalf("row1=%+v", got[1])
	}
	limited, err := ParseProxies(string(b), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 2 {
		t.Fatalf("limit 2 => %d", len(limited))
	}
}
