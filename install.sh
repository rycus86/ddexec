#!/bin/sh

TAG_NAME="$1"

if [ -z "${TAG_NAME}" ]; then
    RELEASES_URL="https://api.github.com/repos/rycus86/ddexec/releases/latest"

    if command -v curl >/dev/null; then
        RELEASE_DATA="$(curl -fsSL "${RELEASES_URL}")"
    elif command -v wget >/dev/null; then
        RELEASE_DATA="$(wget -qO- "${RELEASES_URL}")"
    else
        echo "Can not find either curl or wget"
        exit 1
    fi

    if command -v jq >/dev/null; then
        TAG_NAME="$(echo "${RELEASE_DATA}" | jq -r '.tag_name')"
    elif command -v python2 >/dev/null; then
        TAG_NAME="$(echo "${RELEASE_DATA}" | python2 -c 'import sys; import json; r=json.load(sys.stdin); print r["'"tag_name"'"]')"
    elif command -v python3 >/dev/null; then
        TAG_NAME="$(echo "${RELEASE_DATA}" | python3 -c 'import sys; import json; r=json.load(sys.stdin); print(r["'"tag_name"'"])')"
    elif command -v python >/dev/null; then
        TAG_NAME="$(echo "${RELEASE_DATA}" | python -c 'from __future__ import print_function; import sys; import json; r=json.load(sys.stdin); print(r["'"tag_name"'"])')"
    else
        TAG_NAME="$(echo "${RELEASE_DATA}" | grep -oE '"tag_name": "[0-9.]+"' | sed -E 's/.*: "([0-9.]+)"/\1/')"
    fi
fi

if [ -z "${TAG_NAME}" ]; then
    echo "Can not find the latest release version number"
    exit 1
else
    echo "Downloading ddexec version ${TAG_NAME} ..."
fi

RELEASE_URL="https://github.com/rycus86/ddexec/releases/tag/${TAG_NAME}"
BINARY_URL="https://github.com/rycus86/ddexec/releases/download/${TAG_NAME}/ddexec-${TAG_NAME}.linux-amd64"
SHASUM_URL="${BINARY_URL}.sha256sum"

if command -v curl >/dev/null; then
    DOWNLOAD_COMMAND="curl -fsSL -o"
elif command -v wget >/dev/null; then
    DOWNLOAD_COMMAND="wget -q -O"
fi

${DOWNLOAD_COMMAND} ddexec "${BINARY_URL}" &&
    ${DOWNLOAD_COMMAND} ddexec.sha256sum "${SHASUM_URL}" &&
    cat ddexec | sha256sum -c ddexec.sha256sum >/dev/null &&
    rm ddexec.sha256sum &&
    chmod +x ddexec ||
    {
        echo "Failed to download the release binary from ${RELEASE_URL}"
        [ -f ddexec ] && rm ddexec
        [ -f ddexec.sha256sum ] && rm ddexec.sha256sum
        exit 1
    }

echo
echo "./ddexec successfully downloaded"
echo

MOVE_TARGET="/usr/local/bin/"
if command -v ddexec >/dev/null; then
    MOVE_TARGET="$(command -v ddexec)"
fi

echo "You might want to move it to ${MOVE_TARGET} :"

if [ "$(id -u)" = "0" ]; then
    printf '# mv ddexec %s\n' "$MOVE_TARGET"
else
    printf '$ sudo mv ddexec %s\n' "$MOVE_TARGET"
fi
