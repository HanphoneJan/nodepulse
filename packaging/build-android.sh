#!/usr/bin/env bash
# Build the Flutter Android app plus the Go sidecar for arm64.
# Run from the repository root:
#   VERSION=0.3.3 bash packaging/build-android.sh
#
# Requires Flutter, Go and the Android SDK. The APK is signed with the debug
# key; configure a release keystore before publishing to a store.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-0.3.3}"
FLUTTER="${FLUTTER:-flutter}"
DIST="$ROOT/dist"
BUNDLE="$ROOT/desktop"
JNI="$BUNDLE/android/app/src/main/jniLibs/arm64-v8a"
OUT="$DIST/NodePulse-android-arm64-${VERSION}.apk"

mkdir -p "$DIST" "$JNI"

cd "$ROOT"
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X github.com/HanphoneJan/nodepulse/internal/version.Version=$VERSION" \
  -o "$JNI/libnodepulse_core.so" ./cmd/nodepulse

cd "$BUNDLE"
"$FLUTTER" build apk --release --target-platform android-arm64 --build-name="$VERSION"

APK="$BUNDLE/build/app/outputs/flutter-apk/app-release.apk"
if [[ ! -f "$APK" ]]; then
  echo "APK not found: $APK" >&2
  exit 1
fi
cp "$APK" "$OUT"
echo "wrote $OUT"
