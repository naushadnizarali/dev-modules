#!/usr/bin/env bash
set -euo pipefail

if [[ "$GITHUB_REF" != refs/heads/main ]]; then
  echo 'Releases must run from main.' >&2
  exit 1
fi

latest=$(git tag --list 'v1.*.*' --sort=-version:refname | awk '/^v1\.[0-9]+\.[0-9]+$/ { if (!found++) print }')
tag=''
notes_file=$(mktemp)
changelog_file=$(mktemp)
trap 'rm -f "$notes_file" "$changelog_file"' EXIT
if [[ -n "$latest" ]]; then
  if [[ $(git rev-parse "$latest^{commit}") == "$(git rev-parse HEAD)" ]]; then
    # Reuse the tag when retrying a failed release or proxy request.
    tag=$latest
  elif ! git merge-base --is-ancestor "$latest" HEAD; then
    echo 'Refusing to release a commit behind or divergent from the latest tag.' >&2
    exit 1
  fi
fi

if [[ -z "$tag" ]]; then
  if [[ -z "$latest" ]]; then
    tag=v1.0.0
  else
    IFS=. read -r major minor patch <<< "${latest#v}"
    case "${BUMP:-patch}" in
      patch) tag="v1.$((10#$minor)).$((10#$patch + 1))" ;;
      minor) tag="v1.$((10#$minor + 1)).0" ;;
      *) echo 'Expected patch or minor.' >&2; exit 1 ;;
    esac
  fi
  git config user.name 'github-actions[bot]'
  git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
  if [[ -n "$latest" ]]; then
    range="$latest..HEAD"
  else
    range=HEAD
  fi
  git log --reverse --format="- %s ([%h](https://github.com/${GITHUB_REPOSITORY}/commit/%H))" \
    "$range" > "$notes_file"
  {
    printf '# Changelog\n\n## %s (%s)\n\n' "$tag" "$(date -u +%F)"
    cat "$notes_file"
    if [[ -f CHANGELOG.md ]]; then
      printf '\n'
      # Preserve older entries, dropping only the existing document heading.
      awk 'NR == 1 && $0 == "# Changelog" { next } !started && /^$/ { next } { started=1; print }' CHANGELOG.md
    fi
  } > "$changelog_file"
  cp "$changelog_file" CHANGELOG.md
  git add -- CHANGELOG.md
  git commit -m "docs: update changelog for $tag"
  git tag -a "$tag" -m "Release $tag"
  # Publish the changelog commit and its tag together; never overwrite a tag.
  git push --atomic origin HEAD:main "refs/tags/$tag"
fi

# Read the persisted entry on retries so release notes always match the changelog.
if [[ ! -s "$notes_file" ]]; then
  awk -v tag="$tag" '
    /^## / { if (found) exit; if ($2 == tag) found=1; next }
    found { print }
  ' CHANGELOG.md > "$notes_file"
  if [[ ! -s "$notes_file" ]]; then
    echo "Missing changelog entry for $tag." >&2
    exit 1
  fi
fi

# Keep publication in this run: bot-created tags do not trigger push workflows.
if ! gh release view "$tag" >/dev/null 2>&1; then
  gh release create "$tag" --verify-tag --notes-file "$notes_file" --title "$tag"
fi

module=$(go list -m)
# Public Go modules are published by pushing a tag and requesting it from a proxy.
# Retry to allow GitHub and the module proxy to discover the newly pushed tag.
for delay in 0 5 10 20 40; do
  if ((delay)); then sleep "$delay"; fi
  if GOPROXY=https://proxy.golang.org GOPRIVATE= GONOPROXY= \
      go list -m "$module@$tag"; then
    printf 'Published %s@%s to the Go module proxy.\n' "$module" "$tag"
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
      printf 'Published `%s@%s`.\n\nInstall with `go get %s@%s`.\n' \
        "$module" "$tag" "$module" "$tag" >> "$GITHUB_STEP_SUMMARY"
    fi
    exit 0
  fi
done

echo "Proxy publication failed for $module@$tag. Rerun this workflow to retry the same tag." >&2
exit 1
