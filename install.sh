#!/usr/bin/env bash

set -e

echo "Installing Oar..."

# Create installation directories
OAR_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/oar"
BIN_DIR="${HOME}/.local/bin"
mkdir -p "$OAR_DIR" "$BIN_DIR"
cd "$OAR_DIR"

# Get latest release information
echo "Downloading latest Docker Compose configuration..."
RELEASE_INFO=$(curl -sSL https://api.github.com/repos/ch00k/oar/releases/latest)
LATEST_VERSION=$(echo "$RELEASE_INFO" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)
COMPOSE_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*compose\.yaml"' | cut -d'"' -f4)
CLI_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*oar-linux-amd64"' | cut -d'"' -f4)

if [ -z "$COMPOSE_URL" ] || [ -z "$CLI_URL" ]; then
    echo "Error: Could not find required files in latest release"
    exit 1
fi

# Download files from GitHub release
echo "Downloading files from release $LATEST_VERSION..."
curl -sSL "$COMPOSE_URL" -o compose.yaml
curl -sSL "$CLI_URL" -o "$BIN_DIR/oar"
chmod +x "$BIN_DIR/oar"

# Start Oar
echo "Starting Oar with Docker Compose..."
docker compose --project-name oar up -d

# Create version file
echo "$LATEST_VERSION" >VERSION

echo ""
echo "Oar installed successfully ($LATEST_VERSION)!"
echo "Access Oar at: http://127.0.0.1:8080"
echo "Installation directory: $OAR_DIR"
echo ""
echo "To manage the installation:"
echo "  oar update             # Update to latest version"
echo "  oar project list       # List projects via CLI"
echo "  cd $OAR_DIR"
echo "  docker compose logs    # View logs"
echo "  docker compose stop    # Stop Oar"
echo "  docker compose up -d   # Start Oar"
echo ""
echo "Note: Make sure $BIN_DIR is in your PATH to use 'oar' commands."
