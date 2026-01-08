#!/bin/bash

# Vyom Middleware - One-Command Installer
# Usage: curl -fsSL https://code-execute-rishi.github.io/VG_Cloud_Middleware_Download/install.sh | sudo bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}==================================================${NC}"
echo -e "${BLUE}       Vyom Middleware Installer (RPi OS)         ${NC}"
echo -e "${BLUE}==================================================${NC}"

# 1. Check Root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Please run as root (use sudo).${NC}"
  exit 1
fi

# 2. Detect Architecture
ARCH=$(dpkg --print-architecture)
echo -e "${GREEN}[+] Detected Architecture: ${ARCH}${NC}"

if [[ "$ARCH" != "arm64" && "$ARCH" != "armhf" ]]; then
    if [[ "$ARCH" == "armv7l" ]]; then
        echo -e "${GREEN}[+] Detected armv7l. Mapping to armhf...${NC}"
        ARCH="armhf"
    else
        echo -e "${RED}[!] Warning: This script is optimized for Raspberry Pi (arm64/armhf). Detected: $ARCH${NC}"
        echo "Proceeding anyway in 3 seconds..."
        sleep 3
    fi
fi

# 3. Install Dependencies
echo -e "${GREEN}[+] Updating APT repositories...${NC}"
apt-get update -qq

echo -e "${GREEN}[+] Installing System Dependencies (GStreamer, ZeroTier, Tools)...${NC}"
apt-get install -y -qq \
    curl \
    wget \
    jq \
    v4l-utils \
    libgstreamer1.0-0 \
    gstreamer1.0-tools \
    gstreamer1.0-plugins-base \
    gstreamer1.0-plugins-good \
    gstreamer1.0-plugins-bad \
    gstreamer1.0-libav \
    psmisc \
    zerotier-one

# 4. Enable ZeroTier
echo -e "${GREEN}[+] Enabling ZeroTier Service...${NC}"
systemctl enable zerotier-one
systemctl start zerotier-one

# 5. Fetch Latest Release
# 5. Fetch Latest Release
REPO_OWNER="code-execute-rishi"
REPO_NAME="VG_Cloud_Middleware_Download"
LATEST_RELEASE_URL="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"

echo -e "${GREEN}[+] Fetching latest release info from GitHub...${NC}"
RELEASE_JSON=$(curl -s $LATEST_RELEASE_URL)
VERSION=$(echo "$RELEASE_JSON" | jq -r .tag_name)

if [ "$VERSION" == "null" ] || [ -z "$VERSION" ]; then
    echo -e "${RED}Latest release not found (might be a new repo). Checking for Nightly tag...${NC}"
    LATEST_RELEASE_URL="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/tags/nightly"
    RELEASE_JSON=$(curl -s $LATEST_RELEASE_URL)
    VERSION=$(echo "$RELEASE_JSON" | jq -r .tag_name)
fi

if [ "$VERSION" == "null" ] || [ -z "$VERSION" ]; then
    echo -e "${RED}Failed to fetch any release (Latest or Nightly). Check repo visibility or internet connection.${NC}"
    exit 1
fi

echo -e "${GREEN}[+] Latest Version: ${VERSION}${NC}"

# Find asset for the architecture
ASSET_URL=$(echo "$RELEASE_JSON" | jq -r --arg arch "$ARCH" '.assets[] | select(.name | contains($arch)) | .browser_download_url' | head -n 1)

# Fallback: If strict arch match fails, try generic or ask user
if [ -z "$ASSET_URL" ]; then
    echo -e "${RED}No pre-built package found for architecture: $ARCH in release $VERSION${NC}"
    echo "This installer expects files named like 'vyom-middleware_X.X.X_arm64.deb'"
    exit 1
fi

FILE_NAME=$(basename "$ASSET_URL")
echo -e "${GREEN}[+] Downloading $FILE_NAME...${NC}"
wget -q --show-progress -O "/tmp/$FILE_NAME" "$ASSET_URL"

# 6. Install Package
echo -e "${GREEN}[+] Installing Deb Package...${NC}"
dpkg -i "/tmp/$FILE_NAME" || apt-get install -f -y

# 7. Cleanup
rm "/tmp/$FILE_NAME"

echo -e "${BLUE}==================================================${NC}"
echo -e "${GREEN}   Vyom Middleware Installed Successfully!       ${NC}"
echo -e "${BLUE}==================================================${NC}"
echo "Services should be starting now."
echo "Check status: sudo systemctl status vyom-api"
