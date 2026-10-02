# Proxy Tray — Design Spec

**Date:** 2026-10-02  
**Status:** Approved for planning  
**Project:** `proxy-tray` (new Go project, Windows-first)

## Goal

A lightweight Windows system-tray application that periodically fetches free proxies from [free-proxy-list.net](https://free-proxy-list.net/), validates them (same idea as [free-proxy-hunter/fpc.py](https://github.com/RecNes/free-proxy-hunter/blob/main/fpc.py)), lists the best working ones in the tray menu (with country flag + country code), and lets the user apply or clear an OS-wide proxy.

This is **not** a Windows Service. It is a single tray process that can optionally start with Windows.

## Non-goals (v1)

- macOS / Linux support (architecture must allow adding later)
- Real Windows Service / Session 0 worker
- Separate Settings GUI window
- Paid / official proxy APIs
- Guaranteed anonymity or security (free open proxies are untrusted by nature)

## Decisions summary

| Topic | Choice |
|-------|--------|
| Architecture | Single-process tray app (option C), layered packages |
| Language | Go |
| Platform (v1) | Windows only |
| Refresh | Every 30 minutes + manual Refresh |
| Concurrent scans | Only one at a time; manual blocked while scanning |
| Menu list size | Top 10 fastest that pass filters |
| Candidate pool | Test first 100 table rows (all types); filter successes in menu |
| Filters UI | Systray **Filters** submenu (checkable) |
| Start with Windows | Optional, **off by default**, toggle in menu |
| Quit | Always restore original system proxy settings |
| Active proxy missing from new scan | Keep applied; do not auto-Off |

## Architecture

Single Go binary with clear package boundaries:

```
cmd/proxytray/          # entrypoint
internal/core/          # scrape, test, sort, filter, scan lock
internal/proxyos/       # Windows system proxy apply/restore + backup
internal/tray/          # systray icon, menu, user actions
internal/config/        # persisted filters, startup preference, cache paths
```

- Network/scan work runs in goroutines.
- Menu rebuilds happen under a mutex after scan or filter changes.
- OS-specific code stays in `proxyos` (and startup helpers) so Mac/Linux backends can be added later without rewriting core.

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│ tray (UI)   │────▶│ core (scan)  │────▶│ free-proxy-list │
│ menu/actions│◀────│ filter/sort  │     │ + probe target  │
└──────┬──────┘     └──────────────┘     └─────────────────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐
│ proxyos     │────▶│ WinINET /    │
│ apply/Off   │     │ system proxy │
└─────────────┘     └──────────────┘
```

## Components

### core

1. **Scrape** `https://free-proxy-list.net/` HTML table fields: IP, Port, Code, Country, Anonymity, Https (and related columns as available).
2. **Test window:** take at most the **first 100** table rows per scan (same window size as fpc.py). Within that window, test **all** rows (every anonymity/HTTPS combination) so Filters can work on a real result set without re-scraping.
3. **Test** those candidates concurrently against a simple IP echo endpoint (primary: `https://httpbin.org/ip`), timeout ~10s, concurrency cap ~10.
4. **Record** success latency; sort ascending by time; keep the full successful set in cache (not only top 10).
5. **Scan lock:** mutex/flag so automatic and manual scans never overlap. If a scan is running, Refresh is a no-op (menu shows Scanning… / disabled).
6. **Cache** last successful result set in memory and on disk for fast menu after restart.

Filter flags affect **which of the validated results appear in the menu**, not which rows were tested in the last scan. Changing filters re-ranks/re-slices the cached successes into the top 10 without requiring a new network scan.

### proxyos (Windows)

1. Before first apply in a session (or before any apply if no backup exists), snapshot current user proxy settings.
2. Apply selected `host:port` as OS-wide HTTP and HTTPS proxy via WinINET registry settings and broadcast with `InternetSetOption` (so running apps pick up the change where supported).
3. **Off** restores the snapshot (or disables proxy if there was no prior proxy).
4. **Quit** always runs the same restore path as Off, then exits.

### tray

Menu order (top → bottom):

1. **Off** — fixed; checked when no app-managed proxy is applied
2. Separator
3. Up to **10** proxy items: `🇹🇷 TR  1.2.3.4:8080  (0.8s)`  
   - Country flag via regional-indicator emoji from ISO country code  
   - Country code, address, latency  
   - Checked item = currently applied
4. If an applied proxy is no longer in the filtered top 10, still show it (e.g. keep in list or as an “Active” row) so the user can see what is applied; do not auto-clear
5. Separator
6. **Refresh now** — starts scan if idle; disabled/labeled while scanning
7. **Filters ▸** — checkable: HTTPS, Elite, Anonymous, Transparent (persist to config)
8. **Start with Windows** — checkable; default unchecked; toggles current-user Startup entry
9. **Quit** — restore proxy, exit

Tooltip can show status: idle / scanning / active proxy / last error summary.

### config

Path: `%AppData%\proxytray\`

- `config.json` — filter toggles, start-with-Windows preference  
- Optional cache file for last proxy list  
- `proxytray.log` — operational log

## Data flows

### Startup

1. Load config  
2. Show tray  
3. Show cached proxies if present  
4. Start background scan  
5. Do not change system proxy on launch (remain Off unless we later add “remember applied” — **out of scope**; v1 always starts logically Off regarding *new* apply, and Quit always cleared proxy on previous exit)

### Periodic scan (every 30 minutes)

If not scanning: run full scrape → test → update cache → rebuild menu.

### Manual Refresh

If scanning: ignore. Else: same as periodic scan.

### Apply proxy

User clicks item → `proxyos.Apply` → update checked state. On failure: keep previous state; log + tooltip error.

### Off

`proxyos.Restore` → check Off; clear active selection.

### Quit

Restore → exit.

## Error handling

| Failure | Behavior |
|---------|----------|
| Site unreachable / HTML changed | Keep previous cache; show refresh-failed status |
| Individual proxy test fails | Skip silently |
| Apply fails | Do not change checked state; log error |
| Restore fails | Log error; still attempt best-effort disable |

## Testing

**Unit (core):**

- Parse fixture HTML  
- Filter combinations  
- Sort + top-10  
- Scan lock rejects overlapping start  

**Windows manual / integration:**

- Apply changes system proxy  
- Off restores original  
- Quit leaves proxy restored  
- Start with Windows toggle creates/removes Startup entry  

**Smoke:**

- First launch, Refresh, change filters, apply, Off, Quit  

## Future extension hooks

- `internal/proxyos` interface with Windows implementation only in v1  
- Same tray/core on other OSes later  
- Optional stricter defaults (e.g. elite+HTTPS only) via filter presets  

## Open implementation notes (resolved in planning, not ambiguous)

- Probe URL may be `httpbin.org/ip` or a small fallback set if httpbin is rate-limited; primary behavior matches fpc.py-style latency check.  
- Systray library choice (e.g. `getlantern/systray` or maintained fork) is an implementation detail; menu must support checkable items and dynamic rebuild.  
- Project module path: `proxy-tray` under `C:\Users\Botano User\Projects\proxy-tray`.
