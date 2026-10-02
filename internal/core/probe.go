package core

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	DefaultProbeURL     = "https://httpbin.org/ip"
	DefaultConcurrency  = 10
	DefaultProbeTimeout = 10 * time.Second
	DefaultScanLimit    = 100
)

func probeWithClient(probeURL string, client *http.Client) (float64, error) {
	start := time.Now()
	resp, err := client.Get(probeURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	return time.Since(start).Seconds(), nil
}

func ProbeAll(ctx context.Context, candidates []Proxy, probeURL string, concurrency int, timeout time.Duration) []Proxy {
	if concurrency < 1 {
		concurrency = 1
	}
	if probeURL == "" {
		probeURL = DefaultProbeURL
	}
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var ok []Proxy
	var wg sync.WaitGroup
	for _, c := range candidates {
		c := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			proxyURL, err := url.Parse("http://" + c.Address())
			if err != nil {
				return
			}
			client := &http.Client{
				Timeout: timeout,
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
			}
			lat, err := probeWithClient(probeURL, client)
			if err != nil {
				return
			}
			c.LatencyS = float64(int(lat*10+0.5)) / 10
			mu.Lock()
			ok = append(ok, c)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return ok
}
