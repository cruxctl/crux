#!/bin/sh
set -eu

REPO="${CRUX_REPO:-github.com/cruxctl/crux}"
VERSION="${CRUX_VERSION:-latest}"
BIN_DIR="${CRUX_BIN_DIR:-$HOME/.local/bin}"
CONFIG_DIR="${CRUX_CONFIG_DIR:-$HOME/.config/crux}"
CRUXD_INSTALL_URL_DEFAULT="https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.sh"
CRUXD_INSTALL_URL="${CRUXD_INSTALL_URL:-$CRUXD_INSTALL_URL_DEFAULT}"
CRUXD_INSTALL_REF="${CRUXD_INSTALL_REF:-main}"
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

is_sha() {
  case "$1" in
    *[!0-9a-fA-F]*|"")
      return 1
      ;;
    *)
      [ "${#1}" -eq 40 ]
      ;;
  esac
}

resolve_github_ref() {
  ref="$1"
  if is_sha "$ref"; then
    printf '%s' "$ref"
    return 0
  fi

  if command -v git >/dev/null 2>&1; then
    sha="$(git ls-remote https://github.com/cruxctl/cruxd.git "refs/heads/$ref" "refs/tags/$ref" | awk 'NR == 1 { print $1 }')"
    if [ -n "$sha" ]; then
      printf '%s' "$sha"
      return 0
    fi
  fi

  curl -fsSL \
    -H 'Accept: application/vnd.github+json' \
    -H 'User-Agent: crux-installer' \
    "https://api.github.com/repos/cruxctl/cruxd/commits/$ref" |
    sed -n 's/^[[:space:]]*"sha":[[:space:]]*"\([0-9a-fA-F]\{40\}\)".*/\1/p' |
    head -n 1
}

resolve_cruxd_install_url() {
  if [ "$CRUXD_INSTALL_URL" != "$CRUXD_INSTALL_URL_DEFAULT" ]; then
    printf '%s' "$CRUXD_INSTALL_URL"
    return 0
  fi

  sha="$(resolve_github_ref "$CRUXD_INSTALL_REF")"
  [ -n "$sha" ] || fail "could not resolve cruxd installer ref: $CRUXD_INSTALL_REF"
  printf 'https://raw.githubusercontent.com/cruxctl/cruxd/%s/scripts/install-cruxd.sh' "$sha"
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
  CRUXD_INSTALL_URL="$(resolve_cruxd_install_url)"
  curl -fsSL "$CRUXD_INSTALL_URL" | CRUXD_VERSION="${CRUXD_VERSION:-$VERSION}" sh -s -- $cruxd_args
fi

tmp_bin="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_bin"
}
trap cleanup EXIT

echo "Installing crux from $REPO@$VERSION"
GOPROXY="${GOPROXY:-direct}" GOBIN="$tmp_bin" go install "$REPO/cmd/crux@$VERSION"
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
