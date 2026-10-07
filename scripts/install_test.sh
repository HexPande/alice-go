#!/usr/bin/env bash
set -euo pipefail

# Test the sh -c entry point without network access or installing anything.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bin" "$tmp/work"
cat > "$tmp/bin/curl" <<'MOCK'
#!/bin/sh
if [ "${FAIL_DOWNLOAD:-0}" = 1 ]; then exit 22; fi
while [ "$#" -gt 0 ]; do
  if [ "$1" = --output ]; then shift; output=$1; fi
  shift
done
printf '#!/bin/bash\nprintf "installer:%%s\\n" "$*"\n' > "$output"
MOCK
chmod +x "$tmp/bin/curl"
export PATH="$tmp/bin:$PATH" TMPDIR="$tmp/work"
entrypoint=$(cat install.sh)
[[ $(sh -c "$entrypoint") == 'installer:' ]]
[[ $(sh -c "$entrypoint" -- v0.1.0) == 'installer:v0.1.0' ]]
status=0
FAIL_DOWNLOAD=1 sh -c "$entrypoint" > "$tmp/output" || status=$?
[[ $status == 22 && ! -s $tmp/output ]]
shopt -s nullglob
leftovers=("$tmp/work/"*)
[[ ${#leftovers[@]} == 0 ]]
printf 'Installer entry point: 3 tests passed.\n'

# shellcheck source=scripts/install.sh
source scripts/install.sh
NO_COLOR=1 init_colors
output=$(print_admin_url 192.168.1.50 9000)
[[ $output == $'\nhttp://192.168.1.50:9000/_/' ]]
output=$(print_admin_url alice.example.com 80)
[[ $output == $'\nhttp://alice.example.com/_/' ]]
output=$(print_admin_url fd00::1 9000)
[[ $output == $'\nhttp://[fd00::1]:9000/_/' ]]
printf 'Admin URLs: 3 tests passed.\n'

TERM=xterm FORCE_COLOR=1 NO_COLOR='' init_colors
[[ $(success 'ready') == $'\033[32m✓ ready\033[0m' ]]
[[ $(error 'failed' 2>&1) == $'\033[31m✗ failed\033[0m' ]]
[[ $(warn 'warning' 2>&1) == $'\033[33m! warning\033[0m' ]]
NO_COLOR=1 FORCE_COLOR=1 init_colors
[[ $(success 'ready') == '✓ ready' ]]
TERM=dumb FORCE_COLOR=1 NO_COLOR='' init_colors
[[ $(info 'step') == '→ step' ]]
printf 'Colors: 5 tests passed.\n'
