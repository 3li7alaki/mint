#!/usr/bin/env bash
# mint installer/updater
# Install: curl -fsSL https://raw.githubusercontent.com/3li7alaki/mint/main/install.sh | bash
# Update:  same command — it's idempotent
#
# Safe: only touches ~/.claude/plugins/. Never modifies project files or .mint/ configs.
set -euo pipefail

REPO="3li7alaki/mint"
PLUGIN_NAME="mint"
MARKETPLACE_DIR="$HOME/.claude/plugins/marketplaces/mint"
CACHE_DIR="$HOME/.claude/plugins/cache/mint"

# Check claude CLI exists
if ! command -v claude &>/dev/null; then
  echo "Error: claude CLI not found. Install Claude Code first:"
  echo "  https://docs.anthropic.com/en/docs/claude-code"
  exit 1
fi

# Detect install vs update
if [ -d "$MARKETPLACE_DIR" ]; then
  echo "mint — updating..."
  MODE="update"
else
  echo "mint — installing..."
  MODE="install"
fi

# Add marketplace (idempotent — safe to run twice)
if [ "$MODE" = "install" ]; then
  echo "  Adding marketplace..."
  claude plugin marketplace add "$REPO" 2>/dev/null || true
fi

# Pull latest from GitHub
if [ -d "$MARKETPLACE_DIR/.git" ]; then
  echo "  Fetching latest..."
  git -C "$MARKETPLACE_DIR" fetch origin main -q 2>/dev/null || true
  git -C "$MARKETPLACE_DIR" reset --hard origin/main -q 2>/dev/null || true
fi

# Clear stale cache so plugin install picks up new version
rm -rf "$CACHE_DIR" 2>/dev/null || true

# Install/reinstall plugin
echo "  Installing plugin..."
claude plugin install "${PLUGIN_NAME}@${PLUGIN_NAME}" 2>/dev/null || true

# Verify
VERSION=$(grep -o '"version": "[^"]*"' "$CACHE_DIR/${PLUGIN_NAME}/"*/.claude-plugin/plugin.json 2>/dev/null | head -1 | grep -o '[0-9][^"]*' || echo "unknown")

echo ""
if [ "$MODE" = "update" ]; then
  echo "mint v${VERSION} updated successfully."
else
  echo "mint v${VERSION} installed successfully."
fi
echo ""
echo "Start a new Claude Code session to activate."
echo "Run 'mint init' in any project to configure gates and reviewers."
