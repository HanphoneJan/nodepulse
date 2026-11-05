# Build the Flutter desktop app and the Go sidecar for Windows.
# Run from the repository root in PowerShell:
#   $env:VERSION = "0.3.7"; .\packaging\build-windows.ps1
#
# Requires Flutter, Go and NSIS (makensis) on PATH.
$ErrorActionPreference = "Stop"

$Version = if ($env:VERSION) { $env:VERSION } else { "0.3.7" }
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$Dist = Join-Path $Root "dist"
$Stage = Join-Path $Dist "NodePulse"
$Bundle = Join-Path $Root "desktop\build\windows\x64\runner\Release"
$Installer = Join-Path $Dist "NodePulse-windows-amd64-${Version}.exe"
$Zip = Join-Path $Dist "NodePulse-windows-amd64-${Version}.zip"

Remove-Item -Recurse -Force $Stage -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $Dist | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $Root "bin") | Out-Null

Push-Location (Join-Path $Root "desktop")
try {
  flutter build windows --release --build-name=$Version
  if ($LASTEXITCODE -ne 0) { throw "flutter build failed" }
} finally {
  Pop-Location
}

Push-Location $Root
try {
  go build -trimpath -ldflags "-s -w -H windowsgui -X github.com/HanphoneJan/nodepulse/internal/version.Version=$Version" `
    -o (Join-Path $Root "bin\nodepulse.exe") .\cmd\nodepulse
  if ($LASTEXITCODE -ne 0) { throw "go build failed" }
} finally {
  Pop-Location
}

New-Item -ItemType Directory -Force -Path $Stage | Out-Null
Copy-Item (Join-Path $Bundle "*") $Stage -Recurse -Force
Copy-Item (Join-Path $Root "bin\nodepulse.exe") (Join-Path $Stage "nodepulse-core.exe") -Force
Copy-Item (Join-Path $Root "packaging\nodepulse.ico") $Stage -Force
Copy-Item (Join-Path $Root "LICENSE") $Stage -Force
Copy-Item (Join-Path $Root "THIRD_PARTY_NOTICES.md") $Stage -Force

if (Test-Path $Zip) { Remove-Item $Zip -Force }
Compress-Archive -Path (Join-Path $Stage "*") -DestinationPath $Zip
Write-Host "wrote $Zip"

$Makensis = Get-Command makensis -ErrorAction SilentlyContinue
if ($Makensis) {
  Push-Location (Join-Path $Root "packaging")
  try {
    & $Makensis.Source "-INPUTCHARSET" "UTF8" "/DVERSION=$Version" "/DSOURCE=$Stage" "/DOUT=$Installer" "nodepulse.nsi"
    if ($LASTEXITCODE -ne 0) { throw "makensis failed" }
  } finally {
    Pop-Location
  }
  Write-Host "wrote $Installer"
} else {
  Write-Warning "NSIS not found; portable zip produced without an installer."
}
