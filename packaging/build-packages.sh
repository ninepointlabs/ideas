#!/usr/bin/env bash
# Build .deb, .rpm and Arch packages for idea into dist/.
# Usage: packaging/build-packages.sh [version]   (default 0.1.0)
# Packaging tools run inside a debian:trixie container; only go and docker are needed locally.
set -euo pipefail

VERSION="${1:-0.1.0}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
MAINTAINER="Ninepoint Labs"
DESCRIPTION="Terminal app for capturing and managing numbered ideas in markdown"
URL="https://github.com/ninepointlabs/ideas"

rm -rf "$DIST"
mkdir -p "$DIST/pkg/deb/DEBIAN" "$DIST/pkg/deb/usr/local/bin"

cd "$ROOT"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$DIST/idea" .
[ -x "$DIST/idea" ] || { echo "binary missing or not executable" >&2; exit 1; }

install -m755 "$DIST/idea" "$DIST/pkg/deb/usr/local/bin/idea"
cat > "$DIST/pkg/deb/DEBIAN/control" <<EOF
Package: idea
Version: $VERSION
Architecture: amd64
Maintainer: $MAINTAINER
Homepage: $URL
Description: $DESCRIPTION
EOF

docker run --rm -v "$DIST:/dist" -w /dist \
  -e VERSION="$VERSION" -e MAINTAINER="$MAINTAINER" -e DESCRIPTION="$DESCRIPTION" -e URL="$URL" \
  -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
  debian:trixie bash -euo pipefail -c '
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq ruby rpm zstd libarchive-tools >/dev/null
    gem install --no-document fpm >/dev/null

    dpkg-deb --root-owner-group --build pkg/deb "idea_${VERSION}_amd64.deb"

    fpm_common=(-s dir -n idea -v "$VERSION" -a amd64 --license MIT
      --maintainer "$MAINTAINER" --description "$DESCRIPTION" --url "$URL")
    fpm "${fpm_common[@]}" -t rpm    -p "idea-${VERSION}-1.x86_64.rpm"         idea=/usr/local/bin/idea
    fpm "${fpm_common[@]}" -t pacman -p "idea-${VERSION}-1-x86_64.pkg.tar.zst" idea=/usr/local/bin/idea

    chown -R "$HOST_UID:$HOST_GID" /dist
  '

for f in "$DIST"/idea_*.deb "$DIST"/idea-*.rpm "$DIST"/idea-*.pkg.tar.zst; do
  [ -s "$f" ] || { echo "missing or empty: $f" >&2; exit 1; }
done
ls -l "$DIST"/idea_*.deb "$DIST"/idea-*.rpm "$DIST"/idea-*.pkg.tar.zst
