#!/usr/bin/env bash

# This is a simple installer that gets the latest version of gollama from Github and installs it to /usr/local/bin

INSTALL_DIR="/usr/local/bin"
INSTALL_PATH="${INSTALL_PATH:-$INSTALL_DIR/gollama}"
ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# Map architecture to short form
case $ARCH in
    x86_64)
        SHORT_ARCH="amd64"
        ;;
    aarch64)
        SHORT_ARCH="arm64"
        ;;
    *)
        SHORT_ARCH="unknown"
        ;;
esac

# Ensure the user is not root
if [ "$EUID" -eq 0 ]; then
  echo "Please do not run as root"
  exit 1
fi

# Get the latest release from Github
VER=$(curl --silent -qI https://github.com/sammcj/gollama/releases/latest | awk -F '/' '/^location/ {print  substr($NF, 1, length($NF)-1)}')

echo "Downloading gollama ${VER} for ${OS}-${ARCH}..."

if [ "${OS}" == "darwin" ]; then
  URL="https://github.com/sammcj/gollama/releases/download/$VER/gollama-macos.zip"
else
  URL="https://github.com/sammcj/gollama/releases/download/$VER/gollama-${OS}-${SHORT_ARCH}.zip"
fi

wget -q --show-progress -O /tmp/gollama.zip "${URL}"

# Unzip the binary
unzip -q /tmp/gollama.zip -d /tmp

# # # Move the binary to the install directory
mv gollama "${INSTALL_PATH}"

# # # Make the binary executable
chmod +x "${INSTALL_PATH}"

echo "gollama has been installed to ${INSTALL_PATH}"
