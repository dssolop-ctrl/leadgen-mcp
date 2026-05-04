# plan_correctives2 — ответ на правки Codex и второй раунд обсуждения

> **Контекст.** Прочитал `PLAN.md` (мой исходный план) и `plan_correctives.md` (правки Codex). Этот файл — мой ответ: где соглашаюсь и беру в финал, где уточняю, где спорю, плюс собственные дополнения и ответы на open questions.
> **Цель.** Свести оба драфта к единому ТЗ, после которого можно стартовать разработку всех волн как полного контура.

---

## 0. Общая оценка правок Codex

Codex поднял **критичный пласт**, которого в моём драфте не было: машинно-проверяемые контракты (idempotency key, action ledger, structured run lifecycle), ownership-модель (управлять только labeled-кампаниями), drift detection (API → STATE как cache), DRAFT-only обязательство (это правило уже есть в проекте — `create-search.md` шаг 11, я упустил его), pacing-контроль вместо одной отсечки 120% MTD, onboarding-паттерн с baseline scan.

Из 14 разделов Codex с большинством соглашаюсь as-is. Спорю с двумя нюансами (deatil ниже). Ниже — построчный разбор.

---

## 1. Где соглашаюсь полностью (берём в финал as-is)

| # | Правка Codex | В финал |
|---|---|---|
| 1.1 | Playbook contract `leadgen-autopilot/references/playbook_contract.md` | ✅ |
| 1.2 | Codex-зеркало автопилота — Claude-only исключение, зафиксировать в `.claude/skills/leadgen-autopilot/CLAUDE.md` и `RECENT-CHANGES.md`; shared-файлы `leadgen/{references,library,branches}` всё равно синхронизируются | ✅ |
| 1.4 | Разделить versioned config и runtime data; `autopilot/runtime/` в `.gitignore` | ✅ |
| 1.5 | Routine cwd = `C:\git\leadgen-mcp` (корень). Prompt: `/leadgen-autopilot city=omsk` | ✅ |
| 2.1 | Deterministic idempotency key `<city>\|<account>\|<entity_type>\|<entity_id>\|<action_type>\|<desired_value>\|<decision_window>` + проверка локального ledger | ✅ |
| 2.2 | Per-city `RUNNING.lock` с TTL 2ч, recovery логика | ✅ |
| 2.3 | Structured run lifecycle (`started → loaded_context → fetched_metrics → planned_actions → approval_checked → applying → applied_partial → memory_written → notified → succeeded/failed/halted`) | ✅ |
| 2.4 | Atomic write для STATE/CURSOR/SUMMARY/pending (tmp → validate → rename → `.bak`) | ✅ |
| 3.1 | API source of truth, STATE = cache, drift detection с `human_override` cooldown | ✅ |
| 3.2 | Ownership model через labels: `autopilot:managed`, `city:<city>`, `topic:<topic>`. Без label — read-only. `campaign.adopt_existing` / `campaign.release` — отдельные действия | ✅ |
| 3.3 | Human change cooldown 24-72ч после ручной правки entity | ✅ |
| 4.1 | Pacing-контроль: `pacing_ratio`, conservation @ >1.1, emergency @ >1.25 или forecast > 120% | ✅ |
| 4.2 | Разделить `managed_spend` / `unmanaged_spend` / `total_account_spend` | ✅ |
| 4.3 | DRAFT-only обязательно. Раздельные действия `campaign.create_draft.*` и `campaign.activate_existing_draft` (последнее — `review_queue` или `block` на пилоте). **Подтверждаю: в проекте это уже правило (`create-search.md` шаг 11, `METRIKA-ADS-RULES.md`).** | ✅ |
| 5A | Onboarding-паттерн: baseline scan → `launch_proposal.{md,yaml}` → human approval / config update → draft creation → manual activation | ✅ |
| 6 (иерархия) | `caps_defaults.yaml` → `city.permissions` → `city.topics.<t>.permissions` → runtime override; resolved permissions писать в run log | ✅ |
| 6.2 | Risk class (`low/medium/high/critical`) для каждого действия в `action_catalog.md` | ✅ |
| 7.1 | LYDC атрибуция, freshness checks, timezone аккаунта, использовать yesterday/rolling 3-7d вместо today для daily decisions | ✅ |
| 7.2 | Per-action `min_evidence` (clicks/conversions/spend/days/window/compare_to) в `action_catalog.md` | ✅ |
| 7.3 | Каждое запланированное действие имеет `signal_id`, `reason_code`, `confidence`, `expected_effect`, `primary_metric`, `guard_metric`, `rollback_trigger` | ✅ |
| 8 | Rollback: `before_snapshot` структурно перед apply, `rollback_plan` сначала dry-run, Telegram `rollback <run_id>` требует `confirm rollback <id>`. `rollback: manual_only` для необратимых | ✅ |
| 9.1 | Telegram-обвязка: HTML/MD escaping, лимит длины с fallback на файл, retry/backoff, `update_offset` для `getUpdates`, allowlist chat_id, маскирование токена в логах, alert при падении notify | ✅ |
| 9.2 | Approval polling в начале прогона + сохранение `message_id` для новых pending после отчёта | ✅ |
| 9.3 | Approval expiry: 24ч ставки/бюджеты, 72ч draft creation; после истечения — пересчитать действие | ✅ |
| 10 | Learnings: validated **не** автоматически меняет поведение; идёт в monthly digest на review; хранить negative evidence; expiry 60 дней; scope city/topic/channel; не переносить между городами без approval | ✅ |
| 11 | Внутри разработки: schema validation, dry-run, mock fixtures, golden run, idempotency test, broken state test, telegram retry/dup test | ✅ |
| 14 | Acceptance criteria финального теста (раздел 14 Codex) — целиком в наш acceptance | ✅ |

