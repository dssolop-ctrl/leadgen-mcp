# MCP Server — Dev Notes

> Dev-инструкции для Go-сервера `leadgen-mcp` (обёртка над Yandex Direct/Metrika/Wordstat и VK Ads). Этот файл грузится автоматически, когда Claude работает в `server/`. Корневой `../CLAUDE.md` тоже остаётся в контексте.

## Структура пакетов

| Путь | Назначение |
|---|---|
| `main.go` | Точка входа: загрузка `config.yaml`, инициализация SQLite, регистрация MCP-инструментов, старт SSE на `:8080`. |
| `config/config.go` | Парсинг `config.yaml` + env-фолбэки (`OPENROUTER_API_KEY` и т. п.). При добавлении новой секции конфига — править здесь. |
| `mcp/setup.go` | **Регистрация всех инструментов сервера.** Сюда добавляются вызовы `Register*Tools` из подпакетов. |
| `platform/direct/` | Yandex Direct API v5: `campaigns.go`, `ads.go`, `adgroups.go`, `keywords.go`, `summarize.go`, `images.go`, `content.go`, `extensions.go` и т. д. |
| `platform/metrika/` | Yandex Metrika API. |
| `platform/wordstat/` | Wordstat API (часть единого Yandex Direct токена). |
| `platform/vk/` | VK Ads API (отдельный токен). |
| `platform/imagegen/` | Генерация креативов через OpenRouter (`client.go`, `tools.go`). |
| `auth/` | Token storage helpers. |
| `data/` | Bind-mount: SQLite (`filters.db`, `change_history.db`) + JSON-сиды (`filter_values.json`, `network_benchmarks.json`). |
| `config.yaml.example` | Шаблон конфига. Перед первым запуском: `cp config.yaml.example config.yaml` и заполнить токены. |
| `Dockerfile`, `Dockerfile.simple` | Multi-stage build (Go → distroless/alpine). |
| `Makefile` | Локальные сокращения. |

## Build & Run

```bash
docker compose build         # пересборка образа из текущего worktree
docker compose up -d         # запуск (порт 8080)
docker logs leadgen-mcp -f   # логи
docker compose down          # остановка
```

После пересборки SSE-клиент (Claude Code, Codex) **не подхватит новые инструменты автоматически** — требуется перезапуск клиента или хотя бы реконнект SSE-сессии.

## Добавить новый MCP-инструмент

1. **Реализация** — в подходящем `platform/<area>/<file>.go`. Сигнатура: handler принимает `mcp.CallToolRequest`, возвращает `*mcp.CallToolResult, error`.
2. **Регистрация** — добавить вызов `mcp.AddTool(s, ...)` либо в `Register*Tools` пакета, либо напрямую в `mcp/setup.go`.
3. **Описание** — короткое, с явными `required` параметрами. Длинные описания съедают контекст у клиента.
4. **Если возвращается массив > 10 записей** — сделать парный `summarize_*` (паттерн в `platform/direct/summarize.go`). Это норма, не опция: на обзорных запросах экономит 70–90% токенов клиента.
5. **Перебилд + рестарт** контейнера, перезапуск MCP-клиента.

## Конвенция: компактные ответы

- `get_*` — полные структуры (для точечной работы).
- `summarize_*` — сжатые сводки: `id`, `name`, `state` + ключевые поля. Для любого инструмента, возвращающего > 10 записей.
- `field_names` — для обзоров `["Id", "Name"]`, для анализа — нужный набор. Не запрашивать «все поля» без цели.

## Конфиг и секреты

- `server/config.yaml` (bind-mount, read-only внутри контейнера) — токены Yandex/VK.
- `tokens.env` (env_file в docker-compose) — секреты, нужные как env vars (`OPENROUTER_API_KEY`, `YANDEX_API_KEY`/`YANDEX_CLOUD_FOLDER_ID` для SERP). **Файл должен существовать**, иначе Docker создаёт пустую директорию вместо bind-mount файла.
- `tokens.env.example` — шаблон.

## SQLite

- `data/filters.db` — фильтры посадочных URL. Сид: `data/filter_values.json`. После каждого upsert сервер переписывает JSON для git-трекинга — историю изменений видно в обычном `git diff`.
- `data/change_history.db` — журнал MCP-операций (`update_daily_summary`, `log_change_event`). Не коммитится.
- Оба открываются в `main.go` при старте.

## CI/CD

`.github/workflows/release-on-push.yml` — при push в `main` создаётся GitHub Release с архивом и тегом `vX.Y.Z+1` (patch-инкремент от последнего тега). Для minor/major — тег вручную:

```bash
git tag v1.3.0 && git push origin v1.3.0
```

Дальше patch-релизы пойдут от него.

## Известные нюансы API

См. `../TECHNICAL.md`, раздел «Yandex Direct quirks» и «Imagegen — известные ограничения».

## История

См. `../RECENT-CHANGES.md` — там пункты с тегами `MCP-сервер` (изменения Go-кода).
