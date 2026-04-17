#!/usr/bin/env bash
set -euo pipefail

ADA_INSTALL_REPO="${ADA_INSTALL_REPO:-yesabhishek/ada}"
ADA_INSTALL_VERSION="${ADA_INSTALL_VERSION:-latest}"
ADA_INSTALL_BASE_URL="${ADA_INSTALL_BASE_URL:-}"

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *) echo "Unsupported operating system: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

resolve_version() {
  if [ "${ADA_INSTALL_VERSION}" != "latest" ]; then
    printf '%s\n' "${ADA_INSTALL_VERSION}"
    return
  fi
  if [ -n "${ADA_INSTALL_BASE_URL}" ]; then
    echo "ADA_INSTALL_VERSION must be set when ADA_INSTALL_BASE_URL is used" >&2
    exit 1
  fi
  curl -fsSL "https://api.github.com/repos/${ADA_INSTALL_REPO}/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1
}

choose_bin_dir() {
  if [ -n "${ADA_INSTALL_BIN_DIR:-}" ]; then
    printf '%s\n' "${ADA_INSTALL_BIN_DIR}"
    return
  fi
  if [ -w /usr/local/bin ]; then
    printf '%s\n' "/usr/local/bin"
    return
  fi
  printf '%s\n' "${HOME}/.local/bin"
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
VERSION="$(resolve_version)"
BIN_DIR="$(choose_bin_dir)"
ASSET="ada_${VERSION}_${OS}_${ARCH}.tar.gz"

if [ -n "${ADA_INSTALL_BASE_URL}" ]; then
  DOWNLOAD_URL="${ADA_INSTALL_BASE_URL}/${ASSET}"
else
  DOWNLOAD_URL="https://github.com/${ADA_INSTALL_REPO}/releases/download/${VERSION}/${ASSET}"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${BIN_DIR}"

echo "Installing Ada ${VERSION} for ${OS}/${ARCH}"
echo "Downloading ${DOWNLOAD_URL}"

curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${ASSET}"
tar -xzf "${TMP_DIR}/${ASSET}" -C "${TMP_DIR}"
install "${TMP_DIR}/ada" "${BIN_DIR}/ada"

echo "Installed to ${BIN_DIR}/ada"
"${BIN_DIR}/ada" version

case ":${PATH}:" in
  *":${BIN_DIR}:"*) ;;
  *)
    echo "Note: ${BIN_DIR} is not currently on PATH."
    ;;
esac
