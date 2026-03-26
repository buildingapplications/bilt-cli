# Bilt.me CLI

Build your Bilt app and install it on your iPhone.

## Install

```sh
brew install bilt-dev/tap/bilt
```

Or download from [Releases](https://github.com/buildingapplications/bilt-cli/releases).

## Usage

1. Get a build code from [bilt.me](https://bilt.me)
2. Run:

```sh
bilt build <code>      # build and install on your device
```

Prerequisites (Xcode, Node.js, CocoaPods, etc.) are checked automatically before the build starts.

## Requirements

- macOS with Xcode installed
- A connected iPhone (USB)
- A free or paid Apple Developer account

## Development

Requires Go 1.26+, [Task](https://taskfile.dev), and [golangci-lint](https://golangci-lint.run).

```sh
npm install          # set up husky pre-commit hooks
task build           # build for current platform
task check           # lint + vet + test
task --list-all      # see all tasks
```

See [RELEASE.md](RELEASE.md) for the release process.
