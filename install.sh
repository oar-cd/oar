#!/usr/bin/env bash

set -e

echo "Installing Oar..."

# Get docker group GID for container access to Docker socket
DOCKER_GID=$(getent group docker | cut -d: -f3)
if [ -z "$DOCKER_GID" ]; then
    echo "Error: docker group not found. Make sure Docker is installed and you are in the docker group."
    exit 1
fi

# Generate encryption key and create .env file
echo "Generating encryption key..."
if [ -r /dev/urandom ]; then
    ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64 | tr -d '\n')
else
    echo "Error: Cannot generate encryption key. /dev/urandom is not available."
    exit 1
fi

# Get latest release information
echo "Fetching latest release information..."
RELEASE_INFO=$(curl -sSL https://api.github.com/repos/oar-cd/oar/releases/latest)
LATEST_VERSION=$(echo "$RELEASE_INFO" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)

if [ -n "$LATEST_VERSION" ]; then
    echo "Latest release: $LATEST_VERSION"
else
    echo "Error: Could not fetch latest version"
    exit 1
fi

COMPOSE_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*compose\.yaml"' | cut -d'"' -f4)
CLI_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*oar-linux-amd64"' | cut -d'"' -f4)

if [ -z "$COMPOSE_URL" ] || [ -z "$CLI_URL" ]; then
    echo "Error: Could not find required files in latest release"
    exit 1
fi

# Create installation directories
echo "Setting up installation directories..."
OAR_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/oar"
BIN_DIR="${HOME}/.local/bin"
mkdir -p "$OAR_DIR/data" "$BIN_DIR"

echo "Installation directory: $OAR_DIR"
echo "Data directory: $OAR_DIR/data"
echo "CLI executable directory: $BIN_DIR"

cd "$OAR_DIR"

# Download files from GitHub release
echo "Downloading files from release $LATEST_VERSION..."
curl -sSL "$COMPOSE_URL" -o compose.yaml
curl -sSL "$CLI_URL" -o "$BIN_DIR/oar"
chmod +x "$BIN_DIR/oar"

# Create .env file
echo "Creating .env file..."

cat >.env <<EOF
OAR_ENCRYPTION_KEY=$ENCRYPTION_KEY
EOF

# Start Oar
echo "Starting Oar with Docker Compose..."
docker compose --project-name oar up -d

# Create version file
echo "$LATEST_VERSION" >VERSION

echo ""
echo "Oar installed successfully ($LATEST_VERSION)!"
echo "Access Oar web UI at http://127.0.0.1:8080"
echo "Installation directory: $OAR_DIR"
echo "Data directory: $OAR_DIR/data"
echo "CLI executable: $BIN_DIR/oar"
echo ""
echo "To manage the installation:"
echo "  oar status    # Show status"
echo "  oar logs      # View logs"
echo "  oar stop      # Stop Oar"
echo "  oar start     # Start Oar"
echo "  oar update    # Update to latest version"
echo ""
echo "Note: Make sure $BIN_DIR is in your PATH to use 'oar' commands."
