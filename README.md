# Tray Proxy

Windows system-tray app that fetches free proxies from [free-proxy-list.net](https://free-proxy-list.net/), tests them, and lets you apply/clear an OS-wide proxy.

## Requirements

- Windows 10/11
- Go 1.22+ (to build from source)
- [Inno Setup 6](https://jrsoftware.org/isinfo.php) (to build the installer)

## Install (end users)

1. Download `TrayProxy-Setup-x.y.z.exe` from this repo’s **GitHub Releases** (personal account).
2. Run the setup and choose **current user** or **all users**.
3. Finish with **Launch Tray Proxy ** checked (default) to start the tray app.

Uninstall removes the program files and Start Menu shortcut.  
`%AppData%\proxytray` (config, cache, logs) is **kept** on purpose.

## Build from source

```powershell
$env:Path = "C:\Program Files\Go\bin;" + $env:Path
cd proxy-tray
go mod tidy
go build -o proxytray.exe ./cmd/proxytray
.\proxytray.exe
```

## Build installer (local)

```powershell
# Requires Go + Inno Setup 6 (ISCC.exe)
.\scripts\build-installer.ps1 -Version 0.1.0
# Output: dist\TrayProxy-Setup-0.1.0.exe
```

Optional code signing (skipped if unset):

```powershell
$env:SIGN_CERT = "C:\path\to\cert.pfx"
$env:SIGN_CERT_PASSWORD = "..."
# Or base64 PFX:
# $env:SIGN_CERT_BASE64 = "..."
.\scripts\build-installer.ps1 -Version 0.1.0
```

## Release (personal GitHub)

Push a version tag; Actions builds and attaches the setup exe to the Release:

```powershell
git tag v0.1.0
git push origin v0.1.0
```

Optional repo secrets: `SIGN_CERT_BASE64`, `SIGN_CERT_PASSWORD`.

## Tray menu

- **Off** — restore original system proxy
- Proxy list — click to apply (flag icon + country code + latency)
- **Refresh now** — manual scan (skipped if a scan is already running)
- **Filters** — HTTPS / Elite / Anonymous / Transparent
- **Start with Windows** — optional (off by default; not set by the installer)
- **Quit** — restores proxy settings, then exits

Automatic refresh runs every 30 minutes.  
Config/cache/logs: `%AppData%\trayproxy\`

## Tests

```powershell
go test ./...
```

## Note

Free open proxies are untrusted. Prefer for non-sensitive browsing only.
