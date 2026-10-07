#!/usr/bin/env bash
set -euo pipefail

# Test the sh -c entry point without network access or installing anything.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bin" "$tmp/work"
cat > "$tmp/bin/uname" <<'MOCK'
#!/bin/sh
case "$1" in
  -s) printf '%s\n' "${TEST_OS:-Linux}" ;;
  -m) printf '%s\n' "${TEST_MACHINE:-aarch64}" ;;
esac
MOCK
chmod +x "$tmp/bin/uname"
cat > "$tmp/bin/curl" <<'MOCK'
#!/bin/sh
printf 'Unexpected network call in installer tests.\n' >&2
exit 99
MOCK
chmod +x "$tmp/bin/curl"
export PATH="$tmp/bin:$PATH" TMPDIR="$tmp/work"
entrypoint=$(cat install.sh)
[[ $(sh -c "$entrypoint" -- --help) == *'Использование:'* ]]
[[ $(sh install.sh --help) == *'Использование:'* ]]
status=0
sh -c "$entrypoint" -- v0.1.0 extra > "$tmp/output" 2>&1 || status=$?
[[ $status == 1 && $(cat "$tmp/output") == *'не более одного тега'* ]]
shopt -s nullglob
leftovers=("$tmp/work/"*)
[[ ${#leftovers[@]} == 0 ]]
printf 'Installer entry point: 3 tests passed.\n'

# Check and source the exact embedded payload, rather than a duplicate implementation.
sed '1,/^# BEGIN BASH INSTALLER$/d; /^# END BASH INSTALLER$/,$d' install.sh > "$tmp/payload.sh"
bash -n "$tmp/payload.sh"
if command -v shellcheck >/dev/null 2>&1; then
  shellcheck --shell=bash "$tmp/payload.sh"
fi
# shellcheck source=/dev/null
source "$tmp/payload.sh"
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

# OS checks run before sudo or any changes to the installed service.
mkdir -p "$tmp/systemd"
printf 'ID=fedora\nPRETTY_NAME="Fedora Linux"\n' > "$tmp/os-release"
status=0
(check_system "$tmp/os-release" "$tmp/systemd") > "$tmp/output" 2>&1 || status=$?
[[ $status == 1 && $(cat "$tmp/output") == *'не поддерживается'* ]]
printf 'ID=debian\nPRETTY_NAME="Debian GNU/Linux"\n' > "$tmp/os-release"
status=0
(check_system "$tmp/os-release" "$tmp/no-systemd") > "$tmp/output" 2>&1 || status=$?
[[ $status == 1 && $(cat "$tmp/output") == *'работающий systemd'* ]]
status=0
(check_system "$tmp/missing-os-release" "$tmp/systemd") > "$tmp/output" 2>&1 || status=$?
[[ $status == 1 && $(cat "$tmp/output") == *'определить ОС'* ]]
status=0
TEST_OS=Darwin sh -c "$entrypoint" > "$tmp/output" 2>&1 || status=$?
[[ $status == 1 && $(cat "$tmp/output") == *'только Linux'* ]]
printf 'System requirements: 4 tests passed.\n'