---

## 2. Где соглашаюсь, но с уточнением

### 2.1. Машинный ledger (Codex 1.3) — **как реализовать без размывания идеи lazy-md-памяти**

**Согласен:** для apply-фазы и idempotency нужен структурный источник правды. Markdown-валидация хрупкая.

**Уточнение.** Не заменять md, а ввести **двухслойную модель**:

| Слой | Формат | Кто пишет | Кто читает | Назначение |
|---|---|---|---|---|
| **Operational** (источник правды) | `state.yaml`, `action_ledger.jsonl`, `metrics_snapshots.jsonl`, `pending_approvals.yaml`, `before_snapshots/<action_id>.json` | Apply engine | Decision/idempotency/rollback логика | Машинная валидация, apply, recovery |
| **Narrative** (для агента) | `STATE.md`, `CURSOR.md`, `SUMMARY.md`, `runs/<id>.md`, `campaigns/<id>.md`, `decisions/*.md` | Memory-write фаза (генерируется из operational + контекст агента) | Агент в начале прогона | Понятная агенту память + lazy load по тегам |

**Ключевой момент.** Markdown-память остаётся "главной фичей" для агента (как просил пользователь — это удобно для лениво-загружаемой контекстной памяти), но **источник правды для apply — `state.yaml` + ledger**. Если md разойдётся со state.yaml — пересобираем md из state.yaml + lazy-файлов. Md — производное.

Конкретно:
- `state.yaml` — текущее состояние (бюджет, тематики, кампании со статусами и ownership). Регенерируется в конце прогона из API + ledger.
- `action_ledger.jsonl` — append-only лог всех попыток применить действие: `{run_id, idempotency_key, action, before_snapshot_ref, status, applied_at, error?}`. Никогда не редактируется, только приписывается.
- `metrics_snapshots.jsonl` — ежедневный срез ключевых метрик города (для pacing, decision precision, rollback evidence).
- `pending_approvals.yaml` — структурный, с полями `id, action, payload, evidence, telegram_message_id, expires_at, status`. `pending_approvals.md` — рендер из yaml для удобства чтения.

**JSON Schema** для каждого артефакта в `autopilot/schemas/{city_config,state,action,approval,ledger_entry,metrics_snapshot,launch_proposal}.schema.json`. Валидация на каждом write.

### 2.2. Trust profiles (Codex 6.3) — **гибрид с явными overrides**

**Согласен** ввести `trust_profile: conservative | balanced | aggressive | custom`.

**Уточнение.** Профиль — это шаблон, не замена. Resolution-порядок:

```
1. caps_defaults.yaml.permissions      # глобальный baseline (самый консервативный)
2. trust_profiles/<profile>.yaml       # профильный набор (новый файл)
3. cities/<city>.yaml.permissions      # точечные city overrides
4. cities/<city>.yaml.topics.<t>.permissions  # топик-специфичные
5. runtime override через approval     # временные с expiry
```

В `autopilot/config/trust_profiles/` — три preset-файла (`conservative.yaml`, `balanced.yaml`, `aggressive.yaml`), плюс возможность `custom` (тогда profile-слой пропускается, всё из city).

Это:
- Снимает проблему "заполнять десятки permissions для каждого нового города".
- Сохраняет полный контроль (любой permission можно переопределить точечно в `city.yaml`).
- Делает прозрачным: в run log пишем `permissions_resolution: { campaign.create.in_new_topic: { value: review_queue, source: city.yaml }, bid.increase.within_cap: { value: auto, source: trust_profiles/balanced.yaml }, ... }`.

