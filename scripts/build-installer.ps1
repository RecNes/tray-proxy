<#
.SYNOPSIS
  Build Tray Proxy Windows installer (Go binary + Inno Setup).

.PARAMETER Version
  Semver without leading v (e.g. 0.1.0). Used for artifact name and AppVersion.

.EXAMPLE
  .\scripts\build-installer.ps1 -Version 0.1.0
#>
[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [ValidatePattern('^\d+\.\d+\.\d+')]
  [string]$Version,

  [string]$Configuration = "release"
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$Dist = Join-Path $Root "dist"
$Iss = Join-Path $Root "installer\trayproxy.iss"
$ExeOut = Join-Path $Dist "trayproxy.exe"
$SetupOut = Join-Path $Dist "TrayProxy-Setup-$Version.exe"

function Find-ISCC {
  $paths = New-Object System.Collections.Generic.List[string]
  $cmd = Get-Command ISCC.exe -ErrorAction SilentlyContinue
  if ($cmd -and $cmd.Source) { [void]$paths.Add($cmd.Source) }

  $pf86 = [Environment]::GetFolderPath("ProgramFilesX86")
  $pf = [Environment]::GetFolderPath("ProgramFiles")
  foreach ($root in @($pf86, $pf, $env:LOCALAPPDATA)) {
    if ([string]::IsNullOrWhiteSpace($root)) { continue }
    [void]$paths.Add((Join-Path $root "Inno Setup 6\ISCC.exe"))
    [void]$paths.Add((Join-Path $root "Programs\Inno Setup 6\ISCC.exe"))
  }

  foreach ($p in $paths) {
    if ($p -and (Test-Path -LiteralPath $p)) { return $p }
  }
  throw "ISCC.exe not found. Install Inno Setup 6: https://jrsoftware.org/isinfo.php"
}

function Invoke-OptionalSign([string]$Path) {
  $certPath = $env:SIGN_CERT
  $certPass = $env:SIGN_CERT_PASSWORD
  if (-not $certPath) {
    if ($env:SIGN_CERT_BASE64) {
      $certPath = Join-Path $Dist "codesign.pfx"
      [IO.File]::WriteAllBytes($certPath, [Convert]::FromBase64String($env:SIGN_CERT_BASE64))
    }
  }
  if (-not $certPath -or -not (Test-Path $certPath)) {
    Write-Host "Signing skipped (no SIGN_CERT / SIGN_CERT_BASE64)."
    return
  }
  $signtool = Get-Command signtool.exe -ErrorAction SilentlyContinue
  if (-not $signtool) {
    $kits = Get-ChildItem "C:\Program Files (x86)\Windows Kits\10\bin" -Recurse -Filter signtool.exe -ErrorAction SilentlyContinue |
      Sort-Object FullName -Descending | Select-Object -First 1
    if ($kits) { $signtool = $kits.FullName }
  }
  if (-not $signtool) {
    Write-Warning "signtool.exe not found; skipping sign for $Path"
    return
  }
  $args = @(
    "sign", "/fd", "SHA256", "/tr", "http://timestamp.digicert.com", "/td", "SHA256",
    "/f", $certPath
  )
  if ($certPass) { $args += @("/p", $certPass) }
  $args += $Path
  & $signtool @args
  if ($LASTEXITCODE -ne 0) { throw "signtool failed for $Path" }
  Write-Host "Signed $Path"
}

Push-Location $Root
try {
  New-Item -ItemType Directory -Force -Path $Dist | Out-Null

  Write-Host "==> go test"
  & go test ./...
  if ($LASTEXITCODE -ne 0) { throw "go test failed" }

  Write-Host "==> go build -> $ExeOut"
  $ldflags = "-X main.version=$Version"
  & go build -ldflags $ldflags -o $ExeOut ./cmd/trayproxy
  if ($LASTEXITCODE -ne 0) { throw "go build failed" }

  Invoke-OptionalSign $ExeOut

  $iscc = Find-ISCC
  Write-Host "==> Inno Setup ($iscc) version $Version"
  & $iscc "/DMyAppVersion=$Version" $Iss
  if ($LASTEXITCODE -ne 0) { throw "ISCC failed" }

  if (-not (Test-Path $SetupOut)) {
    throw "Expected setup missing: $SetupOut"
  }

  Invoke-OptionalSign $SetupOut

  Write-Host "==> Done: $SetupOut"
  Get-Item $SetupOut | Format-List FullName, Length, LastWriteTime
}
finally {
  Pop-Location
}
