# Последние изменения (2026-04-15)

> Этот файл — контекст для продолжения работы с другого компьютера.
> После синхронизации контекста — файл можно удалить.

## Что сделано

### 1. MCP-сервер: оптимизация описаний инструментов
- Сжаты описания `account` (147 вхождений) и `client_login` (99 вхождений) во всех 36 Go-файлах
- 12+ описаний инструментов сокращены
- Результат: 107553 → 98604 символов (-8.3%, ~2237 токенов сэкономлено)
- Файлы: все `server/platform/**/*.go`

### 2. MCP-сервер: исправлен update_autotargeting
- **Проблема:** отправлял `AutotargetingCategories` через `AdGroups.update`, API возвращал ошибку
- **Решение:** переписан на работу через `Keywords` сервис (get→delete `---autotargeting`→add с категориями YES/NO)
- Файл: `server/platform/direct/keywords.go`

### 3. MCP-сервер: сжатие ответа get_ads
- **Проблема:** 150-258 КБ на кампанию из-за UTM-параметров в Href (~300 символов каждый)
- **Решение:** `stripUTMFromAdsResponse()` удаляет `utm_*` и `yclid` из Href в ответе
- Результат: 150 КБ → 63.5 КБ (-58%) для Омска (192 объявления)
- UTM трекинг не теряется — он задаётся через `tracking_params` на уровне группы
- Файл: `server/platform/direct/ads.go`

### 4. MCP-сервер: SQLite-модуль фильтров сайта
- Новый модуль `server/platform/filters/` — SQLite БД для фильтров посадочных URL
- 3 инструмента: `build_landing_url`, `upsert_site_filters`, `get_site_filters`
- Резолвит районы, комнатность, типы объектов в URL-параметры
- Файлы: `server/platform/filters/store.go`, `server/platform/filters/tools.go`

### 5. Skill.md: кросс-минусация
- **ВЕТКА 1 (Создание):** добавлен Шаг 8 — кросс-минусация между группами после создания
- **ВЕТКА 3 (Оптимизация):** добавлена подветка O3.6 — диагностика каннибализации

### 6. Skill.md: 3 объявления на группу
- Было: `add_ad × 2` → стало: `add_ad × 3` (базовое + 2 гипотезы)
- Обновлены все чеклисты во всех ветках

### 7. Skill.md: якоря для навигации
- Каждый шаг помечен HTML-комментарием: `<!-- C1 -->`, `<!-- A2 -->`, `<!-- O3.6 -->`
- Позволяет точечно указывать шаг для доработки: «доработай C7»
- 26 точек навигации: INIT, C1-C10, A1-A6, O0-O1, O3.1-O3.7

### 8. Skill.md: реорганизация файлов
- MCP-дублированные файлы перемещены в `mcp/` подпапку (reference-only)
- Удалены из основного скилла: counters, goals, utm_reference, blocked_placements, semantic_clusters, negative_keywords, sitelinks, site_filters
- В скилле остались только файлы, загружаемые агентом в контекст

### 9. Документация флоу
- `flow-steps.md` — структурированное описание каждого шага (якорь, MCP, файлы, вход/выход)
- `flow-diagram.excalidraw` — обзорная схема 3 веток
- `flow-steps-detailed.excalidraw` — детальная карта всех 26 шагов с инструментами и файлами

### 10. Skill.md: исправлены баги
- `add_ad_extension`: `callouts=` → отдельные вызовы `callout_text=`
- `get_ads`: добавлен `states="ON"` ко всем вызовам
- `update_autotargeting`: `true/false` → `YES/NO`
- Расширены триггерные слова роутера для всех 3 веток

### 16. Обратный инжиниринг yandex-performance-ops → leadgen (2026-04-20)

Волна по мотивам анализа стороннего скилла `yandex-performance-ops`. 4 волны изменений.

**W1. Переименование скилла**
- `.claude/skills/etazhi-direct/` → `.claude/skills/leadgen/` (через `git mv`)
- Обновлены ссылки: `SKILL.md` (frontmatter name + description), `README.md`, `METRIKA-ADS-RULES.md`
- Создано:
  - `.claude/skills/leadgen/references/lessons_registry.md` — реестр API-гочи (21 запись: `ads.add` vs `ads.get` по extensions, read-back после resume, `AVERAGE_CPA` без 10+ конв, batch-лимиты и др.)
  - `.claude/skills/leadgen/references/copy_blacklist.md` — запрещённые слова в текстах (жёсткий + мягкий списки: WhatsApp, VPN, «официальный сайт», «без СМС», «гарантия», «№1» и т.д.)