### 2.3. Цикл daily/weekly/monthly (Codex раздел 5) — **с одной поправкой**

Codex: «monthly day = light safety + monthly, weekly пропускается». Правка пользователя в исходном ТЗ: «месячное подведение итогов и планирование следующего месяца **без ежедневной и недельной рутины**».

**Принимаю Codex**, но конкретизирую:

| День | Что выполняется |
|---|---|
| Обычный день | `daily` (полный) |
| `weekly_rollup_dow` (например monday) | `daily_safety_check` (HALT, drift, overrun, pending_approvals) + `weekly` (без полного daily-decide) |
| `monthly_rollup_dom` (например 1-е) | `daily_safety_check` + `monthly` (без weekly, без daily-decide) |
| Совпадение weekly + monthly (например 1 число — понедельник) | `daily_safety_check` + `monthly` (weekly merged into monthly digest) |

`daily_safety_check` — минимальный набор: HALT-проверка, hard pacing cap, drift detection с алертами, обработка expired approvals. Никаких изменений тактики.

Принципиально: в один прогон **не** меняем тактику в нескольких циклах. Daily может предложить +bid 10%, но в weekly-день это решение откладывается до следующего обычного дня.

### 2.4. Status тематик (Codex 5A) — **принимаю с одной добавкой**

Codex: `candidate | active | paused | blocked`. Это лучше, чем `enabled: true/false`.

**Добавка.** Ввести ещё `experimental` для тем, которые специалист хочет дать боту "пощупать" с жёсткими caps:
- `experimental` — можно создать draft, но активация только через `review_queue`, бюджет ≤ 30% от tier_min, period ≤ 14 дней, после — авто-перевод в `paused` для review.

Это разгружает `candidate → active` переход (часто хочется попробовать тему, а не сразу запускать на полном бюджете).

Итог: `candidate | experimental | active | paused | blocked`.

### 2.5. Baseline permissions (Codex 6.1) — **частично спорю**

Codex предлагает консервативные дефолты. Соглашаюсь с большинством, но:

| Permission | Codex предлагает | Я предлагаю | Почему |
|---|---|---|---|
| `creative.generate_new_image` | `review_queue` | `auto_with_notify` для `balanced`, `review_queue` для `conservative` | Картинки создаются в DRAFT-объявлении, активацию делает специалист → риск контролируемый. Для `conservative` соглашусь. |
| `campaign.archive.no_traffic_30days` | `review_queue` | `auto_with_notify` (порог 30 дней — уже сильный сигнал) | 30 дней без трафика — это уже не learning-период, архивация безопасна. На 14d согласен с `review_queue`. |
| `placement.unblock` | `review_queue` | Согласен с `review_queue` | OK |
| `audience.create` / `retargeting.create_list` | `review_queue` | `auto_with_notify` для `balanced` | Создание аудитории/списка — read-only действие в Direct (никакие показы не стартуют автоматически). Применение в кампании — отдельное действие. |
| `keyword.add_new_group_in_existing_topic` | `auto_with_notify` или `review_queue` | `auto_with_notify` для `balanced` | Согласен на `auto_with_notify`. |
| `ad.update_text` | `review_queue` или `auto_with_notify` | `auto_with_notify` | До тех пор, пока бот не правит юр.дисклеймер (это `block`), правка текста — обычная итерация. Через `copy_blacklist.md` уже фильтруем. |

**Принципиальная позиция.** Дефолтный профиль — `balanced`. Создавать `conservative` отдельно для пилота. Дефолт `aggressive` — после 30+ дней стабильной работы и пользовательского approval на смену профиля.

Сводная таблица baseline permissions по профилям — добавляется в финальный план как раздел 6A.

---

## 3. Где спорю / предлагаю отложить

### 3.1. Codex 7.3 (`expected_effect`, `guard_metric`, `rollback_trigger` для каждого действия)

**Спор.** Согласен с `signal_id`, `reason_code`, `confidence`. Но `expected_effect` + `primary_metric` + `guard_metric` + `rollback_trigger` для **каждого** действия — overengineering для пилота.

**Предложение.**
- W3 (analyze): обязательны `signal_id`, `reason_code`, `confidence`, `evidence` (для пер-action minimum).
- W5 (apply): добавить `expected_effect` (компактная строка: "−10% CPA за 7d" — генерируется по reason_code).
- W7 (rollups): добавить `guard_metric` для high/critical risk class только.
- W11 (hardening): `rollback_trigger` авто-вычисляется по policy (`if metric > threshold for X days → flag for review`). Не требует ручного заполнения для каждого действия.

