#!/bin/sh
set -e

# Bilt CLI installer
# Usage:
#   curl -fsSL https://bilt.me/install.sh | sh
#   curl -fsSL https://bilt.me/install.sh | sh -s -- build --project <id> --token <token>

GITHUB_REPO="buildingapplications/bilt-cli"
BINARY_NAME="bilt"
INSTALL_DIR="/usr/local/bin"

# Colors (if terminal supports it)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

info()  { printf "${GREEN}==>${NC} ${BOLD}%s${NC}\n" "$1"; }
warn()  { printf "${YELLOW}==> Warning:${NC} %s\n" "$1"; }
error() { printf "${RED}==> Error:${NC} %s\n" "$1" >&2; }

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux)  echo "linux" ;;
    *)      error "Unsupported OS: $(uname -s)"; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)             error "Unsupported architecture: $(uname -m)"; exit 1 ;;
  esac
}

get_latest_version() {
  curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"v?([^"]+)".*/\1/'
}

install_via_brew() {
  if command -v brew >/dev/null 2>&1; then
    info "Installing via Homebrew..."
    if brew install buildingapplications/tap/bilt 2>/dev/null; then
      return 0
    fi
    warn "Homebrew install failed, falling back to direct download..."
    return 1
  fi
  return 1
}

install_via_download() {
  local os="$1"
  local arch="$2"

  info "Fetching latest version..."
  local version
  version=$(get_latest_version)
  if [ -z "$version" ]; then
    error "Could not determine latest version"
    exit 1
  fi
  info "Latest version: v${version}"

  local archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
  if [ "$os" = "windows" ]; then
    archive_name="${BINARY_NAME}_${os}_${arch}.zip"
  fi

  local download_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/${archive_name}"

  info "Downloading ${archive_name}..."
  local tmp_dir
  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT

  curl -fsSL "$download_url" -o "${tmp_dir}/${archive_name}"

  info "Extracting..."
  tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"

  # Install binary
  if [ -w "$INSTALL_DIR" ]; then
    mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  else
    info "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  fi

  chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
  info "Installed ${BINARY_NAME} v${version} to ${INSTALL_DIR}/${BINARY_NAME}"
}

main() {
  local os arch

  os=$(detect_os)
  arch=$(detect_arch)

  # Check if bilt is already installed
  if command -v bilt >/dev/null 2>&1; then
    info "bilt is already installed: $(bilt --version 2>/dev/null || echo 'unknown version')"
  else
    # Try brew first on macOS, fall back to direct download
    if [ "$os" = "darwin" ]; then
      install_via_brew || install_via_download "$os" "$arch"
    else
      install_via_download "$os" "$arch"
    fi
  fi

  # Verify installation
  if ! command -v bilt >/dev/null 2>&1; then
    error "Installation failed — 'bilt' not found in PATH"
    error "You may need to add ${INSTALL_DIR} to your PATH"
    exit 1
  fi

  printf "\n"
  info "bilt is ready!"

  # If arguments were passed (e.g., build --project <id> --token <token>), run them
  if [ $# -gt 0 ]; then
    printf "\n"
    info "Running: bilt $*"
    printf "\n"
    exec bilt "$@"
  fi
}

main "$@"
