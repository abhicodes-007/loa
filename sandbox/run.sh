#!/bin/bash
set -e

SANDBOX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_PROJECT="${TARGET_PROJECT:-$PWD}"
USER_ID=$(id -u)
GROUP_ID=$(id -g)
INSTALL_MODE="local" # Setup script changes this to "remote" if installing a release
LOA_SRC="/home/entity/go/src/github.com/laughingmandev/loa"

if [ "$INSTALL_MODE" == "local" ]; then
    echo "Syncing Loa source from $LOA_SRC..."
    mkdir -p "$SANDBOX_DIR/loa-src"
    rsync -a --exclude='.git' --exclude='memory.json' "$LOA_SRC/" "$SANDBOX_DIR/loa-src/"
else
    # In remote mode, we just copy the host's compiled binary into the context
    echo "Fetching local Loa binary..."
    if [ -f "$HOME/.loa/bin/loa" ]; then
        cp "$HOME/.loa/bin/loa" "$SANDBOX_DIR/loa-bin"
    elif command -v loa >/dev/null 2>&1; then
        cp "$(command -v loa)" "$SANDBOX_DIR/loa-bin"
    else
        echo "Error: Could not find loa executable. Are you sure it is installed?"
        exit 1
    fi
fi

cd "$SANDBOX_DIR"

# Build the container (caches will make this fast after the first run)
echo "Building Loa Sandbox image..."
docker compose build

# Determine config mount strategy based on host env
if [ -d "$HOME/.config/loa" ]; then
    echo "Host config found. Mounting it to share settings."
    export CONFIG_MOUNT="$HOME/.config/loa:/home/loa/.config/loa:rw"
else
    echo "No host config found. Isolating sandbox settings."
    mkdir -p "$SANDBOX_DIR/config"
    export CONFIG_MOUNT="./config:/home/loa/.config/loa:rw"
fi

# Run the container attached
echo "Starting Loa Sandbox attached to: $TARGET_PROJECT"
TARGET_PROJECT="$TARGET_PROJECT" USER_ID="$USER_ID" GROUP_ID="$GROUP_ID" CONFIG_MOUNT="$CONFIG_MOUNT" docker compose up
