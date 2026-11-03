#!/usr/bin/env bash
# Build a single-file AppImage for NodePulse.
# Requires: appimagetool in PATH (or set APPIMAGETOOL).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-0.3.0}"
APPDIR="$ROOT/dist/NodePulse.AppDir"
OUT="$ROOT/dist/NodePulse-linux-amd64-${VERSION}.AppImage"

rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/applications" \
  "$APPDIR/usr/share/icons/hicolor/scalable/apps" \
  "$APPDIR/usr/share/doc/nodepulse/third-party"

cp "$ROOT/bin/nodepulse" "$APPDIR/usr/bin/"
cp "$ROOT/bin/nodepulse-gui" "$APPDIR/usr/bin/"
cp "$ROOT/packaging/nodepulse.desktop" "$APPDIR/usr/share/applications/"
cp "$ROOT/packaging/nodepulse.svg" "$APPDIR/usr/share/icons/hicolor/scalable/apps/"
cp "$ROOT/packaging/nodepulse.desktop" "$APPDIR/nodepulse.desktop"
cp "$ROOT/packaging/nodepulse.svg" "$APPDIR/nodepulse.svg"
cp "$ROOT/packaging/nodepulse-1024.png" "$APPDIR/nodepulse.png"
ln -sf nodepulse.png "$APPDIR/.DirIcon"
cp "$ROOT/LICENSE" "$APPDIR/usr/share/doc/nodepulse/LICENSE"
cp "$ROOT/THIRD_PARTY_NOTICES.md" "$APPDIR/usr/share/doc/nodepulse/THIRD_PARTY_NOTICES.md"
cp "$ROOT/internal/uiassets/dist/material-color-utilities.LICENSE.txt" \
  "$APPDIR/usr/share/doc/nodepulse/third-party/material-color-utilities-LICENSE.txt"
cp "$ROOT/internal/uiassets/dist/vendor/LICENSES/preact-LICENSE.txt" \
  "$APPDIR/usr/share/doc/nodepulse/third-party/preact-LICENSE.txt"
cp "$ROOT/internal/uiassets/dist/vendor/LICENSES/htm-LICENSE.txt" \
  "$APPDIR/usr/share/doc/nodepulse/third-party/htm-LICENSE.txt"
cp "$ROOT/internal/uiassets/dist/vendor/LICENSES/floating-ui-LICENSE.txt" \
  "$APPDIR/usr/share/doc/nodepulse/third-party/floating-ui-LICENSE.txt"
cp "$ROOT/internal/uiassets/dist/icons.LICENSE.txt" \
  "$APPDIR/usr/share/doc/nodepulse/third-party/material-symbols-LICENSE.txt"

cat > "$APPDIR/AppRun" <<'EOF'
#!/usr/bin/env bash
HERE="$(dirname "$(readlink -f "${0}")")"
exec "$HERE/usr/bin/nodepulse-gui" "$@"
EOF
chmod +x "$APPDIR/AppRun"

export ARCH=x86_64
RUNTIME_ARGS=()
if [[ -n "${APPIMAGE_RUNTIME_FILE:-}" ]]; then
  RUNTIME_ARGS=(--runtime-file "$APPIMAGE_RUNTIME_FILE")
fi
"${APPIMAGETOOL:-appimagetool}" "${RUNTIME_ARGS[@]}" "$APPDIR" "$OUT"
echo "wrote $OUT"