Иначе action_catalog раздуется до 60×7 полей и станет неподдерживаемым.

### 3.2. Codex 8 (rollback модель)

Согласен с dry-run и confirm flow. Но **спорю** с фразой "перед каждым apply сохранять `before_snapshot` структурно". Для лёгких действий (минусование, +1 placement в blocked) snapshot — это лишний JSON на ~200 действий/день. 

**Предложение.** Snapshot обязателен только для `risk_class >= medium`. Для `low` (минусование, blacklist placement, мелкие ставки в cap) — достаточно записи в ledger с `before_value`/`after_value` (одной строкой, без отдельного snapshot файла).

### 3.3. Codex 6.2 (risk class)

Согласен ввести risk_class. Но **спорю с привязкой rollback policy к risk class на этапе action_catalog**. На пилоте rollback policy = "ledger-driven, manual через Telegram confirm". Когда наберём данные — придумаем differentiated policies.

То есть: risk_class в каталоге **есть**, но используется для outline-ровности (snapshot yes/no, evidence threshold, approval expiry length), а не для авто-rollback policy.

---

## 4. Мои собственные дополнения (не было ни в моём, ни в Codex)

### 4.1. Onboarding для **существующего** аккаунта (с уже работающими кампаниями)

Codex описал onboarding для нового города. Но у Этажей все 60 городов — уже работают с кампаниями. Бот никогда не подойдёт к "чистому листу".

Поэтому baseline scan должен включать:

1. **Inventory.** Все campaigns/adgroups/keywords/audiences с разделением:
   - Owned (имеют label `autopilot:managed`) — управляемые ботом.
   - Adoptable (без label, но в разрешённой тематике) — кандидаты на adoption.
   - Foreign (без label, не в разрешённой тематике) — read-only, никогда не трогаем.
2. **Adoption proposal.** В `launch_proposal.yaml` отдельный блок `adoptable_campaigns`: список кандидатов с метриками (last 30d), causal hint ("эту тему мы хотели запустить — кампания уже есть"), и предложение `adopt | leave_readonly`.
3. **Tagging step.** При `adopt_existing` бот делает `add_labels(campaign_id, ["autopilot:managed", "city:omsk", "topic:vtorichka"])`. С этого момента кампания под управлением.

Это критично для пилота — без adoption бот будет работать на пустом аккаунте, что бессмысленно. Но adoption — **ручное решение специалиста через approval**.

### 4.2. Метрики Метрики (goals) — единый каталог в city.yaml

Codex упомянул вопрос "есть ли единый список целей". Предлагаю:

```yaml
# в city.yaml
metrika:
  counter_id: 12345678
  goals:
    lead_form: { id: 100001, name: "Заявка форма", attribution: LYDC, value_type: leads }
    call: { id: 100002, name: "Звонок calltouch", attribution: LYDC, value_type: leads }
    chat: { id: 100003, name: "Чат WhatsApp", attribution: LYDC, value_type: leads }
    qualified_lead: { id: 100004, name: "Квал. лид CRM", attribution: LYDC, value_type: qualified_leads }
  primary_conversion_goal: lead_form  # для расчёта CPA по умолчанию
  secondary_goals: [call, chat]
```

Бот всегда работает с этими alias-ами, а не голыми ID. В отчётах — человечные имена. Это снимает Codex.13.6.

### 4.3. Двойная атрибуция: lead vs qualified_lead

CPA по `lead_form` ≠ CPA по `qualified_lead` (часто x3-5 разница). Reconcile целевых CPA должен учитывать обе метрики:

- `target_cpa_form` (из city.yaml) — оптимизация ставок/бюджета.
- `target_cpa_qualified` — для weekly анализа эффективности тематики, не для daily-действий (delay 7-14 дней).

### 4.4. Search vs RSYA — разные defaults для ряда действий

Codex 13.8 спросил. Подтверждаю: **да**, нужны отдельные дефолты:

| Action | Search default | RSYA default |
|---|---|---|
| `min_clicks_for_decision` | 30 | 100 |
| `min_conversions_for_decision` | 3 | 5 |
| `cooldown_hours_after_bid_change` | 24 | 48 |
| `placement.block.low_performing` | n/a | первоочередное действие |
| `negatives.add_from_search_queries` | первоочередное действие | вторично |
| `learning_period_days` | 7 | 14 |

В `caps_defaults.yaml` ввести вложенные блоки `caps_defaults.search.*` и `caps_defaults.rsya.*` с fallback на корневые значения.

### 4.5. Snapshots для расчёта decision precision

