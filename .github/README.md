# Automation

- Pull requests: check `gofmt`, run tests and `go vet`, and run `govulncheck`.
- Pushes to `main`: apply `gofmt` and commit any changes, run all checks on the
  resulting commit, then run Release Please using its Go release strategy.
- Release Please maintains a release pull request with a version and changelog
  based on Conventional Commits (`fix:` for patches, `feat:` for minor versions,
  and `feat!:` or `BREAKING CHANGE:` for major versions). Merge the release PR
  to create the version tag and publish the GitHub Release on the next run.
- Every Monday at 08:23 UTC: rerun checks, including the current vulnerability
  database. **Go checks** can also be run manually.

The workflows use Go from `go.mod` and the built-in `GITHUB_TOKEN`; no extra
secret is needed. Enable **Allow GitHub Actions to create and approve pull
requests** in Settings > Actions > General. Repository rules must allow the
Actions bot to push formatting commits to `main` and create version tags.
Protected branches may require source formatting before merging instead.
Fork pull requests receive read-only checks.

Release Please PRs created with the built-in token may require manual approval
or a manually triggered check run. If automatic PR checks are required, configure
Release Please with a GitHub App token or a suitable personal access token.
Checks on `main` still run before Release Please publishes a release.

This is a library, so releases contain GitHub's source archives rather than
binaries. Consumers can use `go get github.com/naushadnizarali/dev-modules@v1.x.y`.
Major releases beyond v1 require a Go module path migration before merging the
release PR.
