# Release Strategy

`crux-console` uses semver tags (`vX.Y.Z`) for releases.

## Changelog

- `CHANGELOG.md` is the human-curated release history.
- Keep new work under `Unreleased`.
- Before tagging, move relevant entries from `Unreleased` into a dated `vX.Y.Z` section.

## GitHub Releases

- Tags matching `v*.*.*` run `.github/workflows/release.yml`.
- GitHub Release notes are generated from merged changes using `.github/release.yml`.
- Release notes must call out view, workflow, API, RBAC, and redaction changes.

## Container Images

- `main` pushes `ghcr.io/cruxctl/crux-console:main` and `:<commit-sha>`.
- A release tag pushes `ghcr.io/cruxctl/crux-console:vX.Y.Z` and `:latest`.
- The current image is a scaffold image until console implementation lands.
