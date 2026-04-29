# Technical Notes — отладка и тонкости

> ⚠️ **НЕ ГРУЗИТЬ ПРИ СТАРТЕ СЕССИИ.** Этот файл — на лениво, по требованию. Открывать только когда:
> - что-то сломалось и нужно понять архитектуру runtime;
> - нужна специфика API (поля, лимиты, attribution и т. п.);
> - разбор конкретного известного сбоя или нестандартного кейса.
>
> Файл большой; превентивная загрузка съедает контекст. Если задача — типичная dev-работа (фикс кода, правка скилла, чистка), сюда не лезть.

## Что искать здесь

| Тема | Раздел |
|---|---|
| Yandex Direct API нюансы | «Yandex Direct quirks» + полный справочник `METRIKA-ADS-RULES.md` |
| VK Ads API нюансы | «VK Ads quirks» + `VK-ADS-RULES.md` |
| Архитектура MCP-сервера в runtime | «MCP runtime» |
| Bind-mounts, SQLite, env vars | «Конфиг и хранилище» |
| Imagegen — известные проблемы | «Imagegen quirks» |
| Сбои контейнера / SSE / Docker | «Известные сбои» |
| Юр. правила контента | `LEGAL.md` (ссылка) |
| Бизнес-настройки CPA, целей | `PROJECTS.md` (ссылка) |
| Roadmap / backlog скилла | `docs/skill_improvement_backlog.md` |

---

## MCP runtime

- **Транспорт:** SSE на `:8080/sse`. Клиент держит долгое соединение, сервер пушит события `tool_call_result`. Новые `mcp.AddTool` появляются у клиента **только после реконнекта** — недостаточно перезапустить сервер.
- **Bind-mounts (docker-compose.yml):**
  - `./server/config.yaml` → `/app/config.yaml` (read-only)
  - `./server/data/` → `/app/data/` (rw, SQLite + JSON-сиды)
  - `./docs/campaign_previews/` → `/app/previews/` (rw, генерируемые баннеры из imagegen)
- **env_file:** `./tokens.env` — обязателен. Если файл отсутствует, Docker создаёт пустую директорию `tokens.env/` вместо bind-mount файла (см. «Известные сбои»).

## Конфиг и хранилище

- `server/config.yaml`: токены (Yandex `default + agent`, VK `agency_client`), `bearer_token` сервера (опц.).
- `tokens.env`: `OPENROUTER_API_KEY` (imagegen), `YANDEX_API_KEY` + `YANDEX_CLOUD_FOLDER_ID` (SERP через Yandex Search API), `SITE_DOMAIN`, `YDIRECT_TOKEN` (для скриптов в `docs/`).
- **SQLite:**
  - `data/filters.db` — фильтры посадочных URL (район/комнаты/цена). Сид — `data/filter_values.json` (коммитится в git). При первом старте с пустой БД сервер импортирует JSON. После каждого upsert переписывает JSON, чтобы git-история отражала изменения.
  - `data/change_history.db` — журнал MCP-операций (`update_daily_summary`, `log_change_event`). В git не коммитится.

## Yandex Direct quirks (краткая выжимка, полные правила в METRIKA-ADS-RULES.md)

- **Бюджет** — недельный, в **рублях** (число), не умножать на 1 000 000 (старая ошибка с микроюнитами).
- **Атрибуция отчётов** — всегда `LYDC` в `get_campaign_stats` и `get_search_queries`.
- **UTM** — через `tracking_params` на уровне группы.
- **Места показа поисковой кампании** — `network_strategy: "SERVING_OFF"`.
- **РСЯ Network-стратегии** требуют вложенной структуры:
  - `WB_MAXIMUM_CONVERSION_RATE { WeeklySpendLimit, GoalId }`
  - `AVERAGE_CPA { WeeklySpendLimit, AverageCpa, GoalId }`
  - `WB_MAXIMUM_CLICKS { WeeklySpendLimit }`
  - Без вложенной структуры API возвращает ошибку «Стратегия должна содержать структуру с настройками». Реализовано в `server/platform/direct/campaigns.go` (switch по `networkStrategy`).
- **GEO-кампании** — отключать `ENABLE_AREA_OF_INTEREST_TARGETING` (иначе показы за пределами региона).
- **Запрет авто-активации:** после `add_campaign` НЕ вызывать `resume_campaign`. Кампания остаётся в DRAFT/SUSPENDED. Активирует пользователь после ревью текстов/креативов/настроек. То же — для `update_campaign` (не должен случайно перевести в ON).
- **Получение кампаний:** дефолт `states: ["ON"]`, не ставить маленький `limit`. `field_names` — для обзора `["Id", "Name"]`, для анализа — нужный набор.
- **Автотаргетинг** — только целевые + узкие, остальные выключены.
- **Стратегии** — старт с `WB_MAXIMUM_CONVERSION_RATE`, переход на `AVERAGE_CPA` после 10+ конверсий/нед.
- **Изменения** цели/стратегии/бюджета >30% — только с подтверждением пользователя, лучше в чт-пт (даём выходные на стабилизацию).
- **Имена целей и счётчиков** — никогда голым ID. Формат: «цель Отправка заявки (123456789)».

