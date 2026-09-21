#!/bin/sh
set -eu

# WPTUI installer for macOS
# Installs the latest stable release into a user-local directory.

REPO_OWNER="HungNth"
REPO_NAME="wordpress-tui"
APP_NAME="wptui"

BASE_DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download"

# Determine installation directory
INSTALL_DIR="${WPTUI_INSTALL_DIR:-${HOME}/.local/bin}"

# Target executable
TARGET_BIN="${INSTALL_DIR}/${APP_NAME}"

info() {
    printf "\033[1;34m==>\033[0m %s\n" "$*"
}

error() {
    printf "\033[1;31merror:\033[0m %s\n" "$*" >&2
}

fatal() {
    error "$@"
    exit 1
}

# 1. Platform verification
OS="$(uname -s)"
if [ "${OS}" != "Darwin" ]; then
    fatal "This installer is designed for macOS (Darwin). Current OS is '${OS}'."
fi

# 2. Architecture detection (handling Rosetta translation)
ARCH="$(uname -m)"
IS_TRANSLATED=0
if command -v sysctl >/dev/null 2>&1; then
    IS_TRANSLATED="$(sysctl -in sysctl.proc_translated 2>/dev/null || echo 0)"
fi

if [ "${IS_TRANSLATED}" = "1" ] || [ "${ARCH}" = "arm64" ]; then
    TARGET_ARCH="arm64"
elif [ "${ARCH}" = "x86_64" ] || [ "${ARCH}" = "amd64" ]; then
    TARGET_ARCH="amd64"
else
    fatal "Unsupported architecture '${ARCH}'. WPTUI supports arm64 and amd64 on macOS."
fi

# 3. Tool checks
for tool in curl unzip shasum; do
    if ! command -v "${tool}" >/dev/null 2>&1; then
        fatal "Required command '${tool}' not found. Please install or make it available in PATH."
    fi
done

# 4. Setup staging directory and cleanup trap
STAGING_DIR="$(mktemp -d -t wptui-install.XXXXXX)"
cleanup() {
    rm -rf "${STAGING_DIR}"
}
trap cleanup EXIT INT TERM

# 5. Fetch checksums manifest
CHECKSUMS_FILE="${STAGING_DIR}/checksums.txt"
info "Fetching release checksums..."
if ! curl -fsSL "${BASE_DOWNLOAD_URL}/checksums.txt" -o "${CHECKSUMS_FILE}"; then
    fatal "Failed to download checksums from '${BASE_DOWNLOAD_URL}/checksums.txt'."
fi

# 6. Locate asset entry for darwin/${TARGET_ARCH}
# Expected line: <64-hex sha256><whitespace>wptui_<semver>_darwin_<arch>.zip
# grep -E is used because `awk -v` performs escape processing on backslashes,
# which corrupts the pattern's `\.` and `\+` sequences.
ASSET_NAME_PATTERN="wptui_[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?_darwin_${TARGET_ARCH}\.zip"
LINE_PATTERN="^[^[:space:]]+[[:space:]]+${ASSET_NAME_PATTERN}\$"

MATCH_LINE="$(grep -E "${LINE_PATTERN}" "${CHECKSUMS_FILE}" | awk '{print $1, $2}' || true)"

MATCH_COUNT="$(echo "${MATCH_LINE}" | grep -c . || true)"
if [ "${MATCH_COUNT}" -eq 0 ] || [ -z "${MATCH_LINE}" ]; then
    fatal "No asset matching '${ASSET_NAME_PATTERN}' found in checksums.txt."
fi
if [ "${MATCH_COUNT}" -gt 1 ]; then
    fatal "Multiple assets matching '${ASSET_NAME_PATTERN}' found in checksums.txt."
fi

EXPECTED_HASH="$(echo "${MATCH_LINE}" | awk '{print $1}')"
ASSET_NAME="$(echo "${MATCH_LINE}" | awk '{print $2}')"

# Validate SHA-256 format (64 hex characters)
case "${EXPECTED_HASH}" in
    [0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F])
        ;;
    *)
        fatal "Malformed SHA-256 digest in checksums.txt: '${EXPECTED_HASH}'."
        ;;
esac

# 7. Download release archive
ARCHIVE_PATH="${STAGING_DIR}/${ASSET_NAME}"
info "Downloading ${ASSET_NAME}..."
if ! curl -fsSL "${BASE_DOWNLOAD_URL}/${ASSET_NAME}" -o "${ARCHIVE_PATH}"; then
    fatal "Failed to download release archive '${ASSET_NAME}' from '${BASE_DOWNLOAD_URL}'."
fi

