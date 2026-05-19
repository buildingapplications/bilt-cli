#!/bin/sh
set -e

# Bilt CLI installer
# Usage:
#   curl -fsSL https://bilt.me/install.sh | sh
#   curl -fsSL https://bilt.me/install.sh | sh -s -- build <code>

GITHUB_REPO="buildingapplications/bilt-cli"
BINARY_NAME="bilt"
INSTALL_DIR="/usr/local/bin"
DEFAULT_BASE_URL="https://app.bilt.me"

# Colors (if terminal supports it)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

info()  { printf "${GREEN}==>${NC} ${BOLD}%s${NC}\n" "$1"; }
warn()  { printf "${YELLOW}==> Warning:${NC} %s\n" "$1"; }
error() { printf "${RED}==> Error:${NC} %s\n" "$1" >&2; }

normalize_base_url() {
  printf "%s" "$1" | sed 's#/*$##'
}

requested_base_url() {
  if [ -n "${BILT_BASE_URL:-}" ]; then
    normalize_base_url "$BILT_BASE_URL"
    return 0
  fi
  printf "%s" "$DEFAULT_BASE_URL"
}

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

# Find bilt in PATH or common install locations
find_bilt() {
  if command -v bilt >/dev/null 2>&1; then
    command -v bilt
    return 0
  fi
  for dir in "$HOME/go/bin" "/usr/local/bin" "/opt/homebrew/bin" "$HOME/.local/bin"; do
    if [ -x "$dir/bilt" ]; then
      echo "$dir/bilt"
      return 0
    fi
  done
  return 1
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
    error "Install manually: go install github.com/bilt-dev/bilt-cli@latest"
    exit 1
  fi
  info "Latest version: v${version}"

  local archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
  local download_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}/${archive_name}"

  info "Downloading ${archive_name}..."
  local tmp_dir
  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT

  curl -fsSL "$download_url" -o "${tmp_dir}/${archive_name}"

  info "Extracting..."
  tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"

  if [ -w "$INSTALL_DIR" ]; then
    mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  else
    info "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  fi

  chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
  info "Installed ${BINARY_NAME} v${version} to ${INSTALL_DIR}/${BINARY_NAME}"
}

build_for_base_url() {
  local base_url="$1"
  local cache_dir safe_name bilt_path tmp_dir source_dir

  if ! command -v go >/dev/null 2>&1; then
    error "This build code was created on ${base_url}, but the installed bilt CLI targets ${DEFAULT_BASE_URL}."
    error "Install Go and run the command again, or open Build on Device from ${DEFAULT_BASE_URL}."
    exit 1
  fi

  cache_dir="${XDG_CACHE_HOME:-$HOME/.cache}/bilt"
  safe_name=$(printf "%s" "$base_url" | sed 's#[^A-Za-z0-9._-]#_#g')
  bilt_path="${cache_dir}/${BINARY_NAME}-${safe_name}"
  source_dir=$(find_local_source) || source_dir=""

  if [ -z "$source_dir" ] && ! command -v git >/dev/null 2>&1; then
    error "Git is required to prepare bilt for ${base_url}."
    exit 1
  fi

  if [ -z "$source_dir" ] && [ -x "$bilt_path" ]; then
    echo "$bilt_path"
    return 0
  fi

  mkdir -p "$cache_dir"

  if [ -n "$source_dir" ]; then
    info "Preparing local bilt from ${source_dir} for ${base_url}..." >&2
    (
      cd "$source_dir"
      go build -ldflags "-s -w -X main.baseURL=${base_url}" -o "$bilt_path" . >&2
    )
  else
    tmp_dir=$(mktemp -d)
    info "Preparing bilt for ${base_url}..." >&2
    git clone --depth 1 "https://github.com/${GITHUB_REPO}.git" "$tmp_dir" >/dev/null
    (
      cd "$tmp_dir"
      go build -ldflags "-s -w -X main.baseURL=${base_url}" -o "$bilt_path" . >&2
    )
    rm -rf "$tmp_dir"
  fi

  chmod +x "$bilt_path"

  echo "$bilt_path"
}

find_local_source() {
  local dir

  if [ -n "${BILT_CLI_SOURCE:-}" ]; then
    if [ -f "${BILT_CLI_SOURCE}/go.mod" ]; then
      echo "$BILT_CLI_SOURCE"
      return 0
    fi
    error "BILT_CLI_SOURCE is set but does not point to a Go module: ${BILT_CLI_SOURCE}"
    exit 1
  fi

  for dir in "$PWD/../bilt-cli" "$PWD/bilt-cli" "$HOME/bilt/bilt-cli"; do
    if [ -f "${dir}/go.mod" ]; then
      echo "$dir"
      return 0
    fi
  done

  return 1
}

main() {
  local os arch bilt_path base_url

  os=$(detect_os)
  arch=$(detect_arch)
  base_url=$(requested_base_url)

  if [ "$base_url" != "$DEFAULT_BASE_URL" ] && [ $# -gt 0 ]; then
    bilt_path=$(build_for_base_url "$base_url")
    info "bilt ready for ${base_url}: $bilt_path"
  else
    bilt_path=$(find_bilt) || bilt_path=""

    if [ -n "$bilt_path" ]; then
      info "bilt found: $bilt_path"
    else
      # Try brew first on macOS, fall back to direct download
      if [ "$os" = "darwin" ]; then
        install_via_brew || install_via_download "$os" "$arch"
      else
        install_via_download "$os" "$arch"
      fi
      bilt_path=$(find_bilt) || bilt_path=""
    fi
  fi

  if [ -z "$bilt_path" ]; then
    error "Installation failed — 'bilt' not found"
    error "You may need to add ${INSTALL_DIR} to your PATH"
    exit 1
  fi

  # Run bilt with any arguments passed to the script
  if [ $# -gt 0 ]; then
    printf "\n"
    exec "$bilt_path" "$@"
  fi

  printf "\n"
  info "bilt is ready! Run 'bilt build <code>' to build your app."
}

main "$@"
