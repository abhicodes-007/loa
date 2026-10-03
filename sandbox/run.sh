#!/bin/bash
set -e

SANDBOX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_PROJECT="${TARGET_PROJECT:-$PWD}"
USER_ID=$(id -u)
GROUP_ID=$(id -g)
LOA_SRC="/home/entity/go/src/github.com/laughingmandev/loa"

# Determine if we can use a pre-compiled binary or if we must build from source
USE_BINARY=false

# First check if there is a local binary
if [ -f "$HOME/.loa/bin/loa" ]; then
    HOST_BIN="$HOME/.loa/bin/loa"
elif command -v loa >/dev/null 2>&1; then
    HOST_BIN="$(command -v loa)"
else
    HOST_BIN=""
fi

if [ -n "$HOST_BIN" ] && command -v file >/dev/null 2>&1; then
    # Verify it is a Linux ELF binary (compatible with Debian container)
    if file "$HOST_BIN" | grep -q "ELF"; then
        USE_BINARY=true
    fi
fi

# Clean previous state
rm -rf "$SANDBOX_DIR/loa-src" "$SANDBOX_DIR/loa-bin"

if [ "$USE_BINARY" = true ]; then
    echo "Compatible Linux binary found. Fetching $HOST_BIN to skip compilation..."
    cp "$HOST_BIN" "$SANDBOX_DIR/loa-bin"
else
    if [ -d "$LOA_SRC" ]; then
        echo "Compatible Linux binary not found. Syncing Loa source from $LOA_SRC to build inside container..."
        mkdir -p "$SANDBOX_DIR/loa-src"
        rsync -a --exclude='.git' --exclude='memory.json' "$LOA_SRC/" "$SANDBOX_DIR/loa-src/"
    else
        echo "Error: No compatible binary found, and source directory $LOA_SRC does not exist. Cannot build sandbox."
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
    export CONFIG_MOUNT="$HOME/.config/loa:/home/loa/.config/loa:ro"
else
    echo "No host config found. Isolating sandbox settings."
    mkdir -p "$SANDBOX_DIR/config"
    export CONFIG_MOUNT="./config:/home/loa/.config/loa:rw"
fi

# Determine global .loa dir mount strategy
if [ -d "$HOME/.loa" ]; then
    echo "Host .loa directory found. Mounting it to share binaries and models."
    export LOA_DIR_MOUNT="$HOME/.loa:/home/loa/.loa:ro"
else
    mkdir -p "$SANDBOX_DIR/loa_env"
    export LOA_DIR_MOUNT="./loa_env:/home/loa/.loa:ro"
fi

# Run the container attached
echo "Starting Loa Sandbox attached to: $TARGET_PROJECT"
TARGET_PROJECT="$TARGET_PROJECT" USER_ID="$USER_ID" GROUP_ID="$GROUP_ID" CONFIG_MOUNT="$CONFIG_MOUNT" LOA_DIR_MOUNT="$LOA_DIR_MOUNT" docker compose up
