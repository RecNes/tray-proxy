# Proxy Tray Windows Installer — Design Spec

**Date:** 2026-10-02  
**Status:** Approved for planning  
**Project:** `proxy-tray` (personal GitHub account — not Botano)

## Goal

Ship Proxy Tray as an installable Windows application via an Inno Setup installer, built reproducibly by a PowerShell script and published through GitHub Actions Releases on the maintainer’s **personal** GitHub repository.

## Non-goals (v1)

- Botano organization CI or Botano-owned remotes
- Microsoft Store packaging
- Auto-update channel
- Mandatory code signing (optional only)
- Desktop shortcut by default

## Decisions summary

| Topic | Choice |
|-------|--------|
| Installer tech | Inno Setup → `.exe` setup |
| Scope | Wizard chooses per-user **or** per-machine |
| Shortcuts | Start Menu only (no desktop) |
| Post-install | “Launch Proxy Tray” checkbox on finish page, **checked by default** |
| Signing | Optional placeholder (`signtool` when cert secrets/env present; skip otherwise) |
| Distribution | GitHub Actions release on personal repo (approach 3) |
| Local build | Same `scripts/build-installer.ps1` CI uses |
| User data on uninstall | Keep `%AppData%\proxytray` |
| Start with Windows | App menu only; installer does not enable it |

## Install behavior

### Paths

- **Per-user:** `%LocalAppData%\Programs\ProxyTray\`
- **Per-machine:** `%ProgramFiles%\ProxyTray\` (requires elevation)

### Wizard

1. Privileges / install mode: current user vs all users (Inno `PrivilegesRequiredOverridesAllowed` / equivalent)
2. Install directory (default as above)
3. Finish page: checkbox **Launch Proxy Tray** (default checked)
4. Creates Start Menu shortcut to `proxytray.exe`
5. Registers uninstaller in Windows Apps & Features

### Uninstall

- Removes install directory and Start Menu shortcut
- Does **not** delete `%AppData%\proxytray` (config, cache, logs, proxy backup)

### Signing

- If `SIGN_CERT` / CI secrets (`SIGN_CERT_BASE64`, `SIGN_CERT_PASSWORD`) available → sign setup (and optionally exe) with `signtool`
- If absent → produce unsigned installer and continue (SmartScreen may warn)

## Repository layout (additions)

```
installer/proxytray.iss          # Inno script (AppId GUID stable across releases)
scripts/build-installer.ps1      # test → build exe → compile setup → optional sign
.github/workflows/release.yml    # tag-triggered release on personal GitHub
```

Existing app binary entrypoint remains `cmd/proxytray` → `proxytray.exe`.

## Versioning & identity

- Display name: **Proxy Tray**
- Executable: `proxytray.exe`
- Output artifact: `ProxyTray-Setup-<version>.exe` (e.g. `ProxyTray-Setup-0.1.0.exe`)
- Version source: Git tag `vMAJOR.MINOR.PATCH` → version `MAJOR.MINOR.PATCH`
- Inno `AppId`: fixed GUID generated once and never changed (enables upgrades)
- Prefer embedding matching version info into `proxytray.exe` when practical in the build script

## CI / release flow (personal GitHub)

**Remote:** `github.com/<personal-username>/proxy-tray` (not Botano).

**Trigger:** push of tag matching `v*` (example: `git tag v0.1.0 && git push origin v0.1.0`).

**Job (`windows-latest`):**

1. Checkout tagged commit
2. Setup Go
3. Install Inno Setup (chocolatey or official silent installer)
4. Run `scripts/build-installer.ps1 -Version <from tag>`
5. Upload `ProxyTray-Setup-<version>.exe` to the GitHub Release for that tag

**Optional secrets** on the personal repo:

- `SIGN_CERT_BASE64`
- `SIGN_CERT_PASSWORD`

Absence of secrets must not fail the release.

## Local developer flow

```powershell
./scripts/build-installer.ps1 -Version 0.1.0
```

Requires: Go, Inno Setup Compiler (`ISCC.exe` on PATH or standard install path).

## README updates

Document:

1. Personal-repo tag → Release download
2. Local installer build prerequisites and script
3. Optional signing env vars
4. Uninstall leaves AppData intact

## Testing

- Build script produces setup exe locally
- Install per-user without admin; Start Menu entry works; Launch checkbox starts tray app
- Install per-machine with elevation
- Upgrade same `AppId` replaces previous version
- Uninstall removes app files; `%AppData%\proxytray` remains
- CI dry-run / tag release uploads artifact (unsigned OK)

## Open implementation notes (resolved)

- Inno must be installed in CI each run (or cached); script documents expected `ISCC` path.
- Personal GitHub username is chosen when creating/pushing the remote; workflow paths are repo-relative and account-agnostic.
