#!/usr/bin/env bash
# Build locally by default; publishing requires an explicit destination.
set -euo pipefail
cd "$(dirname "$0")"
version=$(cat .release-version)
PLATFORMS=${PLATFORMS:-linux/amd64}
IMAGE=${IMAGE:-trojan-panel:$version}
mode=${1:---load}
case "$mode" in
  --load) [[ "$PLATFORMS" != *,* ]] || { echo '--load accepts one platform' >&2; exit 2; } ;;
  --push) [[ -n "${PUBLISH_IMAGE:-}" ]] || { echo 'Set PUBLISH_IMAGE explicitly to publish your own image.' >&2; exit 2; }; IMAGE=$PUBLISH_IMAGE ;;
  *) echo 'Usage: build.sh [--load|--push]' >&2; exit 2 ;;
esac
echo 'Build target binaries into build/ first, or use the GitHub release workflow.'
docker buildx build --platform "$PLATFORMS" -t "$IMAGE" "$mode" .
