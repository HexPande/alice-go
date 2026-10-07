#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ ${1:-} == --help ]]; then
  printf 'Usage: bash install.sh [latest|v0.1.0]\nRun on Raspberry Pi Debian as a user with sudo access.\n'
  exit 0
fi
[[ $# -le 1 ]] || { printf 'Expected at most one release tag.\n' >&2; exit 1; }
for cmd in curl dpkg sha256sum tar systemctl flock; do
  command -v "$cmd" >/dev/null || { printf 'Missing command: %s\n' "$cmd" >&2; exit 1; }
done
[[ $(uname -s) == Linux ]] || { printf 'Run on Raspberry Pi (Debian).\n' >&2; exit 1; }
case $(dpkg --print-architecture) in
  arm64) target=linux-arm64 ;;
  armhf)
    [[ $(uname -m) != armv6* ]] || { printf 'ARMv6 is not supported.\n' >&2; exit 1; }
    target=linux-armv7 ;;
  *) printf 'Supported architectures: arm64, armhf (ARMv7+).\n' >&2; exit 1 ;;
esac

root=()
if [[ $EUID -ne 0 ]]; then
  sudo -v
  root=(sudo)
fi
"${root[@]}" install -d -m 0755 /opt/alice-go
"${root[@]}" touch /opt/alice-go/install.lock
"${root[@]}" chmod 0644 /opt/alice-go/install.lock
exec 9</opt/alice-go/install.lock
flock -n 9 || { printf 'Another installation is running.\n' >&2; exit 1; }

tmp=$(mktemp -d)
restart_old=false
cleanup() {
  local status=$?
  if [[ $restart_old == true ]]; then
    "${root[@]}" systemctl start alice-go || true
  fi
  rm -rf "$tmp"
  return "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

version=${1:-latest}
if [[ $version == latest ]]; then
  # Pin the resolved tag so both files come from the same release.
  release_url=$(curl --fail --silent --show-error --location --connect-timeout 15 --max-time 60 \
    https://github.com/HexPande/alice-go/releases/latest --output /dev/null --write-out '%{url_effective}')
  [[ $release_url == https://github.com/HexPande/alice-go/releases/tag/* ]] || {
    printf 'No published stable release found.\n' >&2; exit 1;
  }
  version=${release_url##*/}
fi
[[ $version =~ ^v[0-9][a-zA-Z0-9._+-]*$ ]] || {
  printf 'Invalid release tag: %s\n' "$version" >&2; exit 1;
}
base="https://github.com/HexPande/alice-go/releases/download/$version"
archive="alice-go_${target}.tar.gz"
printf 'Downloading %s (%s)...\n' "$version" "$target"
for name in "$archive" SHA256SUMS; do
  curl --fail --silent --show-error --location --retry 3 --connect-timeout 15 --max-time 600 \
    "$base/$name" --output "$tmp/$name"
done

# Validate the selected archive specifically, not merely any file in the manifest.
matches=0
while read -r digest name extra; do
  if [[ $name == "$archive" ]]; then
    [[ $digest =~ ^[0-9a-fA-F]{64}$ && -z $extra ]] || {
      printf 'Invalid checksum entry.\n' >&2; exit 1;
    }
    printf '%s  %s\n' "$digest" "$archive" > "$tmp/selected.sha256"
    matches=$((matches + 1))
  fi
done < "$tmp/SHA256SUMS"
[[ $matches == 1 ]] || { printf 'Expected exactly one archive checksum.\n' >&2; exit 1; }
(cd "$tmp" && sha256sum --check selected.sha256)
# Extract only a regular binary, ignoring all other paths and links.
tar -tvzf "$tmp/$archive" alice > "$tmp/members"
members=0
while IFS= read -r entry; do
  [[ $entry == -* ]] || { printf 'The alice archive entry must be a regular file.\n' >&2; exit 1; }
  members=$((members + 1))
done < "$tmp/members"
[[ $members == 1 ]] || { printf 'Expected exactly one alice binary.\n' >&2; exit 1; }
tar -xOzf "$tmp/$archive" alice > "$tmp/alice"
chmod 755 "$tmp/alice"
"$tmp/alice" --help >/dev/null

if ! id alice-go >/dev/null 2>&1; then
  "${root[@]}" useradd --system --user-group --home-dir /var/lib/alice-go --shell /usr/sbin/nologin alice-go
fi
"${root[@]}" install -d -o alice-go -g alice-go -m 0750 /var/lib/alice-go
cat > "$tmp/alice-go.service" <<'UNIT'
[Unit]
Description=Alice skill (PocketBase)
After=network.target

[Service]
Type=simple
User=alice-go
Group=alice-go
WorkingDirectory=/var/lib/alice-go
ExecStart=/opt/alice-go/alice serve --http=127.0.0.1:8090 --dir=/var/lib/alice-go/pb_data --dev=false
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
UMask=0027

[Install]
WantedBy=multi-user.target
UNIT

if "${root[@]}" systemctl is-active --quiet alice-go; then
  restart_old=true
  "${root[@]}" systemctl stop alice-go
elif [[ -f /etc/systemd/system/alice-go.service ]]; then
  "${root[@]}" systemctl stop alice-go
fi
backup="/var/backups/alice-go/$(date +%Y%m%d-%H%M%S)-$$"
"${root[@]}" install -d -m 0700 "$backup"
if [[ -x /opt/alice-go/alice ]]; then
  "${root[@]}" cp /opt/alice-go/alice "$backup/alice"
fi
if "${root[@]}" test -d /var/lib/alice-go/pb_data; then
  "${root[@]}" tar -C /var/lib/alice-go -czf "$backup/pb_data.tar.gz" pb_data
fi
printf 'Backup: %s\n' "$backup"
"${root[@]}" install -o root -g root -m 0755 "$tmp/alice" /opt/alice-go/alice.new
"${root[@]}" mv /opt/alice-go/alice.new /opt/alice-go/alice
# A failed new start may migrate the DB, so do not automatically roll back binaries.
restart_old=false
if [[ ! -f /etc/systemd/system/alice-go.service ]]; then
  "${root[@]}" install -o root -g root -m 0644 "$tmp/alice-go.service" /etc/systemd/system/alice-go.service
fi
"${root[@]}" systemctl daemon-reload
"${root[@]}" systemctl enable --now alice-go
for ((i=0; i<30; i++)); do
  if "${root[@]}" systemctl is-active --quiet alice-go && \
    curl --fail --silent --max-time 2 http://127.0.0.1:8090/healthz >/dev/null; then
    printf 'Installed %s. Health: http://127.0.0.1:8090/healthz\n' "$version"
    exit 0
  fi
  sleep 2
done
"${root[@]}" systemctl stop alice-go
printf 'Startup failed; service stopped. Backup: %s\nInspect: sudo journalctl -u alice-go -n 100 --no-pager\n' "$backup" >&2
exit 1