**W2. Процессный контур в SKILL.md**
Новая сквозная секция «Процессный контур: режимы работы, валидация, отчётность»:
- Session Modes (4 режима): `research` / `pre-apply-review` / `live-apply` / `post-apply-monitoring`. Agent называет режим в первом сообщении.
- Validation Mesh: три чеклиста перед live-apply (domain / coverage / apply-safety).
- Read-back отчёт после live-apply: было → сделал → стало, с read-back через `get_*`.
- Output Contract: единая структура для аудита/оптимизации (scope / findings / proposed / applied / next steps).

**W3. Централизованная история изменений (MCP)**
- Новый модуль Go: `server/platform/history/{store.go,tools.go}` — SQLite-хранилище.
- Две таблицы: `change_events` (append-only детальный лог критичных мутаций) + `daily_summaries` (одна запись на город на день, UPDATE в течение дня, INSERT на новый день).
- Древовидный ключ: `agency_account → city_login → campaign_id → event`.
- БД: `server/data/change_history.db` (в `.gitignore`).
- 4 MCP-инструмента:
  - `log_change_event` — записать критичное live-изменение (смена цели/стратегии, бюджет >30%, пауза/запуск)
  - `get_change_history` — получить детальную историю с фильтрами (campaign / city / agency / date / correlation_key)
  - `update_daily_summary` — обновить суточное саммари по городу (append/replace)
  - `get_daily_summary` — получить недавние саммари по городу
- Правила работы в SKILL.md секции «История изменений»: когда писать, когда читать перед правками.

**W4. Дополнительные инструменты**
- Wordstat L1→L2 waves: в разделе «Шаг 4. Сбор семантики» зафиксирован канонический порядок `ПРОДУКТ-МАПА → МАСКИ L1 → РЕВЬЮ → WAVE 1 → GAP → МАСКИ L2 → WAVE 2 → ВАЛИДАЦИЯ → КЛАСТЕРИЗАЦИЯ` и уровни масок L1-L4.
- `.claude/skills/leadgen/scripts/check_copy.py` — Python-валидатор текстов против `copy_blacklist.md`. Работает с CLI-параметрами или TSV. Exit code: 0 (чисто) / 1 (hard) / 2 (soft).
- `.claude/skills/leadgen/scripts/render_audit_report.py` — HTML-рендерер аудита с фиксированной версткой (шапка с KPI, badge-цвета для verdict, таблицы findings + proposals + applied + next_steps). Предсказуемый вид между запусками. Пример JSON-входа: `audit_example.json`.
- `server/platform/direct/forecast.go` — новый MCP-инструмент `forecast_campaign(campaign_id, horizon_days)`. Горизонты 3/7/15/30/90 дней, 95% CI (z=1.96), сезонный множитель по месяцам, опциональный override. Готовит baseline из daily-статистики через Reports API.

**Сборка и деплой**
- Docker-образ пересобран, контейнер перезапущен (`docker compose build && docker compose up -d`)
- Все 5 новых инструментов проверены через `test_tool.sh`:
  - `log_change_event` — событие записано, id=1
  - `update_daily_summary` — саммари создано на 2026-04-20
  - `get_change_history` — возвращает events
  - `get_daily_summary` — возвращает results
  - `forecast_campaign` — компилируется (проверка в проде — на реальной кампании)

**Итоги по инструментам MCP**
- Было: ~150 инструментов. Стало: **~155** (+4 history + 1 forecast).
- Новые таблицы SQLite: `change_events`, `daily_summaries` в `server/data/change_history.db`.

## Что НЕ сделано / на обсуждение
- vCard/визитка — решено НЕ добавлять
- Корректировки на мобильные — решено НЕ добавлять
- Поступательная доработка каждого шага флоу — следующий этап работы

## Docker
- Контейнер `leadgen-mcp` пересобран и работает с последними изменениями
- Команда запуска: `MSYS_NO_PATHCONV=1 docker run -d --rm --name leadgen-mcp -p 8080:8080 -v leadgen-mcp-data:/app/data -v "C:/git/leadgen-mcp/server/config.yaml:/app/config.yaml:ro" leadgen-mcp`

## Тестовые сущности в Яндекс Директе (Омск)
- Sitelink set `1471491080` (4 ссылки)
- Ad extension (callout) `42489359` ("Без комиссии")
- Autotargeting обновлён на группе `5742398648` (Нефтяники)
