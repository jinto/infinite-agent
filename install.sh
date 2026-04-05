#!/bin/sh
set -e

REPO="jinto/infinite-agent"
INSTALL_DIR="${INA_INSTALL_DIR:-$HOME/.ina/bin}"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
esac

# Get latest release tag
TAG=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$TAG" ]; then
  echo "Error: could not fetch latest release" >&2
  exit 1
fi

echo "Installing ina ${TAG} (${OS}/${ARCH})..."

mkdir -p "$INSTALL_DIR"

for BIN in ina ina-mcp; do
  URL="https://github.com/${REPO}/releases/download/${TAG}/${BIN}-${OS}-${ARCH}"
  echo "  Downloading ${BIN}..."
  curl -sSfL "$URL" -o "${INSTALL_DIR}/${BIN}"
  chmod +x "${INSTALL_DIR}/${BIN}"
done

# Add to PATH hint
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Add to your shell profile:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

echo ""
echo "Installed: ${INSTALL_DIR}/ina"
echo "Run 'ina setup' to configure Claude Code integration."
