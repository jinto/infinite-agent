#!/bin/sh
set -e

REPO="jinto/infinite-angel"
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

# 1. Download binaries
mkdir -p "$INSTALL_DIR"

for BIN in ina ina-mcp; do
  URL="https://github.com/${REPO}/releases/download/${TAG}/${BIN}-${OS}-${ARCH}"
  echo "  Downloading ${BIN}..."
  curl -sSfL "$URL" -o "${INSTALL_DIR}/${BIN}"
  chmod +x "${INSTALL_DIR}/${BIN}"
  # macOS: remove quarantine + ad-hoc codesign
  xattr -d com.apple.quarantine "${INSTALL_DIR}/${BIN}" 2>/dev/null || true
  codesign -s - -f "${INSTALL_DIR}/${BIN}" 2>/dev/null || true
done

# 2. Remove legacy local plugin (conflicts with marketplace install)
rm -rf "$HOME/.claude/plugins/ina" 2>/dev/null || true

# 3. Detect shell profile
CURRENT_SHELL=$(basename "${SHELL:-/bin/sh}")
case "$CURRENT_SHELL" in
  zsh)  PROFILE="$HOME/.zshrc" ;;
  bash)
    if [ -f "$HOME/.bash_profile" ]; then
      PROFILE="$HOME/.bash_profile"
    elif [ -f "$HOME/.bashrc" ]; then
      PROFILE="$HOME/.bashrc"
    else
      PROFILE="$HOME/.bash_profile"
    fi
    ;;
  *)    PROFILE="$HOME/.profile" ;;
esac

# 4. Add to PATH if not already present
PATH_LINE="export PATH=\"${INSTALL_DIR}:\$PATH\""
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    if [ -n "$PROFILE" ]; then
      if ! grep -q ".ina/bin" "$PROFILE" 2>/dev/null; then
        echo "" >> "$PROFILE"
        echo "# ina — Infinite Angel" >> "$PROFILE"
        echo "$PATH_LINE" >> "$PROFILE"
        echo "  PATH added to ${PROFILE}"
      fi
    else
      echo ""
      echo "Add to your shell profile:"
      echo "  $PATH_LINE"
    fi
    export PATH="${INSTALL_DIR}:$PATH"
    ;;
esac

# 5. Record installed version
echo "$TAG" > "${INSTALL_DIR}/../version"

# 6. Auto-configure Claude Code (hooks + MCP + statusline)
echo ""
echo "Configuring Claude Code..."
"${INSTALL_DIR}/ina" setup || echo "  Warning: auto-setup failed. Run 'ina setup' manually."

echo ""
echo "Installed:"
echo "  Binaries: ${INSTALL_DIR}/ina, ina-mcp"
echo ""
echo "Next steps:"
if [ -n "$PROFILE" ]; then
  echo "  source ${PROFILE}              # reload PATH (or open a new terminal)"
fi
echo "  ina install                        # register daemon (auto-start on login)"
echo ""
echo "  # In Claude Code:"
echo "  /plugin marketplace add https://github.com/jinto/infinite-angel"
echo "  /plugin install ina"
echo ""
echo "  # HUD (statusline):"
echo "  ina hud off                        # hide context bar"
echo "  ina hud on                         # show it again"