В `metrics_snapshots.jsonl` каждый день писать:
```json
{
  "date": "2026-04-29",
  "city": "omsk",
  "topic": "vtorichka",
  "channel": "search",
  "campaign_id": "8765432",
  "spend": 4200, "clicks": 84, "impressions": 1200,
  "leads_form": 6, "leads_call": 2, "leads_qualified": 1,
  "cpa_form": 700, "cpa_call": 2100, "cpa_qualified": 4200,
  "active_actions_today": ["budget.increase.within_cap"]
}
```

Это позволит:
- В monthly считать **decision precision** (сделанное действие → результат через 14 дней).
- В weekly считать pacing/forecast по фактическим данным.
- В авто-обучении строить evidence с конкретными цифрами вместо общих ощущений.

### 4.6. Где живёт runtime — ответ на 13.1

Предлагаю: **в репо `autopilot/runtime/` с `.gitignore`**. Аргументы:
- Один корень для всего проекта (упрощает Claude Desktop routine cwd).
- Удобно делать backup всего autopilot одним архивом.
- Не размазывать по диску (никаких %LOCALAPPDATA% или OneDrive).
- При переносе на другую машину — `git clone` + копирование `autopilot/runtime/` + `autopilot/config/secrets.env`.

`.gitignore` дополнения:
```
autopilot/runtime/
autopilot/config/secrets.env
autopilot/reports/
```

`autopilot/learnings/` — **не** в gitignore: validated learnings — артефакт для review/PR.

### 4.7. Как помечать кампанию ownership

Yandex Direct поддерживает `Labels` (см. mcp-инструменты `add_labels` / `get_labels`). Используем три label per managed campaign:

```
autopilot:managed
city:omsk
topic:vtorichka
```

Опционально `channel:search` или `channel:rsya`, `cohort:2026-Q2` для трекинга.

При adoption — `add_labels`. При release — `delete labels` через update_campaign (нужно проверить, поддерживает ли API; если нет — добавить инструмент `remove_labels` в MCP). Это open question в раздел 6 ниже.

### 4.8. Holdout / эксперименты

Чтобы бот не "увлёкся" самооптимизацией, специалист может оставить **holdout-кампанию** (не под управлением, явный label `autopilot:holdout`). Она работает по ручным настройкам, и в monthly бот сравнивает CPA managed vs holdout. Если managed хуже — alert.

Это опциональная фича, реализовать в W7 или W10.

---

## 5. Ответы на open questions Codex (раздел 13)

| # | Вопрос Codex | Моё предложение | Спорно? |
|---|---|---|---|
| 1 | Где runtime? | `autopilot/runtime/` в репо с `.gitignore` (см. 4.6) | нет |
| 2 | Какие existing campaigns adoptить? | Только через явный onboarding approval с tagging (см. 4.1). Без label — read-only | нет |
| 3 | Daily/weekly в monthly day? | Только `daily_safety_check` + `monthly`. Полный daily-decide пропускается (см. 2.3) | да, обсудим |
| 4 | MCP для rollback/snapshots? | Имеющиеся достаточны: `get_change_history`, `get_*_stats` для restore evidence. Доп.инструмент `remove_labels` при release. Скриптовая обвязка snapshot — в `autopilot/lib/` | нет |
| 5 | log_change_event — где? | Локальный `action_ledger.jsonl` обязателен (для idempotency). API-вызов `log_change_event` — для кросс-проверки drift и audit trail в Direct UI. Оба места, но source of truth — локальный ledger | нет |
| 6 | Цели Метрики — единый список? | `city.yaml.metrika.goals` (см. 4.2). Один формат для всех городов | нет |
| 7 | Атрибуция delay? | LYDC. Daily decisions по `yesterday + rolling 7d`. Weekly — `last 7 full days`. Monthly — `last 30 full days`. Qualified leads (CRM-задержка 5-7 дней) — только в weekly/monthly, не в daily | нет |
| 8 | Разные rules search/rsya? | Да (см. 4.4) | нет |
| 9 | SLA routine? | Пилот: ожидаемое время прогона 5-15 мин/город. Если 24ч без прогона — Telegram alert (через external scheduled-check, отдельный routine). Retry — следующий routine-cycle | да, обсудим watch-mechanism |

---

## 6. Новые open questions (от меня после анализа)

1. **Один Telegram-бот на все города или per-city?** Codex говорит "один чат на город" — согласен. Но **бот** один или N? Один бот, разные чаты — проще (один токен в `secrets.env`). Подтверждаешь?

2. **Holdout-кампании.** Хочешь ли иметь возможность держать holdout (не под управлением, для сравнения)? Если да — добавляем 4.8 в W7.

