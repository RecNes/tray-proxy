# Proxy Tray Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Windows systray Go app that scrapes/tests free proxies from free-proxy-list.net and applies/clears an OS-wide proxy from the tray menu.

**Architecture:** Single-process Go binary with `internal/core` (scrape/test/filter), `internal/config` (persist settings + cache), `internal/proxyos` (WinINET apply/restore), `internal/tray` (systray UI), and `cmd/proxytray` entrypoint.

**Tech Stack:** Go 1.22+, `github.com/PuerkitoBio/goquery`, `github.com/energye/systray`, standard library `net/http`, Windows registry via `golang.org/x/sys/windows/registry`, WinINET via `golang.org/x/sys/windows`.

## Global Constraints

- Windows-only for v1; OS calls isolated behind `proxyos.Manager` interface
- Not a Windows Service — tray app only
- Scan interval: 30 minutes; manual Refresh allowed only when idle
- Test at most first 100 table rows per scan; concurrency 10; probe timeout 10s
- Menu shows top 10 after filters; country flag emoji + ISO code
- Filters: HTTPS, Elite, Anonymous, Transparent — checkable submenu; persist to config
- Start with Windows: off by default; toggle in menu
- Quit always restores original system proxy
- Active proxy missing from new top-10: keep applied, still visible
- Data dir: `%AppData%\proxytray\` (`config.json`, `cache.json`, `proxytray.log`)
- Module path: `proxy-tray` (local module name)
- Spec: `docs/superpowers/specs/2026-10-02-proxy-tray-design.md`

## File structure

| Path | Responsibility |
|------|----------------|
| `go.mod` | Module + deps |
| `cmd/proxytray/main.go` | Entrypoint, wire deps, start tray |
| `internal/core/types.go` | `Proxy`, `Filters`, anonymity constants |
| `internal/core/parse.go` | HTML scrape/parse |
| `internal/core/filter.go` | Filter + top-N by latency |
| `internal/core/probe.go` | Concurrent latency probe |
| `internal/core/scanner.go` | Scan lock, scrape→probe→cache update |
| `internal/core/flag.go` | ISO country code → flag emoji |
| `internal/core/testdata/sample_table.html` | Parse fixture |
| `internal/config/config.go` | Load/save config + cache paths |
| `internal/proxyos/proxyos.go` | `Manager` interface |
| `internal/proxyos/windows.go` | WinINET apply/restore (`//go:build windows`) |
| `internal/proxyos/stub.go` | No-op stub for non-Windows tests (`//go:build !windows`) |
| `internal/proxyos/startup_windows.go` | Start with Windows registry (`//go:build windows`) |
| `internal/tray/app.go` | Systray menu build + handlers |
| `internal/tray/icon.go` | Embedded tray icon bytes |
| `README.md` | Build/run instructions |

---

### Task 1: Module scaffold + core types

**Files:**
- Create: `go.mod`
- Create: `internal/core/types.go`
- Create: `internal/core/types_test.go`
- Create: `internal/core/flag.go`
- Create: `internal/core/flag_test.go`

**Interfaces:**
- Consumes: nothing
- Produces: `core.Proxy`, `core.Anonymity`, `core.Filters`, `core.FlagEmoji(code string) string`, `Proxy.Address() string`

- [ ] **Step 1: Init module**

```bash
cd "C:/Users/Botano User/Projects/proxy-tray"
go mod init proxy-tray
```

Expected: `go.mod` exists with `module proxy-tray`.

- [ ] **Step 2: Write failing tests**

Create `internal/core/types_test.go`:

```go
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
```

Create `internal/core/flag_test.go`:

```go
package core

import "testing"

func TestFlagEmojiTR(t *testing.T) {
	got := FlagEmoji("TR")
	want := string([]rune{0x1F1F9, 0x1F1F7}) // 🇹🇷
	if got != want {
		t.Fatalf("FlagEmoji(TR) = %q want %q", got, want)
	}
}

func TestFlagEmojiEmpty(t *testing.T) {
	if FlagEmoji("") != "" {
		t.Fatal("empty code should yield empty flag")
	}
}
```