## VK Ads quirks (полные правила — `VK-ADS-RULES.md`)

- Запрещённые символы в текстах, лимиты длины, формат API — см. справочник.
- Отдельный токен от Yandex (хранится в `config.yaml` под `accounts.vk`).

## Imagegen quirks

- Модель по умолчанию: `google/gemini-2.5-flash-image`. Иногда возвращает текст вместо картинки на сложных промптах (top-down floor plan и т. п.) — есть авто-фолбэк на резервную модель (см. `platform/imagegen/client.go`).
- **Pre-upload валидация** — формат + размер ≤ 10 МБ — до вызова `adimages/add`. Чтобы не ловить ошибку API.
- **Soft-лимит** на стороне скилла — 20 генераций на одну кампанию (см. `references/image_prompts.md`).
- **Modalities-aware:** клиент проверяет, поддерживает ли модель `modalities:["image","text"]`. Если нет — переключается на direct URL fallback.

## Места показа: поиск vs РСЯ

| Параметр | Поиск | РСЯ |
|---|---|---|
| `network_strategy` (Search) | n/a | `SERVING_OFF` |
| Search-стратегия | автостратегия по конверсиям | n/a |
| Network-стратегия | n/a | `WB_MAXIMUM_CONVERSION_RATE` / `AVERAGE_CPA` / `WB_MAXIMUM_CLICKS` |
| `BidCeiling` | опционально (Москва, дорогой трафик) | опционально |
| Креативы | текстовые объявления + sitelinks | + картинки (через imagegen + `add_ad_image`) |
| Коэф. tCPA | 1.0 от целевого CPA | × 0.7 от поискового |

## Известные сбои

### 1. Контейнер запущен из другого worktree, отстаёт от `main`

**Симптом:** клиент не видит новых инструментов (`summarize_*`, `add_ad_image` и т. п.).
**Причина:** `docker compose up` был сделан из старого worktree, в котором отсутствуют последние коммиты.
**Лечение:** `docker stop leadgen-mcp && docker rm leadgen-mcp`, перейти в актуальный worktree (последний main), скопировать рабочее состояние (`server/config.yaml`, `server/data/*.db`, при необходимости — `tokens.env`), `docker compose up --build -d`. См. `RECENT-CHANGES.md` #20.

### 2. `tokens.env` как директория

**Симптом:** контейнер не стартует или стартует без env vars.
**Причина:** docker-compose делает bind-mount `./tokens.env`. Если файла нет, Docker создаёт пустую директорию с этим именем.
**Лечение:** удалить директорию, `cp tokens.env.example tokens.env`, заполнить нужные ключи.

### 3. Битые пути с двоеточием на Windows

**Симптом:** в корне репо появляются директории `tokens.env;C\`, `config.yaml;C\`.
**Причина:** некорректное экранирование пути с двоеточием в каком-то batch/PS-скрипте.
**Лечение:** удалить руками — это пустые артефакты.

### 4. SSE-клиент не подхватывает новые инструменты после rebuild

**Симптом:** сервер пересобран, новые tools зарегистрированы в `mcp/setup.go`, но Claude Code их не видит.
**Причина:** SSE-клиент держит сессию, инициализированную до rebuild.
**Лечение:** перезапустить Claude Code (или хотя бы переоткрыть сессию). После этого клиент сделает свежий `tools/list`.

### 5. CalloutSetting не обновляется через `update_ad`

**Симптом:** `update_ad` с `ad_extension_ids` не подменяет callouts.
**Причина:** API требует SET вместо ADD/REMOVE для CalloutSetting (исправлено в `df5361f`).
**Уроки:** см. `references/lessons_registry.md`.

## Ссылки на специализированные справочники

- **`METRIKA-ADS-RULES.md`** (~37 КБ) — полные правила Yandex Direct API + Метрика. Грузить только при работе с конкретным API-вызовом, для которого нужна точность.
- **`VK-ADS-RULES.md`** (~3.6 КБ) — VK Ads.
- **`LEGAL.md`** (~3.4 КБ) — юр. правила контента (запрещённые формулировки, альтернативы).
- **`PROJECTS.md`** (~13.7 КБ) — бизнес-настройки: счётчик Метрики, цели, CPA-пороги по проектам.
- **`agent-direct_wordstat.md`** (~16 КБ) — агентские инструкции по работе с Yandex Direct + Wordstat.
- **`agent-vk.md`** (~10 КБ) — агентские инструкции по VK Ads.
- **`docs/skill_improvement_backlog.md`** (~72 КБ) — план доработки скилла.
- **`docs/rsya-research/`** (~8 МБ jpg) — визуальный анализ 395 РСЯ-кампаний Этажей.

## История проблем и решений

В `RECENT-CHANGES.md` (журнал волн изменений) и `references/lessons_registry.md` каждой ветки скилла (узкие уроки от ошибок API).
