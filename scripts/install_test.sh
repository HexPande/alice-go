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
