#!/usr/bin/env bash
# Script: install.sh
# Purpose: Downloads and installs the latest arcup binary according to system OS and architecture.
#
# Key Components:
#   - Detects operating system (linux vs darwin)
#   - Detects architecture (amd64 vs arm64)
#   - Checks and prompts for rclone dependency installation
#   - Checks compression tools (zstd, gzip, zip) and warns if missing
#   - Downloads latest binary from GitHub releases
#   - Installs binary to /usr/local/bin/arcup
#   - Prompts user to apply default configuration via arcup --install
#
# Dependencies:
#   - curl or wget
#   - sudo or root privileges
#
# Error Types:
#   - Unsupported OS, unsupported architecture, or failed download

set -e

REPO="Dlcuy22/arcup"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)
        OS="linux"
        ;;
    darwin)
        OS="darwin"
        ;;
    *)
        echo "Error: Unsupported operating system: $OS" >&2
        exit 1
        ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

BINARY="arcup_${OS}_${ARCH}"

# Check for rclone dependency
if ! command -v rclone >/dev/null 2>&1; then
    echo "Notice: rclone is required by arcup for uploading backups to remote storage."
    if [ -e /dev/tty ]; then
        read -p "Do you want to install rclone now? (y/n): " rclone_resp </dev/tty
    else
        rclone_resp="n"
    fi

    case "$rclone_resp" in
        [yY][eE][sS]|[yY])
            echo "Installing rclone..."
            if command -v sudo >/dev/null 2>&1; then
                sudo -v && curl https://rclone.org/install.sh | sudo bash
            else
                curl https://rclone.org/install.sh | bash
            fi
            ;;
        *)
            echo "Skipping rclone installation. Please install rclone manually before using remote uploads."
            ;;
    esac
fi

# Check for compression tool dependencies (zstd, gz/gzip, zip)
for algo in zstd gzip zip; do
    if ! command -v "$algo" >/dev/null 2>&1; then
        echo "Warning: Compression tool '$algo' was not found on your system. Please install it via your OS package manager if you intend to use algorithm '$algo'."
    fi
done

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${BINARY} from latest release..."
if command -v curl >/dev/null 2>&1; then
    curl -sSL "$DOWNLOAD_URL" -o "$TMP_DIR/arcup"
elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TMP_DIR/arcup" "$DOWNLOAD_URL"
else
    echo "Error: Neither curl nor wget is installed." >&2
    exit 1
fi

chmod +x "$TMP_DIR/arcup"

echo "Installing arcup to /usr/local/bin/arcup..."
if [ "$(id -u)" -eq 0 ]; then
    install -Dm755 "$TMP_DIR/arcup" /usr/local/bin/arcup
else
    sudo install -Dm755 "$TMP_DIR/arcup" /usr/local/bin/arcup
fi

echo "arcup binary successfully installed."

if [ -e /dev/tty ]; then
    read -p "Do you want to apply default configuration? (y/n): " response </dev/tty
else
    response="n"
fi

case "$response" in
    [yY][eE][sS]|[yY])
        arcup --install
        ;;
    *)
        echo "Skipping default configuration installation."
        ;;
esac
