# Последние изменения

> **Постоянный лог.** Этот файл живёт в репо и пополняется новыми пунктами после каждой значимой сессии. Цель — переносить контекст между машинами (работа с двух ПК) и между сессиями Claude/Codex без потерь. **Не удалять.**
> Формат: пункты пронумерованы по убыванию (свежие — сверху). Каждый пункт = одна сессия / одна логическая волна изменений.

## Что сделано

### 21. Разделение CLAUDE.md по режимам и иерархии (2026-04-29)

**Контекст.** Параллельно ведём разработку MCP-сервера и скилла `leadgen` (плюс смежные скиллы). Корневой `CLAUDE.md` (144 строки, 11.7 КБ) на 95% содержал runtime-инструкции для агента-пользователя скилла («бюджет — недельный в рублях», «атрибуция — LYDC», «не вызывай resume_campaign» и т. п.). На каждой dev-сессии этот балласт грузился в hot-path, плюс риск применить runtime-правило к dev-задаче.

**Решение.** Раздельные `CLAUDE.md` по уровням + явное разделение режимов **DEV (default)** и **RUNTIME (только по слэш-команде)**.

**W1. Новая структура CLAUDE.md (4 файла)**

- **`CLAUDE.md` (root, переписан)** — DEV-режим по умолчанию, ~80 строк. Объясняет: проект — разработка скилла + MCP, runtime-правила скилла применяются только по `/leadgen`, `/yandex-direct`, `/yandex-metrika`, `/vk-ads`, `/demand-research`, `/serp-monitor`, `/seo`. Без слэш-команды runtime-документы (`METRIKA-ADS-RULES.md`, `VK-ADS-RULES.md`, `LEGAL.md`, `PROJECTS.md`, `agent-*.md`) в контекст не грузить. Карта проекта с явной колонкой «Грузить при старте?». Старт-чеклист: читать последние 1–2 пункта `RECENT-CHANGES.md` первым делом. Двойное дерево скиллов и правило записи в журнал — оставлены здесь.
- **`server/CLAUDE.md` (новый)** — dev-инструкции MCP-сервера. Структура пакетов (`platform/direct`, `platform/imagegen` и т. д.), build/run/test, как добавить новый MCP-инструмент (с явным шагом «если массив > 10 — делай парный `summarize_*`»), конвенция компактных ответов, конфиг и секреты, SQLite, CI/CD (auto-release при пуше в main). Грузится автоматически только когда cwd в `server/`.
- **`.claude/skills/leadgen/CLAUDE.md` (новый)** — dev-инструкции скилла. **Явный заголовок: «Загружать только когда работаем над содержимым скилла. При обычной dev-сессии MCP — не загружать. При активации `/leadgen` — также не загружать (грузится `skill.md`).»** Правило двойного дерева, структура файлов скилла с пометками когда грузится в runtime, frontmatter ≤ 50 слов, lazy-load mantra, якоря шагов как стабильный контракт.
- **`TECHNICAL.md` (новый, ~150 строк)** — технические детали для отладки. **Заголовок: «НЕ ГРУЗИТЬ ПРИ СТАРТЕ СЕССИИ. Открывать только при сбое или поиске специфики API.»** Содержит: MCP runtime-архитектура (SSE, bind-mounts), Yandex Direct quirks (выжимка), VK Ads quirks, Imagegen quirks, поиск vs РСЯ — таблица отличий, известные сбои (5 кейсов: контейнер из старого worktree, `tokens.env` как директория, битые `;C\`-пути, SSE не подхватывает после rebuild, CalloutSetting через SET). Ссылается на `METRIKA-ADS-RULES.md`, `VK-ADS-RULES.md`, `LEGAL.md`, `PROJECTS.md`, `agent-*.md` как на «справочники по требованию».

**W2. Sync `AGENTS.md` для Codex**

`AGENTS.md` переписан под ту же логику: DEV по умолчанию, RUNTIME по слэш-команде, указатели на новые `CLAUDE.md`-уровни и `TECHNICAL.md`. Остался тонким — без дублирования контента.

**W3. Auto-memory обновлена**

В `MEMORY.md` (auto-memory Claude Code) добавлены два пойнтера, чтобы новые сессии моментально знали контекст:
- `project_history_log.md` — RECENT-CHANGES.md = канонический журнал, читать первым.
- `project_dev_vs_runtime_modes.md` — DEV/RUNTIME режимы и что не грузить превентивно.

**Эффект.**
- Hot-path dev-сессии: с ~12 КБ runtime-инструкций до ~3 КБ навигации. Освободилось ~9 КБ контекста под рабочие данные.
- Mode confusion устранена: dev-задача не подбирает runtime-правила «не вызывай resume_campaign» и т. п. Если они понадобятся — пользователь зовёт `/leadgen` или агент сам грузит `TECHNICAL.md` при поломке.
- Иерархия каскадирует автоматически: правишь `server/foo.go` — Claude получает root + server-CLAUDE; правишь скилл — root + skill-CLAUDE. Правильные инструкции в нужный момент.
- Cross-PC синхронизация: всё лежит в репо, новый ПК после `git pull` работает идентично.

**Не тронуто.** `skill.md` всех скиллов, `library/`, `references/`, `branches/`, `flow-steps.md` — это runtime-контент, остался как был. Ссылки на них из root `CLAUDE.md` теперь явно условные («только при активации скилла»).

**Файлы созданы / изменены:**
- ✏️ `CLAUDE.md` (переписан с нуля)
- ✏️ `AGENTS.md` (переписан)
- 🆕 `server/CLAUDE.md`
- 🆕 `.claude/skills/leadgen/CLAUDE.md`
- 🆕 `TECHNICAL.md`
- 🆕 `~/.claude/projects/C--git-leadgen-mcp/memory/project_history_log.md`
- 🆕 `~/.claude/projects/C--git-leadgen-mcp/memory/project_dev_vs_runtime_modes.md`
- ✏️ `~/.claude/projects/C--git-leadgen-mcp/memory/MEMORY.md` (добавлены два пойнтера)

### 20. Пересборка MCP + большая уборка корня (2026-04-29)

**Контекст.** Контейнер `leadgen-mcp` крутился из worktree `condescending-lumiere` на коммите `8bceda0` — отставал от `main` (`ab21581`) на 7 коммитов. Не было `summarize_*` инструментов, Network-стратегий РСЯ, `imagegen`. Параллельно — корень основного репо `C:\git\leadgen-mcp\` зарос мусором (34 untracked-объекта).

**W1. Миграция рантайма + пересборка**
- Остановил старый контейнер. Перенёс рабочие файлы из `condescending-lumiere/server/` в текущий worktree (`intelligent-chatelet-76b590/server/`): `config.yaml`, `filter_values.json`, SQLite — `filters.db` (фильтры посадочных URL) и `change_history.db` (журнал MCP).
- Создал `tokens.env` из шаблона (старый контейнер тоже работал без секретов в env — всё в `config.yaml`).
- `docker compose build` + `up -d` из этого worktree. Бинарь `/app/leadgen-mcp` от 2026-04-29, обе БД открылись, `:8080/sse` отвечает 200.
- Теперь активны: `summarize_campaigns/ads/adgroups/keywords` (−70–90% токенов на обзорах), `add_ad_image`/`delete_ad_images`, `generate_image`/`generate_banner_set`, Network-стратегии (`WB_MAXIMUM_CONVERSION_RATE` / `AVERAGE_CPA` / `WB_MAXIMUM_CLICKS`), `BidCeiling` для авто-стратегий, расширенный `get_ad_images` с `OriginalUrl`/`PreviewUrl`.

**W2. Чистка корня worktree (10 файлов, ~1.5 МБ)**

Удалены файлы без единой входящей ссылки в скиллах / MCP-коде / hot-path-документах:
- `usecases.md` (52 КБ), `parse_goals.py` (одноразовый, с захардкоженным путём к удалённой `tool-results`), `test_tool.sh`.
- `docs/tz-get_conversion_values-v2.md` (24 КБ, ТЗ закрытой задачи).
- `docs/history-architecture.{png,excalidraw}` (960 КБ), `docs/rsya-branch-flow.excalidraw`.
- `campaigns/omsk_poisk_vtorichka_odnokomnatnye_preview.md` (временный preview-черновик).
- `docs/setup-*.md` (8 файлов: claude-code, cline, codex, cursor, gemini-cli, openclaw, vscode-copilot, windsurf).

**Сохранены** (на них есть ссылки): `docs/rsya-research/` (упомянут в `rsya_defaults.md:266` как источник анализа 395 РСЯ-кампаний), `docs/skill_improvement_backlog.md` (упомянут в `skill.md`), `docs/project-flow-cjm.excalidraw` (восстановлен после ошибочного удаления), `docs/campaign_previews/` (mount-таргет imagegen). Все hot-path .md (`CLAUDE.md`, `AGENTS.md`, `agent-direct_wordstat.md`, `agent-vk.md`, `LEGAL.md`, `PROJECTS.md`, `METRIKA-ADS-RULES.md`, `VK-ADS-RULES.md`) — на месте.

**W3. Чистка корня основного репо `C:\git\leadgen-mcp\` (34 объекта)**

Корень оброс артефактами старой ad-hoc отладки (15.04.2026):
- 25 `.mjs`-скриптов: `add_keywords`, `add_negatives`, `analyze_wordstat`, 3× `check_campaign*`, `check_adgroup_detail`, `collect_benchmarks`, `create_campaign`, `debug_labels`, `deploy_groups`, `expand_semantics`, `fix_remaining`, `mcp_call`, `run_flow`, 7× `test_*`, 2× `update_omsk_*`.
- 1 `.sh`: `test_tool.sh`.
- 4 JSON-дампа: `semantics_final.json`, `wordstat_omsk_data.json`, `wordstat_round{1,2}_raw.json`.
- 1 лог: `campaign_creation_log.md`.
- 3 ломаные docker-артефакт-директории: `tokens.env\` (пустая, создалась когда docker не нашёл файл и сделал bind-mount как директорию), `tokens.env;C\` и `config.yaml;C\` (битые Windows-пути с двоеточием).

Все были untracked в git. Tracked-файлы (`*.md`, `LICENSE`, `.gitignore`, `.mcp.json`, директории) не тронуты — они принадлежат main-ветке (`f8399f2`, отстаёт от обновлённой main; синхронизировать отдельно).

**W4. Решение про лог-файл**

Поначалу удалил `RECENT-CHANGES.md` — старая шапка говорила «после синхронизации можно удалить». Пользователь поправил: файл нужен **постоянно**, как механизм переноса контекста между двумя ПК и между сессиями Claude/Codex. Шапка переписана: «Постоянный лог, не удалять». Дальше каждая значимая сессия добавляет новый пункт сверху.

**Эффект.** MCP работает на свежем коде с экономичными `summarize_*`. Корни обоих рабочих директорий чистые — меньше шума при `ls`, меньше токенов на навигационные команды.

### 19. Рефакторинг скилла leadgen для экономии контекста (2026-04-24)

Мотивация: один монолитный `skill.md` на 1893 строки грузился целиком при каждом запросе — ≈ 22k токенов на скилл без учёта references/library/mcp. Часть файлов в `mcp/` дублировали данные, которые и так отдают MCP-инструменты (`get_city_config`, `get_conversion_values`, `get_utm_config` и др.). Описание скилла в frontmatter занимало ~140 слов — висело в контексте на каждом ходу.

**Что сделано**

1. **Разбил `skill.md` на `branches/`** — каждая ветка в своём файле, лениво подгружается после Роутера 1+2:
   - `branches/create-search.md` (667 строк, C1–C10)
   - `branches/create-rsya.md` (251 строка, R1–R11)
   - `branches/analyze.md` (88 строк, A1–A6)
   - `branches/optimize-search.md` (390 строк, O0/O1/O3.1–O3.10)
   - `branches/optimize-rsya.md` (94 строки, OR.1–OR.5)
2. **`skill.md` стал роутером: 1893 → 349 строк (−82%)**. Оставлены только: шапка, Роутер 1, Роутер 2, таблица тематик, правила коммуникации, процессный контур (режимы / auto-review / validation mesh / read-back / output contract), INIT, правила безопасности, инструкции по централизованной истории, источники данных (MCP vs файлы), связь со скиллами, будущие расширения.
3. **Новый раздел «Правила ленивой загрузки»** в skill.md — явно прописано, когда какой reference/library-файл читать (`image_prompts.md` только на R6.5, `lessons_registry.md` только при ошибках API и т. д.). Агент больше не грузит всё «на всякий случай».
4. **Сжал `description` в frontmatter:** ~140 слов → ~50 слов. Триггеры сохранены, но плотнее — экономия ~200 токенов на каждом сообщении, где скилл активен.
5. **Вычистил `mcp/` от дублей MCP API.** Удалены 6 файлов (`counters.md`, `goals.md`, `utm_reference.md`, `semantic_clusters.md`, `sitelinks.md`, `site_filters.md`) — те же данные возвращают MCP-инструменты `get_*`. Оставлены `negative_keywords.md` и `blocked_placements.md` (их нет в MCP). Добавлен `mcp/README.md` с таблицей «что было → чем заменено».

**Эффект на контекст** (жадная загрузка на «создай РСЯ»):
- skill.md: 22k → 6k токенов (−73%)
- mcp/ дубли: 8–10k токенов → 0 (вычищено)
- описание в frontmatter: 200 → 70 токенов (на каждое сообщение)
- Суммарная экономия на старте сессии ≈ **25–30k токенов** (≈ 30% контекстного окна освобождается под рабочие данные).

**Обратная совместимость:** все якоря шагов (C1–C10, R1–R11, A1–A6, O0/O1/O3.*, OR.1–OR.5) сохранены внутри бранчей. Ссылки `mcp/goals.md` в бранчах заменены на `get_conversion_values`. Ссылки `mcp/negative_keywords.md` остались — файл не удалялся. IMPROVEMENT_PLAN.md не трогали (архивный документ).

### 18. Ветка R (РСЯ) в скилле leadgen + генерация картинок через OpenRouter (2026-04-24)

Большая волна: отдельная ветка для создания РСЯ-кампаний, генерация креативов через сторонний API, новые MCP-инструменты, первый e2e-тест.

**W1. Исследование: как Этажи настраивают РСЯ сейчас**

- Проанализировано 395 активных РСЯ-кампаний в 61 из 77 городов (общий расход ≈ 6.25M ₽/мес). 92% расхода — `TEXT_CAMPAIGN` с сетями, 8% — `CPM_BANNER_CAMPAIGN` (медийка).
- Разобраны настройки топ-10 РСЯ по расходу. Выявлены разбросы: UTM отсутствует в 40% кампаний (Краснодар/Казань/Омск/Курган), `ENABLE_AREA_OF_INTEREST_TARGETING` — разная политика, `BidCeiling` — только в Москве.
- Скачаны 15 картинок из топ-кампаний в `docs/rsya-research/` для визуального анализа. Выявлены 3 кластера стилей: фото/рендеры объектов (валидно), текст-тяжёлые баннеры (не наш путь), готовые фото. Источник найден: API adimages `OriginalUrl` / `PreviewUrl` (в MCP раньше не возвращались).

**W2. Скилл — новые файлы и правила**

Создано:
- `.claude/skills/leadgen/references/rsya_defaults.md` — дефолты РСЯ: консервативные бюджеты (Микро 3–5k, Стандарт 5–10k, Расширенный 15–25k, Премиум 40–80k ₽/нед), дерево решений по стратегиям, **коэффициент tCPA_РСЯ = CPA_поиска × 0.7**, правила ретаргета/LAL (опция, не дефолт), minusи (инфо-интент), чеклист валидации.
- `.claude/skills/leadgen/references/image_prompts.md` — промпт-билдер для OpenRouter: 7 визуальных направлений (V1 интерьер / V2 3D-план / V3 фасад / V4 дом / V5 участок / V6 коммерческая / V7 ключ), константы PHOTO_STYLE_BASE / NEGATIVE_HARD_BAN / CITY_ARCHITECTURE_HINTS, маппинг тематика → визуал, правило лимита 20 генераций на кампанию, валидация.
- `.claude/skills/leadgen/library/banner_titles.md` — 4 формулы заголовков РСЯ (отличаются от поиска — короче, benefit-forward), заготовки по 5 тематикам.
- `.claude/skills/leadgen/library/banner_texts.md` — 3 формулы текстов + уточнения callouts (доверие / скорость / финансы).
- `campaigns/template_rsya.md` — отдельный шаблон для РСЯ-кампаний с обязательной секцией «Креативы» и счётчиком генераций.

Обновлено:
- `.claude/skills/leadgen/skill.md`:
  - **Роутер 2 (канал размещения)** — ветвление «поиск vs РСЯ» по триггерам пользователя, таблица критичных отличий каналов.
  - **Автономный режим (auto vs review)** — явные триггеры «запусти», «сам», «без модерации» пропускают preview-шаги; validation mesh остаётся обязательной.
  - **Ветка 1-R (C1–C10 → R1–R11)** — полный флоу создания РСЯ: параметры, конфиг, анализ аналогов, широкая семантика, кластеризация по аудитории (не интенту), preview R5.5, контент R6 с визуал-стилем, генерация картинок R6.5, создание через API R7 с add_ad_image, аудиторные таргеты R8 (опция), метки, документирование, read-back/мониторинг.
  - **Ветка 3-R (O-RSYA)** — OR.1 чистка площадок, OR.2 A/B картинок, OR.3 аудиторные таргеты, OR.4 частотное ограничение, OR.5 корректировки bid_modifiers.
- `.claude/skills/leadgen/flow-steps.md` — детальные таблицы для R1–R11 и OR.1–OR.5.

**W3. MCP-инструменты — новые и расширенные**

- **Расширен `get_ad_images`** (`server/platform/direct/content.go`): добавлены `OriginalUrl`, `PreviewUrl` в FieldNames + параметр `with_urls`. Было — только хеши без URL, теперь можно сразу получать ссылки на изображения.
- **Новый `server/platform/direct/images.go`** — 2 инструмента:
  - `add_ad_image` — загрузка картинки в `adimages/add`. Источник: URL (скачает сам), file_path, или image_base64. Возвращает `AdImageHash`. Лимит 10 МБ.
  - `delete_ad_images` — удаление неиспользуемых хешей.
- **Новый пакет `server/platform/imagegen/`** — генерация картинок через OpenRouter:
  - `client.go` — HTTP-клиент к `openrouter.ai/api/v1/chat/completions` с `modalities:["image","text"]`, парсит ответ (`choices[0].message.images[].image_url.url` с `data:image/...;base64,...`), поддерживает direct URL fallback.
  - `tools.go` — 2 MCP-инструмента: `generate_image` (одиночная генерация с сохранением в preview dir), `generate_banner_set` (пакетная генерация одного промпта в нескольких aspect ratios с вариантами).
- **Регистрация:** `server/mcp/setup.go` расширен для приёма `*imagegen.Client` и `preview_dir`. `server/main.go` создаёт клиента из конфига. `server/platform/direct/tools.go` регистрирует `RegisterImageTools`.
- **Конфиг:** `server/config/config.go` — новая секция `openrouter.api_key` + `server.preview_dir`. Env fallback `OPENROUTER_API_KEY`. `docker-compose.yml` монтирует `./docs/campaign_previews:/app/previews` — generated files видны на хосте.

**W4. Доработка add_campaign — Network-стратегия для РСЯ**

- До: `Network` формировался как `{BiddingStrategyType: <name>}`, без вложенной структуры. Для РСЯ с `WB_MAXIMUM_CONVERSION_RATE` / `AVERAGE_CPA` API возвращал ошибку «Стратегия должна содержать структуру с настройками».
- После (`server/platform/direct/campaigns.go`): добавлен switch по `networkStrategy` с полями `WbMaximumConversionRate{WeeklySpendLimit, GoalId}`, `AverageCpa{WeeklySpendLimit, AverageCpa, GoalId}`, `WbMaximumClicks{WeeklySpendLimit}`. Новые параметры `network_weekly_budget`, `network_average_cpa` (если не заданы — наследуются от search-аналогов). `start_date` по умолчанию — сегодня (избегаем ошибки 5003).

**W5. Первый e2e-тест: Омск-вторичка**

Создана тестовая РСЯ-кампания `Омск | РСЯ | Вторичка | Общая | [site]` (id `709316305`) в клиенте `porg-cyjuzztm`:
- Бюджет 10 000 ₽/нед, `Search=SERVING_OFF`, `Network=WB_MAXIMUM_CONVERSION_RATE`, `PriorityGoals` = форма 6443 ₽ + звонок 1869 ₽ (из `get_conversion_values`).
- `ENABLE_AREA_OF_INTEREST_TARGETING=YES` (регион — ловим приезжих), UTM на уровне кампании, 47 минус-слов.
- 2 группы (`Общие-купить` 5745089037, `2-комнатные` 5745089038) с 12 и 6 ключевыми, автотаргетингом EXACT+ALTERNATIVE.
- 6 объявлений с 3 картинками (картинки переиспользуются между группами — 1 картинка → 2 группы). Sitelinks (1472621032) + 4 callouts (42591619..22).
- Метки: `Лидген`, `Вторичка`, `Покупатель`, `РСЯ`.
- Status: DRAFT — ожидает модерации.

**Картинки** сгенерированы через OpenRouter, модель `google/gemini-2.5-flash-image`:
- V1.1 гостиная 1:1 (`uguH254sX390VOmwaN_TcA`) — фото современного интерьера.
- V1.1 гостиная 16:9 (`0CJ7iBxiwezPNQusP22xUA`) — широкоформат.
- V3 фасад ЖК Омск 16:9 (`qLoxFHy1BPe6D2glOzgDmA`) — панельно-кирпичный дом + двор + детская площадка.
- Итого **3 картинки** из лимита 20, стоимость **~$0.12**. Попытка V2 top-down floor plan — Gemini 2.5 Flash вернул текст вместо картинки, отложено на Gemini 3 Pro.

Превью-файлы: `docs/campaign_previews/omsk_rsya_vtorichka/` (примонтировано в контейнер через `/app/previews`).
Документация: `campaigns/omsk_rsya_vtorichka.md`.
Запись в history: `update_daily_summary` + `log_change_event` зафиксированы.

**Известные ограничения тестового запуска:**
- В текущей Claude-сессии MCP-клиент не видит новые инструменты (SSE-соединение инициализируется один раз). Тест проведён: картинки сгенерированы + загружены через прямой curl/Python к OpenRouter и `adimages/add`; кампания создана через Python-обёртку над API v5 (т.к. до фикса Go-кода Network-стратегия не строилась). В следующей сессии Claude Code новые MCP-инструменты (`add_ad_image`, `generate_image`, `generate_banner_set`) будут доступны нативно.

**W6. BidCeiling для Network и Search автостратегий**

Добавлена полная поддержка верхнего потолка ставки клика (`BidCeiling`) для автостратегий в `add_campaign` и `update_campaign`.

- **Что это:** жёсткий лимит ставки клика, которую автобиддер Директа не может превысить. Страховка от перекрута CPC на горячих аукционах (особенно в первые дни обучения стратегии). Анализ показал, что все московские РК Этажей используют BidCeiling 200–1200 ₽, но ни одна региональная — пропасть в настройках.
- **Где применим:** `WB_MAXIMUM_CONVERSION_RATE`, `AVERAGE_CPA`, `WB_MAXIMUM_CLICKS`. Не применим к `SERVING_OFF`, `NETWORK_DEFAULT`, ручным стратегиям.
- **Новые параметры `add_campaign`:** `search_bid_ceiling`, `network_bid_ceiling` (рубли). Нижний порог API — 0.3 ₽ (300 000 микро), контролируется helper-функцией `bidCeilingMicros()`.
- **Новые параметры `update_campaign`:** те же + блок Network полностью переработан (раньше покрывал только Search). Теперь можно обновлять `network_strategy`, `network_weekly_budget`, `network_average_cpa`, `network_bid_ceiling` через MCP. Нюанс API: при частичном обновлении BidCeiling нужно продублировать `network_strategy` — Yandex требует Network-блок целиком.
- **Формула для ветки R:** `network_bid_ceiling = tCPA × 1.5`, где `tCPA = CPA_поиска_факт × 0.7`. Для Омска-вторички: CPA_звонок 1869 ₽ → tCPA 1308 ₽ → потолок ≈ 1962 ₽.
- **Обновлены файлы скилла:**
  - `.claude/skills/leadgen/references/rsya_defaults.md` — раздел «Параметры создания для API» переписан, добавлен пункт в чеклист валидации.
  - `.claude/skills/leadgen/skill.md` — шаг R7 теперь включает `network_bid_ceiling` в вызов `add_campaign`.
- **Сборка и рестарт:** `docker compose up -d --build`, health=200. Новые инструменты доступны в схемах MCP (`mcp__yandex-direct__add_campaign`, `update_campaign`) в следующей Claude-сессии.

### 17. Синхронизация проектной информации (2026-04-24)

- Просмотрена текущая структура репозитория: Go MCP-сервер, skills, campaign-файлы, docs, правила Direct/VK/Legal.
- Зафиксирован фактический состав MCP-инструментов: всего 162, из них Direct 109, Metrika 11, Wordstat 5, VK Ads 30, filters 3, history 4.
- Заполнен `PROJECTS.md`: продукт Этажи, сайты, аккаунты, счётчики, цели, CPA-ориентиры, гео, naming и правила кампаний.
- Обновлён `README.md`: актуальные счётчики инструментов, модули filters/history/forecast, структура данных.
- Исправлена компиляционная ошибка в `server/platform/direct/references.go`: в `fmt.Sprintf` для fallback-сообщения `get_city_config` передан отсутствовавший `cityName`.
- Проверка `go test ./...` в `server/` проходит.
- Текущее рабочее дерево уже содержит незакоммиченные изменения в `.claude/skills/*`, `campaigns/template.md`, `tokens.env.example`, новые audit/SERP артефакты в `campaigns/` и новую CJM-схему в `docs/`. Эти изменения не откатывались и считаются текущей рабочей реальностью.

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
- Было: ~150 инструментов. Сейчас: **162** (Direct 109, Metrika 11, Wordstat 5, VK Ads 30, filters 3, history 4).
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