- [ ] **Step 3: Run tests — expect FAIL**

```bash
go test ./internal/core/ -count=1
```

Expected: FAIL (undefined types / functions).

- [ ] **Step 4: Implement types + flag**

`internal/core/types.go`:

```go
package core

type Anonymity string

const (
	AnonymityElite       Anonymity = "elite proxy"
	AnonymityAnonymous   Anonymity = "anonymous"
	AnonymityTransparent Anonymity = "transparent"
)

type Proxy struct {
	IP        string    `json:"ip"`
	Port      string    `json:"port"`
	Code      string    `json:"code"`
	Country   string    `json:"country"`
	Anonymity Anonymity `json:"anonymity"`
	HTTPS     bool      `json:"https"`
	LatencyS  float64   `json:"latency_s"` // set after probe; 0 if untested
}

func (p Proxy) Address() string {
	return p.IP + ":" + p.Port
}

type Filters struct {
	HTTPS       bool `json:"https"`
	Elite       bool `json:"elite"`
	Anonymous   bool `json:"anonymous"`
	Transparent bool `json:"transparent"`
}

func DefaultFilters() Filters {
	return Filters{HTTPS: true, Elite: true, Anonymous: true, Transparent: true}
}
```

`internal/core/flag.go`:

```go
package core

import "unicode"

// FlagEmoji maps an ISO 3166-1 alpha-2 code to a regional-indicator flag emoji.
func FlagEmoji(code string) string {
	if len(code) != 2 {
		return ""
	}
	a := unicode.ToUpper(rune(code[0]))
	b := unicode.ToUpper(rune(code[1]))
	if a < 'A' || a > 'Z' || b < 'A' || b > 'Z' {
		return ""
	}
	return string([]rune{0x1F1E6 + (a - 'A'), 0x1F1E6 + (b - 'A')})
}
```

- [ ] **Step 5: Run tests — expect PASS**

```bash
go test ./internal/core/ -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod internal/core/
git commit -m "feat: add core types and country flag helper"
```

---

### Task 2: HTML parse (first 100 rows)

**Files:**
- Create: `internal/core/parse.go`
- Create: `internal/core/parse_test.go`
- Create: `internal/core/testdata/sample_table.html`
- Modify: `go.mod` / `go.sum` (add goquery)

**Interfaces:**
- Consumes: `core.Proxy`, `core.Anonymity`
- Produces: `ParseProxies(html string, limit int) ([]Proxy, error)`, `FetchProxyList(client *http.Client) ([]Proxy, error)`

- [ ] **Step 1: Add dependency + fixture**

```bash
go get github.com/PuerkitoBio/goquery@v1.9.2
```

Create `internal/core/testdata/sample_table.html` with a minimal table matching free-proxy-list columns:

```html
<html><body><table><tbody>
<tr><td>1.1.1.1</td><td>8080</td><td>US</td><td>United States</td><td>elite proxy</td><td>no</td><td>yes</td><td>1 min ago</td></tr>
<tr><td>2.2.2.2</td><td>80</td><td>TR</td><td>Turkey</td><td>anonymous</td><td>no</td><td>no</td><td>1 min ago</td></tr>
<tr><td>3.3.3.3</td><td>3128</td><td>DE</td><td>Germany</td><td>transparent</td><td>yes</td><td>yes</td><td>1 min ago</td></tr>
</tbody></table></body></html>
```

- [ ] **Step 2: Write failing parse test**

```go
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
```

- [ ] **Step 3: Run test — expect FAIL**

```bash
go test ./internal/core/ -run TestParseProxies -count=1
```

Expected: FAIL undefined `ParseProxies`.

- [ ] **Step 4: Implement parse**

`internal/core/parse.go`:

```go
package core

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const ProxyListURL = "https://free-proxy-list.net/"

func ParseProxies(html string, limit int) ([]Proxy, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	var out []Proxy
	doc.Find("tbody tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		if limit > 0 && len(out) >= limit {
			return false
		}
		tds := tr.Find("td")
		if tds.Length() < 7 {
			return true
		}
		ip := strings.TrimSpace(tds.Eq(0).Text())
		port := strings.TrimSpace(tds.Eq(1).Text())
		if ip == "" || port == "" {
			return true
		}
		anon := Anonymity(strings.ToLower(strings.TrimSpace(tds.Eq(4).Text())))
		https := strings.EqualFold(strings.TrimSpace(tds.Eq(6).Text()), "yes")
		out = append(out, Proxy{
			IP:        ip,
			Port:      port,
			Code:      strings.TrimSpace(tds.Eq(2).Text()),
			Country:   strings.TrimSpace(tds.Eq(3).Text()),
			Anonymity: anon,
			HTTPS:     https,
		})
		return true
	})
	return out, nil
}

func FetchProxyList(client *http.Client) ([]Proxy, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Get(ProxyListURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("proxy list HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ParseProxies(string(body), 100)
}
```

- [ ] **Step 5: Run test — expect PASS**

```bash
go test ./internal/core/ -run TestParseProxies -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/core/
git commit -m "feat: parse free-proxy-list HTML table"
```

---

### Task 3: Filters + top-N

**Files:**
- Create: `internal/core/filter.go`
- Create: `internal/core/filter_test.go`

**Interfaces:**
- Consumes: `Proxy`, `Filters`
- Produces: `ApplyFilters(proxies []Proxy, f Filters) []Proxy`, `TopN(proxies []Proxy, n int) []Proxy`

- [ ] **Step 1: Write failing tests**

```go
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
	if len(got) != 2 { // elite https + transparent https; anonymous HTTP excluded
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
```

**Filter semantics (locked):**
- Anonymity: proxy included if its class checkbox is true.
- HTTPS: if `f.HTTPS` is true, only `proxy.HTTPS == true`; if false, no HTTPS restriction (HTTP and HTTPS both allowed).

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/core/ -run "TestApplyFilters|TestTopN" -count=1
```

Expected: FAIL undefined.

- [ ] **Step 3: Implement**

```go
package core

import "sort"

func ApplyFilters(proxies []Proxy, f Filters) []Proxy {
	out := make([]Proxy, 0, len(proxies))
	for _, p := range proxies {
		if f.HTTPS && !p.HTTPS {
			continue
		}
		switch p.Anonymity {
		case AnonymityElite:
			if !f.Elite {
				continue
			}
		case AnonymityAnonymous:
			if !f.Anonymous {
				continue
			}
		case AnonymityTransparent:
			if !f.Transparent {
				continue
			}
		default:
			continue
		}
		out = append(out, p)
	}
	return out
}

func TopN(proxies []Proxy, n int) []Proxy {
	cp := append([]Proxy(nil), proxies...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].LatencyS < cp[j].LatencyS })
	if n < 0 {
		n = 0
	}
	if len(cp) > n {
		cp = cp[:n]
	}
	return cp
}

// MenuProxies applies filters then returns fastest n.
func MenuProxies(validated []Proxy, f Filters, n int) []Proxy {
	return TopN(ApplyFilters(validated, f), n)
}
```

Fix test `TestApplyFiltersHTTPSOnly` expectation: with HTTPS true and all anonymity true → elite HTTPS + transparent HTTPS = 2. Anonymous HTTP excluded. Good.

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/core/ -run "TestApplyFilters|TestTopN" -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/filter.go internal/core/filter_test.go
git commit -m "feat: filter proxies and select top-N by latency"
```

---

### Task 4: Probe + scanner lock

**Files:**
- Create: `internal/core/probe.go`
- Create: `internal/core/probe_test.go`
- Create: `internal/core/scanner.go`
- Create: `internal/core/scanner_test.go`

**Interfaces:**
- Consumes: `Proxy`, `ParseProxies` / `FetchProxyList`
- Produces:
  - `ProbeAll(ctx, candidates []Proxy, probeURL string, concurrency int, timeout time.Duration) []Proxy`
  - `type Scanner struct` with `TryStart() bool`, `Finish()`, `IsScanning() bool`, `Run(ctx) ([]Proxy, error)`
  - Constants: `DefaultProbeURL = "https://httpbin.org/ip"`, `DefaultConcurrency = 10`, `DefaultProbeTimeout = 10*time.Second`, `DefaultScanLimit = 100`

