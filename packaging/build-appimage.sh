#!/usr/bin/env bash
# Build a single-file AppImage for NodePulse.
# Requires: appimagetool in PATH (or set APPIMAGETOOL).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-0.3.3}"
BUNDLE="$ROOT/desktop/build/linux/x64/release/bundle"
APPDIR="$ROOT/dist/NodePulse.AppDir"
OUT="$ROOT/dist/NodePulse-linux-amd64-${VERSION}.AppImage"

if [[ ! -x "$BUNDLE/nodepulse" ]]; then
  echo "Flutter bundle missing: $BUNDLE/nodepulse" >&2
  exit 1
fi
if [[ ! -x "$BUNDLE/nodepulse-core" ]]; then
  echo "NodePulse sidecar missing: $BUNDLE/nodepulse-core" >&2
  exit 1
fi

rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/lib/nodepulse" "$APPDIR/usr/share/applications" \
  "$APPDIR/usr/share/icons/hicolor/scalable/apps" \
  "$APPDIR/usr/share/doc/nodepulse"

cp -R "$BUNDLE/." "$APPDIR/usr/lib/nodepulse/"
cp "$ROOT/packaging/nodepulse.desktop" "$APPDIR/usr/share/applications/"
cp "$ROOT/packaging/nodepulse.svg" "$APPDIR/usr/share/icons/hicolor/scalable/apps/"
cp "$ROOT/packaging/nodepulse.desktop" "$APPDIR/nodepulse.desktop"
cp "$ROOT/packaging/nodepulse.svg" "$APPDIR/nodepulse.svg"
cp "$ROOT/packaging/nodepulse-1024.png" "$APPDIR/nodepulse.png"
ln -sf nodepulse.png "$APPDIR/.DirIcon"
cp "$ROOT/LICENSE" "$APPDIR/usr/share/doc/nodepulse/LICENSE"
cp "$ROOT/THIRD_PARTY_NOTICES.md" "$APPDIR/usr/share/doc/nodepulse/THIRD_PARTY_NOTICES.md"

cat > "$APPDIR/AppRun" <<'RUN'
#!/usr/bin/env bash
HERE="$(dirname "$(readlink -f "${0}")")"
exec "$HERE/usr/lib/nodepulse/nodepulse" "$@"
RUN
chmod +x "$APPDIR/AppRun"

export ARCH=x86_64
RUNTIME_ARGS=()
if [[ -n "${APPIMAGE_RUNTIME_FILE:-}" ]]; then
  RUNTIME_ARGS=(--runtime-file "$APPIMAGE_RUNTIME_FILE")
fi
"${APPIMAGETOOL:-appimagetool}" "${RUNTIME_ARGS[@]}" "$APPDIR" "$OUT"
echo "wrote $OUT"
