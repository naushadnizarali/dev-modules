# Automation

- Pull requests: check `gofmt`, run tests and `go vet`, and run `govulncheck`.
- Pushes to `main`: apply `gofmt` and commit any changes, run all checks on the
  resulting commit, prepend dated commit notes to `CHANGELOG.md`, commit the
  changelog, create the next stable v1 patch tag, publish a GitHub Release,
  and request the version from `proxy.golang.org`. No release pull request is used.
  After `v1.0.0`, the next new commit is released as `v1.0.1`.
- Manual releases: run **Format and publish Go module** from `main` and choose
  `patch` or `minor`. Retrying an already tagged commit reuses its version and
  retries any incomplete release or proxy publication.
- Every Monday at 08:23 UTC: rerun checks, including the current vulnerability
  database. **Go checks** can also be run manually.

The workflows use Go from `go.mod` and the built-in `GITHUB_TOKEN`; no extra
secret or permission to create pull requests is needed. Repository rules must
allow the Actions bot to push formatting commits to `main` and create version
tags. Protected branches may require source formatting before merging instead.
Fork pull requests receive read-only checks.

The module repository must be public for publication through `proxy.golang.org`.
The workflow retries proxy requests briefly after pushing the tag and fails if
publication cannot be verified. Tags are never moved or overwritten.

Each release includes `CHANGELOG.md` in its tagged source. The GitHub Release uses
that same entry, listing commit subjects and links since the preceding tag. Older
entries are preserved. The changelog commit and tag are pushed atomically using
the built-in token, so the generated commit does not start another release run.
Repository rules must also allow the bot to push changelog commits to `main`.

Consumers install a published version with:

```sh
go get github.com/naushadnizarali/dev-modules@v1.0.1
```

GitHub Releases contain source archives because this is a library. Major releases
beyond v1 require a Go module path migration before enabling them in automation.