- [ ] **Step 1: Write failing probe + scanner tests**

Create `internal/core/probe_test.go`:

```go
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
	if len(got) != 1 || got[0].LatencyS <= 0 {
		t.Fatalf("%+v", got)
	}
}
```

Create `internal/core/scanner_test.go`:

```go
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
```

Use HTTP httptest URLs only (avoid HTTPS CONNECT).

- [ ] **Step 2: Run tests — expect FAIL**

```bash
go test ./internal/core/ -run "TestProbe|TestScanner" -count=1
```

Expected: FAIL (undefined symbols).

- [ ] **Step 3: Implement probe.go**

```go
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
```

- [ ] **Step 4: Implement scanner.go**

```go
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
```

- [ ] **Step 5: Run tests — PASS**

```bash
go test ./internal/core/ -count=1
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/core/probe.go internal/core/probe_test.go internal/core/scanner.go internal/core/scanner_test.go
git commit -m "feat: concurrent proxy probe and scan lock"
```

### Task 5: Config + disk cache

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Interfaces:**
- Consumes: `core.Filters`, `core.Proxy`
- Produces:
  - `type Config struct { Filters core.Filters; StartWithWindows bool }`
  - `Dir() (string, error)` → `%AppData%\proxytray`
  - `Load() (Config, error)`, `Save(Config) error`
  - `LoadCache() ([]core.Proxy, error)`, `SaveCache([]core.Proxy) error`
  - `LogPath() (string, error)`

- [ ] **Step 1: Failing test with t.TempDir override**

Implement `config.SetRootForTest(dir string)` or pass root into `Store`:

```go
type Store struct{ Root string }

func (s Store) Load() (Config, error)
func (s Store) Save(Config) error
func (s Store) LoadCache() ([]core.Proxy, error)
func (s Store) SaveCache([]core.Proxy) error
```

Test:

```go
func TestSaveLoadRoundTrip(t *testing.T) {
	st := Store{Root: t.TempDir()}
	cfg := Config{Filters: core.DefaultFilters(), StartWithWindows: false}
	cfg.Filters.Transparent = false
	if err := st.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Filters.Transparent || !got.Filters.Elite {
		t.Fatalf("%+v", got)
	}
	proxies := []core.Proxy{{IP: "1.1.1.1", Port: "80", LatencyS: 0.5}}
	if err := st.SaveCache(proxies); err != nil {
		t.Fatal(err)
	}
	c2, err := st.LoadCache()
	if err != nil || len(c2) != 1 || c2[0].IP != "1.1.1.1" {
		t.Fatalf("%v %+v", err, c2)
	}
}
```

Default `Root`: `filepath.Join(os.Getenv("APPDATA"), "proxytray")` on Windows; fallback `filepath.Join(home, ".proxytray")`.

- [ ] **Step 2–4: Implement, pass tests, commit**

```bash
git commit -m "feat: persist config and proxy cache under AppData"
```

Implementation sketch:

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"proxy-tray/internal/core"
)

type Config struct {
	Filters          core.Filters `json:"filters"`
	StartWithWindows bool         `json:"start_with_windows"`
}

func Default() Config {
	return Config{Filters: core.DefaultFilters(), StartWithWindows: false}
}

type Store struct{ Root string }

func DefaultStore() Store {
	root := filepath.Join(os.Getenv("APPDATA"), "proxytray")
	if os.Getenv("APPDATA") == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".proxytray")
	}
	return Store{Root: root}
}

func (s Store) ensure() error { return os.MkdirAll(s.Root, 0o755) }

