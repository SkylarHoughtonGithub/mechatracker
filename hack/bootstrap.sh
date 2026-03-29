#!/usr/bin/env bash
# bootstrap.sh — installs the minimum needed to run `task`.
# Run once on a fresh Mac, then use `task setup` for everything else.

set -euo pipefail

# ── Homebrew ──────────────────────────────────────────────────────────────────
if ! command -v brew &>/dev/null; then
  echo "==> Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  eval "$(/opt/homebrew/bin/brew shellenv)" 2>/dev/null || eval "$(/usr/local/bin/brew shellenv)"
else
  echo "==> Homebrew already installed"
fi

# ── Task ──────────────────────────────────────────────────────────────────────
if ! command -v task &>/dev/null; then
  echo "==> Installing go-task..."
  brew install go-task
else
  echo "==> task already installed"
fi

echo ""
echo "Ready. Run: task setup"
