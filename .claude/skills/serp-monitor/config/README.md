# Настройка Yandex Search API v2

Старый `xml.yandex.ru` закрывается (API v1 отключают 30.09.2025). Рабочий путь один — **Search API v2 через Yandex Cloud / AI Studio**. Авторизация — **API-ключ сервисного аккаунта** (не истекает, проще всего).

## Шаг 1. Аккаунт Yandex Cloud

1. Зайти в https://console.yandex.cloud/, залогиниться Яндекс-аккаунтом.
2. Принять условия, создать **платёжный аккаунт** — без него API не даст ключ, даже в пределах промо-гранта.
3. Новому пользователю обычно даётся грант ~4000 ₽ / 60 дней — этого хватит на тесты.
4. Автоматически создаются **облако** (cloud) и **каталог** (folder). Скопировать `folder_id`:
   - Console → слева дерево → ваш каталог → правая панель → «Скопировать ID».
   - Формат: `b1gxxxxxxxxxxxxxxxxx`.
   - **→ `YANDEX_CLOUD_FOLDER_ID` в `tokens.env`**

## Шаг 2. Подключить Search API в каталоге

1. Cloud Console → меню сервисов → раздел **AI Studio** → **Search API**.
2. Нажать «Начать пользоваться» / «Подключить», принять соглашение.

Без этого шага запросы вернут `PERMISSION_DENIED` или `NOT_FOUND`.

## Шаг 3. Создать сервисный аккаунт

1. Cloud Console → каталог → слева **Identity and Access Management** → **Сервисные аккаунты** → **Создать сервисный аккаунт**.
2. Имя: `serp-monitor` (любое, латиница).
3. **Назначить роль**: `ai.searchApi.executor`.
   - Если в UI не нашли — добавить на уровне каталога: IAM → «Права доступа» → «Настроить доступ» → выбрать сервисный аккаунт → роль `ai.searchApi.executor`.
4. Сохранить.

## Шаг 4. Получить API-ключ

**Вариант А — через UI:**
1. Открыть сервисный аккаунт → вкладка **API-ключи** → **Создать ключ**.
2. **Скопировать `secret` один раз** (повторно не покажут).
   - **→ `YANDEX_API_KEY` в `tokens.env`**

**Вариант Б — через `yc` CLI:**
```bash
yc iam api-key create --service-account-name serp-monitor
```
Секрет — поле `secret` в ответе.

## Шаг 5. Заполнить `tokens.env`

В корне проекта:
```env
YANDEX_API_KEY=<секрет из Шага 4>
YANDEX_CLOUD_FOLDER_ID=<folder_id из Шага 1>
SITE_DOMAIN=etagi.com
```

## Шаг 6. Проверка

```bash
scripts/search.sh --query "купить квартиру омск" --region 66
```

Успех — markdown-таблица top-20 с подсветкой ★ etagi.com.

**Ошибки:**

| Код / симптом | Причина | Фикс |
|---|---|---|
| `UNAUTHENTICATED` / HTTP 401 | API-ключ невалиден или неверно скопирован | Пересоздать ключ в Шаге 4, вставить без пробелов |
| `PERMISSION_DENIED` / HTTP 403 | Нет роли `ai.searchApi.executor` или Search API не подключён в каталоге | Шаги 2 и 3 |
| `NOT_FOUND` (endpoint) | Search API не подключён в каталоге | Шаг 2 |
| `QUOTA` / `LIMIT` / HTTP 429 | Исчерпана квота / RPS | Повторить через минуту, проверить лимиты каталога |
| `ERROR: YANDEX_API_KEY не задан` | Нет `tokens.env` или переменной | `cp tokens.env.example tokens.env`, заполнить |

## Квоты и цены

Search API v2 — платный. Цены и лимиты RPS/сутки смотреть в каталоге: Cloud Console → AI Studio → Search API → «Лимиты». Промо-гранта новому аккаунту хватает на сотни запросов.

Бесплатных 10 запросов/день, как было на `xml.yandex.ru`, больше нет.

## Устаревшее (не использовать)

В старой версии скилла были переменные `YANDEX_XML_USER`, `YANDEX_XML_KEY`, `YANDEX_CLOUD_OAUTH_TOKEN` и скрипт `iam_token.sh`. Всё это больше не нужно:

- `xml.yandex.ru` закрывается — XML user+key отклоняются новым API
- OAuth-токен физлица заменён на API-ключ сервисного аккаунта (не истекает vs 12h для OAuth→IAM)
- `iam_token.sh` удалён — с API-ключом IAM не нужен

Если ранее заполняли эти переменные в `tokens.env` — можно удалить.

## Ссылки

- Search API docs: https://yandex.cloud/ru/docs/search-api/
- AI Studio docs: https://aistudio.yandex.ru/docs/ru/search-api/quickstart
- Коды регионов (`lr`): https://yandex.cloud/ru/docs/search-api/concepts/regions
- IAM API keys: https://yandex.cloud/ru/docs/iam/concepts/authorization/api-key
