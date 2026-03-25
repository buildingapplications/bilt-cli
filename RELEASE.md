# Release Process

Releases use [goreleaser](https://goreleaser.com) and are driven by git tags.

## Steps

### 1. Pre-release

Run all checks, create and push a signed tag:

```sh
task pre-release TAG=v0.1.0
```

This will:
- Run lint, vet, and tests
- Verify you're on a clean `main` branch, up to date with origin
- Create an annotated tag and push it

### 2. Release

Build cross-platform binaries and publish the GitHub release:

```sh
task release TAG=v0.1.0
```

This runs `goreleaser release`, which:
- Builds for darwin/linux/windows (amd64 + arm64)
- Creates a GitHub release with changelog
- Updates the Homebrew tap (`bilt-dev/homebrew-tap`)

### Dry run

Test the release build locally without publishing:

```sh
task release:dry
```

## Versioning

We use [semver](https://semver.org). Bump accordingly:
- `v0.x.y` — pre-1.0, minor = breaking, patch = fixes/features
- `v1.0.0+` — major = breaking, minor = features, patch = fixes
