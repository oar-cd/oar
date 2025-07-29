#!/bin/bash
set -e

echo "Installing Oar..."

# Create installation directory
OAR_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/oar"
mkdir -p "$OAR_DIR"

# Download compose.yaml
echo "Downloading Docker Compose configuration..."
curl -sSL https://raw.githubusercontent.com/ch00k/oar/main/compose.yaml -o "$OAR_DIR/compose.yaml"

# Start Oar
echo "Starting Oar with Docker Compose..."
cd "$OAR_DIR"
docker compose up -d

# Wait for Oar to be ready
echo "Waiting for Oar to start..."
until curl -sSf http://localhost:8080/health >/dev/null 2>&1; do
    sleep 2
done

# Bootstrap Oar as a self-managed project
echo "Bootstrapping Oar as a self-managed project..."
curl -X POST http://localhost:8080/projects/create \
    -d "name=oar" \
    -d "git_url=https://github.com/ch00k/oar.git" \
    -d "compose_files=compose.yaml"

echo ""
echo "Oar installed successfully!"
echo "Access Oar at: http://localhost:8080"
echo "Installation directory: $OAR_DIR"
echo ""
echo "To manage the installation:"
echo "  cd $OAR_DIR"
echo "  docker compose logs    # View logs"
echo "  docker compose stop    # Stop Oar"
echo "  docker compose up -d   # Start Oar"
