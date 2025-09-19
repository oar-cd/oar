#!/usr/bin/env bash
set -e

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo "Warning: Running as root. This script will use sudo commands for system operations."
else
    echo "This script requires sudo privileges for system operations."

    # Test sudo availability
    echo "Testing sudo access..."
    if ! sudo -v; then
        echo "Error: Cannot obtain sudo privileges. Please ensure you have sudo access."
        exit 1
    fi
    echo "Sudo access confirmed."
fi

echo

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
echo
if [ "$UPGRADE_MODE" = true ]; then
    echo "Existing Oar installation detected (version: $CURRENT_VERSION)"

    # Check if versions are the same
    if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ]; then
        echo
        echo "You already have the latest version ($CURRENT_VERSION) installed."
        echo "No upgrade needed. Exiting."
        exit 0
    else
        echo "Upgrading Oar to version: $LATEST_VERSION"
    fi
else
    echo "Installing Oar version: $LATEST_VERSION"
fi

echo

# Get download URLs for all files
BINARY_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*oar-linux-amd64"' | cut -d'"' -f4)
SERVICE_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*oar\.service"' | cut -d'"' -f4)
ASSETS_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*web-assets\.tar\.gz"' | cut -d'"' -f4)

if [ -z "$BINARY_URL" ] || [ -z "$SERVICE_URL" ] || [ -z "$ASSETS_URL" ]; then
    echo "Error: Could not find required files (oar-linux-amd64, oar.service, web-assets.tar.gz) in latest release"
    exit 1
fi

# Stop service during upgrade
if [ "$UPGRADE_MODE" = true ]; then
    echo "Stopping Oar service for upgrade..."
    sudo systemctl stop oar

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
    sudo mkdir -p /opt/oar/data/backups
else
    echo "Creating /opt/oar directory structure..."
    sudo mkdir -p /opt/oar/{bin,data/backups}
fi

# Create database backup during upgrade
if [ "$UPGRADE_MODE" = true ] && [ -f "/opt/oar/data/oar.db" ]; then
    BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_FILE="/opt/oar/data/backups/oar.db.${CURRENT_VERSION}.${BACKUP_TIMESTAMP}"

    echo "Creating database backup..."
    sudo cp /opt/oar/data/oar.db "$BACKUP_FILE"
    echo "Database backed up to: $BACKUP_FILE"
fi

# Backup current binary during upgrade
if [ "$UPGRADE_MODE" = true ]; then
    BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BINARY_BACKUP="/opt/oar/data/backups/oar.${CURRENT_VERSION}.${BACKUP_TIMESTAMP}"
    sudo cp /opt/oar/bin/oar "$BINARY_BACKUP"
    echo "Current binary backed up to: $BINARY_BACKUP"
fi

# Download and install binary
echo "Installing Oar command line tool..."
sudo curl -sSL "$BINARY_URL" -o /opt/oar/bin/oar
sudo chmod +x /opt/oar/bin/oar

# Download and extract web assets
echo "Installing Oar web assets..."
curl -sSL "$ASSETS_URL" | sudo tar -xzf - -C /opt/oar

# Download systemd service (skip if upgrading and file already exists)
if [ "$UPGRADE_MODE" = false ] || [ ! -f "/opt/oar/oar.service" ]; then
    echo "Downloading Oar systemd service file..."
    sudo curl -sSL "$SERVICE_URL" -o /opt/oar/oar.service
fi

# Install systemd service
echo "Installing Oar systemd service..."
sudo ln -sf /opt/oar/oar.service /etc/systemd/system/oar.service

# Create configuration file (only for fresh installs)
if [ "$UPGRADE_MODE" = false ]; then
    # Generate encryption key
    echo "Generating Oar encryption key..."
    if [ -r /dev/urandom ]; then
        ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64 | tr -d '\n')
    else
        echo "Error: Cannot generate encryption key. /dev/urandom is not available."
        exit 1
    fi

    echo "Creating Oar configuration file..."
    sudo tee /opt/oar/config.yaml >/dev/null <<EOF
data_dir: /opt/oar/data
log_level: info

http:
  host: 127.0.0.1
  port: 4777

watcher:
  enabled: true
  poll_interval: 5m

encryption_key: $ENCRYPTION_KEY
EOF
fi

# Reload systemd and enable service
echo "Enabling Oar service..."
sudo systemctl daemon-reload
sudo systemctl enable oar

# Start the service
echo "Starting Oar service..."
sudo systemctl start oar

# Wait for service to be active (timeout after 30 seconds)
echo "Waiting for Oar service to start..."
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
    echo "Oar upgrade complete!"
    echo "Oar upgraded from $CURRENT_VERSION to $LATEST_VERSION"
    echo
    echo "Backups created in: /opt/oar/data/backups/"
else
    echo "Oar installation complete!"
    echo "Oar $LATEST_VERSION is now running."
fi
echo
echo "Check status with:"
echo "  sudo systemctl status oar"
echo
echo "View logs with:"
echo "  sudo journalctl -u oar -f"
echo
echo -e "Web interface:\t\thttp://127.0.0.1:4777"
echo ""
echo -e "Installation directory:\t/opt/oar"
echo -e "Command line tool:\t/opt/oar/bin/oar"
echo -e "Configuration file:\t/opt/oar/config.yaml"
echo -e "Data directory:\t\t/opt/oar/data"
echo ""
echo "Add oar executable to your PATH for easier access:"
# shellcheck disable=SC2016
echo '  export PATH=$PATH:/opt/oar/bin'
