#!/bin/sh
set -eu
command -v bash >/dev/null 2>&1 || {
  printf 'Для запуска установщика нужен Bash.\n' >&2
  exit 1
}

# Keep sh -c compatibility; the Bash implementation is embedded in this file.
bash -s -- "$@" <<'ALICE_GO_INSTALLER'
# BEGIN BASH INSTALLER
#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

init_colors() {
  red='' green='' yellow='' cyan='' bold='' reset=''
  if [[ -z ${NO_COLOR:-} && ${TERM:-} != dumb ]] && [[ -t 1 || ${FORCE_COLOR:-0} == 1 ]]; then
    red=$'\033[31m' green=$'\033[32m' yellow=$'\033[33m'
    cyan=$'\033[36m' bold=$'\033[1m' reset=$'\033[0m'
  fi
}
info() { printf '%s→ %s%s\n' "$cyan" "$*" "$reset"; }
success() { printf '%s✓ %s%s\n' "$green" "$*" "$reset"; }
warn() { printf '%s! %s%s\n' "$yellow" "$*" "$reset" >&2; }
error() { printf '%s✗ %s%s\n' "$red" "$*" "$reset" >&2; }
die() { error "$*"; exit 1; }
init_colors

print_admin_url() {
  local host=$1 port=$2 authority
  [[ $host != *:* ]] || host="[$host]"
  authority=$host
  [[ $port == 80 ]] || authority="$host:$port"
  printf '\n%s%shttp://%s/_/%s\n' "$bold" "$cyan" "$authority" "$reset"
}

check_system() {
  local release_file=${1:-/etc/os-release} systemd_dir=${2:-/run/systemd/system}
  local ID='' PRETTY_NAME='' cmd arch
  [[ $(uname -s) == Linux ]] || die 'Поддерживается только Linux.'
  [[ -r $release_file ]] || die 'Не удалось определить ОС: отсутствует /etc/os-release.'
  # os-release is a system-owned file using shell-compatible assignments.
  # shellcheck source=/dev/null
  source "$release_file"
  case "$ID" in
    debian|ubuntu|raspbian) ;;
    *) die "ОС ${PRETTY_NAME:-$ID} не поддерживается. Нужна Debian, Ubuntu или Raspberry Pi OS." ;;
  esac
  [[ -d $systemd_dir ]] || die 'Для установки нужен работающий systemd. Запустите скрипт на хосте с systemd.'
  for cmd in curl dpkg sha256sum tar systemctl flock install mktemp hostname useradd id date cp mv chmod rm cat sleep; do
    command -v "$cmd" >/dev/null || die "Не найдена обязательная команда: $cmd"
  done
  if [[ $EUID -ne 0 ]]; then
    command -v sudo >/dev/null || die 'Нужны права root или установленный sudo.'
  fi
  arch=$(dpkg --print-architecture)
  case "$arch" in
    arm64) target=linux-arm64 ;;
    armhf)
      [[ $(uname -m) != armv6* ]] || die 'ARMv6 не поддерживается; нужен ARMv7 или ARM64.'
      target=linux-armv7 ;;
    *) die "Архитектура $arch не поддерживается. Доступны arm64 и armhf (ARMv7+)." ;;
  esac
  success "Система: ${PRETTY_NAME:-$ID} ($arch)"
}

cleanup() {
  local status=$?
  if [[ $restart_old == true ]]; then
    warn 'Возобновление прежнего сервиса после прерванного обновления.'
    "${root[@]}" systemctl start alice-go || error 'Не удалось запустить прежний сервис.'
  fi
  rm -rf "$tmp"
  return "$status"
}

