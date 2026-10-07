#!/bin/sh
set -eu

red= cyan= reset=
if [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != dumb ]; then
    if [ -t 1 ] || [ "${FORCE_COLOR:-0}" = 1 ]; then
        red=$(printf '\033[31m')
        cyan=$(printf '\033[36m')
        reset=$(printf '\033[0m')
    fi
fi

# POSIX entry point: also works when passed to sh -c by curl.
for cmd in bash curl mktemp; do
    command -v "$cmd" >/dev/null 2>&1 || {
        printf '%s✗ Не найдена команда: %s%s\n' "$red" "$cmd" "$reset" >&2
        exit 1
    }
done

umask 077
installer=$(mktemp)
trap 'rm -f "$installer"' 0
trap 'exit 130' INT
trap 'exit 143' TERM
printf '%s→ Загрузка установщика alice-go%s\n' "$cyan" "$reset" >&2
curl --fail --silent --show-error --location --connect-timeout 15 --max-time 60 \
    https://raw.githubusercontent.com/HexPande/alice-go/main/scripts/install.sh \
    --output "$installer" || {
        status=$?
        printf '%s✗ Не удалось скачать установщик.%s\n' "$red" "$reset" >&2
        exit "$status"
    }
bash "$installer" "$@"