func (s Store) Load() (Config, error) {
	b, err := os.ReadFile(filepath.Join(s.Root, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (s Store) Save(c Config) error {
	if err := s.ensure(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Root, "config.json"), b, 0o644)
}

func (s Store) LoadCache() ([]core.Proxy, error) {
	b, err := os.ReadFile(filepath.Join(s.Root, "cache.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var p []core.Proxy
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s Store) SaveCache(p []core.Proxy) error {
	if err := s.ensure(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Root, "cache.json"), b, 0o644)
}

func (s Store) LogPath() string { return filepath.Join(s.Root, "proxytray.log") }
```

---

### Task 6: proxyos interface + Windows WinINET

**Files:**
- Create: `internal/proxyos/proxyos.go`
- Create: `internal/proxyos/windows.go` (`//go:build windows`)
- Create: `internal/proxyos/stub.go` (`//go:build !windows`)
- Create: `internal/proxyos/windows_test.go` (`//go:build windows`) — optional skip in CI without admin; prefer unit-testing backup struct encode/decode with registry mock if hard — at minimum test `formatProxyServer` helper.

**Interfaces:**
- Consumes: none from core except address string
- Produces:
```go
type Manager interface {
	Apply(addr string) error    // host:port
	Restore() error
	IsApplied() bool
	Active() string             // current applied addr or ""
}
```

Windows implementation details:
- Registry key: `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
- Values: `ProxyEnable` (DWORD), `ProxyServer` (string `host:port`), optionally `ProxyOverride`
- Before first Apply: read and store original `ProxyEnable`, `ProxyServer`, `ProxyOverride` in memory (and optionally a sidecar file under AppData `proxy_backup.json` so crash mid-session can recover — **include sidecar** for safety)
- After write: call `InternetSetOptionW` with `INTERNET_OPTION_SETTINGS_CHANGED` (39) and `INTERNET_OPTION_REFRESH` (37)
- Restore: write backup values back + InternetSetOption; if no backup, set ProxyEnable=0

- [ ] **Step 1: Write helper test (portable)**

Put `formatProxyServer` / backup JSON round-trip in `backup.go` (all platforms):

```go
type Snapshot struct {
	ProxyEnable   uint32 `json:"proxy_enable"`
	ProxyServer   string `json:"proxy_server"`
	ProxyOverride string `json:"proxy_override"`
	Valid         bool   `json:"valid"`
}
```

Test marshal round-trip.

- [ ] **Step 2: Implement interface + windows apply/restore**

Use `golang.org/x/sys/windows` and `registry` package.

```bash
go get golang.org/x/sys@latest
```

`windows.go` outline:

```go
//go:build windows

package proxyos

type WinManager struct {
	backupPath string
	snap       Snapshot
	active     string
}

func New(backupPath string) *WinManager { ... }

func (m *WinManager) Apply(addr string) error {
	if !m.snap.Valid {
		if err := m.capture(); err != nil {
			return err
		}
	}
	if err := m.write(1, addr, m.snap.ProxyOverride); err != nil {
		return err
	}
	if err := notify(); err != nil {
		return err
	}
	m.active = addr
	return nil
}

func (m *WinManager) Restore() error {
	if m.snap.Valid {
		_ = m.write(m.snap.ProxyEnable, m.snap.ProxyServer, m.snap.ProxyOverride)
	} else {
		_ = m.write(0, "", "")
	}
	_ = notify()
	m.active = ""
	return nil
}
```

- [ ] **Step 3: Manual verification notes in commit message** (automated registry test may be skipped):

```bash
go test ./internal/proxyos/ -count=1
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat: Windows WinINET proxy apply and restore"
```

---

### Task 7: Start with Windows

**Files:**
- Create: `internal/proxyos/startup_windows.go` (`//go:build windows`)
- Create: `internal/proxyos/startup_stub.go` (`//go:build !windows`)
- Create: `internal/proxyos/startup_test.go` — test path quoting helper only

**Interfaces:**
```go
func EnableStartWithWindows(exePath string) error  // HKCU\...\Run value "ProxyTray"
func DisableStartWithWindows() error
func IsStartWithWindowsEnabled() (bool, error)
```

Registry: `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, value name `ProxyTray`, data = quoted exe path.

- [ ] **Step 1–4: Implement, test helpers, commit**

```bash
git commit -m "feat: optional Start with Windows via HKCU Run key"
```

---

### Task 8: Tray UI + main wiring

**Files:**
- Create: `internal/tray/icon.go` (embed simple generated ICO/PNG bytes — 16x16; can use a minimal embedded PNG)
- Create: `internal/tray/app.go`
- Create: `cmd/proxytray/main.go`
- Create: `README.md`
- Modify: `go.mod` (add energye/systray)

**Interfaces:**
- Consumes: `core.Scanner`, `core.MenuProxies`, `config.Store`, `proxyos.Manager`, startup helpers
- Produces: `tray.Run(deps Deps)` blocking

Menu labels exactly:
- `Off`
- proxy lines: `fmt.Sprintf("%s %s  %s  (%.1fs)", core.FlagEmoji(p.Code), p.Code, p.Address(), p.LatencyS)`
- `Refresh now` / `Scanning…`
- Filters submenu: `HTTPS`, `Elite`, `Anonymous`, `Transparent`
- `Start with Windows`
- `Quit`

Behavior wiring:
- On start: load config + cache; show menu; `go scanAndRefresh()`; start `time.Ticker` 30m
- Refresh click: if `scanner.IsScanning()` return; else `go scanAndRefresh()`
- Filter toggle: update config, Save, rebuild menu from cache (no scan)
- Proxy click: `manager.Apply(addr)`; set active; rebuild
- Off: `manager.Restore()`; clear active; rebuild
- Quit: `manager.Restore()`; `systray.Quit()`
- Start with Windows: enable/disable + save config
- Active proxy not in top-10: prepend/keep a checked entry for active address (find full `Proxy` from cache by address; if missing, show address-only row)

Use `systray.ResetMenu()` then rebuild (energye/systray).

```bash
go get github.com/energye/systray@latest
```

Minimal `main.go`:

```go
package main

import (
	"log"
	"os"

	"proxy-tray/internal/config"
	"proxy-tray/internal/core"
	"proxy-tray/internal/proxyos"
	"proxy-tray/internal/tray"
)

func main() {
	store := config.DefaultStore()
	cfg, err := store.Load()
	if err != nil {
		log.Fatal(err)
	}
	cache, _ := store.LoadCache()
	mgr := proxyos.New(filepath.Join(store.Root, "proxy_backup.json"))
	scanner := core.NewScanner(nil, nil)
	tray.Run(tray.Deps{
		Store:   store,
		Config:  cfg,
		Cache:   cache,
		Scanner: scanner,
		Proxy:   mgr,
	})
}
```

- [ ] **Step 1: Implement tray + main**
- [ ] **Step 2: Build**

```bash
go build -o proxytray.exe ./cmd/proxytray
```

Expected: binary builds on Windows.

- [ ] **Step 3: Smoke (manual)**
  - Launch exe → tray icon
  - Wait/Refresh → proxies appear with flags
  - Toggle filters → list changes
  - Apply → system proxy set (check Windows Proxy settings)
  - Off → restored
  - Quit → restored

- [ ] **Step 4: Commit**

```bash
git add cmd/ internal/tray/ README.md go.mod go.sum
git commit -m "feat: systray UI and application entrypoint"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Scrape free-proxy-list.net | 2 |
| Test first 100, concurrency 10, ~10s | 4 |
| Top 10 + filters | 3, 8 |
| Scan lock / manual vs auto | 4, 8 |
| 30 min refresh | 8 |
| Flag + country code in menu | 1, 8 |
| Off restores | 6, 8 |
| Quit restores | 8 |
| Keep active if not in list | 8 |
| Start with Windows optional default off | 7, 8 |
| AppData config/cache/log | 5 |
| Windows-only, layered pkgs | 6–8 |
| Not a Windows Service | 8 |

## Self-review notes

- Filter HTTPS semantics locked: checked ⇒ HTTPS only; unchecked ⇒ no HTTPS restriction.
- Probe unit test uses HTTP httptest upstream to avoid CONNECT complexity.
- `FetchProxyList` already limits to 100 via `ParseProxies(..., 100)`; scanner also caps `DefaultScanLimit`.
