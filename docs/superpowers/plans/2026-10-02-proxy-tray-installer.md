# Proxy Tray Installer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Inno Setup packaging, a reproducible PowerShell build script, and a personal-GitHub tag-triggered release workflow.

**Architecture:** `installer/proxytray.iss` defines install UX; `scripts/build-installer.ps1` builds the Go binary and compiles the setup; `.github/workflows/release.yml` runs the same script on `v*` tags.

**Tech Stack:** Go, Inno Setup 6, PowerShell, GitHub Actions (`windows-latest`).

## Global Constraints

- Personal GitHub only (not Botano)
- Inno Setup `.exe` installer; wizard chooses per-user or per-machine
- Start Menu only; Launch checkbox default checked
- Optional signing; missing cert must not fail build
- Uninstall keeps `%AppData%\proxytray`
- Spec: `docs/superpowers/specs/2026-10-02-proxy-tray-installer-design.md`

---

### Task 1: Inno script + build script + workflow + README

Create `installer/proxytray.iss`, `scripts/build-installer.ps1`, `.github/workflows/release.yml`, update README/gitignore, commit.
