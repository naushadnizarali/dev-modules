# Automation

- Pull requests: check `gofmt`, run tests and `go vet`, and run `govulncheck`.
- Pushes to `main`: apply `gofmt` and commit any changes, run all checks on the
  resulting commit, then create the next `v1.x.y` patch tag and publish a GitHub
  Release with generated notes. The first release after `v1.0.0` is `v1.0.1`.
- Manual releases: run **Format and release** from `main`, choosing `patch` or
  `minor`. An already tagged commit reuses its tag, allowing failed publication
  to be retried without incrementing the version again.
- Manually pushed stable `v1.x.y` tags: run checks and publish a GitHub Release.
- Every Monday at 08:23 UTC: rerun checks, including the current vulnerability
  database. **Go checks** can also be run manually.

The workflows use Go from `go.mod` and the built-in `GITHUB_TOKEN`; no extra
secret is needed. Repository rules must allow the Actions bot to push formatting
commits to `main` and create version tags. Protected branches may require source
formatting before merging instead. Fork pull requests receive read-only checks.

This is a library, so releases contain GitHub's source archives rather than
binaries. Consumers can use `go get github.com/naushadnizarali/dev-modules@v1.x.y`.
Major releases beyond v1 require a Go module path migration before enabling them.
