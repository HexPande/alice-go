# alice-go

Основа для собственного навыка Яндекс Алисы на Go. Начните с простого диалога и добавляйте свои сценарии.

[Релизы](https://github.com/HexPande/alice-go/releases) · [Ошибки и предложения](https://github.com/HexPande/alice-go/issues)

## Возможности

- Подключение к Яндекс Диалогам.
- Базовые сценарии: приветствие, помощь и завершение разговора.

## Системные требования

- **ОС:** Debian или Ubuntu с systemd.
- **Архитектура:** ARM64 (`arm64`) или ARMv7 (`armhf`).
- Доступ к интернету и права администратора (`sudo` или root).

Готовые сборки доступны для Linux ARM. Установка Go не требуется.

## Установка

Установите `curl` и корневые сертификаты:

```sh
sudo apt update
sudo apt install -y curl ca-certificates
```

Запустите установщик и следуйте подсказкам:

```sh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/HexPande/alice-go/main/install.sh)"
```

## Обновление

Повторно запустите установщик:

```sh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/HexPande/alice-go/main/install.sh)"
```

## Подключение к Алисе

1. Настройте публичный HTTPS-адрес приложения через nginx или другой reverse proxy.
2. Создайте навык в [консоли Яндекс Диалогов](https://dialogs.yandex.ru/developer/).
3. Укажите адрес обработчика: `https://ваш-домен/alice`.
4. Проверьте диалог на вкладке тестирования.

## Обратная связь

Нашли ошибку или хотите предложить новый сценарий? [Создайте issue](https://github.com/HexPande/alice-go/issues/new) и опишите ожидаемое поведение, версию приложения и операционную систему.
