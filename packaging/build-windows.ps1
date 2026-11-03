# Build the Wails desktop app for Windows.
# Run from the repository root in PowerShell:
#   $env:VERSION = "0.3.0"; .\packaging\build-windows.ps1
$ErrorActionPreference = "Stop"

$Version = if ($env:VERSION) { $env:VERSION } else { "0.3.0" }
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$Dist = Join-Path $Root "dist"
New-Item -ItemType Directory -Force -Path $Dist | Out-Null

Push-Location (Join-Path $Root "gui")
try {
  go build -trimpath -tags "production,desktop" `
    -ldflags "-s -w -H windowsgui -X github.com/HanphoneJan/nodepulse/internal/version.Version=$Version" `
    -o (Join-Path $Dist "nodepulse.exe") .
  if ($LASTEXITCODE -ne 0) { throw "go build failed" }
} finally {
  Pop-Location
}

Copy-Item (Join-Path $Root "LICENSE") $Dist -Force
Copy-Item (Join-Path $Root "THIRD_PARTY_NOTICES.md") $Dist -Force
$Zip = Join-Path $Dist "nodepulse_${Version}_windows_amd64.zip"
if (Test-Path $Zip) { Remove-Item $Zip -Force }
Compress-Archive -Path (Join-Path $Dist "nodepulse.exe"), (Join-Path $Dist "LICENSE"), (Join-Path $Dist "THIRD_PARTY_NOTICES.md") -DestinationPath $Zip
Write-Host "wrote $Zip"