3. **Adoption на пилоте.** Сразу adoptим существующие кампании omsk через onboarding-approval? Или начинаем с пустого аккаунта (создаём draft с нуля → специалист активирует)? Второе — чище для проверки контура, но менее полезно бизнесу.

4. **`remove_labels` в MCP.** Yandex Direct API не имеет прямого `remove_labels`. Нужно через `update_campaign` с пустым массивом labels (или через delete? проверить). Если потребуется доработка MCP — это блокер для adoption-release? Проверим в W1.

5. **Schema-валидация на каждый write.** Это медленно (особенно для ledger append). Соглашаемся на overhead, или валидируем только при значимых артефактах (state, approvals), а ledger пишем без валидации (jsonl row уже простая структура)?

6. **`experimental` статус тематики (см. 2.4).** Нужен или избыточен? Если тебе достаточно `candidate → active`, я уберу.

7. **`trust_profile` дефолт для пилотного omsk?** Я бы взял `balanced` для нашей baseline-разработки и потом тестировали корректность. Но если хочешь — `conservative`.

8. **Управление через Claude Desktop routine.** На сколько города параллельно? 1 routine per city, не одновременно? Или один routine на все города последовательно? Это влияет на per-city lock и расписание.

9. **Куда писать `RECENT-CHANGES.md` от автопилота?** В корневой `RECENT-CHANGES.md` (как сейчас) или отдельный `autopilot/RECENT-CHANGES.md`? Пред: автопилот не пишет в общий журнал dev-режима — у него свой.

10. **PIDfile / session lock.** На Windows нет honest pid-блокировок — Claude Desktop может пересекаться. RUNNING.lock с TTL — единственный способ. Alert если lock протух дважды подряд (явно зависший прогон)?

---

## 7. Пересобранные волны (полный контур, разработка без пилотных пауз)

> Учтены: правки Codex по структуре волн (W1-W10), мои добавки (operational+narrative ledger, onboarding adopting), пользовательское "разрабатываем все волны сразу, тестим итоговый вариант".

### W1 — Каркас + schemas + gitignore + lock + Telegram hello
- Структура `autopilot/{config,runtime,reports,learnings,lib,schemas}/`.
- `.gitignore` для `runtime/`, `secrets.env`, `reports/`.
- JSON Schema: `city_config`, `state`, `action`, `approval`, `ledger_entry`, `metrics_snapshot`, `launch_proposal`.
- Skill `.claude/skills/leadgen-autopilot/skill.md` с роутером шагов.
- `autopilot/CLAUDE.md` корневой роутер.
- `caps_defaults.yaml`, `trust_profiles/{conservative,balanced,aggressive}.yaml`, `cities/_example.yaml`, `secrets.env.example`.
- `lib/telegram_send.sh`, `lib/telegram_send_doc.sh`, `lib/atomic_write.sh`, `lib/lock.sh`.
- HALT.flag (global + per-city).
- Per-city `RUNNING.lock` с TTL 2ч.
- Запуск из `C:\git\leadgen-mcp`, prompt `/leadgen-autopilot city=omsk`.
- **Self-check:** `/leadgen-autopilot city=omsk` запускается, читает HALT, берёт lock, шлёт `Hello from autopilot, city=omsk` в Telegram, отпускает lock, выходит.

### W2 — Память: operational + narrative + atomic + recovery
- Templates narrative md (`STATE.md`, `CURSOR.md`, `SUMMARY.md`, `pending_approvals.md`, `runs/<id>.md`, `campaigns/<id>.md`, `decisions/*.md`).
- Operational артефакты: `state.yaml`, `action_ledger.jsonl`, `metrics_snapshots.jsonl`, `pending_approvals.yaml`, `before_snapshots/`.
- `lib/atomic_write.sh` (tmp → validate → rename → .bak).
- `lib/memory_lookup.sh` (grep по тегам).
- Branch `branches/memory_write.md` — алгоритм синхронизации operational → narrative.
- Алгоритм компрессии SUMMARY (>30/>90/>365 дней).
- Crash recovery: при старте сверять lock + ledger.last_status, при `applied_partial` → drift check.
- **Self-check:** ручной прогон создаёт корректные файлы; повторный запуск читает их; вручную corrupt state.yaml → recovery из .bak; компрессия запускается из weekly mock.

