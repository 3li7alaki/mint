#!/usr/bin/env bash
# mint installer/updater
# Install: curl -fsSL https://raw.githubusercontent.com/3li7alaki/mint/main/install.sh | bash
# Update:  same command — it's idempotent
#
# Installs the mint CLI globally (always) and the Claude Code plugin (if claude CLI exists).
set -euo pipefail

REPO="3li7alaki/mint"
PLUGIN_NAME="mint"
MINT_HOME="$HOME/.mint"
LINK_DIR="$HOME/.local/bin"
MARKETPLACE_DIR="$HOME/.claude/plugins/marketplaces/mint"
CACHE_DIR="$HOME/.claude/plugins/cache/mint"

echo ""
echo "  mint — disciplined agentic development"
echo ""

# ─── Step 1: Clone or update mint repo ────────────────────────────────────────

if [ -d "$MINT_HOME/.git" ]; then
  echo "  Updating mint..."
  git -C "$MINT_HOME" fetch origin main -q 2>/dev/null || true
  git -C "$MINT_HOME" reset --hard origin/main -q 2>/dev/null || true
  MODE="update"
else
  echo "  Installing mint..."
  git clone -q "https://github.com/$REPO.git" "$MINT_HOME" 2>/dev/null || {
    # If clone fails (dir exists but not git), remove and retry
    rm -rf "$MINT_HOME"
    git clone -q "https://github.com/$REPO.git" "$MINT_HOME"
  }
  MODE="install"
fi

# ─── Step 2: Make CLI globally available ──────────────────────────────────────

MINT_BIN="$MINT_HOME/cli/mint.js"
if [ -f "$MINT_BIN" ]; then
  chmod +x "$MINT_BIN"

  # Install dependencies
  if command -v bun &>/dev/null; then
    (cd "$MINT_HOME" && bun install --frozen-lockfile 2>/dev/null || bun install 2>/dev/null) || true
  elif command -v npm &>/dev/null; then
    (cd "$MINT_HOME" && npm install --omit=dev 2>/dev/null) || true
  else
    echo "  ⚠ Neither bun nor npm found — install one to use the CLI"
  fi

  mkdir -p "$LINK_DIR"
  ln -sf "$MINT_BIN" "$LINK_DIR/mint"
  echo "  ✓ CLI installed: mint init, mint config, mint doctor, mint update"
else
  echo "  ✗ CLI not found at $MINT_BIN"
fi

# Check PATH
case ":$PATH:" in
  *":$LINK_DIR:"*) ;;
  *)
    echo ""
    echo "  NOTE: $LINK_DIR is not in your PATH."
    echo "  Add to your shell profile (~/.bashrc or ~/.zshrc):"
    echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    ;;
esac

# ─── Step 3: Claude Code plugin (optional) ───────────────────────────────────

if command -v claude &>/dev/null; then
  echo "  Claude CLI detected — installing plugin..."

  # Add marketplace (idempotent)
  if [ ! -d "$MARKETPLACE_DIR" ]; then
    claude plugin marketplace add "$REPO" 2>/dev/null || true
  fi

  # Pull latest in marketplace
  if [ -d "$MARKETPLACE_DIR/.git" ]; then
    git -C "$MARKETPLACE_DIR" fetch origin main -q 2>/dev/null || true
    git -C "$MARKETPLACE_DIR" reset --hard origin/main -q 2>/dev/null || true
  fi

  # Clear cache + reinstall
  rm -rf "$CACHE_DIR" 2>/dev/null || true
  claude plugin install "${PLUGIN_NAME}@${PLUGIN_NAME}" 2>/dev/null || true
  echo "  ✓ Claude Code plugin installed"
else
  echo "  Claude CLI not found — skipping plugin install."
  echo "  The mint CLI works standalone. Install Claude Code later for full integration:"
  echo "    https://docs.anthropic.com/en/docs/claude-code"
fi

# ─── Done ─────────────────────────────────────────────────────────────────────

VERSION=$(grep -o '"version": "[^"]*"' "$MINT_HOME/.claude-plugin/plugin.json" 2>/dev/null | head -1 | grep -o '[0-9][^"]*' || echo "unknown")

echo ""
if [ "$MODE" = "update" ]; then
  echo "  mint v${VERSION} updated successfully."
else
  echo "  mint v${VERSION} installed successfully."
fi
echo ""
echo "  Quick start:"
echo "    cd your-project"
echo "    mint init          # interactive setup wizard"
echo "    mint init --yes    # auto-detect everything, no prompts"
echo "    mint doctor        # check your setup"
echo ""