main() {
if [[ ${1:-} == --help ]]; then
  printf '%salice-go — установка и обновление%s\n' "$bold" "$reset"
  printf 'Использование: sh install.sh [latest|v0.1.0]\nЗапустите с правами root или sudo.\n'
  exit 0
fi
stage='подготовка'
trap 'error "Не удалось выполнить этап: $stage"' ERR
[[ $# -le 1 ]] || die 'Укажите не более одного тега релиза.'
info 'Проверка системы'
check_system /etc/os-release /run/systemd/system

root=()
if [[ $EUID -ne 0 ]]; then
  root=(sudo)
fi
"${root[@]}" install -d -m 0755 /opt/alice-go
"${root[@]}" touch /opt/alice-go/install.lock
"${root[@]}" chmod 0644 /opt/alice-go/install.lock
exec 9</opt/alice-go/install.lock
flock -n 9 || die 'Другой установщик уже запущен.'

tmp=$(mktemp -d)
restart_old=false
trap 'cleanup' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

version=${1:-latest}
stage='скачивание релиза'
if [[ $version == latest ]]; then
  info 'Поиск последнего стабильного релиза'
  # Pin the resolved tag so both files come from the same release.
  release_url=$(curl --fail --silent --show-error --location --connect-timeout 15 --max-time 60 \
    https://github.com/HexPande/alice-go/releases/latest --output /dev/null --write-out '%{url_effective}')
  [[ $release_url == https://github.com/HexPande/alice-go/releases/tag/* ]] || {
    die 'Стабильный релиз ещё не опубликован: https://github.com/HexPande/alice-go/releases'
  }
  version=${release_url##*/}
fi
[[ $version =~ ^v[0-9][a-zA-Z0-9._+-]*$ ]] || {
  die "Некорректный тег релиза: $version"
}
base="https://github.com/HexPande/alice-go/releases/download/$version"
archive="alice-go_${target}.tar.gz"
info "Скачивание $version ($target)"
for name in "$archive" SHA256SUMS; do
  curl --fail --silent --show-error --location --retry 3 --connect-timeout 15 --max-time 600 \
    "$base/$name" --output "$tmp/$name"
done

# Validate the selected archive specifically, not merely any file in the manifest.
stage='проверка дистрибутива'
info 'Проверка SHA-256 и содержимого архива'
matches=0
while read -r digest name extra; do
  if [[ $name == "$archive" ]]; then
    [[ $digest =~ ^[0-9a-fA-F]{64}$ && -z $extra ]] || {
      die 'Некорректная запись контрольной суммы.'
    }
    printf '%s  %s\n' "$digest" "$archive" > "$tmp/selected.sha256"
    matches=$((matches + 1))
  fi
done < "$tmp/SHA256SUMS"
[[ $matches == 1 ]] || die 'В манифесте должна быть ровно одна контрольная сумма выбранного архива.'
(cd "$tmp" && sha256sum --check --status selected.sha256) || die 'Контрольная сумма архива не совпадает.'
# Extract only a regular binary, ignoring all other paths and links.
tar -tvzf "$tmp/$archive" alice > "$tmp/members"
members=0
while IFS= read -r entry; do
  [[ $entry == -* ]] || die 'Запись alice в архиве должна быть обычным файлом.'
  members=$((members + 1))
done < "$tmp/members"
[[ $members == 1 ]] || die 'В архиве должен быть ровно один бинарник alice.'
tar -xOzf "$tmp/$archive" alice > "$tmp/alice"
chmod 755 "$tmp/alice"
"$tmp/alice" --help >/dev/null
success 'Дистрибутив проверен'

stage='настройка приложения'
config_file=/etc/alice-go/config.yaml
if "${root[@]}" test -f "$config_file"; then
  "${root[@]}" cat "$config_file" > "$tmp/config.yaml"
  info "Сохранение существующих настроек: $config_file"
else
  port=${ALICE_PORT:-}
  while :; do
    if [[ -z $port ]]; then
      read -r -p "${bold}${cyan}Порт сервиса [8090]: ${reset}" port </dev/tty || {
        die 'Терминал недоступен. Задайте порт через ALICE_PORT.'
      }
      port=${port:-8090}
    fi
    if [[ $port =~ ^[0-9]{1,5}$ ]] && ((10#$port >= 1 && 10#$port <= 65535)); then
      port=$((10#$port))
      break
    fi
    warn 'Порт должен быть числом от 1 до 65535.'
    [[ -z ${ALICE_PORT:-} ]] || exit 1
    port=
  done
  printf 'port: %s\n' "$port" > "$tmp/config.yaml"
fi
# The application parses YAML through Viper; the shell does not parse YAML.
port=$("$tmp/alice" config-port --config "$tmp/config.yaml")
if [[ ! $port =~ ^[0-9]{1,5}$ ]] || ((port < 1 || port > 65535)); then
  die 'Релиз не вернул корректный порт из YAML-конфига.'
fi
success "Порт сервиса: $port"

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
ExecStart=/opt/alice-go/alice serve --config=/etc/alice-go/config.yaml --dir=/var/lib/alice-go/pb_data --dev=false
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
UMask=0027

[Install]
WantedBy=multi-user.target
UNIT

stage='резервное копирование'
info 'Подготовка резервной копии'
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
if "${root[@]}" test -f "$config_file"; then
  "${root[@]}" cp "$config_file" "$backup/config.yaml"
fi
if [[ -f /etc/systemd/system/alice-go.service ]]; then
  "${root[@]}" cp /etc/systemd/system/alice-go.service "$backup/alice-go.service"
fi
if "${root[@]}" test -d /var/lib/alice-go/pb_data; then
  "${root[@]}" tar -C /var/lib/alice-go -czf "$backup/pb_data.tar.gz" pb_data
fi
success "Резервная копия: $backup"
stage='установка приложения'
info 'Установка приложения и настройка автозапуска'
"${root[@]}" install -d -o root -g alice-go -m 0750 /etc/alice-go
if ! "${root[@]}" test -f "$config_file"; then
  "${root[@]}" install -o root -g alice-go -m 0640 "$tmp/config.yaml" "$config_file"
fi
"${root[@]}" install -o root -g root -m 0755 "$tmp/alice" /opt/alice-go/alice.new
"${root[@]}" mv /opt/alice-go/alice.new /opt/alice-go/alice
# A failed new start may migrate the DB, so do not automatically roll back binaries.
restart_old=false
"${root[@]}" install -o root -g root -m 0644 "$tmp/alice-go.service" /etc/systemd/system/alice-go.service
"${root[@]}" systemctl daemon-reload
"${root[@]}" systemctl enable --now alice-go
stage='проверка запуска'
info 'Проверка запуска сервиса'
for ((i=0; i<30; i++)); do
  if "${root[@]}" systemctl is-active --quiet alice-go && \
    curl --fail --silent --max-time 2 "http://127.0.0.1:$port/healthz" >/dev/null; then
    success "alice-go $version установлен и запущен"
    host=
    read -r -a addresses <<< "$(hostname -I 2>/dev/null || true)"
    for address in "${addresses[@]}"; do
      if [[ $address == *.* && $address != 127.* ]]; then
        host=$address
        break
      fi
    done
    host=${host:-${addresses[0]:-$(hostname)}}
    print_admin_url "$host" "$port"
    exit 0
  fi
  sleep 2
done
"${root[@]}" systemctl stop alice-go
warn "Резервная копия: $backup"
warn 'Журнал: sudo journalctl -u alice-go -n 100 --no-pager'
die 'Приложение не прошло проверку запуска. Сервис остановлен.'
}

# Sourcing exposes the formatters to tests without starting an installation.
if [[ -z ${BASH_SOURCE[0]:-} || ${BASH_SOURCE[0]} == "$0" ]]; then
  main "$@"
fi
# END BASH INSTALLER
ALICE_GO_INSTALLER