### W3 — Аналитика, метрики, signals, dry-run plan
- Branch `branches/analyze.md` — daily-цикл (read-only).
- Сбор метрик: `get_campaign_stats`, `metrika_get_direct_report` с LYDC, `get_search_queries`, `get_change_history`, `get_blocked_placements`.
- Goals из `city.yaml.metrika.goals`.
- Freshness checks (timezone, data delay).
- `signal_catalog.md` — каталог сигналов (CPA jump, overrun, no conv, learning ended, drift detected, …).
- `action_catalog.md` — каталог действий с `risk_class`, `min_evidence`, `cooldown`, `default_permission`.
- Action plan c reason_code/confidence/evidence.
- HTML-отчёт через `lib/render_html.sh`.
- Telegram daily summary (compact).
- Onboarding scan для нового/существующего города → `launch_proposal.{md,yaml}`.
- **Self-check:** на dev-окружении `/leadgen-autopilot city=omsk_dev` собирает state без apply, генерит план с reason_codes, выводит drift report.

### W4 — Approval queue + Telegram replies + expiry
- Branch `branches/approval.md`.
- `pending_approvals.yaml` структура с `expires_at`, `evidence`, `telegram_message_id`.
- `lib/telegram_check_replies.sh` с persisted `update_offset` (защита от dup updates), allowlist chat_id.
- Парсинг команд: `approve <id>`, `reject <id>`, `defer <id> <Nd>`, `confirm rollback <run_id>`.
- Expiry: 24ч (bid/budget), 72ч (draft create), пересчёт при истечении.
- Применение approved actions при следующем прогоне (вход в W5 apply pipeline).
- **Self-check:** mock approval цикл end-to-end: bot → pending → reply в TG → next run → apply.

### W5 — Apply engine + ownership + low-risk auto
- Branch `branches/apply.md`.
- Deterministic idempotency key (см. 2.1 Codex).
- Ownership check: трогать только campaigns с label `autopilot:managed`.
- Drift check перед apply (API state vs expected before_state).
- Snapshot policy: structured snapshot для risk_class>=medium, ledger-only для low.
- `log_change_event` в API + локальный ledger.
- Запуск low-risk auto-actions: negatives.add_from_search_queries, blacklist placements, мелкие bid/budget в cap при high confidence.
- **Self-check:** apply одного безопасного действия (negatives), проверка idempotency (повторный run не дублирует).

### W6 — Reconciliation + onboarding + adoption + DRAFT-only
- Branch `branches/reconcile_config.md`.
- Diff config↔state↔API.
- Сценарии: новая `active` тематика → launch_proposal (если `candidate`) или draft creation (если `active`); смена `active→paused` → review_queue для pause; budget/target_cpa changes.
- **DRAFT-only**: `campaign.create_draft.*` создаёт SUSPENDED, `campaign.activate_existing_draft` — отдельное действие (по baseline `review_queue`).
- Adoption: `campaign.adopt_existing` через onboarding approval, `add_labels`.
- Drift detection с `human_override` cooldown (24-72ч).
- Pacing-контроль: `pacing_ratio`, conservation @ >1.1, emergency @ >1.25.
- Hard cap: forecast > 120% → emergency pause + alert.
- **Self-check:** добавление новой тематики в city.yaml → следующий run → draft created → approval pending → confirm → active.

### W7 — Weekly/monthly rollups + compression + quality metrics
- Weekly: сравнение неделя/неделя по тематикам, обновление CURSOR с тактикой на след. неделю, компрессия SUMMARY, weekly HTML отчёт с графиками (matplotlib png inline).
- Monthly: стратегические выводы, бюджет на след. месяц, quality metrics (decision precision, rollback rate, approval queue health, coverage, telegram noise).
- Holdout-сравнение (опционально, см. 4.8): managed vs holdout campaigns.
- Cycle priority (см. 2.3): `daily_safety_check` + `weekly|monthly`, без полного daily-decide в эти дни.
- **Self-check:** mock weekly run на тестовых данных формирует корректный отчёт и обновляет CURSOR.

### W8 — Medium/high-risk actions
- Включить (по permissions): pause/resume, ad variants, keyword expansion, audiences, retargeting, creative generation (DRAFT-only).
- Усиленные уведомления.
- `before_snapshot` обязателен для risk>=medium.
- Per-action `min_evidence` строго применяется.
- **Self-check:** в dev mock сценарий "high CPA → pause low_performing" с full evidence trail.

### W9 — Learnings (proposed → digest)
- Branch `branches/learnings.md`.
- Naissance: паттерн → `proposed/<id>.md` с structured evidence.
- Negative evidence tracking.
- **НЕ автоматическое применение** — только в monthly digest предложением для review.
- Specialist принимает через PR в `lessons_registry.md` (как сегодня).
- Expiry: re-check каждые 60 дней; rejected — навсегда.
- Scope: city/topic/channel — не переносить между городами без approval.
- **Self-check:** mock сценарий с 3 повторами паттерна за 14 дней → proposed; rejection через approval → moved to rejected.

