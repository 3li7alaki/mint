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

# ─── Helpers ──────────────────────────────────────────────────────────────────

ok()   { echo "  ✓ $1"; }
fail() { echo "  ✗ $1"; }
info() { echo "  · $1"; }

echo ""
echo "  mint — disciplined agentic development"
echo ""

# ─── Step 1: Clone or update mint repo ────────────────────────────────────────

if [ -d "$MINT_HOME/.git" ]; then
  info "Updating mint..."
  # Ensure LF line endings (WSL with Windows git may default to CRLF)
  git -C "$MINT_HOME" config core.autocrlf input 2>/dev/null || true
  if git -C "$MINT_HOME" fetch origin main -q 2>&1; then
    git -C "$MINT_HOME" clean -fd -q
    git -C "$MINT_HOME" reset --hard origin/main -q
    ok "Updated ~/.mint"
  else
    fail "Failed to fetch latest mint — check your internet connection"
    echo "    Try: git -C $MINT_HOME fetch origin main"
    exit 1
  fi
  MODE="update"
else
  info "Installing mint..."
  if git clone -q -c core.autocrlf=input "https://github.com/$REPO.git" "$MINT_HOME" 2>&1; then
    ok "Cloned to ~/.mint"
  else
    # If directory exists but isn't git, remove and retry
    if [ -d "$MINT_HOME" ]; then
      info "Removing stale $MINT_HOME and retrying..."
      rm -rf "$MINT_HOME"
      if git clone -q -c core.autocrlf=input "https://github.com/$REPO.git" "$MINT_HOME" 2>&1; then
        ok "Cloned to ~/.mint"
      else
        fail "Git clone failed"
        echo "    Check: internet connection, DNS, firewall"
        echo "    Manual: git clone https://github.com/$REPO.git $MINT_HOME"
        exit 1
      fi
    else
      fail "Git clone failed"
      echo "    Check: internet connection, DNS, firewall"
      echo "    Manual: git clone https://github.com/$REPO.git $MINT_HOME"
      exit 1
    fi
  fi
  MODE="install"
fi

# ─── Step 2: Ensure Bun is installed ─────────────────────────────────────────

if ! command -v bun &>/dev/null; then
  info "Bun not found — installing (required for mint CLI)..."
  if curl -fsSL https://bun.sh/install | bash 2>&1; then
    export BUN_INSTALL="$HOME/.bun"
    export PATH="$BUN_INSTALL/bin:$PATH"
    if command -v bun &>/dev/null; then
      ok "Bun $(bun --version) installed"
    else
      fail "Bun installed but not in PATH"
      echo "    Add to your shell profile (~/.bashrc or ~/.zshrc):"
      echo "      export BUN_INSTALL=\"\$HOME/.bun\""
      echo "      export PATH=\"\$BUN_INSTALL/bin:\$PATH\""
      exit 1
    fi
  else
    fail "Bun install failed"
    echo "    Install manually: https://bun.sh"
    echo "    mint CLI requires Bun to run."
    exit 1
  fi
else
  ok "Bun $(bun --version)"
fi

# ─── Step 3: Install dependencies ────────────────────────────────────────────

info "Installing dependencies..."
if (cd "$MINT_HOME" && bun install --frozen-lockfile 2>&1) || (cd "$MINT_HOME" && bun install 2>&1); then
  ok "Dependencies installed"
else
  fail "Dependency install failed — run manually: cd $MINT_HOME && bun install"
fi

# ─── Step 4: Make CLI globally available ──────────────────────────────────────

MINT_BIN="$MINT_HOME/cli/mint.js"
if [ -f "$MINT_BIN" ]; then
  chmod +x "$MINT_BIN"
  mkdir -p "$LINK_DIR"
  ln -sf "$MINT_BIN" "$LINK_DIR/mint"
  ok "CLI linked: $LINK_DIR/mint"
else
  fail "CLI not found at $MINT_BIN — installation may be incomplete"
fi

# Check PATH
case ":$PATH:" in
  *":$LINK_DIR:"*) ;;
  *)
    echo ""
    echo "  NOTE: $LINK_DIR is not in your PATH."

    # Detect shell and offer to add
    SHELL_RC=""
    if [ -n "${ZSH_VERSION:-}" ] || [ "$(basename "$SHELL")" = "zsh" ]; then
      SHELL_RC="$HOME/.zshrc"
    elif [ -n "${BASH_VERSION:-}" ] || [ "$(basename "$SHELL")" = "bash" ]; then
      SHELL_RC="$HOME/.bashrc"
    fi

    if [ -n "$SHELL_RC" ] && [ -f "$SHELL_RC" ]; then
      echo "  Adding to $SHELL_RC..."
      echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$SHELL_RC"
      export PATH="$HOME/.local/bin:$PATH"
      ok "Added to $SHELL_RC — restart your shell or run: source $SHELL_RC"
    else
      echo "  Add to your shell profile:"
      echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
    echo ""
    ;;
esac

# ─── Step 5: Claude Code plugin (optional) ───────────────────────────────────

if command -v claude &>/dev/null; then
  info "Claude CLI detected — installing plugin..."

  # Add marketplace (idempotent)
  if [ ! -d "$MARKETPLACE_DIR" ]; then
    if claude plugin marketplace add "$REPO" 2>&1; then
      ok "Marketplace registered"
    else
      fail "Marketplace registration failed — try: claude plugin marketplace add $REPO"
    fi
  fi

  # Pull latest in marketplace
  if [ -d "$MARKETPLACE_DIR/.git" ]; then
    git -C "$MARKETPLACE_DIR" config core.autocrlf input 2>/dev/null || true
    git -C "$MARKETPLACE_DIR" fetch origin main -q 2>/dev/null || true
    git -C "$MARKETPLACE_DIR" clean -fd -q 2>/dev/null || true
    git -C "$MARKETPLACE_DIR" reset --hard origin/main -q 2>/dev/null || true
  fi

  # Install marketplace deps
  if [ -f "$MARKETPLACE_DIR/package.json" ]; then
    (cd "$MARKETPLACE_DIR" && bun install --frozen-lockfile 2>/dev/null || bun install 2>/dev/null) || true
  fi

  # Clear cache + reinstall
  rm -rf "$CACHE_DIR" 2>/dev/null || true
  if claude plugin install "${PLUGIN_NAME}@${PLUGIN_NAME}" 2>&1; then
    ok "Claude Code plugin installed"
  else
    fail "Plugin install failed — try: claude plugin install ${PLUGIN_NAME}@${PLUGIN_NAME}"
  fi
else
  info "Claude CLI not found — skipping plugin install"
  echo "    The mint CLI works standalone. Install Claude Code later for full integration."
fi

# ─── Step 6: Post-install health check ────────────────────────────────────────

if command -v mint &>/dev/null; then
  info "Running health check..."
  mint doctor --quick 2>/dev/null || true
fi

# ─── Done ─────────────────────────────────────────────────────────────────────

VERSION=$(grep -o '"version": "[^"]*"' "$MINT_HOME/.claude-plugin/plugin.json" 2>/dev/null | head -1 | grep -o '[0-9][^"]*' || echo "unknown")

echo ""
if [ "$MODE" = "update" ]; then
  ok "mint v${VERSION} updated"
else
  ok "mint v${VERSION} installed"
fi
echo ""
echo "  Quick start:"
echo "    cd your-project"
echo "    mint init          # interactive setup wizard"
echo "    mint init --yes    # auto-detect everything, no prompts"
echo "    mint doctor        # check your setup"
echo "    mint status        # quick health at a glance"
echo ""
