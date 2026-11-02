#!/usr/bin/env bash
# Builds the Windows installer from Linux (or WSL / Git Bash with the tools on PATH).
#
#   other/build-installer.sh [display-version]
#
# Needs: go, pnpm, wails3 (go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16),
#        makensis (apt install nsis) and 7z (apt install p7zip-full).
#
# Output: bin/TcNo-Acc-Switcher.exe, bin/TcNo-Acc-Switcher.7z (portable),
#         bin/SHA256SUMS and bin/TcNo Account Switcher Installer.exe.
#
# Unlike other/github-release.ps1 this does not code-sign anything and does not
# write the updater .sig file, so Windows SmartScreen will warn on first run.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

APP_VERSION="$(sed -n 's/^ *version: *"\([0-9.]*\)".*/\1/p' build/config.yml | head -1)"
DISPLAY_VERSION="${1:-$APP_VERSION}"
# NSIS wants four numeric parts.
NSIS_VERSION="$(echo "$DISPLAY_VERSION" | grep -oE '^[0-9]+(\.[0-9]+){0,3}')"
while [ "$(echo "$NSIS_VERSION" | tr -cd '.' | wc -c)" -lt 3 ]; do NSIS_VERSION="$NSIS_VERSION.0"; done

for tool in go pnpm wails3 makensis 7z; do
  command -v "$tool" >/dev/null || { echo "missing tool: $tool" >&2; exit 1; }
done

echo "==> Version $DISPLAY_VERSION (NSIS $NSIS_VERSION)"

echo "==> Bindings and frontend"
wails3 generate bindings -clean=true -ts ./... >/dev/null
(cd frontend && pnpm install --frozen-lockfile >/dev/null && pnpm build >/dev/null)

echo "==> Windows resources (icon, manifest, version info)"
(cd build && wails3 generate syso -arch amd64 -icon windows/icon.ico \
  -manifest windows/wails.exe.manifest -info windows/info.json \
  -out ../wails_windows_amd64.syso)

echo "==> Windows executable"
mkdir -p bin
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags production -trimpath -buildvcs=false \
  -ldflags="-w -s -H windowsgui" -o bin/TcNo-Acc-Switcher.exe .
rm -f wails_windows_amd64.syso

NSIS_DIR="build/windows/nsis"
echo "==> Installer assets"
mkdir -p "$NSIS_DIR/img"
cp other/NSIS/img/{icon.ico,HeaderImage.bmp,SideBanner.bmp} "$NSIS_DIR/img/"
wails3 generate webview2bootstrapper -dir "$NSIS_DIR" >/dev/null

PLUGIN_DIR="$(mktemp -d)"
trap 'rm -rf "$PLUGIN_DIR" "${STAGE:-}"' EXIT
7z x -y -o"$PLUGIN_DIR" other/NSIS/Nsis7z_19.00.7z "Plugins/x86-unicode/nsis7z.dll" >/dev/null

echo "==> Portable archive"
STAGE="$(mktemp -d)"
cp bin/TcNo-Acc-Switcher.exe "$NSIS_DIR/MicrosoftEdgeWebview2Setup.exe" "$STAGE/"
rm -f bin/TcNo-Acc-Switcher.7z
7z a -m0=lzma2 -mx=9 -mfb=273 bin/TcNo-Acc-Switcher.7z "$STAGE/*" >/dev/null
(cd bin && sha256sum TcNo-Acc-Switcher.exe TcNo-Acc-Switcher.7z > SHA256SUMS)

echo "==> NSIS installer"
(cd "$NSIS_DIR" && makensis -V2 \
  -X"!addplugindir /x86-unicode $PLUGIN_DIR/Plugins/x86-unicode" \
  "-DVERSION=$NSIS_VERSION" "-DDISPLAY_VERSION=$DISPLAY_VERSION" project.nsi)
mv -f "$NSIS_DIR/TcNo Account Switcher Installer.exe" "bin/TcNo Account Switcher Installer.exe"

echo "==> Done"
ls -la bin
