# sspt one-line installer (PowerShell, Windows).
#
# Usage:
#   irm https://raw.githubusercontent.com/NerdyGriffin/steam-switch-pro-templates/main/scripts/install.ps1 | iex
#
# What it does:
#   1. Fetches the latest release from GitHub
#   2. Downloads sspt-windows-amd64.exe to a temp dir
#   3. Verifies the SHA256 against the published SHA256SUMS file
#   4. Runs `sspt install` (which copies to %LOCALAPPDATA%\Programs\sspt and
#      registers the HKCU Run key trigger)
#
# All actions are user-scope; no admin / UAC required.

$ErrorActionPreference = 'Stop'
$ProgressPreference    = 'SilentlyContinue'

$Repo  = 'NerdyGriffin/steam-switch-pro-templates'
$Asset = 'sspt-windows-amd64.exe'

Write-Host "==> Fetching latest release info from $Repo ..."
$release = Invoke-RestMethod -UseBasicParsing -Uri "https://api.github.com/repos/$Repo/releases/latest"
$tag     = $release.tag_name
Write-Host "    latest tag: $tag"

$assetUrl    = ($release.assets | Where-Object { $_.name -eq $Asset }     | Select-Object -First 1).browser_download_url
$checksumUrl = ($release.assets | Where-Object { $_.name -eq 'SHA256SUMS' } | Select-Object -First 1).browser_download_url
if (-not $assetUrl)    { throw "release $tag has no asset named $Asset" }
if (-not $checksumUrl) { throw "release $tag has no SHA256SUMS asset" }

$tmp     = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "sspt-install-$([Guid]::NewGuid().Guid.Substring(0,8))") -Force
$binPath = Join-Path $tmp.FullName $Asset
$sumPath = Join-Path $tmp.FullName 'SHA256SUMS'

Write-Host "==> Downloading $Asset ..."
Invoke-WebRequest -UseBasicParsing -Uri $assetUrl    -OutFile $binPath
Invoke-WebRequest -UseBasicParsing -Uri $checksumUrl -OutFile $sumPath

Write-Host "==> Verifying SHA256 ..."
$expectedLine = (Get-Content $sumPath) | Where-Object { $_ -match [regex]::Escape($Asset) } | Select-Object -First 1
if (-not $expectedLine) { throw "SHA256SUMS does not list $Asset" }
$expected = ($expectedLine -split '\s+')[0].ToLower()
$actual   = (Get-FileHash -Algorithm SHA256 -Path $binPath).Hash.ToLower()
if ($expected -ne $actual) {
    throw "checksum mismatch for $Asset`n  expected: $expected`n  actual:   $actual"
}
Write-Host "    checksum OK ($actual)"

Write-Host "==> Running '$Asset install' ..."
& $binPath install
$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) { throw "sspt install exited with code $exitCode" }

Write-Host ""
Write-Host "==> Done. Run 'sspt status' to inspect state." -ForegroundColor Green
Write-Host "    (binary copied into %LOCALAPPDATA%\Programs\sspt; PATH update may require a new shell)"
