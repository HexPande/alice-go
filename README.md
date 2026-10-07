# alice-go

Скелет навыка Яндекс Алисы на Go и [PocketBase](https://github.com/pocketbase/pocketbase).
PocketBase запускает HTTP-сервер, предоставляет SQLite, API коллекций, авторизацию и админ-панель. Webhook навыка подключается через `OnServe`.

## Требования и запуск

Требуется Go **1.27+** (PocketBase `v0.40.4`). При включённом `GOTOOLCHAIN=auto` Go скачает подходящий toolchain автоматически.

```sh
go mod download
make run
```

Или напрямую:

```sh
go run ./cmd/alice serve --http=127.0.0.1:8090 --dir=./pb_data
```

- Webhook: `http://127.0.0.1:8090/alice`
- Проверка доступности: `http://127.0.0.1:8090/healthz`
- Админ-панель: `http://127.0.0.1:8090/_/`
- Штатный API PocketBase: `http://127.0.0.1:8090/api/`

При первом запуске PocketBase выводит ссылку для создания суперпользователя. Откройте её и задайте учётные данные, затем войдите в админ-панель.

Данные и настройки сохраняются в `pb_data/` (исключён из Git). Для развёртывания используйте постоянный диск и резервное копирование этой директории.

Другой адрес через Makefile: `make run HTTP_ADDR=0.0.0.0:8080`.
Параметры запуска: `go run ./cmd/alice serve --help`.

### Raspberry Pi (Debian)

Установка готового GitHub Release, сборка на Mac для Linux ARM, systemd и HTTPS описаны в [инструкции развёртывания](docs/raspberry-pi.md).

### CI и релизы

GitHub Actions запускает тесты с race detector, проверку форматирования, `go vet`, `golangci-lint` и `govulncheck` на push и pull request. После успешных проверок собираются архивы для `linux-arm64` и `linux-armv7`; они доступны в артефакте `raspberry-pi` на странице запуска Actions в течение 14 дней. Сборку также можно запустить вручную через **Actions → CI and Release → Run workflow**.

Для публикации релиза сначала закоммитьте изменения и отправьте ветку с workflow на GitHub. Затем создайте и отправьте новый тег:

```sh
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

После проверок и сборки workflow создаст GitHub Release с файлами:

- `alice-go_linux-arm64.tar.gz` — 64-битный Debian на Pi.
- `alice-go_linux-armv7.tar.gz` — 32-битная ОС на Pi 2/3/4/5.
- `SHA256SUMS` — контрольные суммы обоих архивов.

Внутри архивов — бинарник `alice`, README и инструкция для Pi. Данные `pb_data` в релиз не включаются. Теги с дефисом, например `v0.1.0-rc.1`, публикуются как prerelease. Для каждой новой версии используйте новый тег; повторная публикация существующего релиза завершится ошибкой.

Workflow использует Go `1.27.1`. Для публикации достаточно штатного `GITHUB_TOKEN`; дополнительные секреты не нужны. В репозитории должны быть разрешены GitHub Actions и право `contents: write` для release-job. Автоматического доступа к Pi у CI нет — установка выполняется вручную.

## Проверка навыка

```sh
curl -X POST http://127.0.0.1:8090/alice \
  -H 'Content-Type: application/json' \
  -d '{"version":"1.0","session":{"new":true,"session_id":"local-session","message_id":0,"skill_id":"local-skill"},"request":{"type":"SimpleUtterance","command":""}}'
```

Ответ:

```json
{
  "version": "1.0",
  "response": {
    "text": "Привет! Это заготовка навыка Алисы на Go. Скажите «помощь» или «выход».",
    "end_session": false
  }
}
```

Для следующей реплики установите `session.new: false`, увеличьте `message_id` и передайте `request.command`: `привет`, `помощь`, `что ты умеешь`, `выход`, `пока` или `хватит`.
Неизвестная реплика получает подсказку. `ButtonPressed` пока возвращает подсказку; обработку `payload` можно добавить в сценарии.

## Подключение к Алисе

1. Разверните приложение за HTTPS-прокси или откройте локальный порт через HTTPS-туннель.
2. Создайте навык в [консоли Яндекс Диалогов](https://dialogs.yandex.ru/developer/).
3. В настройках Backend укажите `https://<ваш-домен>/alice`.
4. Сохраните настройки и проверьте диалог на вкладке тестирования.

Webhook публичный: авторизация PocketBase для обращений Алисы не требуется. Обычный `localhost` недоступен серверам Яндекса.

## Структура и расширение

```text
cmd/alice/main.go             # запуск PocketBase и регистрация миграций
internal/alice/protocol.go    # используемая часть JSON-протокола Алисы
internal/alice/skill.go       # сценарии диалога
internal/server/server.go    # маршруты PocketBase и проверка запросов
```

Начните добавлять сценарии в `internal/alice/skill.go`. Логика диалога — обычная функция, независимая от HTTP. Типы протокола описывают только нужные поля; для NLU, карточек и состояния расширяйте их по документации.

Для хранения данных можно создавать коллекции в админ-панели и работать с ними через `core.App`. При запуске через `go run` включена автоматическая генерация Go-миграций в `migrations/`. После создания первой миграции добавьте в `cmd/alice/main.go` импорт:

```go
_ "github.com/HexPande/alice-go/migrations"
```

Храните миграции в Git: они компилируются в приложение и применяются PocketBase при запуске. В собранном бинарнике автогенерация по умолчанию выключена. Пока скелет не создаёт своих коллекций и не сохраняет диалоги.

## Разработка

```sh
make fmt       # форматирование
make test      # тесты с race detector
make build     # бинарник bin/alice
make package   # ARM-архивы и SHA256SUMS в dist/ (та же сборка, что в CI)
go vet ./...   # статический анализ
make lint      # go vet + golangci-lint (нужен golangci-lint v2 с поддержкой Go 1.27)
```

Запуск бинарника:

```sh
./bin/alice serve --http=127.0.0.1:8090 --dir=./pb_data
```

Документация: [PocketBase на Go](https://pocketbase.io/docs/go-overview/), [запрос Алисы](https://yandex.ru/dev/dialogs/alice/doc/ru/request), [ответ Алисы](https://yandex.ru/dev/dialogs/alice/doc/ru/response).
