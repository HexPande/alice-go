#!/bin/sh
set -eu

# POSIX entry point: also works when passed to sh -c by curl.
for cmd in bash curl mktemp; do
    command -v "$cmd" >/dev/null 2>&1 || {
        printf 'Required command not found: %s\n' "$cmd" >&2
        exit 1
    }
done

umask 077
installer=$(mktemp)
trap 'rm -f "$installer"' 0
trap 'exit 130' INT
trap 'exit 143' TERM
curl --fail --silent --show-error --location --connect-timeout 15 --max-time 60 \
    https://raw.githubusercontent.com/HexPande/alice-go/main/scripts/install.sh \
    --output "$installer"
bash "$installer" "$@"
