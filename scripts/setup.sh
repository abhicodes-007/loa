#!/bin/bash
set -e

LOCAL_MODE=0
if [ "$1" == "--local" ]; then
    LOCAL_MODE=1
fi

echo "Setting up Loa..."

ARCH=$(uname -m)
if [ "$ARCH" == "x86_64" ]; then
    LOA_ARCH="amd64"
    ASTGREP_ARCH="x86_64-unknown-linux-gnu.zip"
elif [ "$ARCH" == "aarch64" ] || [ "$ARCH" == "arm64" ]; then
    LOA_ARCH="arm64"
    ASTGREP_ARCH="aarch64-unknown-linux-gnu.zip"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

BIN_DIR="$HOME/.loa/bin"
mkdir -p "$BIN_DIR"

if [ ! -f "$BIN_DIR/ast-grep" ]; then
    echo "Downloading ast-grep (sg) for $ARCH..."
    URL=$(curl -s https://api.github.com/repos/ast-grep/ast-grep/releases/latest | grep browser_download_url | grep "$ASTGREP_ARCH" | cut -d '"' -f 4)
    wget -qO ast-grep.zip "$URL"
    unzip -q ast-grep.zip -d "$BIN_DIR"
    rm ast-grep.zip
    if [ -f "$BIN_DIR/ast-grep" ]; then
        chmod +x "$BIN_DIR/ast-grep"
    elif [ -f "$BIN_DIR/sg" ]; then
        mv "$BIN_DIR/sg" "$BIN_DIR/ast-grep"
        chmod +x "$BIN_DIR/ast-grep"
    fi
    echo "ast-grep installed successfully to $BIN_DIR/ast-grep"
else
    echo "ast-grep is already installed."
fi

echo "Setting up ctags..."
if ! command -v ctags >/dev/null 2>&1; then
    if command -v apt-get >/dev/null 2>&1; then
        echo "Detected apt-get. Installing universal-ctags..."
        sudo apt-get update -y
        sudo apt-get install -y universal-ctags
    elif command -v pacman >/dev/null 2>&1; then
        echo "Detected pacman. Installing universal-ctags..."
        sudo pacman -Sy --noconfirm universal-ctags
    else
        echo "Could not detect apt-get or pacman. Please install universal-ctags manually."
    fi
else
    echo "ctags is already installed."
fi

if [ $LOCAL_MODE -eq 1 ]; then
    echo "Building Loa from source (--local mode)..."
    go build -o "$BIN_DIR/loa" ./cmd/loa
    echo "Loa compiled successfully to $BIN_DIR/loa"
else
    echo "Downloading latest Loa release for $LOA_ARCH..."
    RELEASE_URL="https://github.com/laughingmandev/loa/releases/latest/download/loa-linux-$LOA_ARCH.tar.gz"
    
    if curl --output /dev/null --silent --head --fail "$RELEASE_URL"; then
        TMP_DIR=$(mktemp -d)
        wget -qO "$TMP_DIR/loa.tar.gz" "$RELEASE_URL"
        tar -xzf "$TMP_DIR/loa.tar.gz" -C "$TMP_DIR"
        
        mv "$TMP_DIR/loa" "$BIN_DIR/loa"
        chmod +x "$BIN_DIR/loa"
        echo "Loa downloaded successfully to $BIN_DIR/loa"
        
        TMP_SANDBOX_DIR="$TMP_DIR/sandbox"
    else
        echo "Failed to fetch release from $RELEASE_URL"
        echo "The repository might not have active releases yet."
        echo "Please use 'bash scripts/setup.sh --local' to build from source."
        exit 1
    fi
fi

echo "Setting up path configuration for Loa..."
LOA_BIN="$BIN_DIR/loa"

if [ -f "$LOA_BIN" ]; then
    if [ -d "$HOME/.local/bin" ] && [ -w "$HOME/.local/bin" ]; then
        echo "Found ~/.local/bin. Symlinking loa..."
        ln -sf "$LOA_BIN" "$HOME/.local/bin/loa"
        echo "Successfully linked loa to ~/.local/bin/loa"
    elif [ -d "$HOME/bin" ] && [ -w "$HOME/bin" ]; then
        echo "Found ~/bin. Symlinking loa..."
        ln -sf "$LOA_BIN" "$HOME/bin/loa"
        echo "Successfully linked loa to ~/bin/loa"
    else
        echo "Warning: Could not find a standard user bin directory (~/.local/bin or ~/bin)."
        echo "The 'loa' executable is located at: $LOA_BIN"
        echo "Please manually add it to your PATH or copy it to a directory like /usr/local/bin."
    fi
fi

echo "Done setting up Loa core!"

# Sandbox Installation Prompt
if ! command -v docker >/dev/null 2>&1 || ! (command -v docker-compose >/dev/null 2>&1 || docker compose version >/dev/null 2>&1); then
    echo "================================================================="
    echo "WARNING: Docker and/or Docker Compose were not found on this system."
    echo "You will need to install them before you can run the loa-sandbox."
    echo "You can still complete this setup now and install Docker later."
    echo "================================================================="
fi

read -p "Would you like to install the loa-sandbox environment for safe containerized execution? (y/N) " INSTALL_SANDBOX
if [[ "$INSTALL_SANDBOX" =~ ^[Yy]$ ]]; then
    read -p "Enter installation path for loa-sandbox [default: $HOME/loa-sandbox]: " SANDBOX_TARGET
    SANDBOX_TARGET=${SANDBOX_TARGET:-$HOME/loa-sandbox}
    
    mkdir -p "$SANDBOX_TARGET"
    
    if [ $LOCAL_MODE -eq 1 ]; then
        # Copy from local repo
        cp -r ./sandbox/* "$SANDBOX_TARGET/"
        INSTALL_MODE_VAL="local"
    else
        # Move from extracted release
        if [ -n "$TMP_SANDBOX_DIR" ] && [ -d "$TMP_SANDBOX_DIR" ]; then
            mv "$TMP_SANDBOX_DIR"/* "$SANDBOX_TARGET/"
            INSTALL_MODE_VAL="remote"
        else
            echo "Error: Sandbox files not found in the release archive."
            exit 1
        fi
    fi
    
    # Update INSTALL_MODE in run.sh if it's remote
    if [ "$INSTALL_MODE_VAL" == "remote" ]; then
        sed -i 's/INSTALL_MODE="local"/INSTALL_MODE="remote"/g' "$SANDBOX_TARGET/run.sh"
    fi
    
    chmod +x "$SANDBOX_TARGET/run.sh" "$SANDBOX_TARGET/entrypoint.sh"
    
    echo "Loa Sandbox installed successfully!"
    echo "To use it from anywhere, add this alias to your ~/.bashrc or ~/.zshrc:"
    echo "  alias loa-sandbox='$SANDBOX_TARGET/run.sh'"
fi

# Cleanup
if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
fi

echo "All Done!"
