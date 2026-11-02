#!/usr/bin/env bash
# Build a single-file AppImage for NodePulse.
# Requires: appimagetool in PATH (or set APPIMAGETOOL).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-dev}"
APPDIR="$ROOT/dist/NodePulse.AppDir"
OUT="$ROOT/dist/NodePulse-${VERSION}-x86_64.AppImage"

rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/applications" \
  "$APPDIR/usr/share/icons/hicolor/scalable/apps"

cp "$ROOT/bin/nodepulse" "$APPDIR/usr/bin/"
cp "$ROOT/bin/nodepulse-gui" "$APPDIR/usr/bin/"
cp "$ROOT/packaging/nodepulse.desktop" "$APPDIR/usr/share/applications/"
cp "$ROOT/packaging/nodepulse.svg" "$APPDIR/usr/share/icons/hicolor/scalable/apps/"
cp "$ROOT/packaging/nodepulse.desktop" "$APPDIR/nodepulse.desktop"
cp "$ROOT/packaging/nodepulse.svg" "$APPDIR/nodepulse.svg"

cat > "$APPDIR/AppRun" <<'EOF'
#!/usr/bin/env bash
HERE="$(dirname "$(readlink -f "${0}")")"
exec "$HERE/usr/bin/nodepulse-gui" "$@"
EOF
chmod +x "$APPDIR/AppRun"

export ARCH=x86_64
"${APPIMAGETOOL:-appimagetool}" "$APPDIR" "$OUT"
echo "wrote $OUT"
