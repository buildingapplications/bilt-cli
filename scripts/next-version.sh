#!/usr/bin/env bash
#
# Determine the next semver tag based on conventional commits since the last tag.
#
#   feat: / feat(...):           → minor bump
#   fix: / fix(...):             → patch bump
#   BREAKING CHANGE / feat!: etc → major bump
#   anything else                → patch bump
#
# Usage: ./scripts/next-version.sh        → prints e.g. v0.2.0
#        ./scripts/next-version.sh --dry  → also prints the reason

set -euo pipefail

DRY=false
[[ "${1:-}" == "--dry" ]] && DRY=true

# Latest semver tag, or v0.0.0 if none
LATEST=$(git tag -l 'v*' --sort=-v:refname | head -1)
if [[ -z "$LATEST" ]]; then
  LATEST="v0.0.0"
  RANGE="HEAD"
else
  RANGE="${LATEST}..HEAD"
fi

# Parse version components
BASE="${LATEST#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$BASE"

# Read commit subjects since last tag
COMMITS=$(git log "$RANGE" --pretty=format:"%s" 2>/dev/null || true)

if [[ -z "$COMMITS" ]]; then
  # No new commits — bump patch anyway (allows manual release)
  PATCH=$((PATCH + 1))
  REASON="no new commits, default patch bump"
else
  BUMP="patch"
  REASON=""

  while IFS= read -r MSG; do
    # Check for breaking changes
    if echo "$MSG" | grep -qiE '^[a-z]+(\(.+\))?!:|BREAKING CHANGE'; then
      BUMP="major"
      REASON="breaking: $MSG"
      break
    fi

    # Check for features
    if echo "$MSG" | grep -qE '^feat(\(.+\))?:'; then
      if [[ "$BUMP" != "major" ]]; then
        BUMP="minor"
        REASON="feature: $MSG"
      fi
    fi

    # First fix/other commit sets the patch reason if nothing bigger found yet
    if [[ -z "$REASON" ]]; then
      REASON="$MSG"
    fi
  done <<< "$COMMITS"

  case "$BUMP" in
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
    minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
    patch) PATCH=$((PATCH + 1)) ;;
  esac
fi

NEXT="v${MAJOR}.${MINOR}.${PATCH}"

if $DRY; then
  echo "current: $LATEST"
  echo "next:    $NEXT"
  echo "reason:  $REASON"
else
  echo "$NEXT"
fi
