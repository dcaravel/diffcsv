#!/usr/bin/env bash
set -euo pipefail

latest=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")
echo "Current version: ${latest}"

if [ -n "${1:-}" ]; then
    next="$1"
else
    read -rp "Next version (e.g. v0.2.0): " next
fi

if [[ ! "${next}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: version must match vX.Y.Z" >&2
    exit 1
fi

if git rev-parse "${next}" >/dev/null 2>&1; then
    echo "Error: tag ${next} already exists" >&2
    exit 1
fi

echo ""
echo "This will:"
echo "  1. Tag the current commit as ${next}"
echo "  2. Push the tag to origin"
echo "  3. GitHub Actions will build and create the release"
echo ""
read -rp "Continue? [y/N] " confirm
if [[ "${confirm}" != [yY] ]]; then
    echo "Aborted."
    exit 0
fi

git tag "${next}"
git push origin "${next}"

echo ""
echo "Tag ${next} pushed. GitHub Actions will create the release."
echo "Watch progress: gh run watch"
