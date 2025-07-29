#!/bin/bash
set -e

# Get latest tag from git
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "Latest tag: $LATEST_TAG"

# Extract version numbers (remove 'v' prefix if present)
VERSION=${LATEST_TAG#v}
IFS='.' read -r MAJOR MINOR PATCH <<<"$VERSION"

# Default to patch release
RELEASE_TYPE=${1:-patch}

case $RELEASE_TYPE in
major)
    NEW_VERSION="v$((MAJOR + 1)).0.0"
    ;;
minor)
    NEW_VERSION="v$MAJOR.$((MINOR + 1)).0"
    ;;
patch)
    NEW_VERSION="v$MAJOR.$MINOR.$((PATCH + 1))"
    ;;
*)
    # Custom version provided
    if [[ $1 =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        NEW_VERSION="$1"
        # Add 'v' prefix if not present
        [[ $NEW_VERSION =~ ^v ]] || NEW_VERSION="v$NEW_VERSION"
    else
        echo "Usage: $0 [major|minor|patch|v1.2.3]"
        echo "Examples:"
        echo "  $0 patch    # v1.0.0 -> v1.0.1"
        echo "  $0 minor    # v1.0.0 -> v1.1.0"
        echo "  $0 major    # v1.0.0 -> v2.0.0"
        echo "  $0 v2.1.0   # specific version"
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
    # Remove 'v' prefix for compose.yaml image tag
    IMAGE_VERSION=${NEW_VERSION#v}
    sed -i "s|image: ghcr.io/ch00k/oar:.*|image: ghcr.io/ch00k/oar:$IMAGE_VERSION|" compose.yaml

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
