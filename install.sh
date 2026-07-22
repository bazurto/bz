#!/bin/bash

# Get the system architecture and operating system
ARCH=$(uname -m)
OS=$(uname -s)

# Identify the system architecture and OS
if [[ "$OS" == "Darwin" ]]; then
    if [[ "$ARCH" == "arm64" ]]; then
        binary="bz-darwin-arm64"
    elif [[ "$ARCH" == "x86_64" ]]; then
        binary="bz-darwin-amd64"
    else
        binary="unknown"
    fi
elif [[ "$OS" == "Linux" ]]; then
    if [[ "$ARCH" == "x86_64" ]]; then
        binary="bz-linux-amd64"
    elif [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]]; then
        binary="bz-linux-arm64"
    else
        binary="unknown"
    fi
else
    binary="unknown"
fi

if [[ "$binary" == "unknown" ]]; then
    echo ""
    echo "Unknown OS/Arch $OS/$ARCH"
    echo ""
    exit 1
fi

# Fetch latest release version from GitHub API
VERSION=$(curl -sL https://api.github.com/repos/bazurto/bz/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
if [[ -z "$VERSION" ]]; then
    echo "Error: Could not determine latest version from GitHub"
    exit 1
fi

echo "Installing bz $VERSION ($binary)..."

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

curl -fSL "https://github.com/bazurto/bz/releases/download/${VERSION}/${binary}" -o "$TMPFILE"
if [[ $? -ne 0 ]]; then
    echo "Error: Failed to download bz"
    exit 1
fi

# Verify checksum if available
CHECKSUM_URL="https://github.com/bazurto/bz/releases/download/${VERSION}/checksums.txt"
if curl -fsSL "$CHECKSUM_URL" -o "${TMPFILE}.checksums" 2>/dev/null; then
    EXPECTED=$(grep "$binary" "${TMPFILE}.checksums" | awk '{print $1}')
    if [[ -n "$EXPECTED" ]]; then
        if command -v sha256sum &>/dev/null; then
            ACTUAL=$(sha256sum "$TMPFILE" | awk '{print $1}')
        elif command -v shasum &>/dev/null; then
            ACTUAL=$(shasum -a 256 "$TMPFILE" | awk '{print $1}')
        fi
        if [[ -n "$ACTUAL" && "$ACTUAL" != "$EXPECTED" ]]; then
            echo "Error: Checksum verification failed"
            echo "  Expected: $EXPECTED"
            echo "  Actual:   $ACTUAL"
            exit 1
        fi
    fi
    rm -f "${TMPFILE}.checksums"
fi

sudo install -m 755 "$TMPFILE" /usr/local/bin/bz
echo "bz $VERSION installed to /usr/local/bin/bz"
