# Proxy Tray

Windows system-tray app that fetches free proxies from [free-proxy-list.net](https://free-proxy-list.net/), tests them, and lets you apply/clear an OS-wide proxy.

## Requirements

- Windows 10/11
- Go 1.22+ (to build)

## Build

```powershell
$env:Path = "C:\Program Files\Go\bin;" + $env:Path
cd proxy-tray
go mod tidy
go build -o proxytray.exe ./cmd/proxytray
```

## Run

```powershell
.\proxytray.exe
```

Look for the tray icon. Right-click (or left-click, depending on shell) to open the menu:

- **Off** — restore original system proxy
- Proxy list — click to apply (flag + country code + latency)
- **Refresh now** — manual scan (skipped if a scan is already running)
- **Filters** — HTTPS / Elite / Anonymous / Transparent
- **Start with Windows** — optional (off by default)
- **Quit** — restores proxy settings, then exits

Automatic refresh runs every 30 minutes.

Config/cache/logs: `%AppData%\proxytray\`

## Tests

```powershell
go test ./...
```

## Note

Free open proxies are untrusted. Prefer for non-sensitive browsing only.