# 8. Verify checksum
info "Verifying SHA-256 checksum..."
CALCULATED_HASH="$(shasum -a 256 "${ARCHIVE_PATH}" | awk '{print $1}')"

# Case-insensitive comparison using tr
EXPECTED_LOWER="$(echo "${EXPECTED_HASH}" | tr '[:upper:]' '[:lower:]')"
CALCULATED_LOWER="$(echo "${CALCULATED_HASH}" | tr '[:upper:]' '[:lower:]')"

if [ "${EXPECTED_LOWER}" != "${CALCULATED_LOWER}" ]; then
    fatal "Checksum mismatch for '${ASSET_NAME}'.
Expected:   ${EXPECTED_LOWER}
Calculated: ${CALCULATED_LOWER}"
fi

# 9. Verify archive structure: must contain exactly 1 member named 'wptui'
ARCHIVE_MEMBERS="$(unzip -Z1 "${ARCHIVE_PATH}")"
MEMBER_COUNT="$(echo "${ARCHIVE_MEMBERS}" | grep -c . || true)"
if [ "${MEMBER_COUNT}" -ne 1 ] || [ "${ARCHIVE_MEMBERS}" != "${APP_NAME}" ]; then
    fatal "Archive '${ASSET_NAME}' must contain exactly one member '${APP_NAME}', found: ${ARCHIVE_MEMBERS}"
fi

EXTRACT_DIR="${STAGING_DIR}/extracted"
mkdir -p "${EXTRACT_DIR}"
if ! unzip -q "${ARCHIVE_PATH}" -d "${EXTRACT_DIR}"; then
    fatal "Failed to extract release archive '${ASSET_NAME}'."
fi

STAGE_BIN="${EXTRACT_DIR}/${APP_NAME}"
if [ ! -f "${STAGE_BIN}" ]; then
    fatal "Extracted executable '${STAGE_BIN}' not found."
fi

chmod +x "${STAGE_BIN}"

# 10. Atomic replacement into destination
mkdir -p "${INSTALL_DIR}"
TEMP_TARGET="${TARGET_BIN}.tmp.$$"
cp "${STAGE_BIN}" "${TEMP_TARGET}"
chmod +x "${TEMP_TARGET}"
mv -f "${TEMP_TARGET}" "${TARGET_BIN}"

info "Installed ${APP_NAME} to ${TARGET_BIN}"

# 11. PATH configuration check
DIR_IN_PATH=0
case ":${PATH}:" in
    *":${INSTALL_DIR}:"*)
        DIR_IN_PATH=1
        ;;
esac

PROFILE_MODIFIED=""
if [ "${DIR_IN_PATH}" -eq 0 ]; then
    USER_SHELL="$(basename "${SHELL:-}")"
    TARGET_PROFILE=""

    if [ "${USER_SHELL}" = "zsh" ]; then
        TARGET_PROFILE="${HOME}/.zprofile"
    elif [ "${USER_SHELL}" = "bash" ]; then
        if [ -f "${HOME}/.bash_profile" ]; then
            TARGET_PROFILE="${HOME}/.bash_profile"
        elif [ -f "${HOME}/.bash_login" ]; then
            TARGET_PROFILE="${HOME}/.bash_login"
        else
            TARGET_PROFILE="${HOME}/.profile"
        fi
    fi

    EXPORT_CMD="export PATH=\"${INSTALL_DIR}:\$PATH\""

    if [ -n "${TARGET_PROFILE}" ]; then
        ALREADY_IN_PROFILE=0
        if [ -f "${TARGET_PROFILE}" ] && grep -F -q "${EXPORT_CMD}" "${TARGET_PROFILE}"; then
            ALREADY_IN_PROFILE=1
        fi

        if [ "${ALREADY_IN_PROFILE}" -eq 0 ]; then
            printf "\n# Added by wptui installer\n%s\n" "${EXPORT_CMD}" >> "${TARGET_PROFILE}"
            PROFILE_MODIFIED="${TARGET_PROFILE}"
        fi
    fi

    printf "\n"
    info "\033[1;33mAction required:\033[0m '${INSTALL_DIR}' is not currently in your PATH."
    if [ -n "${PROFILE_MODIFIED}" ]; then
        info "Updated ${PROFILE_MODIFIED}. Please restart your terminal or run:"
        printf "    %s\n\n" "${EXPORT_CMD}"
    else
        info "Please add it to your shell configuration or run:"
        printf "    %s\n\n" "${EXPORT_CMD}"
    fi
else
    printf "\n"
    info "WPTUI is ready to run! Execute '${APP_NAME}' to get started."
fi
