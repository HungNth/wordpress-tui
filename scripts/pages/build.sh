#!/bin/sh
set -eu

MANIFEST_PATH="manifest.json"
SITE_DIR="website"
OUT_DIR="_site"
EXPECTED_TAG=""

usage() {
    cat <<'EOF' >&2
Usage: build.sh [--out DIR] [--tag TAG] [--manifest PATH] [--website DIR]
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        --out)
            [ $# -ge 2 ] || { usage; exit 1; }
            OUT_DIR="$2"
            shift 2
            ;;
        --tag)
            [ $# -ge 2 ] || { usage; exit 1; }
            EXPECTED_TAG="$2"
            shift 2
            ;;
        --manifest)
            [ $# -ge 2 ] || { usage; exit 1; }
            MANIFEST_PATH="$2"
            shift 2
            ;;
        --website|--site)
            [ $# -ge 2 ] || { usage; exit 1; }
            SITE_DIR="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "error: unknown argument: $1" >&2
            usage
            exit 1
            ;;
    esac
done

command -v jq >/dev/null 2>&1 || {
    echo "error: jq is required to build the Pages artifact" >&2
    exit 1
}

[ -f "$MANIFEST_PATH" ] || {
    echo "error: manifest not found at '$MANIFEST_PATH'" >&2
    exit 1
}

[ -f "$SITE_DIR/index.html" ] || {
    echo "error: website entrypoint not found at '$SITE_DIR/index.html'" >&2
    exit 1
}

MANIFEST_NAME="$(jq -r '.name // empty' "$MANIFEST_PATH")"
if [ "$MANIFEST_NAME" != "wptui" ]; then
    echo "error: manifest name '$MANIFEST_NAME' does not match expected 'wptui'" >&2
    exit 1
fi

MANIFEST_VERSION="$(jq -r '.version // empty' "$MANIFEST_PATH")"
SEMVER_REGEX='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$'

if ! jq -e --arg re "$SEMVER_REGEX" '.version | strings and test($re)' "$MANIFEST_PATH" >/dev/null; then
    echo "error: manifest version '$MANIFEST_VERSION' is not valid semantic versioning" >&2
    exit 1
fi

if [ -n "$EXPECTED_TAG" ] && [ "$EXPECTED_TAG" != "v${MANIFEST_VERSION}" ]; then
    echo "error: release tag '$EXPECTED_TAG' does not match manifest version '$MANIFEST_VERSION'" >&2
    exit 1
fi

if ! grep -q '__WPTUI_VERSION__' "$SITE_DIR/index.html"; then
    echo "error: '$SITE_DIR/index.html' is missing '__WPTUI_VERSION__' placeholder" >&2
    exit 1
fi

STAGE_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "$STAGE_DIR"
}
trap cleanup EXIT INT TERM

cp -R "$SITE_DIR"/. "$STAGE_DIR"/
rm -f "$STAGE_DIR"/*.tape "$STAGE_DIR/.DS_Store" "$STAGE_DIR/Thumbs.db"
: > "$STAGE_DIR/.nojekyll"

awk -v ver="$MANIFEST_VERSION" '{ gsub(/__WPTUI_VERSION__/, ver); print }' \
    "$SITE_DIR/index.html" > "$STAGE_DIR/index.html.tmp"
mv "$STAGE_DIR/index.html.tmp" "$STAGE_DIR/index.html"

if grep -q '__WPTUI_VERSION__' "$STAGE_DIR/index.html"; then
    echo "error: unresolved '__WPTUI_VERSION__' placeholder remains in built index.html" >&2
    exit 1
fi

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"
cp -R "$STAGE_DIR"/. "$OUT_DIR"/

echo "Built GitHub Pages artifact for wptui v${MANIFEST_VERSION} at ${OUT_DIR}"
