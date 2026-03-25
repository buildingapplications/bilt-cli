# Release Process

Versions are auto-bumped from conventional commit messages since the last tag.

| Commit prefix | Bump |
|---|---|
| `feat:` / `feat(scope):` | minor |
| `fix:` / anything else | patch |
| `feat!:` / `BREAKING CHANGE` | major |

## Quick release

```sh
task release
```

This will: run all checks, compute the next version, tag, push, and publish via goreleaser.

## Other commands

```sh
task version       # preview what the next version would be
task release:dry   # test goreleaser build locally, no publish
task release:tag   # tag and push only (no goreleaser)
```

## Versioning

We use [semver](https://semver.org). Pre-1.0: minor = breaking, patch = fixes/features.
