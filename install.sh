#!/usr/bin/env bash
set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Check if Docker is installed and running
if ! command -v docker &>/dev/null; then
    echo "Error: Docker is not installed. Please install Docker first."
    exit 1
fi

if ! systemctl is-active --quiet docker; then
    echo "Error: Docker service is not running. Please start Docker first."
    exit 1
fi

# Detect installation mode
UPGRADE_MODE=false
CURRENT_VERSION="unknown"
if [ -d "/opt/oar" ] && [ -f "/opt/oar/bin/oar" ]; then
    UPGRADE_MODE=true
    CURRENT_VERSION=$(/opt/oar/bin/oar version 2>/dev/null || echo "unknown")
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

# Show installation mode and ask for confirmation
if [ "$UPGRADE_MODE" = true ]; then
    echo
    echo "Existing Oar installation detected (version: $CURRENT_VERSION)"
    echo "This will upgrade to version: $LATEST_VERSION"
    echo
    read -p "Continue with upgrade? (y/N): " -n 1 -r
else
    echo
    echo "This will install Oar version: $LATEST_VERSION"
    echo
    read -p "Continue with installation? (y/N): " -n 1 -r
fi
echo
[[ ! $REPLY =~ ^[Yy]$ ]] && exit 0

# Get download URLs for both files
BINARY_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*oar-linux-amd64"' | cut -d'"' -f4)
SERVICE_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*oar\.service"' | cut -d'"' -f4)

if [ -z "$BINARY_URL" ] || [ -z "$SERVICE_URL" ]; then
    echo "Error: Could not find required files (oar-linux-amd64, oar.service) in latest release"
    exit 1
fi

# Stop service during upgrade
if [ "$UPGRADE_MODE" = true ]; then
    echo "Stopping Oar service for upgrade..."
    systemctl stop oar

    # Wait for service to actually stop (timeout after 30 seconds)
    echo "Waiting for service to stop..."
    timeout 30 sh -c 'while systemctl is-active --quiet oar; do sleep 0.5; done'

    if systemctl is-active --quiet oar; then
        echo "Service failed to stop within 30 seconds"
        exit 1
    fi
    echo "Service stopped successfully"
fi

# Create installation directory structure
if [ "$UPGRADE_MODE" = true ]; then
    echo "Preparing upgrade..."
    # Ensure backups directory exists
    mkdir -p /opt/oar/data/backups
else
    echo "Creating /opt/oar directory structure..."
    mkdir -p /opt/oar/{bin,data/backups}
fi

# Create database backup during upgrade
if [ "$UPGRADE_MODE" = true ] && [ -f "/opt/oar/data/oar.db" ]; then
    BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_FILE="/opt/oar/data/backups/oar.db.${CURRENT_VERSION}.${BACKUP_TIMESTAMP}"

    echo "Creating database backup..."
    cp /opt/oar/data/oar.db "$BACKUP_FILE"
    echo "Database backed up to: $BACKUP_FILE"
fi

# Backup current binary during upgrade
if [ "$UPGRADE_MODE" = true ]; then
    BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BINARY_BACKUP="/opt/oar/data/backups/oar.${CURRENT_VERSION}.${BACKUP_TIMESTAMP}"
    cp /opt/oar/bin/oar "$BINARY_BACKUP"
    echo "Current binary backed up to: $BINARY_BACKUP"
fi

# Download and install binary
echo "Installing Oar binary..."
curl -sSL "$BINARY_URL" -o /opt/oar/bin/oar
chmod +x /opt/oar/bin/oar

# Download systemd service (skip if upgrading and file already exists)
if [ "$UPGRADE_MODE" = false ] || [ ! -f "/opt/oar/oar.service" ]; then
    echo "Downloading systemd service file..."
    curl -sSL "$SERVICE_URL" -o /opt/oar/oar.service
fi

# Install systemd service
echo "Installing systemd service..."
ln -sf /opt/oar/oar.service /etc/systemd/system/oar.service

# Create configuration file (only for fresh installs)
if [ "$UPGRADE_MODE" = false ]; then
    # Generate encryption key
    echo "Generating encryption key..."
    if [ -r /dev/urandom ]; then
        ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64 | tr -d '\n')
    else
        echo "Error: Cannot generate encryption key. /dev/urandom is not available."
        exit 1
    fi

    echo "Creating configuration file..."
    cat >/opt/oar/config.yaml <<EOF
data_dir: /opt/oar/data
log_level: info

http:
  host: 127.0.0.1
  port: 3333

watcher:
  poll_interval: 5m

encryption_key: $ENCRYPTION_KEY
EOF
fi

# Reload systemd and enable service
echo "Enabling Oar service..."
systemctl daemon-reload
systemctl enable oar

# Start the service
echo "Starting Oar service..."
systemctl start oar

# Wait for service to be active (timeout after 30 seconds)
echo "Waiting for service to start..."
timeout 30 sh -c 'while ! systemctl is-active --quiet oar; do sleep 0.5; done'

if systemctl is-active --quiet oar; then
    echo "Oar service is running"
else
    echo "Failed to start Oar service within 30 seconds"
    echo "Check logs: journalctl -u oar -f"
    exit 1
fi

# Show completion message
echo
if [ "$UPGRADE_MODE" = true ]; then
    echo "Upgrade complete!"
    echo "Oar upgraded from $CURRENT_VERSION to $LATEST_VERSION"
    echo
    echo "Backups created in: /opt/oar/data/backups/"
else
    echo "Installation complete!"
    echo "Oar $LATEST_VERSION is now running."
fi
echo
echo "Check status with:"
echo "  sudo systemctl status oar"
echo
echo "View logs with:"
echo "  sudo journalctl -u oar -f"
echo
echo "Web interface available at: http://127.0.0.1:3333"
echo "Installation directory: /opt/oar"
echo "Data directory: /opt/oar/data"
