# Запуск на Raspberry Pi с Debian

Рекомендуемая схема: **GitHub Actions → GitHub Release → systemd на Pi → HTTPS через Caddy**. Для локальной разработки можно собирать на Mac. На Raspberry Pi не нужны Go, компилятор C или отдельный сервер базы данных. PocketBase использует встроенную SQLite; данные хранятся отдельно от бинарника.

В примерах `pi@raspberrypi.local` — SSH-пользователь и адрес Raspberry Pi. Замените их своими. Команды помечены по месту выполнения.

## 1. Определить архитектуру ОС

На **Raspberry Pi**:

```sh
dpkg --print-architecture
uname -m
```

- `arm64` — используйте сборку `GOARCH=arm64`; рекомендуется для Pi 3/4/5 с 64-битной ОС.
- `armhf` — нужна 32-битная сборка `GOARCH=arm`. Для Pi 2/3/4/5 используйте `GOARM=7`.
- Старые Pi 1 / Zero первого поколения имеют ARMv6: для них нужен `GOARM=6`, даже если Raspberry Pi OS сообщает `armhf`. На таких устройствах работу текущей версии PocketBase нужно проверять отдельно.

Выбирайте архитектуру установленной ОС, а не только процессора: на 64-битной Pi может стоять 32-битная система.

## 2. Получить бинарник

### Скачать GitHub Release (рекомендуется)

