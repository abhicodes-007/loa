FROM debian:trixie-slim

# Prevent interactive prompts during apt
ENV DEBIAN_FRONTEND=noninteractive

# Update and install all required system tools and language runtimes
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential gcc g++ make cmake \
    universal-ctags git curl wget jq unzip tar sqlite3 ripgrep \
    tmux htop tree nano vim \
    python3 python3-pip python3-venv \
    nodejs npm yarnpkg \
    cargo rustc \
    default-jdk \
    php-cli php-curl php-mbstring php-xml php-sqlite3 php-zip composer \
    gosu ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Symlink yarnpkg to yarn if needed
RUN ln -s /usr/bin/yarnpkg /usr/local/bin/yarn || true

# Install Go (latest)
ENV GO_VERSION=1.23.0
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o go.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

# Set up the 'loa' user (uid/gid will be modified by entrypoint.sh)
RUN useradd -ms /bin/bash loa
ENV PATH="/home/loa/go/bin:${PATH}"

# Support both local build from source and remote pre-compiled binary
WORKDIR /opt/sandbox_ctx
COPY . /opt/sandbox_ctx
RUN if [ -d "./loa-src" ] && [ -f "./loa-src/go.mod" ]; then \
        echo "Building from source..." && \
        cd ./loa-src && \
        LLAMA_GO_VER=$(go list -m -f '{{.Version}}' github.com/tcpipuk/llama-go) && \
        if [ -n "$LLAMA_GO_VER" ]; then \
            TMP_LLAMA_DIR=$(mktemp -d) && \
            LLAMA_GO_COMMIT=$(echo "$LLAMA_GO_VER" | awk -F'-' '{print $NF}') && \
            git clone https://github.com/tcpipuk/llama-go.git "$TMP_LLAMA_DIR" && \
            (cd "$TMP_LLAMA_DIR" && git checkout "$LLAMA_GO_COMMIT" && git submodule update --init --recursive && make libbinding.a) && \
            export C_INCLUDE_PATH="$TMP_LLAMA_DIR" && export LIBRARY_PATH="$TMP_LLAMA_DIR"; \
        fi && \
        go build -tags localembed -o /usr/local/bin/loa ./cmd/loa; \
    elif [ -f "./loa-bin" ]; then \
        echo "Using pre-compiled binary..." && \
        cp ./loa-bin /usr/local/bin/loa && chmod +x /usr/local/bin/loa; \
    else \
        echo "Error: Neither loa-src nor loa-bin found in build context." && exit 1; \
    fi

# Set up the workspace directory
WORKDIR /workspace

# Add the entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["loa", "-addr", "0.0.0.0:8104", "."]
