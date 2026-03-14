#!/bin/sh
# curl -fsSL https://raw.githubusercontent.com/yoorquezt-labs/yqctl/master/scripts/install.sh | sh
set -e

REPO="yoorquezt-labs/yqctl"
INSTALL_DIR="${YQCTL_INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version"
  exit 1
fi

ARCHIVE="yqctl_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"

echo "Installing yqctl v${VERSION} (${OS}/${ARCH})..."

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

curl -fsSL "$URL" -o "$TMP_DIR/$ARCHIVE"
tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR" yqctl

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_DIR/yqctl" "$INSTALL_DIR/yqctl"
else
  echo "Need sudo to install to $INSTALL_DIR"
  sudo mv "$TMP_DIR/yqctl" "$INSTALL_DIR/yqctl"
fi

chmod +x "$INSTALL_DIR/yqctl"

echo ""
echo "  yqctl v${VERSION} installed to $INSTALL_DIR/yqctl"
echo ""
echo "  Get started:"
echo "    yqctl --help"
echo "    yqctl -gw ws://your-gateway:9099/ws health"
echo ""
