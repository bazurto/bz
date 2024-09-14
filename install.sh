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
    echo "Unknow OS/Arch $OS/$ARCH"
    echo ""
    exit
fi

sudo curl https://github.com/bazurto/bz/releases/download/v0.1.7/$binary -o /usr/local/bin/bz
sudo chmod +x /usr/local/bin/bz

