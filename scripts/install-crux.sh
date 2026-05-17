#!/bin/sh
set -eu

REPO="${CRUX_REPO:-github.com/cruxctl/crux}"
VERSION="${CRUX_VERSION:-latest}"
BIN_DIR="${CRUX_BIN_DIR:-$HOME/.local/bin}"
CONFIG_DIR="${CRUX_CONFIG_DIR:-$HOME/.config/crux}"
CRUXD_INSTALL_URL="${CRUXD_INSTALL_URL:-https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.sh}"
INSTALL_CRUXD="${CRUX_INSTALL_CRUXD:-1}"
FORCE="${CRUX_FORCE:-0}"
START="${CRUXD_START:-1}"

usage() {
  cat <<'EOF'
Usage: install-crux.sh [--version VERSION] [--force] [--skip-cruxd] [--no-start]

Installs the crux CLI for the current user and, by default, installs cruxd too.
Requires Go because preview builds are installed with `go install`.
EOF
}

fail() {
  echo "install-crux: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || fail "--version requires a value"
      VERSION="$2"
      shift 2
      ;;
    --force|-f)
      FORCE=1
      shift
      ;;
    --skip-cruxd)
      INSTALL_CRUXD=0
      shift
      ;;
    --no-start)
      START=0
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

case "$(uname -s)" in
  Linux|Darwin) ;;
  *) fail "unsupported OS for this shell installer; use scripts/install-crux.ps1 on Windows" ;;
esac

need go
need curl
mkdir -p "$BIN_DIR" "$CONFIG_DIR"

if [ "$INSTALL_CRUXD" = "1" ]; then
  cruxd_args=""
  [ "$FORCE" = "1" ] && cruxd_args="$cruxd_args --force"
  [ "$START" = "0" ] && cruxd_args="$cruxd_args --no-start"
  curl -fsSL "$CRUXD_INSTALL_URL" | CRUXD_VERSION="${CRUXD_VERSION:-$VERSION}" sh -s -- $cruxd_args
fi

tmp_bin="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_bin"
}
trap cleanup EXIT

echo "Installing crux from $REPO@$VERSION"
GOBIN="$tmp_bin" go install "$REPO/cmd/crux@$VERSION"
install -m 755 "$tmp_bin/crux" "$BIN_DIR/crux"

if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
  cat >"$CONFIG_DIR/config.yaml" <<'EOF'
currentContext: local
contexts:
  local:
    serverUrl: http://127.0.0.1:7700
    namespace: default
EOF
fi

echo "crux installed at $BIN_DIR/crux"

