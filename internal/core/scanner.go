package core

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

type Fetcher func(client *http.Client) ([]Proxy, error)
type Prober func(ctx context.Context, candidates []Proxy) []Proxy

type Scanner struct {
	fetch  Fetcher
	probe  Prober
	client *http.Client
	busy   atomic.Bool
}

func NewScanner(fetch Fetcher, probe Prober) *Scanner {
	if fetch == nil {
		fetch = FetchProxyList
	}
	if probe == nil {
		probe = func(ctx context.Context, cands []Proxy) []Proxy {
			return ProbeAll(ctx, cands, DefaultProbeURL, DefaultConcurrency, DefaultProbeTimeout)
		}
	}
	return &Scanner{
		fetch:  fetch,
		probe:  probe,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Scanner) TryStart() bool {
	return s.busy.CompareAndSwap(false, true)
}

func (s *Scanner) Finish() {
	s.busy.Store(false)
}

func (s *Scanner) IsScanning() bool {
	return s.busy.Load()
}

func (s *Scanner) Run(ctx context.Context) ([]Proxy, error) {
	if !s.TryStart() {
		return nil, ErrScanInProgress
	}
	defer s.Finish()
	list, err := s.fetch(s.client)
	if err != nil {
		return nil, err
	}
	if len(list) > DefaultScanLimit {
		list = list[:DefaultScanLimit]
	}
	return s.probe(ctx, list), nil
}

var ErrScanInProgress = errScanInProgress{}

type errScanInProgress struct{}

func (errScanInProgress) Error() string { return "scan already in progress" }
