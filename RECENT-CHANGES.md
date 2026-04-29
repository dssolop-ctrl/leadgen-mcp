# Последние изменения

> **Постоянный лог.** Этот файл живёт в репо и пополняется новыми пунктами после каждой значимой сессии. Цель — переносить контекст между машинами (работа с двух ПК) и между сессиями Claude/Codex без потерь. **Не удалять.**
> Формат: пункты пронумерованы по убыванию (свежие — сверху). Каждый пункт = одна сессия / одна логическая волна изменений.

## Что сделано

### 26. Тестовый прогон РСЯ #2 (Claude Desktop, opus 4.7): 6 фиксов по результатам (2026-04-29)

**Контекст.** Второй тестовый прогон создания РСЯ-кампании в Claude Desktop с обновлённым скиллом leadgen (после #24, #25). Сессия-лог пользователя зафиксировал 6 проблем итоговой кампании. Половина — Go-код MCP, половина — скилл.

**Проблема 1: чёрный список площадок не применился.** Корень: MCP имел только read-only `get_blocked_placements`. Скилл по `rsya_defaults` раздел 10 говорил «при создании пусто, чистка через 3 дня» — это дарило бюджет ~400 мусорным площадкам в самые дорогие дни обучения автостратегии.
Фикс: добавлен `apply_blocked_placements(client_login, campaign_id)` в `server/platform/direct/blocked_placements.go` — одним вызовом применяет 400+ площадок Этажи через `excludedsites.set`. Также `set_excluded_sites(...)` для произвольного списка (лимит 1000 хостов API). В `branches/create-rsya.md` добавлен обязательный шаг **R7.5** сразу после R7.

**Проблема 2: 406 минус-слов для РСЯ — слишком много.** Корень: `get_negative_keyword_guidance` для `placement="rsya"` всё равно подмешивал `other_cities_rf` (~120), `international_cis` (~21), `abroad_realty` (~23). В РСЯ это бесполезно — таргетинг идёт по `region_id` пользователя, не по тексту запроса; раздутый `negative_keywords` пожирает 20 КБ-лимит API.
Фикс: новый параметр `include_geo_blocks` в инструменте, дефолт **false для rsya**, true для search. Хард-гейт для rsya без гео-блоков снижен до ≥80 слов. Smoke-test для Омск-вторички-РСЯ: 406 → **243** слова (−40%). Скилл R4 обновлён под новый объём.

**Проблема 3: 3 группы по 4 keyword (семантика «не собиралась»).** Корень: `check_search_volume` имеет реальный API-лимит 10 фраз (описание врало «до 128»). Агент попробовал 12 → ошибка 241 → ручной чанкинг по 10 → собрал всего 20 кандидатов → после фильтра по Shows ≥100 осталось 16.
Фикс: chunking на стороне MCP. Tool теперь авто-разбивает любой список на батчи по 10, склеивает результаты, добавляет partial-error-репортинг. Описание обновлено («до 100 фраз за вызов, авто-чанкинг»). Скилл R4: рекомендация увеличена до 25–40 кандидатов на проверку, целевой объём 15–25 валидных, хард-гейт `<12 фраз → не запускать в auto, спросить пользователя`.

**Проблема 4: не было генераций изображений.** Корень: `OPENROUTER_API_KEY` не задан в контейнере. Агент откатился к переиспользованию существующих хешей через `get_ad_images` — формально картинки есть, но не верифицированы под тематику.
Фикс: документация. В `TECHNICAL.md` явный раздел «OpenRouter API key — обязательная конфигурация» с двумя путями установки (`tokens.env` или `server/config.yaml`). Это не code-fix — нужно действие пользователя.

**Проблема 5: ENABLE_AREA_OF_INTEREST_TARGETING=true.** Корень: в `rsya_defaults.md` раздел 5 регионы по дефолту получали YES «чтобы ловить приезжих». Для лидогенерации недвижимости это в основном утечка.
Фикс: дефолт переписан на **false**. Включение только по явному запросу пользователя. В R7 settings строка обновлена; раздел 5 в `rsya_defaults.md` переписан.

**Проблема 6: нет видео-креативов.** Корень: Yandex Direct авто-генерирует видео из картинок на серверной стороне после старта показов. Явного API-toggle нет (проверено: ни в `Campaigns.Settings`, ни в `AdGroupSettings`).
Фикс: документация. Добавлен раздел 11 в `rsya_defaults.md` и раздел в `TECHNICAL.md`. Кампанию не трогаем; диагностика через `creatives` инструмент при необходимости.

**Сборка и проверка.** `docker compose build` + рестарт. `tools/list` показывает новые `apply_blocked_placements`, `set_excluded_sites`. Smoke-test `get_negative_keyword_guidance(вторичка/омск/rsya)` → 243 words, include_geo_blocks=false, hard_gate=PASS.

**Файлы изменены:**
- 🆕 `server/platform/direct/blocked_placements.go` (~135 строк, 2 новых tool)
- ✏️ `server/platform/direct/negatives.go` (include_geo_blocks param + placement-aware default + динамический хард-гейт)
- ✏️ `server/platform/wordstat/handlers.go` (chunking по 10 + partial-error reporting)
- ✏️ `server/mcp/setup.go` (+1 регистрация)
- ✏️ `.claude/skills/leadgen/branches/create-rsya.md` (R4 целевые объёмы, R7 settings false, R7.5 apply_blocked_placements)
- ✏️ `.claude/skills/leadgen/references/rsya_defaults.md` (раздел 5 переписан, 10 расширен, 11 добавлен)
- ✏️ Codex-зеркала обоих скилл-файлов
- ✏️ `TECHNICAL.md` (OpenRouter + видео-генерация)
- ✏️ `RECENT-CHANGES.md` (этот пункт)

### 25. Поправки правил R2.5 и R5 в create-rsya: автономность вместо паузы, посадочная как мягкий сигнал (2026-04-29)

**Контекст.** После #24 пользователь дал две правки по существу:
1. Правило R5 «посадочная под сегмент существует» как **обязательное** ломает кейсы, где семантическая выделенность сегмента очевидна, а специализированной фильтрованной посадочной нет. В таких случаях группа всё равно осмысленна (использует базовую посадочную тематики).
2. Правило R2.5 «при ratio benchmark/network > 1.3 — стоп и спросить пользователя даже в auto» ломает автономную работу скилла (фразы «делай сам», «без вмешательства», ночные/массовые прогоны). Лучше иметь детерминированное правило выбора target.

**W1. R5 — посадочная стала мягким сигналом.**

В `branches/create-rsya.md` правило выделения сегментной группы переписано:
- **Жёсткое правило (единственное):** ≥3 фраз в R4 под сегмент.
- **Мягкий сигнал (не блокирует):** наличие специализированной посадочной через `build_landing_url` / `get_site_filters`. Если есть — используем как `href` объявлений группы. Если нет — fallback на базовую тематическую посадочную (`/realty/` для вторички), группу всё равно создаём.

Анти-паттерн «свернуть сегмент только из-за отсутствия фильтра» добавлен в явный список запрещённого. Таблица групп обновлена: «посадочная — желательна, но не обязательна».

**W2. R2.5 — детерминированное правило выбора target, без пауз.**

Старое правило вынуждало агента «спросить пользователя» при ratio > 1.3, что блокировало автономную работу. Новое правило применяется одинаково в `auto` и `review`, без пауз:

| ratio | target_form_cpa |
|---|---|
| < 0.7 | `BenchmarkFormCPA` (нормально низкий) |
| 0.7 – 1.3 | `BenchmarkFormCPA` (норма) |
| **1.3 – 2.0** | **`NetworkExpected`** (ориентир на норму tier-а) |
| > 2.0 | `NetworkExpected × 0.9` (сильная аномалия — стартуем чуть ниже нормы) |

Это правило:
- **Безопасно для бюджета** — если benchmark задран (например, 6000 ₽ при ожидании 3500 ₽), новая кампания не унаследует исторический потолок, а будет учить автостратегию на норме tier-а.
- **Детерминировано** — никаких «угадываний», для одинаковых входов всегда одинаковый target. Воспроизводимо в массовых прогонах.
- **Автоматизируемо** — не требует человека в петле, пригодно для cron-задач, оркестраторов, ночных пакетных запусков.
- В режиме `review` агент всё равно показывает расчёт в таблице решений — пользователь может перебить через `target_form_cpa_override`.

То же — для `BenchmarkCallCPA` против `NetworkExpected.call`.

**W3. lessons_registry.md — поправка к Блоку 0.**

Запись «0. РСЯ: BidCeiling — это клик, не CPA. И группы — после семантики, не до» обновлена:
- В пункте про R2.5 пауза «спросить пользователя» заменена на детерминированный выбор; явная пометка «(Поправка 2026-04-29: ранее в этом блоке стояла пауза — она ломала автономную работу)».
- В пункте про R5 хард-правило про посадочную смягчено: «при ≥3 фразах группа создаётся; посадочная — soft-сигнал для `href`, fallback на базовую тематическую».

**W4. Зеркало в `.codex/skills/leadgen-codex/`.**

Все три файла (`branches/create-rsya.md`, `references/rsya_defaults.md`, `references/lessons_registry.md`) синхронизированы. `diff -r` чистый.

**Эффект.**
- Скилл теперь полностью автономен в RSYA-флоу — пакетные прогоны не блокируются на каждой кампании с задранным benchmark.
- Сегментные группы не сворачиваются необоснованно при отсутствии фильтрованных посадочных — более точное использование собранной семантики.

**Файлы изменены:**
- ✏️ `.claude/skills/leadgen/branches/create-rsya.md` — R2.5 (детерминированный выбор), R5 (посадочная как soft-сигнал)
- ✏️ `.claude/skills/leadgen/references/rsya_defaults.md` — раздел «Sanity-check бенчмарка» — таблица + объяснение «почему детерминированно»
- ✏️ `.claude/skills/leadgen/references/lessons_registry.md` — поправки в Блоке 0
- ✏️ Codex-зеркала всех трёх
- ✏️ `RECENT-CHANGES.md` (этот пункт)

### 24. Фикс ветки create-rsya: BidCeiling, кластеризация, sanity-check на target (2026-04-29)

**Контекст.** Тестовый прогон РСЯ-кампании Омск/вторичка (запрос «делай сам без вмешательства») показал три бага одновременно: (1) три группы созданы по упоминанию пользователя без проверки семантики/посадочных, (2) `network_bid_ceiling ≈ 1800 ₽` — нереально для Омска (реальный клик 30–100 ₽), (3) target form CPA взят как ~6000 ₽ из benchmark без сверки с network-ожиданием 3500 ₽ для tier_1 вторички. Все три причины — в скилле, не в MCP.

**W1. Формула BidCeiling — переписана.**
В `references/rsya_defaults.md` старая формула `BidCeiling = tCPA × 1.5` (концептуально неверна — путает потолок клика с целью конверсии) заменена на:
```
expected_click = tCPA × CR_estimate     # CR=0.05 для РСЯ-недвижимости
BidCeiling     = expected_click × 2.5   # запас сверху
BidCeiling_final = clamp(BidCeiling, tier_min, tCPA × 0.30)
```
tier_min: tier_1=80₽, tier_2=40₽, tier_3=20₽. Хард-кап: BidCeiling ≤ tCPA × 0.30 (если выше — что-то не так с CR или tCPA). Раздел расширен таблицей примеров и анти-паттернами («НИКОГДА BidCeiling = CallCPA × что-то», «НИКОГДА без `network_bid_ceiling`»).

**W2. Sanity-check benchmark vs network — обязательный шаг R2.5.**
Раньше скилл брал `BenchmarkFormCPA` из `get_conversion_values` как target. Теперь — обязательная сверка с `NetworkExpected[theme/tier]` (значения захардкожены в `server/data/network_benchmarks.json` + Go-fallback в benchmarks.go):

| ratio = benchmark / network_expected | Действие |
|---|---|
| 0.7–1.3 | Норма, использовать benchmark. |
| 1.3–2.0 | Стоп, спросить пользователя даже в `auto`-режиме. |
| > 2.0 | Не использовать benchmark, брать `NetworkExpected × 0.9`. |

В режиме `auto` при ratio > 1.3 — приостановить автономный поток и запросить пользователя. Защита от слива бюджета на аномальном target. Зафиксировано как обязательный гейт в `branches/create-rsya.md` и параллельно в Codex-зеркале.

**W3. Группы — после семантики, не до.**
В R5 переписан алгоритм: упомянутые пользователем сегменты («1-комн», «2-комн») — сигнал, не решение. Хард-правила:
- ≥3 фраз в R4 под этот сегмент;
- наличие посадочной (через `build_landing_url` / `get_site_filters`);
- если хотя бы одно «нет» → сегмент сворачивается в `Общие-купить`.

Минимум для первой кампании теперь — одна группа `Общие-купить`. Сегментные добавляются ТОЛЬКО при выполнении правил выше. Анти-паттерны прописаны явно.

**W4. Объявлений на группу — 5–8 (стандарт), до 45 (LAL).**
В R7 было `add_ad × 3` — ниже минимума обучения автостратегии РСЯ. Заменено на `5–8` для стандартных групп, `до 45` для LAL. Каждое объявление = свой комбо (заголовок × текст × визуал 1:1 или 16:9).

**W5. Использование новых MCP-инструментов в R7.**
- `network_bid_ceiling` теперь читается из формулы R2.6, не `tCPA × 1.5`.
- `negative_keywords` ← `summary_string` из `get_negative_keyword_guidance(theme, city, placement="rsya")` (вместо ручного склеивания файла).
- `daily_budget_amount` ← `get_default_budgets(channel="rsya", tier, theme, target_cpa)`.
- Read-back: `summarize_campaign_snapshot(campaign_id)` для компактной сводки.

**W6. Lessons registry + dual-tree sync.**
Запись «0. РСЯ: BidCeiling — это клик, не CPA. И группы — после семантики, не до» добавлена в `references/lessons_registry.md` как повышенный приоритет (блок 0, перед существующими). Все три файла (`branches/create-rsya.md`, `references/rsya_defaults.md`, `references/lessons_registry.md`) синхронизированы между `.claude/skills/leadgen/` и `.codex/skills/leadgen-codex/` (`diff -r` чистый кроме предсуществующих BOM-разниц).

**Эффект.**
- Будущие прогоны РСЯ-кампаний на любом tier-городе перестанут (а) задирать `network_bid_ceiling` в 5–15 раз; (б) брать аномальный benchmark как target без сверки; (в) предрешать структуру групп по реплике пользователя; (г) создавать недо-обученные группы по 3 объявления.
- В режиме `auto` встроена точка вынужденной паузы: ratio benchmark/network > 1.3 — обязательная пауза. Это нарушает «делай сам без вмешательства», но защищает бюджет.

**Файлы изменены:**
- ✏️ `.claude/skills/leadgen/branches/create-rsya.md` (R2 → R2/R2.5/R2.6, R5 переписан, R7 — фикс ads count и BidCeiling)
- ✏️ `.claude/skills/leadgen/references/rsya_defaults.md` (раздел BidCeiling переписан, добавлен sanity-check)
- ✏️ `.claude/skills/leadgen/references/lessons_registry.md` (новая запись блок 0)
- ✏️ Codex-зеркала всех трёх файлов
- ✏️ `RECENT-CHANGES.md` (этот пункт)

### 23. Новые MCP-инструменты: get_default_budgets + get_negative_keyword_guidance (2026-04-29)

**Контекст.** В пункте #22 PROJECTS.md и LEGAL.md ужаты до policy-only. Из бэклога остались: «бюджеты по уровням × tier» и «гайд по минус-словам по тематике/городу» — это data, и им место в MCP, чтобы скилл при создании кампании звал инструмент, а не читал .md в контекст. Реализованы оба.

**W1. `get_default_budgets(channel, tier, theme?, target_cpa?)`**

Файл: `server/platform/direct/budgets.go` (~155 строк, скомпилирован).

Параметры:
- `channel` (required) — `search` / `rsya` / `vk`.
- `tier` (required) — `tier_1` / `tier_2` / `tier_3` (узнаётся через `get_city_config(city).tier`).
- `theme` (optional) — для коэффициента (`вторичка`=1.0, `новостройки`=1.15, `коммерческая`=1.3, `аренда`=0.7, `ипотека`=0.8, `hr`=0.6).
- `target_cpa` (optional, целое в рублях) — добавляет в ответ `computed_from_target = target_cpa × 10 × 1.2`, округлённый до 500₽.

Матрица (search):

| Уровень | tier_1 | tier_2 | tier_3 |
|---|---:|---:|---:|
| test | 8000 | 5000 | 3000 |
| start | 25000 | 15000 | 8000 |
| scale_min | 80000 | 50000 | 25000 |

РСЯ — ~70% от search (соответствует коэффициенту tCPA × 0.7); VK — ~80% от search.

Hard-floor по каналу (минимум для обучения автостратегий): search 5000, rsya 3000, vk 2000 ₽/нед — поднимается автоматически если `theme_multiplier` опускает ниже.

Ответ включает: `tiers` (test/start/scale_min с применённым multiplier), `min_floor`, `formula`, `theme_guidance` (когда какой уровень брать), `rules` (правила старта/смены), и опциональный `computed_from_target` с разрывом vs `tiers.start`.

**W2. `get_negative_keyword_guidance(theme, city?, placement?, include_competitors?, include_jobs?, include_legal?)`**

Файл: `server/platform/direct/negatives.go` (~330 строк, скомпилирован).

Параметры:
- `theme` (required) — `вторичка` / `новостройки` / `загородка` / `аренда` / `ипотека` / `коммерческая` / `агентство` / `бренд` / `hr`.
- `city` (optional, русский) — собственный город кампании; автоматически исключается из «чужих городов РФ» вместе со словоформами (Омск → `омск`, `омский`, `омская`, `омское`, `в омске`).
- `placement` (default `search`) — для `rsya` подмешивается блок RSYA-минусов с восклицательным знаком (точная форма).
- `include_competitors`, `include_jobs`, `include_legal` (default `true`) — выключи при брендовой/HR/юр.-кампании. Для `theme=hr` блок `jobs` авто-выключается.

Внутри — данные из `mcp/negative_keywords.md` (245 строк) перенесены в Go-литералы:
- `citiesMoscowMO`, `citiesSPbLO`, `citiesMillionnikiAndCenters` — ~80 чужих городов РФ.
- `citiesInternationalCIS`, `citiesAbroadRealty` — международные.
- 7 универсальных блоков: `informational`, `free_download_education`, `irrelevant_services`, `jobs`, `legal_bureaucracy`, `medical_family`, `negative_complaints`.
- 6 тематических блоков: `theme_vtorichka` / `_zagorodka` / `_novostroyki` / `_arenda` / `_ipoteka` / `_kommercheskaya`.
- 2 блока конкурентов: `competitor_aggregators` (Циан, Авито, Домклик, ...), `competitor_agencies` (Инком, Миэль, ...).
- 1 блок РСЯ-специфичных: `!отделка, !ремонт, ...` (с `!` для точной формы в РСЯ).
- `ownCityForms` — словоформы 18 ключевых городов Этажи для авто-исключения.

Ответ включает: `blocks` (полные слова по блокам), `block_sizes`, `total_words`, `summary_string` (готовая строка через запятую — сразу в `negative_keywords`), `hard_gate` (PASS/FAIL по правилу ≥150 слов), `rules` (как использовать).

**W3. Регистрация и сборка**

В `server/platform/direct/references.go::RegisterReferenceTools` добавлены два вызова: `registerGetDefaultBudgets(s)` и `registerGetNegativeKeywordGuidance(s)`. Также добавлено поле `tier` в JSON-ответ `get_city_config(city)` (раньше присутствовало в структуре, но не сериализовалось — нужно для связки с `get_default_budgets`).

`docker compose build` прошёл с одним конфликтом имени: была локальная функция `roundTo` уже в `summarize.go` (с другой сигнатурой) — переименовал свою в `roundToStep`. Без других правок собрался.

**W4. Правки документации**

- `PROJECTS.md` (header table) — добавлены строки про два новых MCP-инструмента.
- `PROJECTS.md` (раздел «Бюджеты по умолчанию») — таблица tier × channel заменена на одно предложение со ссылкой на `get_default_budgets`. Сохранена формула расчёта от целевого CPA.
- `PROJECTS.md` (раздел минусации) — пометка «через MCP `get_negative_keyword_guidance`», полный текстовый референс остаётся для ручного просмотра.
- `.claude/skills/leadgen/mcp/negative_keywords.md` + Codex-зеркало — шапка переписана: «для рантайма зови MCP, файл — только для ручного просмотра».

**W5. Эффект**

- Скилл при `add_campaign` теперь не должен загружать `negative_keywords.md` (245 строк, ~6 КБ) — зовёт `get_negative_keyword_guidance(theme="вторичка", city="омск")` и получает готовую `summary_string` для `negative_keywords` параметра + список блоков по требованию. Экономия контекста на каждой создаваемой кампании.
- При расчёте бюджета — не таблица в скилле, а `get_default_budgets(channel, tier, theme, target_cpa?)`. Расчёт tier-aware и theme-aware прямо из MCP, плюс computed-from-target за один вызов.
- 18 городов Этажей с известными словоформами (для авто-исключения собственного города из чужих) уже встроены в код. Для остальных — базовое имя плюс совпадения по нижнему регистру.

**Файлы изменены / созданы:**
- 🆕 `server/platform/direct/budgets.go` (155 строк)
- 🆕 `server/platform/direct/negatives.go` (330 строк)
- ✏️ `server/platform/direct/references.go` (+ tier в response, +2 register-вызова)
- ✏️ `PROJECTS.md` (3 правки: header table, бюджеты, минусация)
- ✏️ `.claude/skills/leadgen/mcp/negative_keywords.md` (шапка)
- ✏️ `.codex/skills/leadgen-codex/mcp/negative_keywords.md` (шапка, синк)
- ✏️ `RECENT-CHANGES.md` (этот пункт)

### 22. Слим PROJECTS.md и LEGAL.md → MCP как источник правды (2026-04-29)

**Контекст.** В корне основного репо `C:\git\leadgen-mcp\` пользователь увидел почти пустой `PROJECTS.md` — main-репо отставал от свежего main на десятки коммитов (был на `f8399f2`, 15.04). Подтянул через `git pull --ff-only` (FF от `f8399f2` до `55db6c3` — 163 файла обновлено). Заодно нашёл два «забытых шаблона» с placeholder-ами `YOUR_*`, `<!-- Опишите`, медицинскими примерами:
- `LEGAL.md` (48 строк) — шаблон юр. правил «под ваш бизнес», под недвижимость никогда не заполнялся; реальный copy-blacklist живёт в `.claude/skills/leadgen/references/copy_blacklist.md`.
- `.vscode/mcp.json` — конфиг VS Code MCP с `YOUR_API_KEY` и облачным URL `direct-mcp.aatex.ru`; не используется (наш `.mcp.json` бьёт в `localhost:8080/sse`).

**W1. Подтяжка main-репо**
- `git pull --ff-only` в `C:\git\leadgen-mcp\` после удаления конфликтующего untracked `campaigns/omsk_poisk_vtorichka_odnokomnatnye.md` (origin/main имеет более полную версию) и сброса `.mcp.json` (локальная правка совпадала с origin).
- 163 файла обновлено: создание ветви `leadgen` (со всем содержимым, переехавшим из старой `etazhi-direct/`), создание Codex-зеркала, новые MCP-пакеты (`forecast.go`, `images.go`, `summarize.go`, `imagegen/`, `history/`), новые поля в city_config, и т. д.

**W2. Удаление пустых шаблонов**
- `.vscode/mcp.json` — удалён (директория `.vscode` тоже).
- `LEGAL.md` — заменён на 12-строчный stub-redirect: «Был шаблоном, реальный copy-blacklist — `.claude/skills/leadgen/references/copy_blacklist.md` и зеркало Codex». Полный текст не удаляю, чтобы 30 ссылок «см. LEGAL.md» из скиллов и документации не побились — теперь они автоматически перенаправляют на актуальный файл.

**W3. Слим `PROJECTS.md`: 232 → 113 строк (−51%)**

Удалены данные, которые **уже отдаёт MCP**:
- Таблица счётчиков по городам (Омск 22325545 и т. п.) → `get_city_config(city)`.
- ID конкретных целей (34275273, 500485089) → `get_conversion_values(client_login, counter_id, theme)`.
- Численные CPA-ориентиры (1871₽ звонок / 5988₽ форма для Омска) → `get_conversion_values` возвращает target CPA tier-aware.
- Доменный паттерн `{city}.etagi.com` + исключения (Тюмень и др.) → tier и domain в `get_city_config` для всех 33 городов.
- Раздел «Гео-таргетинг» с ID региона Омска → `get_city_config`.
- Раздел «Сезонность» (просто отсылал к `forecast_campaign` + `wordstat_dynamics`).

Шапка файла теперь — таблица «Что нужно → какой MCP-инструмент звать» (8 строк, перекрывает 90% запросов за данными).

Оставлено (это policy, не данные):
- Описание продукта (1 абзац).
- Аккаунты-дефолты (3 строки).
- Правила выбора цели (priority_goals не одна цель, формы + звонки, не использовать микроцели).
- Интерпретация CPA-бакетов 🟢 GOOD / 🟡 WATCH / 🔴 CRITICAL — это правило поверх target-CPA, не данные.
- Таблица дефолтных недельных бюджетов по уровням × tier-у города (теперь tier-aware: tier1/tier2/tier3).
- Соглашения именования кампаний и файлов.
- Минусация: что НЕ минусовать / что минусовать (правила, не списки слов — слова в `mcp/negative_keywords.md`).
- Прочие жёсткие правила (DRAFT-only, UTM на уровне группы, attribution LYDC и т. д.).

**W4. Парсер не сломан**

`.claude/skills/_shared/json_helpers.py:extract_project_value()` парсит ключи `counter_id`, `goal_id`, `cpa_good`, `cpa_bad`, `domain` из `PROJECTS.md`. Проверил: **никто не вызывает** `get_project_value()` / `extract_project_value` в живых скриптах (grep по всему дереву нашёл только определение функций и mirror-копии в worktrees). Парсер мёртвый, слим безопасен. Если когда-нибудь понадобится — те же значения уже отдаёт MCP.

**Эффект.**
- `PROJECTS.md`: 232 → 113 строк (−51%, ~7 КБ контекста).
- `LEGAL.md`: 48 → 12 строк (−75%, и больше не несёт фейкового медицинского контекста, который мог сбить с толку при беглом чтении).
- Источник правды по данным — теперь явно MCP. Документ стал тем, чем должен быть: коротким сводом интерпретаций.
- Пользователь обновил основной локальный репо `C:\git\leadgen-mcp\` до актуального main — синхронизация между двумя ПК упрощается.

**Файлы изменены:**
- ✏️ `PROJECTS.md` (232 → 113 строк, переписан)
- ✏️ `LEGAL.md` (48 → 12 строк, stub-redirect)
- 🗑 `.vscode/mcp.json` (+ директория)
- ✏️ `RECENT-CHANGES.md` (+пункт #22)

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
