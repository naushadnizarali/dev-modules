#!/usr/bin/env bash
set -euo pipefail

if [[ "$GITHUB_REF" == refs/tags/* ]]; then
  tag=${GITHUB_REF#refs/tags/}
  if [[ ! "$tag" =~ ^v1\.[0-9]+\.[0-9]+$ ]]; then
    echo 'Expected a stable v1.x.y tag.' >&2
    exit 1
  fi
else
  # Keep releases in v1: Go modules need a /v2 module path for major version 2.
  latest=$(git tag --list 'v1.*.*' --sort=-version:refname | awk '/^v1\.[0-9]+\.[0-9]+$/ { print; exit }')
  latest=${latest:-v1.0.0}
  if git show-ref --verify --quiet "refs/tags/$latest"; then
    # Retry a failed publication without creating a second version for this commit.
    if [[ $(git rev-parse "$latest^{commit}") == "$(git rev-parse HEAD)" ]]; then
      tag=$latest
    elif ! git merge-base --is-ancestor "$latest" HEAD; then
      echo 'Refusing to release a commit behind or divergent from the latest tag.' >&2
      exit 1
    fi
  fi
  if [[ -z "${tag:-}" ]]; then
    IFS=. read -r major minor patch <<< "${latest#v}"
    case "${BUMP:-patch}" in
      patch) tag="v1.$((10#$minor)).$((10#$patch + 1))" ;;
      minor) tag="v1.$((10#$minor + 1)).0" ;;
      *) echo 'Expected patch or minor.' >&2; exit 1 ;;
    esac
    git config user.name 'github-actions[bot]'
    git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
    git tag -a "$tag" -m "Release $tag"
    git push origin "refs/tags/$tag"
  fi
fi

# Publish here: tags pushed with GITHUB_TOKEN do not trigger another push workflow.
if ! gh release view "$tag" >/dev/null 2>&1; then
  gh release create "$tag" --verify-tag --generate-notes --title "$tag"
fi
