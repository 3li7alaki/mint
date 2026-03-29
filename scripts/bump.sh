#!/usr/bin/env bash
# Usage: ./scripts/bump.sh [major|minor|patch] or ./scripts/bump.sh 1.2.3
#
# Updates version in all locations:
#   - .claude-plugin/plugin.json
#   - .claude-plugin/marketplace.json
#   - package.json
#   - README.md (tagline)
#
# Does NOT commit — you decide when to commit.
set -euo pipefail

PLUGIN_JSON=".claude-plugin/plugin.json"
MARKETPLACE_JSON=".claude-plugin/marketplace.json"
PACKAGE_JSON="package.json"

# Get current version
CURRENT=$(grep -o '"version": "[^"]*"' "$PLUGIN_JSON" | head -1 | grep -o '[0-9][0-9.]*')
if [ -z "$CURRENT" ]; then
  echo "Error: Could not read current version from $PLUGIN_JSON"
  exit 1
fi

IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT"

# Determine new version
case "${1:-patch}" in
  major) NEW="$((MAJOR + 1)).0.0" ;;
  minor) NEW="${MAJOR}.$((MINOR + 1)).0" ;;
  patch) NEW="${MAJOR}.${MINOR}.$((PATCH + 1))" ;;
  [0-9]*) NEW="$1" ;;
  *)
    echo "Usage: $0 [major|minor|patch|X.Y.Z]"
    echo "Current version: $CURRENT"
    exit 1
    ;;
esac

echo "$CURRENT → $NEW"

# Update all version locations
for file in "$PLUGIN_JSON" "$MARKETPLACE_JSON" "$PACKAGE_JSON"; do
  if [ -f "$file" ]; then
    sed -i "s/\"version\": \"$CURRENT\"/\"version\": \"$NEW\"/g" "$file"
    echo "  Updated $file"
  fi
done

echo ""
echo "Version bumped to $NEW."
echo "Commit and merge to main — GitHub Actions will auto-tag and create the release."
