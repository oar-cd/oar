#!/bin/bash
set -e

# Get latest tag from git
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")
echo "Latest tag: $LATEST_TAG"

# Extract version numbers
IFS='.' read -r MAJOR MINOR PATCH <<<"$LATEST_TAG"

# Default to patch release
RELEASE_TYPE=${1:-patch}

case $RELEASE_TYPE in
major)
    NEW_VERSION="$((MAJOR + 1)).0.0"
    ;;
minor)
    NEW_VERSION="$MAJOR.$((MINOR + 1)).0"
    ;;
patch)
    NEW_VERSION="$MAJOR.$MINOR.$((PATCH + 1))"
    ;;
*)
    # Custom version provided
    if [[ $1 =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        NEW_VERSION="$1"
    else
        echo "Usage: $0 [major|minor|patch|1.2.3]"
        echo "Examples:"
        echo "  $0 patch    # 1.0.0 -> 1.0.1"
        echo "  $0 minor    # 1.0.0 -> 1.1.0"
        echo "  $0 major    # 1.0.0 -> 2.0.0"
        echo "  $0 2.1.0   # specific version"
        exit 1
    fi
    ;;
esac

echo "New version: $NEW_VERSION"

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo "Error: You have uncommitted changes. Please commit or stash them first."
    exit 1
fi

read -p "Create release $NEW_VERSION? [y/N] " -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Updating compose.yaml..."
    sed -i "s|image: ghcr.io/ch00k/oar:.*|image: ghcr.io/ch00k/oar:$NEW_VERSION|" compose.yaml

    echo "Committing changes..."
    git add compose.yaml
    git commit -m "Release $NEW_VERSION"

    echo "Creating annotated tag..."
    git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

    echo "Pushing to origin..."
    git push origin main "$NEW_VERSION"

    echo "Release $NEW_VERSION created successfully!"
    echo "Docker build will start automatically via GitHub Actions."
    echo "Monitor at: https://github.com/$(gh repo view --json owner,name -q '.owner.login + "/" + .name')/actions"
else
    echo "Release cancelled"
fi