### W10 — Hardening + chaos + rollback + multi-city checklist
- Stress: 5xx ошибки MCP, битый конфиг, drift между STATE и API.
- Rollback команда из Telegram с двухступенчатым confirm.
- Hard cap budget overrun (emergency).
- Multi-city scaling: lock per city, расписание per city, отчёты per city.
- Документация специалисту: чтение отчётов, управление approval, чтение learnings.
- Чеклист "scale to 3+ cities".
- **Self-check:** chaos test на синтетике 3 дня без вмешательства → 0 критичных инцидентов.

### Финальный end-to-end тест
Сценарий перед боевым запуском (на одном пилотном городе):
1. **Onboarding.** Новый/существующий город → baseline scan → `launch_proposal.yaml` с adoption candidates.
2. **Approval.** Specialist утверждает adoption через config + Telegram approve → bot adoptит campaigns с labels.
3. **Daily cycle.** 7 дней daily без вмешательства; metrics snapshots; pending approvals обрабатываются; report ежедневно.
4. **Crash recovery.** Эмулировать crash в середине apply → следующий run корректно восстанавливается (drift detect, повторно не применяет).
5. **Weekly.** В weekly_dow — отчёт неделя/неделя, обновление CURSOR.
6. **Monthly.** В monthly_dom — стратегический отчёт, learnings digest, бюджет на след. месяц.
7. **Rollback drill.** В Telegram `rollback <run_id>` → bot показывает обратный план → `confirm rollback <id>` → откат успешен / failure properly logged.
8. **Drift drill.** Specialist вручную правит ставку через UI → next run обнаруживает, freeze entity 24ч, не возвращает обратно, отчёт показывает.
9. **Reconcile drill.** Включить новую `active` тематику в config → next run создаёт DRAFT → approval activate → live.

---

## 8. Acceptance criteria финального теста (объединение моего раздела 13 и Codex 14)

- [ ] Каждое proposed action имеет `signal_id`, `reason_code`, `confidence`, `evidence` (см. 3.1 — `expected_effect` обязателен только для apply-фазы).
- [ ] Повторный запуск на тех же входных данных не дублирует action plan (idempotency).
- [ ] Per-city lock работает; recovery после убитого процесса корректен.
- [ ] `action_ledger.jsonl` непротиворечив с фактом в API (drift = 0 или явный `human_override`).
- [ ] Автопилот трогает **только** campaigns с label `autopilot:managed`. Foreign кампании read-only.
- [ ] Все новые кампании созданы как DRAFT/SUSPENDED. Активация — только через `campaign.activate_existing_draft` с `review_queue`.
- [ ] Telegram approval работает с expiry; `update_offset` персистится; повторный approve не применяется дважды.
- [ ] `HALT.flag` (global + per-city) корректно останавливает прогон.
- [ ] Drift report показывает любые расхождения STATE↔API.
- [ ] Onboarding sequence end-to-end: baseline scan → launch_proposal → approval → adoption (с labels) → draft creation → manual activate.
- [ ] Resolved permissions из `caps_defaults` + `trust_profiles/<p>.yaml` + `city.yaml` → пишутся в run log.
- [ ] Pacing-контроль активирует conservation @ >1.1, emergency @ >1.25.
- [ ] Rollback требует двухступенчатого confirm; необратимые помечены `manual_only`.
- [ ] Quality metrics (decision precision, rollback rate, approval queue health, coverage, telegram noise) считаются в monthly.

---

## 9. Что просим тебя решить, прежде чем стартуем W1

Минимум для разморозки (до этого волны не запускаем):

- [ ] Согласовать **двухслойную модель** memory (operational yaml/jsonl + narrative md) — пункт 2.1.
- [ ] Согласовать **trust_profiles** как preset + city overrides — пункт 2.2. Какой профиль ставим на пилотный omsk?
- [ ] Подтвердить **DRAFT-only** для всех `campaign.create_*`. Активация — отдельным `review_queue` действием.
- [ ] Подтвердить **ownership через labels** (`autopilot:managed`, `city:<>`, `topic:<>`). Adoption через onboarding approval.
- [ ] Подтвердить **runtime data в `autopilot/runtime/` (gitignored)**.
- [ ] Подтвердить **cycle priority** в weekly/monthly дни (light safety + rollup, без полного daily-decide).
- [ ] Ответить на новые open questions (раздел 6 этого файла) — как минимум п. 1 (один TG-бот), п. 3 (adoption на пилоте), п. 7 (профиль omsk), п. 8 (1 routine на все города или N).

После этого — собираю финальный объединённый PLAN.md (или редактирую текущий) с уже фиксированными решениями и стартуем разработку всех волн как один пакет.
