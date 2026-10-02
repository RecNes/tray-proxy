package proxyos

import (
	"encoding/json"
	"testing"
)

func TestSnapshotJSONRoundTrip(t *testing.T) {
	s := Snapshot{ProxyEnable: 1, ProxyServer: "1.2.3.4:8080", ProxyOverride: "<local>", Valid: true}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var got Snapshot
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != s {
		t.Fatalf("%+v", got)
	}
}
