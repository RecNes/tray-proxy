package core

import "testing"

func sample() []Proxy {
	return []Proxy{
		{IP: "1.1.1.1", Port: "1", Anonymity: AnonymityElite, HTTPS: true, LatencyS: 2.0},
		{IP: "2.2.2.2", Port: "2", Anonymity: AnonymityAnonymous, HTTPS: false, LatencyS: 0.5},
		{IP: "3.3.3.3", Port: "3", Anonymity: AnonymityTransparent, HTTPS: true, LatencyS: 1.0},
	}
}

func TestApplyFiltersHTTPSOnly(t *testing.T) {
	f := Filters{HTTPS: true, Elite: true, Anonymous: true, Transparent: true}
	got := ApplyFilters(sample(), f)
	if len(got) != 2 {
		t.Fatalf("len=%d got=%+v", len(got), got)
	}
}

func TestApplyFiltersAnonymity(t *testing.T) {
	f := Filters{HTTPS: false, Elite: true, Anonymous: false, Transparent: false}
	got := ApplyFilters(sample(), f)
	if len(got) != 1 || got[0].IP != "1.1.1.1" {
		t.Fatalf("%+v", got)
	}
}

func TestTopNSortsByLatency(t *testing.T) {
	got := TopN(sample(), 2)
	if got[0].IP != "2.2.2.2" || got[1].IP != "3.3.3.3" {
		t.Fatalf("%+v", got)
	}
}
