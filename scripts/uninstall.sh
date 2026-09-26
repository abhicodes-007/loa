#!/bin/bash
set -e

echo "Starting Loa uninstallation..."

# 1. Remove Global Binaries & Configs
echo "Removing global binaries and configurations (~/.loa, ~/.config/loa)..."
rm -rf "$HOME/.loa"
rm -rf "$HOME/.config/loa"

# 2. Remove the Sandbox
SANDBOX_DEFAULT="$HOME/loa-sandbox"
read -p "Did you install the loa-sandbox to the default location ($SANDBOX_DEFAULT)? (y/N) " REMOVE_SANDBOX < /dev/tty
if [[ "$REMOVE_SANDBOX" =~ ^[Yy]$ ]]; then
    echo "Removing sandbox at $SANDBOX_DEFAULT..."
    rm -rf "$SANDBOX_DEFAULT"
else
    echo "Skipping default sandbox removal. If you installed it elsewhere, please remove it manually."
fi

# 3. Remove Symlinks
echo "Removing symlinks..."
rm -f "$HOME/.local/bin/loa"
rm -f "$HOME/bin/loa"

# 4. Local Project State Prompt
read -p "Do you also want to remove local project state (.loa/ and .loa.json) in the CURRENT directory? (y/N) " REMOVE_LOCAL < /dev/tty
if [[ "$REMOVE_LOCAL" =~ ^[Yy]$ ]]; then
    echo "Removing local project state..."
    rm -rf .loa/
    rm -f .loa.json
else
    echo "Skipping local project state."
fi

echo "Uninstallation complete!"
