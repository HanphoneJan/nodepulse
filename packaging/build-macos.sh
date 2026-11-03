#!/usr/bin/env bash
# Build a .app bundle for macOS. Run on macOS (cgo/WebKit is required).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-0.3.0}"
APP="$ROOT/dist/NodePulse.app"
OUT="$ROOT/dist/NodePulse-macos-universal-${VERSION}.zip"

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
(
  cd "$ROOT/gui"
  for arch in amd64 arm64; do
    CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" go build -trimpath -tags "production,desktop" \
      -ldflags "-s -w -X github.com/HanphoneJan/nodepulse/internal/version.Version=$VERSION" \
      -o "$TMP/nodepulse-$arch" .
  done
)
lipo -create "$TMP/nodepulse-amd64" "$TMP/nodepulse-arm64" -output "$APP/Contents/MacOS/nodepulse"
cp "$ROOT/LICENSE" "$APP/Contents/Resources/LICENSE"
cp "$ROOT/THIRD_PARTY_NOTICES.md" "$APP/Contents/Resources/THIRD_PARTY_NOTICES.md"

ICONSET="$TMP/NodePulse.iconset"
mkdir -p "$ICONSET"
for size in 16 32 128 256 512; do
  sips -z "$size" "$size" "$ROOT/packaging/nodepulse-1024.png" --out "$ICONSET/icon_${size}x${size}.png" >/dev/null
  double=$((size * 2))
  sips -z "$double" "$double" "$ROOT/packaging/nodepulse-1024.png" --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/nodepulse.icns"

cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>NodePulse</string>
  <key>CFBundleDisplayName</key><string>NodePulse</string>
  <key>CFBundleIdentifier</key><string>com.nodepulse.app</string>
  <key>CFBundleExecutable</key><string>nodepulse</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleIconFile</key><string>nodepulse.icns</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST

# Ad-hoc signing keeps Gatekeeper from treating the bundle as damaged on Apple
# Silicon. A Developer ID signature and notarization are still required for
# distribution outside the local machine.
codesign --force --deep --sign - "$APP" >/dev/null 2>&1 || true

ditto -c -k --keepParent "$APP" "$OUT"
echo "wrote $OUT"