После публикации тега по [инструкции в README](../README.md#ci-и-релизы) откройте [GitHub Releases](https://github.com/HexPande/alice-go/releases) и выберите версию.

На **Pi** для публичного репозитория выполните следующие команды. Замените `v0.1.0` существующим тегом, а для 32-битной ОС на Pi 2/3/4/5 задайте `TARGET=linux-armv7`:

```sh
sudo apt update
sudo apt install -y ca-certificates curl
VERSION=v0.1.0
TARGET=linux-arm64
mkdir -p "$HOME/alice-releases/$VERSION/$TARGET"
cd "$HOME/alice-releases/$VERSION/$TARGET"
BASE_URL="https://github.com/HexPande/alice-go/releases/download/$VERSION"
curl --fail --location --output "alice-go_${TARGET}.tar.gz" "$BASE_URL/alice-go_${TARGET}.tar.gz"
curl --fail --location --output SHA256SUMS "$BASE_URL/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
```

Продолжайте только если проверка выбранного архива завершилась с `OK`:

```sh
tar -xzf "alice-go_${TARGET}.tar.gz"
install -m 0755 alice "$HOME/alice-upload"
```

Теперь переходите к шагу 3, пропустив сборку на Mac и `scp`.
Если репозиторий приватный, скачайте архив и `SHA256SUMS` через авторизованный GitHub на Mac и перенесите их на Pi через `scp`, затем выполните ту же проверку и распаковку. Анонимный `curl` для приватных релизов не работает.

### Собрать на Mac самостоятельно

В **корне проекта на Mac**, с Go 1.27.1:

```sh
go mod download
go test ./...
mkdir -p bin
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -trimpath -ldflags='-s -w' \
  -o bin/alice-linux-arm64 ./cmd/alice
file bin/alice-linux-arm64
```

`file` должен показать Linux ELF для ARM aarch64, а не macOS Mach-O.
`CGO_ENABLED=0` позволяет собрать проект без Linux C-toolchain. Исходники, Go и зависимости на Pi переносить не нужно.

Для **32-битной ОС на Pi 2/3/4/5** команда сборки:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build \
  -trimpath -ldflags='-s -w' \
  -o bin/alice-linux-armv7 ./cmd/alice
```

Дальше показан вариант `arm64`; для `armhf` замените имя передаваемого файла. Тесты на Mac проверяют логику, но не заменяют проверку запуска на самой Pi.

## 3. Установить приложение

На **Pi**, один раз:

```sh
sudo apt update
sudo apt install -y ca-certificates curl
sudo useradd --system --user-group --home-dir /var/lib/alice-go --shell /usr/sbin/nologin alice-go
sudo install -d -o root -g root -m 0755 /opt/alice-go
sudo install -d -o alice-go -g alice-go -m 0750 /var/lib/alice-go
```

С **Mac**:

```sh
scp bin/alice-linux-arm64 pi@raspberrypi.local:alice-upload
```

На **Pi**:

```sh
sudo install -o root -g root -m 0755 ~/alice-upload /opt/alice-go/alice
/opt/alice-go/alice --help
```

Бинарник хранится в `/opt/alice-go/alice`, база и файлы — в `/var/lib/alice-go/pb_data`.

## 4. Настроить автозапуск через systemd

На **Pi** откройте файл:

```sh
sudo nano /etc/systemd/system/alice-go.service
```

Содержимое:

```ini
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
```

Запустите:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now alice-go
sudo systemctl status alice-go --no-pager
curl --fail http://127.0.0.1:8090/healthz
```

Ожидаемый ответ: `{"status":"ok"}`.

Просмотр логов:

```sh
sudo journalctl -u alice-go -n 100 --no-pager
sudo journalctl -u alice-go -f
```

## 5. Открыть админ-панель

Сервис слушает loopback Pi. Для доступа с **Mac** откройте SSH-туннель и оставьте его работающим:

```sh
ssh -N -L 8090:127.0.0.1:8090 pi@raspberrypi.local
```

Затем откройте `http://127.0.0.1:8090/_/` в браузере на Mac. Для первого создания суперпользователя найдите установочную ссылку в журнале `alice-go` и откройте её через тот же туннель. Если в ссылке указан другой локальный хост, замените только хост и порт на `127.0.0.1:8090`, сохранив путь и токен.

## 6. Подключить HTTPS для Алисы

Для этого варианта нужны домен (например, `alice.example.com`) и публичный IP. DNS должен указывать на этот IP, а TCP-порты **80 и 443** — доходить до Pi. Если Pi за роутером, настройте проброс обоих портов на неё. Запись AAAA добавляйте только при рабочем входящем IPv6.

Если провайдер использует CGNAT и входящие соединения недоступны, понадобится постоянный HTTPS-туннель или внешний сервер-прокси. Обычный проброс портов в таком случае не поможет.

На **Pi**:

```sh
sudo apt install -y caddy
sudo nano /etc/caddy/Caddyfile
```

Для выделенной Pi задайте конфигурацию ниже; если Caddy уже обслуживает сайты, добавьте этот блок к существующей конфигурации. Замените домен своим:

```caddyfile
alice.example.com {
    reverse_proxy 127.0.0.1:8090
}
```

Примените:

```sh
sudo caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
sudo systemctl enable --now caddy
sudo systemctl reload caddy
curl --fail https://alice.example.com/healthz
```

Caddy автоматически получает и обновляет TLS-сертификат. Этот вариант публикует все маршруты PocketBase, включая защищённую авторизацией админ-панель.

Проверьте webhook с **Mac**:

```sh
curl --fail https://alice.example.com/alice \
  -H 'Content-Type: application/json' \
  -d '{"version":"1.0","session":{"new":true,"session_id":"test","message_id":0},"request":{"type":"SimpleUtterance","command":""}}'
```

В [консоли Яндекс Диалогов](https://dialogs.yandex.ru/developer/) укажите Backend URL `https://alice.example.com/alice` и проверьте навык на вкладке тестирования.

## 7. Обновления и резервные копии

Скачайте новый релиз, проверьте SHA-256 и подготовьте `~/alice-upload`, как в шаге 2. Либо соберите бинарник на Mac и передайте через `scp`. Затем на **Pi**:

```sh
sudo systemctl stop alice-go
sudo install -d -m 0700 /var/backups/alice-go
sudo tar -C /var/lib/alice-go -czf "/var/backups/alice-go/pb_data-$(date +%Y%m%d-%H%M%S).tar.gz" pb_data
sudo cp /opt/alice-go/alice /opt/alice-go/alice.previous
sudo install -o root -g root -m 0755 ~/alice-upload /opt/alice-go/alice
sudo systemctl start alice-go
curl --fail http://127.0.0.1:8090/healthz
```

Выполняйте команды последовательно: при ошибке резервного копирования остановитесь и запустите прежний сервис. Копия SQLite в этом примере делается при остановленном приложении. Храните резервные копии также вне Pi. Обновление бинарника не заменяет `pb_data`.

При обновлениях PocketBase учитывайте миграции БД: для отката может понадобиться восстановление и старого бинарника, и совместимой резервной копии данных.

## Типичные проблемы

| Симптом | Что проверить |
| --- | --- |
| `Exec format error` | Бинарник должен быть Linux ARM нужной разрядности, а не macOS |
| `Permission denied` | Права на бинарник, каталог данных и отсутствие `noexec` на файловой системе |
| Сервис завершается сразу | `sudo journalctl -u alice-go -n 100 --no-pager` |
| Caddy возвращает 502 | `systemctl status alice-go` и локальный `/healthz` |
| Сертификат не выпускается | DNS, доступность портов 80/443, CGNAT, журнал `journalctl -u caddy` |
| Навык недоступен Яндексу | Проверьте публичный HTTPS URL из другой сети |

Документация: [PocketBase deployment](https://pocketbase.io/docs/going-to-production/), [Caddy reverse proxy](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy).
