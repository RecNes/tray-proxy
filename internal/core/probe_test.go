package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestProbeWithClientOKAndFail(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer bad.Close()

	lat, err := probeWithClient(ok.URL, &http.Client{Timeout: 2 * time.Second})
	if err != nil || lat <= 0 {
		t.Fatalf("ok probe: lat=%v err=%v", lat, err)
	}
	if _, err := probeWithClient(bad.URL, &http.Client{Timeout: 2 * time.Second}); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestProbeAllUsesProxyTransport(t *testing.T) {
	var sawProxy atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawProxy.Store(true)
		resp, err := http.Get(upstream.URL)
		if err != nil {
			w.WriteHeader(502)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
	}))
	defer proxySrv.Close()

	u, _ := url.Parse(proxySrv.URL)
	cands := []Proxy{{IP: u.Hostname(), Port: u.Port(), Code: "ZZ"}}
	got := ProbeAll(context.Background(), cands, upstream.URL, 2, 3*time.Second)
	if !sawProxy.Load() {
		t.Fatal("expected traffic via proxy")
	}
	if len(got) != 1 || got[0].LatencyS < 0 {
		t.Fatalf("%+v", got)
	}
}
