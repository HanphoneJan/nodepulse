#!/usr/bin/env bash
# Build the Flutter macOS app, embed the Go sidecar, and create zip + dmg.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-0.3.4}"
APP="$ROOT/desktop/build/macos/Build/Products/Release/NodePulse.app"
OUT="$ROOT/dist/NodePulse-macos-universal-${VERSION}.zip"
DMG_OUT="$ROOT/dist/NodePulse-macos-universal-${VERSION}.dmg"

cd "$ROOT/desktop"
flutter build macos --release --build-name="$VERSION"

if [[ ! -d "$APP" ]]; then
  APP="$(find "$ROOT/desktop/build/macos/Build/Products/Release" -maxdepth 1 -name '*.app' -print -quit 2>/dev/null || true)"
fi
if [[ ! -d "$APP" ]]; then
  echo "macOS app bundle not found under $ROOT/desktop/build/macos/Build/Products/Release" >&2
  exit 1
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
(
  cd "$ROOT"
  for arch in amd64 arm64; do
    CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" go build -trimpath \
      -ldflags "-s -w -X github.com/HanphoneJan/nodepulse/internal/version.Version=$VERSION" \
      -o "$TMP/nodepulse-$arch" ./cmd/nodepulse
  done
)
lipo -create "$TMP/nodepulse-amd64" "$TMP/nodepulse-arm64" -output "$APP/Contents/MacOS/nodepulse-core"
chmod +x "$APP/Contents/MacOS/nodepulse-core"
cp "$ROOT/LICENSE" "$APP/Contents/Resources/LICENSE"
cp "$ROOT/THIRD_PARTY_NOTICES.md" "$APP/Contents/Resources/THIRD_PARTY_NOTICES.md"
/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $VERSION" "$APP/Contents/Info.plist" || true
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $VERSION" "$APP/Contents/Info.plist" || true

codesign --force --deep --sign - "$APP" >/dev/null 2>&1 || true

ditto -c -k --keepParent "$APP" "$OUT"
echo "wrote $OUT"

STAGE="$TMP/dmg"
mkdir -p "$STAGE"
cp -R "$APP" "$STAGE/"
ln -s /Applications "$STAGE/Applications"
rm -f "$DMG_OUT"
hdiutil create -volname "NodePulse" -srcfolder "$STAGE" -ov -format UDZO "$DMG_OUT" >/dev/null
echo "wrote $DMG_OUT"
