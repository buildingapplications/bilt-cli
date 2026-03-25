# bilt CLI

Build your AI-generated React Native app and install it on your iPhone.

## Install

```sh
brew install bilt-dev/tap/bilt
```

Or download from [Releases](https://github.com/buildingapplications/bilt-cli/releases).

## Usage

```sh
bilt auth login        # authenticate with your Bilt account
bilt build             # build and install on your device
bilt doctor            # check prerequisites
bilt devices           # list connected iOS devices
bilt projects list     # list your projects
```

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
