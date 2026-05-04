# Autopilot — Итоговое ТЗ автономного агента leadgen

> **Статус.** Финальный план после трёх раундов уточнений (PLAN v1 → corrections (Codex) → corrections2 (Claude) → corrections3 (Codex) → ответы пользователя). Прежний драфт сохранён в `PLAN_v1_archive.md`.
> **Цель.** Автономный агент `/leadgen-autopilot`, который ведёт рекламный аккаунт **с нуля** для нового города 24/7 без участия специалиста: ежедневная операционка, недельные итоги, месячная стратегия. Триггер — Claude Desktop routine.
> **Пилот.** 1 город (новый, аккаунт ещё не определён). Старт с пустого аккаунта — adoption существующих кампаний на пилоте не используется.
> **Эксперимент пилота.** Бот создаёт и ведёт кампании сам, без запроса разрешений. Approval-режим — опция, доступная через `city.yaml`.

---

## 0. Зафиксированные решения

| № | Решение | Источник |
|---|---|---|
| 1 | Runtime в `autopilot/runtime/` (gitignored). | default |
| 2 | Пилотный город — новый, аккаунт определится позднее. Не блокер: до старта пилота в `city.yaml` подставится конкретное имя. | пользователь |
| 3 | Один Telegram bot token, per-city chat (`chat_id` в `city.yaml`). | default |
| 4 | **Старт с нуля.** Adoption-механизм реализуется в W6 как фича для будущего масштабирования, но в финальном e2e тесте не проверяется. | пользователь |
| 5 | Topic statuses: `candidate / experimental / active / paused / blocked`. | default |
| 6 | Routine model: 1 routine на город, staggered. Общий orchestrator — после стабилизации. | default |
| 7 | **Autonomy mode `full_auto` на пилоте.** Бот сам создаёт и ведёт кампании без approval. Approval-режим (`with_approvals`) — опция в `city.yaml`. Block-actions всё равно остаются (legal/account/billing). | пользователь |

---

## 1. Объём и границы

**В скоупе.**
- Скилл `.claude/skills/leadgen-autopilot/` — параллельный к `leadgen`. Запускается по `/leadgen-autopilot city=<city>`. Использует shared playbooks из `leadgen/branches/` через явный playbook contract.
- Папка `autopilot/` в этом же репо: конфиги, schemas, runtime, отчёты, learnings, lib.
- Telegram-уведомления через `curl` без отдельного сервиса.
- Reconciliation: конфиг описывает желаемое состояние, бот сравнивает с фактом.
- Двухслойная память: operational (источник правды) + narrative (для агента).
- Ownership через labels Yandex Direct (`autopilot:managed`, `city:<>`, `topic:<>`).
- Onboarding нового города через baseline scan и launch_proposal.
- Структурный action ledger, deterministic idempotency, per-city lock, drift detection, atomic writes.

**Вне скоупа.**
- Codex-зеркало автопилота (Claude-only исключение зафиксировано в `.claude/skills/leadgen-autopilot/CLAUDE.md`). Shared-файлы `leadgen/{references,library,branches}` всё равно синхронизируются с `.codex/skills/leadgen-codex/`.
- Webhook-сервер для inline-кнопок Telegram.
- Распределённый запуск (всё на одной машине).
- Автоматическое редактирование файлов скилла (только предложения через `learnings/`).
- Изменение существующего скилла `leadgen` под автопилот — он остаётся для специалиста.
- Adoption существующих кампаний в финальном e2e тесте (пилот стартует с нуля). Механизм adoption реализуется в W6 для будущего использования.

**Ключевые ограничения.**
- Не оптимизировать токены/контекст в ущерб качеству решения.
- DRAFT-only обязательно: `campaign.create_draft.*` создаёт SUSPENDED, активация — отдельным действием. Это правило проекта (`create-search.md` шаг 11).
- Block-actions нерушимы даже в `full_auto`: `legal.update_disclaimer`, `legal.change_company_info`, `account.*`, `campaign.delete`.

---

## 2. Архитектура: где что лежит

```
leadgen-mcp/
├── PLAN.md                                 (этот файл)
├── PLAN_v1_archive.md                      исходный драфт
├── plan_correctives.md                     правки Codex раунд 1
├── plan_correctives2.md                    ответ Claude
├── plan_correctives3.md                    финальная позиция Codex
│
├── autopilot/                              ← НОВАЯ папка
│   ├── README.md                            быстрый старт
│   ├── CLAUDE.md                            корневой роутер автопилота
│   ├── HALT.flag                            глобальный стоп всех городов
│   ├── config/
│   │   ├── caps_defaults.yaml               глобальные дефолты caps
│   │   ├── trust_profiles/
│   │   │   ├── conservative.yaml            строгий профиль (review_queue по умолчанию)
│   │   │   ├── balanced.yaml                сбалансированный (auto_with_notify по умолчанию)
│   │   │   ├── aggressive.yaml              минимум approvals
│   │   │   └── pilot_full_auto.yaml         ПИЛОТНЫЙ ПРОФИЛЬ (эксперимент: full autonomy)
│   │   ├── secrets.env.example              шаблон
│   │   ├── secrets.env                      .gitignored
│   │   └── cities/
│   │       ├── _example.yaml                эталон
│   │       └── <pilot_city>.yaml            пилот (имя позднее)
│   ├── schemas/
│   │   ├── city_config.schema.json
│   │   ├── state.schema.json
│   │   ├── action.schema.json
│   │   ├── approval.schema.json
│   │   ├── ledger_entry.schema.json
│   │   ├── metrics_snapshot.schema.json
│   │   └── launch_proposal.schema.json
│   ├── runtime/                             [.gitignore]
│   │   └── <city>/
│   │       ├── HALT.flag                    per-city стоп
│   │       ├── locks/RUNNING.lock
│   │       ├── state.yaml                   [eager] machine source of truth
│   │       ├── action_ledger.jsonl          [append-only]
│   │       ├── metrics_snapshots.jsonl
│   │       ├── pending_approvals.yaml       (только если autonomy=with_approvals)
│   │       ├── before_snapshots/<action_id>.json
│   │       ├── narrative/                   [для агента, lazy + eager]
│   │       │   ├── STATE.md                 регенерируется из state.yaml
│   │       │   ├── CURSOR.md
│   │       │   ├── SUMMARY.md
│   │       │   ├── pending_approvals.md     рендер из yaml
│   │       │   ├── runs/<YYYY-MM>/<DD-HHMM>.md
│   │       │   ├── campaigns/<id>.md
│   │       │   └── decisions/<topic>-<slug>.md
│   │       └── onboarding/
│   │           ├── launch_proposal.md
│   │           └── launch_proposal.yaml
│   ├── reports/                             [.gitignore]
│   │   └── <city>/<YYYY-MM>/
│   │       ├── <DD-HHMM>-daily.html
│   │       ├── week-<NN>.html
│   │       └── month-<MM>.html
│   ├── learnings/                           [НЕ gitignored — это артефакт]
│   │   ├── proposed/<id>.md
│   │   ├── validated/<id>.md
│   │   └── rejected/<id>.md
│   └── lib/
│       ├── telegram_send.sh                 sendMessage
│       ├── telegram_send_doc.sh             sendDocument
│       ├── telegram_check_replies.sh        getUpdates + persisted update_offset
│       ├── render_html.sh                   md→html
│       ├── atomic_write.sh                  tmp → validate → rename → .bak
│       ├── lock.sh                          per-city lock helpers
│       └── memory_lookup.sh                 grep по тегам lazy
│
├── .claude/skills/leadgen/                  ← существующий скилл, не трогаем
│
└── .claude/skills/leadgen-autopilot/         ← НОВЫЙ скилл
    ├── CLAUDE.md                             зафиксированное Claude-only исключение
    ├── skill.md                              роутер скилла
    ├── flow-steps.md                         анкоры шагов (D1..D8 daily, W1..W5 weekly, M1..M6 monthly)
    ├── branches/
    │   ├── analyze.md                        daily/weekly/monthly анализ
    │   ├── reconcile_config.md               diff config↔state↔API
    │   ├── decide.md                         signal → action + caps/permissions/cooldown
    │   ├── apply.md                          выполнение через MCP + log_change_event
    │   ├── memory_write.md                   обновление operational + narrative
    │   ├── notify.md                         формирование отчёта + Telegram
    │   ├── approval.md                       (только если autonomy=with_approvals)
    │   ├── learnings.md                      hypothesis lifecycle
    │   ├── safety.md                         kill-switch, idempotency, rollback
    │   └── onboarding.md                     baseline scan + launch_proposal
    └── references/
        ├── playbook_contract.md              какие leadgen/branches читаем как playbook
        ├── signal_catalog.md                 каталог сигналов
        ├── action_catalog.md                 каталог действий с risk_class и min_evidence
        ├── decision_priorities.md            приоритезация при множественных сигналах
        └── shared_refs.md                    маппинг shared-файлов из leadgen
```

**Принцип.** Скилл = код (правила, флоу). Папка `autopilot/` = runtime-данные. Обновление скилла — через PR. Обновление `autopilot/runtime/` — runtime, делается агентом.

**Связь с `leadgen`.** Параллельные скиллы. Не вызывают друг друга в runtime. Shared playbooks читаются через `playbook_contract.md` — стабильные anchors в `leadgen/branches/{create,optimize}-{search,rsya}.md`. Изменения этих anchors требуют sync проверки.

---

## 3. City config (формат YAML)

`autopilot/config/cities/<city>.yaml`. Все business-параметры здесь.

```yaml
# === Идентификация ===
city: <pilot_city>                         # имя пилота определится отдельно
client_login: <client_login>               # из MCP get_city_config
counter_id: <metrika_counter>
geo_region_id: <direct_geo_id>
domain: <city>.etagi.com
tier: 2                                    # 1/2/3 — для подбора benchmarks
timezone: Asia/Omsk                        # явно для freshness/window расчётов

# === Autonomy mode ===
# full_auto      — бот действует сам без approvals (пилотный эксперимент)
# with_approvals — действия medium+ идут в pending_approvals для review через Telegram
# read_only      — бот только наблюдает и предлагает (никаких apply, любые planned actions логируются)
autonomy_mode: full_auto

# === Trust profile ===
# Базовый набор permissions. Точечные city-overrides идут ниже в `permissions`.
trust_profile: pilot_full_auto

# === Метрика и цели ===
metrika:
  counter_id: <counter_id>
  goals:
    lead_form:
      id: 100001
      name: "Заявка форма"
      attribution: LYDC
      value_type: leads
    call:
      id: 100002
      name: "Звонок"
      attribution: LYDC
      value_type: leads
    qualified_lead:
      id: 100004
      name: "Квал. лид CRM"
      attribution: LYDC
      value_type: qualified_leads
  primary_conversion_goal: lead_form
  secondary_goals: [call]
  qualified_goal: qualified_lead             # отдельная атрибуция для weekly/monthly

# === Тематики ===
# Бот работает только с status=active или experimental, согласно правилам status.
topics:
  vtorichka:
    status: active                          # candidate | experimental | active | paused | blocked
    allowed_channels: [search, rsya]
    monthly_budget: 60000
    target_cpa_form: 3500
    target_cpa_call: 2800
    target_cpa_qualified: 12000             # weekly/monthly only
    notes: "Приоритет — поиск, РСЯ — добивка"
  novostroyki:
    status: active
    allowed_channels: [search, rsya]
    monthly_budget: 80000
    target_cpa_form: 5000
    target_cpa_call: 4000
    target_cpa_qualified: 15000
  zagorodka:
    status: candidate                       # бот может анализировать, не запускать
    allowed_channels: [search, rsya]
    monthly_budget: 0

# === Глобальный бюджет ===
budget:
  total_monthly_limit: 200000
  daily_pacing: linear                      # linear | front_loaded | back_loaded
  weekend_modifier: 0.8
  reserve_pct_for_month_end: 10

# === Расписание ===
schedule:
  daily_runs_per_day: 1
  preferred_hours_msk: [10]
  weekly_rollup_dow: monday
  monthly_rollup_dom: 1

# === Caps overrides ===
# null = взять из caps_defaults.yaml. Город может ужесточить.
caps:
  max_daily_bid_change_pct: null
  max_daily_budget_change_pct: null
  cooldown_hours_after_create: null
  max_actions_per_run: null

# === Permissions overrides ===
# null = взять из trust_profiles/<trust_profile>.yaml. City может переопределить точечно.
permissions:
  campaign.create_draft.in_new_topic: null
  campaign.activate_existing_draft: null
  bid.change.above_cap: null
  # ...

# === Topic-level permissions overrides (опционально) ===
# Если одна тема рискованнее другой, можно переопределить тут.
# topics_permissions:
#   novostroyki:
#     campaign.activate_existing_draft: auto_with_notify

# === Уведомления ===
notify:
  telegram_chat_id: -1001234567890
  daily_summary: true
  weekly_report: true
  monthly_report: true
  alert_thresholds:
    cpa_jump_pct: 50
    budget_overrun_pct: 110
    no_conversions_days: 3
    impressions_drop_pct: 50

# === Кастомные правила (свободный текст) ===
custom_rules:
  - "Не повышать ставки на vtorichka выше 80₽ на клик (конкуренция низкая, выше — слив)"
  - "Не запускать рекламу новостроек до 1 числа месяца"

# === Holdout (опционально) ===
holdout:
  enabled: false
  campaign_ids: []                          # будут помечены label autopilot:holdout

# === Холодный старт ===
baseline_mode: false                        # для пилота "с нуля" сразу false, бот стартует онбординг → создание
```

**Почему YAML, а не роутер:** валидируется JSON Schema, версионируется в git, читается отдельно от логики, легко добавить/выключить тематику без правки скилла.

---

## 4. Trust profiles и autonomy modes

### 4.1. Resolution-порядок permissions

```
1. autopilot/config/caps_defaults.yaml          глобальный baseline
2. autopilot/config/trust_profiles/<profile>.yaml
3. autopilot/config/cities/<city>.yaml.permissions
4. autopilot/config/cities/<city>.yaml.topics_permissions.<topic>
5. runtime override через approval (только если autonomy_mode=with_approvals)
```

В `runs/<id>.md` пишется resolved permissions с источником каждого значения.

### 4.2. Autonomy modes

| Mode | Что делает |
|---|---|
| `full_auto` | Любое действие, не помеченное `block`, выполняется без approval. `auto`/`auto_with_notify` срабатывают как `auto`. `review_queue` срабатывает как `auto_with_notify`. **Block-actions** (legal/account/billing/campaign.delete) остаются `block`. |
| `with_approvals` | Стандартное поведение. `auto`/`auto_with_notify`/`review_queue`/`block` различаются. Approval-цепочка через Telegram активна. |
| `read_only` | Любое действие не применяется. Все proposed actions логируются в `runs/<id>.md` и отчёт — для аудита. |

`autonomy_mode` в `city.yaml` — главный переключатель эксперимента. Hard-блоки нельзя обойти.

### 4.3. Trust profiles

| Profile | Назначение | Ставит большинство actions в… |
|---|---|---|
| `conservative` | Критичные аккаунты, новые городовые, после инцидента | `review_queue` для medium+ |
| `balanced` | Стабильные города после первичной валидации | `auto_with_notify` для medium |
| `aggressive` | Зрелые города с подтверждённой историей | `auto` для medium, `auto_with_notify` для high |
| `pilot_full_auto` | **Пилотный эксперимент:** максимум автономии при сохранении block-границ | `auto`/`auto_with_notify` почти везде, `block` только для critical |

`pilot_full_auto.yaml` — содержит расширенный набор (см. раздел 5). Profile + `autonomy_mode: full_auto` дают эффект "отпустить вожжи".

---

## 5. Каталог действий (action_catalog.md)

Каждое действие имеет:
- `risk_class`: `low | medium | high | critical`
- `default_permission` (зависит от профиля)
- `min_evidence` (clicks/conversions/spend/days/window)
- `cooldown_hours`
- `snapshot_required`: true для `risk>=medium` или bulk/replace/irreversible
- `expected_effect_template` (для applied medium/high)
- `guard_metric` (для medium+)
- `rollback_trigger` (только high/critical)

### 5.1. Полная таблица (компактная) — permissions для `pilot_full_auto`

| # | Action | risk | pilot_full_auto | conservative | balanced |
|---|---|---|---|---|---|
| **Бюджет** |
| 1 | `budget.increase.within_cap` | low | auto | auto | auto |
| 2 | `budget.increase.above_cap` | medium | auto_with_notify | review_queue | review_queue |
| 3 | `budget.decrease.within_cap` | low | auto | auto | auto |
| 4 | `budget.decrease.above_cap` | medium | auto_with_notify | review_queue | review_queue |
| 5 | `budget.set_total_monthly` | medium | auto_with_notify | review_queue | auto_with_notify |
| 6 | `budget.pause_due_to_overrun` | high | auto_with_notify | auto_with_notify | auto_with_notify |
| **Ставки** |
| 7 | `bid.increase.within_cap` | low | auto | auto | auto |
| 8 | `bid.decrease.within_cap` | low | auto | auto | auto |
| 9 | `bid.change.above_cap` | medium | auto_with_notify | review_queue | auto_with_notify |
| 10 | `bid.adjust_strategy_target_cpa.within_cap` | medium | auto_with_notify | auto_with_notify | auto_with_notify |
| 11 | `bid.adjust_strategy_target_cpa.above_cap` | high | auto_with_notify | review_queue | review_queue |
| **Минусы** |
| 12 | `negatives.add_from_search_queries` | low | auto | auto | auto |
| 13 | `negatives.add_global_set` | low | auto | auto | auto |
| 14 | `negatives.remove` | medium | auto_with_notify | review_queue | review_queue |
| **Кампании** |
| 15 | `campaign.create_draft.in_existing_topic` | medium | auto_with_notify | auto_with_notify | auto_with_notify |
| 16 | `campaign.create_draft.in_new_topic` | medium | auto_with_notify | review_queue | auto_with_notify |
| 17 | `campaign.create_draft.outside_config_topics` | critical | **block** | block | block |
| 18 | `campaign.activate_existing_draft` | high | **auto_with_notify** ⚠ эксперимент | review_queue | review_queue |
| 19 | `campaign.pause.low_performance` | medium | auto_with_notify | auto_with_notify | auto_with_notify |
| 20 | `campaign.pause.budget_exhausted` | low | auto | auto | auto |
| 21 | `campaign.resume` | medium | auto_with_notify | review_queue | auto_with_notify |
| 22 | `campaign.archive.no_traffic_30days` | medium | auto_with_notify | auto_with_notify | auto_with_notify |
| 23 | `campaign.adopt_existing` | high | auto_with_notify | review_queue | review_queue |
| 24 | `campaign.release` | medium | auto_with_notify | review_queue | review_queue |
| 25 | `campaign.delete` | critical | **block** | block | block |
| **Группы и объявления** |
| 26 | `adgroup.pause.low_ctr` | medium | auto_with_notify | auto_with_notify | auto_with_notify |
| 27 | `adgroup.split_by_intent` | medium | auto_with_notify | review_queue | auto_with_notify |
| 28 | `ad.add_new_variant` | low | auto | auto | auto |
| 29 | `ad.pause.low_ctr` | low | auto | auto | auto |
| 30 | `ad.update_text` | medium | auto_with_notify | review_queue | auto_with_notify |
| 31 | `ad.update_image_creative` | medium | auto_with_notify | review_queue | auto_with_notify |
| **Ключевые** |
| 32 | `keyword.pause.low_performance` | low | auto | auto | auto |
| 33 | `keyword.add_in_existing_group` | low | auto | auto | auto |
| 34 | `keyword.add_new_group_in_existing_topic` | medium | auto_with_notify | auto_with_notify | auto_with_notify |
| 35 | `keyword.remove` | medium | auto_with_notify | auto_with_notify | auto_with_notify |
| **Стратегии** |
| 36 | `strategy.change_type` | high | auto_with_notify | review_queue | review_queue |
| 37 | `strategy.adjust_constraint` | medium | auto_with_notify | auto_with_notify | auto_with_notify |
| **Площадки РСЯ** |
| 38 | `placement.block.low_performing` | low | auto | auto | auto |
| 39 | `placement.block.from_blacklist` | low | auto | auto | auto |
| 40 | `placement.unblock` | medium | auto_with_notify | review_queue | review_queue |
| **Аудитории** |
| 41 | `audience.create` | low | auto | auto_with_notify | auto |
| 42 | `audience.adjust_bid_modifier` | low | auto | auto | auto |
| 43 | `retargeting.create_list` | low | auto | auto_with_notify | auto |
| **Регионы** |
| 44 | `region.adjust_bid_modifier` | low | auto | auto | auto |
| 45 | `region.add_to_targeting` | high | auto_with_notify | review_queue | review_queue |
| 46 | `region.remove_from_targeting` | high | auto_with_notify | review_queue | review_queue |
| **Расширения** |
| 47 | `extension.add_sitelinks` | low | auto | auto | auto |
| 48 | `extension.update_sitelinks` | low | auto | auto | auto |
| 49 | `extension.update_vcard` | medium | auto_with_notify | review_queue | review_queue |
| **Креативы** |
| 50 | `creative.generate_new_image` | medium | auto_with_notify | review_queue | auto_with_notify |
| 51 | `creative.update_text_variant` | low | auto | auto | auto |
| **Юр-чувствительное** |
| 52 | `legal.update_disclaimer` | critical | **block** | block | block |
| 53 | `legal.change_landing_url` | high | auto_with_notify | review_queue | review_queue |
| 54 | `legal.change_company_info` | critical | **block** | block | block |
| **Аккаунт** |
| 55 | `account.change_settings` | critical | **block** | block | block |
| 56 | `account.change_billing` | critical | **block** | block | block |
| 57 | `account.close` | critical | **block** | block | block |

⚠ **Эксперимент пилота** строка 18: `campaign.activate_existing_draft = auto_with_notify`. В standard профилях остаётся `review_queue`. После пилота — оценка корректности этого решения.

### 5.2. Search vs RSYA — отдельные defaults

В `caps_defaults.yaml`:

```yaml
channels:
  search:
    min_clicks_for_decision: 30
    min_conversions_for_decision: 3
    cooldown_hours_after_bid_change: 24
    learning_period_days: 7
  rsya:
    min_clicks_for_decision: 100
    min_conversions_for_decision: 5
    cooldown_hours_after_bid_change: 48
    learning_period_days: 14
```

Per-action overrides — в `action_catalog.md` отдельной колонкой.

### 5.3. Snapshot policy

| risk_class | Snapshot |
|---|---|
| `low` (single-field additive) | Только ledger row с `before_value/after_value` |
| `medium+` | Structured `before_snapshot` JSON в `before_snapshots/<action_id>.json` |
| bulk/replace/shared-setting | Snapshot независимо от risk_class |
| irreversible | `rollback: manual_only` |

---

## 6. Память: operational + narrative

### 6.1. Operational layer (источник правды)

| Файл | Формат | Назначение |
|---|---|---|
| `state.yaml` | YAML, schema-validated | Текущее состояние (бюджет, тематики, кампании, ownership) |
| `action_ledger.jsonl` | append-only JSONL | Все попытки действий: `{run_id, idempotency_key, action, before_snapshot_ref, status, applied_at, error?}` |
| `metrics_snapshots.jsonl` | append-only JSONL | Ежедневные срезы по city/topic/channel/campaign |
| `pending_approvals.yaml` | YAML | Очередь approvals (только при `autonomy=with_approvals`) |
| `before_snapshots/<action_id>.json` | JSON | Snapshot для `risk>=medium` |

`state.yaml` — кеш. **API — source of truth.** При расхождении — drift flow с `human_override` cooldown.

`action_ledger.jsonl` — append-only, не редактируется. Каждая строка — попытка действия с idempotency key и статусом lifecycle.

### 6.2. Narrative layer (для агента)

| Файл | Объём | Назначение |
|---|---|---|
| `narrative/STATE.md` | ≤200 строк | Регенерируется из `state.yaml` в конце прогона |
| `narrative/CURSOR.md` | ≤80 строк | План: что сделано, что отложено, что pending |
| `narrative/SUMMARY.md` | ≤500 строк (с компрессией) | Хронология: 30 дней детально, далее сжатие |
| `narrative/pending_approvals.md` | рендер из yaml | Для удобства чтения |
| `narrative/runs/<YYYY-MM>/<DD-HHMM>.md` | lazy | Полный лог прогона: контекст, сигналы, решения, действия |
| `narrative/campaigns/<id>.md` | lazy | История изменений по кампании в обратном хроно |
| `narrative/decisions/<topic>-<slug>.md` | lazy | Нестандартные кейсы, гипотезы, эксперименты |

**Lazy lookup** через grep по тегам в шапке:
```markdown
---
tags: [campaign:8765432] [topic:vtorichka] [channel:rsya] [city:omsk]
last_action: 2026-04-29
status: running
---
```

`grep -l "\[campaign:8765432\]" runtime/<city>/narrative/campaigns/`

**Компрессия SUMMARY** — раз в неделю в weekly rollup:
- >30 дней → недельные строки;
- >90 дней → месячные строки;
- >365 дней → перенос в `decisions/historical-<year>.md`.

### 6.3. Atomic writes

Все eager-файлы (`state.yaml`, `STATE.md`, `CURSOR.md`, `SUMMARY.md`, `pending_approvals.{yaml,md}`):
- `tmp` → schema validate → atomic rename → хранить `.bak`
- При corruption — восстановление из `.bak` + drift check с API

`action_ledger.jsonl` — append-only с per-row schema validation.

---

## 7. Цикл daily/weekly/monthly

| День | Что выполняется |
|---|---|
| Обычный | full `daily` |
| `weekly_rollup_dow` (monday) | `daily_safety_check` + `weekly` |
| `monthly_rollup_dom` (1-е) | `daily_safety_check` + `monthly` |
| Совпадение weekly+monthly | `daily_safety_check` + `monthly` (weekly merged) |

`daily_safety_check`:
- HALT (global + per-city);
- lock check + recovery если stale;
- critical drift detection;
- hard pacing cap (`forecast_month_end_spend > monthly_budget * 1.2`);
- expired approvals processing;
- notify failure alerts.

**Никаких tactical daily-decide в weekly/monthly день.** Это исключает противоречивые решения в одном прогоне.

`daily`:
- D1: load state, check HALT/lock
- D2: pull approvals (если applicable)
- D3: fetch metrics (LYDC, full days)
- D4: extract signals
- D5: reconcile config↔state↔API
- D6: build action plan (with evidence/confidence/reason_code/risk)
- D7: apply actions (по autonomy_mode и permissions)
- D8: write memory (operational + narrative) + report + Telegram

`weekly`:
- W1: weekly metrics (week-over-week по тематикам)
- W2: tactical план на след. неделю → CURSOR
- W3: SUMMARY compression
- W4: weekly HTML report с графиками
- W5: Telegram push с приложением

`monthly`:
- M1: monthly metrics + decision precision + rollback rate
- M2: budget plan на след. месяц по тематикам с обоснованием
- M3: learnings monthly digest (proposed → suggest validated)
- M4: holdout comparison (если включён)
- M5: monthly HTML report
- M6: Telegram push

---

## 8. Reconciliation + onboarding

### 8.1. Reconcile-алгоритм (на каждом daily-прогоне)

1. Прочитать `city.yaml`.
2. Прочитать `state.yaml` → `topics_in_state`.
3. Прочитать API: `get_campaigns`, `get_campaign_stats` (LYDC).
4. Дельты config↔state:
   - **Новая `active` тематика** → `campaign.create_draft.in_new_topic` для каждого канала.
   - **`active → paused`** → `campaign.pause.low_performance` для всех её campaigns.
   - **Изменился `monthly_budget`** → `budget.set_total_monthly`.
   - **Изменился `target_cpa_*`** → `bid.adjust_strategy_target_cpa.*`.
5. Дельты state↔API (drift):
   - API state ≠ expected before_state → `get_change_history` → human change?
     - Yes → `human_override` cooldown 24-72ч + alert.
     - No → recovery flag + alert.
6. Каждое запланированное действие проходит decision-фильтр (caps + permissions + cooldown + idempotency).

### 8.2. Onboarding (для пилота: новый аккаунт с нуля)

Шаги первого запуска нового города:

1. **Inventory.** Прочитать `city.yaml`, проверить:
   - Метрика: `metrika_get_counter`, цели, UTM.
   - Direct: `get_campaigns` (на новом аккаунте — пусто).
   - Связка Direct↔Metrika.
2. **Baseline scan** (вне зависимости от того, пуст аккаунт или нет):
   - Если есть кампании — категоризация (`owned/adoptable/foreign/holdout`).
   - Анализ спроса по разрешённым `active`/`experimental`/`candidate` темам через wordstat.
3. **Launch proposal** (`onboarding/launch_proposal.{md,yaml}`):
   - Для каждой `active` темы: предлагаемые кампании, бюджет, target CPA, channels.
   - `adoptable_campaigns` (если есть) — рекомендации `adopt | leave_readonly`.
   - Риски и зависимости.
4. **Approval gate.**
   - Если `autonomy_mode=full_auto` — proposal сохраняется как audit trail, бот **сразу** переходит к шагу 5.
   - Если `autonomy_mode=with_approvals` — proposal в Telegram, ждёт `approve <id>`.
5. **Draft creation.** `campaign.create_draft.*` для каждой одобренной/запланированной темы. Ownership labels: `autopilot:managed`, `city:<>`, `topic:<>`, `channel:<>`.
6. **Activation.**
   - В `pilot_full_auto` + `autonomy_mode=full_auto`: `campaign.activate_existing_draft = auto_with_notify` срабатывает автоматически, кампании становятся живыми с уведомлением в Telegram.
   - В `with_approvals`: ждёт явного approve.

**Принципиально для пилота "с нуля":** в `pilot_full_auto` + `full_auto` весь onboarding-цикл выполняется в одном прогоне без вмешательства специалиста. Telegram получает: launch_proposal + draft creation + activation summary.

---

## 9. Idempotency, lock, drift, rollback

### 9.1. Deterministic idempotency key

```
<city>|<account>|<entity_type>|<entity_id>|<action_type>|<desired_value>|<decision_window>
```

Примеры:
- `omsk|ethaji-omsk|campaign|8765432|budget.set|65000|2026-05`
- `omsk|ethaji-omsk|placement|c876|block|example.com|2026-04-30`
- `omsk|ethaji-omsk|topic|vtorichka|create_draft|search|2026-Q2`

`decision_window` — единица идемпотентности по action_type:
- бюджет month → `YYYY-MM`
- ставка/минусы → `YYYY-MM-DD`
- стратегия → `YYYY-WNN`
- creation → quarter

Перед apply: проверка local `action_ledger.jsonl` по key + cross-check с `get_change_history`. Совпадение — skip.

### 9.2. Per-city lock

```yaml
# autopilot/runtime/<city>/locks/RUNNING.lock
city: omsk
run_id: omsk-2026-04-30-1030
started_at: 2026-04-30T10:30:00+05:00
phase: applying
ttl_minutes: 120
```

- Свежий lock — новый run выходит с alert.
- Stale lock 1 раз → recovery + warning alert.
- Stale lock 2 раза подряд → incident alert.

Recovery читает last `runs/<id>.md` lifecycle phase + сверяет ledger с API.

### 9.3. Run lifecycle (structured phases)

`started → loaded_context → fetched_metrics → planned_actions → approval_checked → applying → applied_partial → memory_written → notified → succeeded | failed | halted`

Каждая транзакция фазы пишется в `runs/<id>.md` + в `state.yaml.last_run`. Recovery понимает, на какой фазе упал прошлый прогон.

### 9.4. Drift handling

```python
expected_before = state.yaml.<entity>
actual = api.get_<entity>()
if expected_before != actual:
    history = get_change_history(entity, last 24h)
    if history.has_human_change():
        mark_human_override(entity, freeze_hours=72)
        alert("Human change detected on <entity>, freezing")
        skip_planned_actions(entity)
    else:
        mark_drift(entity)
        alert("Unexplained drift on <entity>")
        skip_planned_actions(entity)
```

### 9.5. Rollback (двухступенчатый)

- Telegram: `rollback <run_id>` → бот формирует **dry-run plan** обратных действий → отправляет в Telegram.
- Telegram: `confirm rollback <id>` → бот применяет.
- `rollback: manual_only` для необратимых (creative generation, удалённые объявления, активированные стратегии после долгого обучения).
- Failed rollback → alert уровня incident.

---

## 10. Telegram + approvals

### 10.1. Конфигурация

- Один bot token в `secrets.env`.
- Allowlist `chat_id` (per-city из `city.yaml.notify.telegram_chat_id`).
- Persisted `update_offset` в `runtime/_global/telegram_offset` (защита от dup updates после crash).

### 10.2. Защиты

- HTML/Markdown escaping всех динамических полей.
- Лимит длины 4096 символов с fallback на `sendDocument`.
- Retry с exponential backoff (3 попытки).
- Маскирование токена в логах.
- Alert-сообщение при падении notify (попытка через alternate channel или local file).

### 10.3. Daily-уведомление (compact)

```
🟢 Омск · 29 апр 14:30 · full_auto
Бюджет день: 8 420 / 10 000 ₽ (84%)
Лиды: 12 · CPA 702 (цель 800)
Действия: 4 (auto: 3, auto_with_notify: 1)
  · vtorichka-search: +12 минусов, +bid 10%
  · vtorichka-rsya: +34 заблоч.площадки
  · novostroyki-search: ⚠ activated draft "Омск-Новостройки-Поиск"
⚠ novostroyki: CPA 1 450, день 7 обучения, target 1 000
[detail.html прикреплён]
```

### 10.4. Approval-команды (только при `autonomy_mode=with_approvals`)

```
approve <id>
reject <id>
defer <id> <Nd>
rollback <run_id>
confirm rollback <id>
```

Approval expiry:
- 24h для bid/budget;
- 72h для draft creation/adoption;
- после истечения — пересчёт, не apply старого решения.

---

## 11. Pacing-контроль

```python
elapsed_share = days_passed_in_month / days_in_month
expected_spend = monthly_budget * elapsed_share
pacing_ratio = spent_mtd / expected_spend
forecast_month_end = avg_daily_spend_recent * days_in_month
```

| Состояние | Pacing ratio / Forecast | Действие |
|---|---|---|
| Normal | `pacing_ratio < 1.1` and `forecast < 1.1 * budget` | штатно |
| Conservation | `pacing_ratio > 1.1` or `forecast > 1.1 * budget` | агрессивные actions disabled, только защитные минусы/блокировки |
| Emergency | `pacing_ratio > 1.25` or `forecast > 1.2 * budget` | emergency pause всех managed campaigns, alert |
| Hard cap | `spent_mtd > 1.2 * monthly_budget` | принудительная пауза + incident alert |

Раздельный учёт: `managed_spend` / `unmanaged_spend` / `total_account_spend`. Бот принимает решения по managed-бюджету, но в отчёте показывает все три.

---

## 12. Learnings

### 12.1. Lifecycle

```
proposed (≥1 наблюдение)
  → evidence accumulation (positive + negative cases)
  → validated (≥3 повтора, 14 дней без отката, scope-stable)
  → monthly digest (предложение специалисту)
  → specialist review через PR в lessons_registry.md
  → behaviour change (только после кода-правки скилла)

   ↘ rejected (специалист отказал)
   ↘ expired (>60 дней без подтверждения, re-check)
```

### 12.2. Файлы

- `learnings/proposed/<id>.md` — гипотеза с `confidence`, `observed_count`, `pattern`, `evidence`, `negative_evidence`, `scope`, `needs_repeats`.
- `learnings/validated/<id>.md` — после проверки. **НЕ применяется автоматически.** Идёт в monthly digest.
- `learnings/rejected/<id>.md` — отклонено специалистом, больше не предлагается.

### 12.3. Scope

- city/topic/channel — обязательно.
- Не переносить между городами без явного approval специалиста.

### 12.4. Граница

Автопилот никогда сам не пишет в файлы скилла. Только в `autopilot/learnings/`.

---

## 13. Безопасность

| Механизм | Реализация |
|---|---|
| Глобальный kill-switch | `autopilot/HALT.flag` |
| Per-city kill-switch | `autopilot/runtime/<city>/HALT.flag` |
| Hard block actions | `legal.update_disclaimer`, `legal.change_company_info`, `account.*`, `campaign.delete`, `campaign.create_draft.outside_config_topics` — НИКОГДА, ни в одном профиле. |
| Idempotency | Deterministic key + ledger + change_history cross-check |
| Per-city lock | RUNNING.lock с TTL + recovery |
| Action cap | `max_actions_per_run` (мягкий, чтобы не терять решения по большому плану — раздел 14) |
| Atomic writes | Все eager-файлы |
| DRAFT-only | Все `campaign.create_draft.*` создают SUSPENDED. Активация — отдельным действием. |
| Pacing-контроль | Conservation/emergency/hard cap — раздел 11 |
| Drift detection | API > STATE; human_override cooldown 72ч |
| Rollback двухступенчатый | dry-run → confirm |

---

## 14. Caps defaults (`autopilot/config/caps_defaults.yaml`)

```yaml
# === Лимиты темпа изменений ===
max_daily_bid_change_pct: 20             # за прогон не двигать ставку >±20%
max_daily_budget_change_pct: 30
max_daily_target_cpa_change_pct: 15

# === Cooldowns (часы) ===
cooldown_hours_after_create: 72
cooldown_hours_after_bid_change: 24
cooldown_hours_after_strategy_change: 168
cooldown_hours_after_budget_change: 24

# === Пороги статистической значимости (default; channel-specific overrides ниже) ===
channels:
  search:
    min_clicks_for_decision: 30
    min_conversions_for_decision: 3
    cooldown_hours_after_bid_change: 24
    learning_period_days: 7
  rsya:
    min_clicks_for_decision: 100
    min_conversions_for_decision: 5
    cooldown_hours_after_bid_change: 48
    learning_period_days: 14

# === Лимиты на прогон ===
max_actions_per_run: 50                  # мягкий лимит; не отбрасывать качественные решения
max_new_campaigns_per_run: 4             # увеличено для onboarding с нуля
max_pauses_per_run: 10

# === Бюджетные защитные правила ===
budget_overrun_hard_stop_pct: 120
budget_emergency_pct: 120                # forecast threshold
budget_conservation_pct: 110
min_remaining_budget_pct_for_aggressive_actions: 25

# === Пороги уведомлений (override в city.notify.alert_thresholds) ===
alert_cpa_jump_pct: 50
alert_budget_overrun_pct: 110
alert_no_conversions_days: 3
alert_impressions_drop_pct: 50

# === Permissions (overridden by trust_profile) ===
# Базовый, самый строгий — все unknown actions block
default_permission: review_queue
```

Permissions per profile — в `autopilot/config/trust_profiles/<profile>.yaml` (см. раздел 5.1).

---

## 15. Метрики качества автопилота (monthly)

- **Decision precision** — доля действий, через 14 дней давших улучшение (CPA −X% или conv +Y%). Цель: >60%.
- **Rollback rate** — доля действий, откатанных. Цель: <10%.
- **Approval queue health** (только при `with_approvals`) — среднее время в pending. Цель: <72ч.
- **Coverage** — % дней без падений/HALT. Цель: >95%.
- **Telegram noise** — `auto_with_notify` сообщений в день. Если >5 — пересмотреть permissions.
- **Pacing accuracy** — отклонение `forecast_month_end` от факта. Цель: ±10%.
- **Drift incidents** — кол-во `human_override` срабатываний. Информационная метрика.

---

## 16. Волны разработки

> Разрабатываем **полный контур**, тестируем на финальном e2e (не на промежуточной волне). Self-checks внутри каждой волны обязательны, чтобы финальный тест не превратился в отладку базовой инфраструктуры.

### W1 — Каркас, schemas, gitignore, lock, Telegram hello

**Содержание:**
- Структура `autopilot/{config,runtime,reports,learnings,lib,schemas}/`
- `.gitignore` для runtime/secrets/reports
- JSON Schemas всех артефактов (city_config, state, action, approval, ledger_entry, metrics_snapshot, launch_proposal)
- `.claude/skills/leadgen-autopilot/{skill.md,CLAUDE.md,flow-steps.md}`
- `autopilot/CLAUDE.md` — корневой роутер
- `caps_defaults.yaml`, все 4 trust profiles, `cities/_example.yaml`, `secrets.env.example`
- `lib/{telegram_send,telegram_send_doc,atomic_write,lock}.sh`
- HALT.flag (global + per-city), per-city RUNNING.lock с TTL
- cwd = `C:\git\leadgen-mcp`, prompt format `/leadgen-autopilot city=<city>`

**Self-check:** `/leadgen-autopilot city=<test>` запускается, читает HALT, берёт lock, шлёт `Hello from autopilot, city=<test>` в Telegram, отпускает lock, выходит.

### W2 — Память operational + narrative + atomic + recovery

**Содержание:**
- Templates всех narrative md
- Operational артефакты + schema validation per write
- `lib/atomic_write.sh` (tmp → validate → rename → .bak)
- `lib/memory_lookup.sh` (grep по тегам)
- Branch `branches/memory_write.md`
- Алгоритм компрессии SUMMARY
- Crash recovery: lock + ledger.last_status + drift check

**Self-check:** ручной прогон создаёт корректные файлы; повторный run читает их; corrupt state.yaml → recovery из .bak; mock weekly → компрессия.

### W3 — Аналитика, signals, dry-run plan, onboarding scan

**Содержание:**
- Branch `branches/analyze.md` (daily/weekly/monthly режимы)
- Сбор метрик с LYDC, alias через `city.yaml.metrika.goals`
- Freshness checks (timezone, data delay)
- `signal_catalog.md`, `action_catalog.md` с risk_class/min_evidence/cooldown
- Action plan с reason_code/confidence/evidence/risk
- Branch `branches/onboarding.md` — baseline scan + launch_proposal
- HTML render через `lib/render_html.sh`
- Daily Telegram summary (compact)

**Self-check:** в dev-окружении на тестовом аккаунте: scan состояния, генерация launch_proposal с inventory + demand analysis + рекомендациями.

### W4 — Approval queue + Telegram replies + expiry

**Содержание:**
- Branch `branches/approval.md` (активна только при `autonomy=with_approvals`)
- `pending_approvals.yaml` структура + рендер в md
- `lib/telegram_check_replies.sh` с persisted `update_offset`, allowlist
- Команды: `approve/reject/defer/rollback/confirm rollback`
- Expiry: 24h bid/budget, 72h create/adopt
- Связка `message_id` ↔ pending action ↔ run_id

**Self-check:** mock approval-цикл: bot → pending → reply → next run → apply.

### W5 — Apply engine, ownership, low-risk auto

**Содержание:**
- Branch `branches/apply.md`
- Deterministic idempotency key
- Ownership check: только `autopilot:managed`
- Drift check before apply
- Snapshot policy (low — ledger row, medium+ — structured snapshot)
- `log_change_event` в API + локальный ledger
- Автономия по `autonomy_mode`:
  - `full_auto`: применить любое не-block действие
  - `with_approvals`: только approved
  - `read_only`: ничего, всё в plan
- Запуск: negatives auto, blacklist placements, мелкие bid/budget в cap

**Self-check:** apply одного безопасного действия; повторный run — skip по idempotency; `read_only` mode — никаких apply.

### W6 — Reconciliation, onboarding, adoption, DRAFT-only, pacing

**Содержание:**
- Branch `branches/reconcile_config.md`
- Diff config↔state↔API
- Topic status transitions (`active→paused`, `candidate→active`)
- Onboarding flow для нового города (с нуля): scan → proposal → draft create → activate (по autonomy_mode)
- Adoption (для будущего): `campaign.adopt_existing` + `add_labels`. **На пилоте не тестируется.**
- DRAFT-only: `campaign.create_draft.*` всегда SUSPENDED, `campaign.activate_existing_draft` отдельно
- Drift detection + `human_override` cooldown
- Pacing-контроль: conservation/emergency/hard cap

**Self-check:** добавление новой `active` тематики в `city.yaml` → следующий run на пустом аккаунте создаёт draft → активирует (в `full_auto`) или ждёт approval (в `with_approvals`). Затем — `paused` → пауза кампаний.

### W7 — Weekly/monthly rollups, compression, quality metrics, holdout

**Содержание:**
- Weekly режим analyze: week-over-week по тематикам, обновление CURSOR
- Monthly режим: стратегия + budget plan + decision precision/rollback/coverage/noise/pacing accuracy
- Compression SUMMARY (>30/>90/>365)
- Holdout comparison (если включён в city.yaml)
- HTML reports с графиками (matplotlib png inline или ASCII fallback)
- Cycle priority (`daily_safety_check` в weekly/monthly день)

**Self-check:** mock weekly/monthly даты на тестовых данных → корректные отчёты + CURSOR обновлён.

### W8 — Medium/high-risk actions

**Содержание:**
- Включить (по autonomy_mode + permissions): pause/resume, ad variants, keyword/group expansion, audiences, retargeting, creative.generate_new_image (DRAFT-only), strategy.adjust_constraint
- Snapshot обязателен для risk≥medium
- Strict min_evidence enforcement
- Усиленные уведомления

**Self-check:** mock сценарий "high CPA → pause low_performing" с full evidence trail и snapshot.

### W9 — Learnings (proposed → digest)

**Содержание:**
- Branch `branches/learnings.md`
- Naissance: pattern detection → `proposed/<id>.md` с structured evidence + negative evidence
- НЕ авто-apply
- Monthly digest: предложение специалисту
- Rejection через approval/PR
- Expiry/recheck каждые 60 дней
- Scope tracking (city/topic/channel)

**Self-check:** mock pattern с 3 повторами за 14 дней → proposed; rejection → moved to rejected; expired → re-check trigger.

### W10 — Hardening, chaos, rollback, multi-city checklist

**Содержание:**
- Stress: MCP 5xx, битый конфиг, drift API↔STATE, stale lock, partial apply crash
- Rollback Telegram-команда с двухступенчатым confirm
- Hard cap budget overrun (emergency pause всех managed)
- Multi-city scaling: per-city schedule, lock, отчёты
- Документация специалисту (как читать отчёты, переключать autonomy_mode, управлять approval, читать learnings)
- Чеклист "scale to 3+ cities"

**Self-check:** chaos test 3 дня без вмешательства → 0 критичных инцидентов, лог recovery событий чистый.

---

## 17. Финальный e2e тест

Проводится **после** завершения всех волн, на пилотном городе с **новым аккаунтом**.

### 17.1. Подготовка

- `city.yaml` создан для пилотного города (имя/login/counter/geo заполнены)
- `autonomy_mode: full_auto`
- `trust_profile: pilot_full_auto`
- `topics`: 1-2 темы со `status: active`, 1 со `status: candidate` для проверки
- `notify.telegram_chat_id` указан, бот в чате
- `secrets.env` заполнен
- Routine в Claude Desktop настроена на 1 раз/день

### 17.2. Сценарий

1. **Onboarding запуск.** Первый прогон: baseline scan пустого аккаунта → launch_proposal → DRAFT creation для `active` тем → activation (в `full_auto` срабатывает auto_with_notify). Telegram получает сводку. Пользователь видит draft создания + активацию + первые показы.
2. **Daily cycle (7 дней).** Каждый день: метрики, signals, action plan, apply (low+medium-risk), report. Все actions без approval (`full_auto`).
3. **Idempotency drill.** Эмулировать crash в середине apply → следующий run корректно восстанавливается через ledger + drift check, не дублирует.
4. **Drift drill.** Специалист вручную правит ставку через UI Direct → next run обнаруживает, freeze entity на 72ч, не возвращает обратно, отчёт показывает `human_override`.
5. **Reconcile drill.** Изменить `topics.zagorodka.status: candidate → active` в `city.yaml` → следующий run создаёт DRAFT → активирует.
6. **Pacing drill.** Сэмулировать перерасход (manual budget set) → next run переходит в conservation → emergency.
7. **Mode switch drill.** Переключить `autonomy_mode: full_auto → with_approvals` в `city.yaml` → следующий run строит plan, но medium-actions кладёт в `pending_approvals.yaml` + Telegram message. `approve <id>` → apply.
8. **Weekly rollup.** В weekly_dow — отчёт неделя/неделя, обновление CURSOR.
9. **Monthly rollup.** В monthly_dom — стратегический отчёт + learnings digest + budget plan.
10. **Rollback drill.** В Telegram `rollback <run_id>` → бот показывает обратный план → `confirm rollback <id>` → откат успешен / `manual_only` для необратимых.
11. **HALT drill.** Создать `autopilot/HALT.flag` → следующий run выходит без действий, alert в Telegram.

### 17.3. Acceptance criteria

- [ ] Каждое proposed/applied action имеет `signal_id`, `reason_code`, `confidence`, `evidence`, `risk_class`.
- [ ] Applied medium/high дополнительно имеют `expected_effect`, `guard_metric`. High/critical — `rollback_trigger`.
- [ ] Повторный запуск на тех же данных не дублирует action plan (idempotency).
- [ ] Per-city lock работает; recovery после крашa корректен.
- [ ] `action_ledger.jsonl` не противоречит API (drift = 0 или явный `human_override`).
- [ ] Автопилот трогает только campaigns с label `autopilot:managed`. Foreign — read-only.
- [ ] Все новые кампании создаются как DRAFT/SUSPENDED. Активация — отдельным действием через `campaign.activate_existing_draft`.
- [ ] В `full_auto` activation выполняется автоматически; в `with_approvals` — только после `approve`.
- [ ] Block-actions не выполняются ни в одном профиле.
- [ ] Telegram approval (в `with_approvals` mode) работает с expiry; `update_offset` персистится; повторный approve не применяется дважды.
- [ ] HALT (global + per-city) корректно останавливают прогон.
- [ ] Drift report показывает любые расхождения STATE↔API с классификацией (human/unexplained).
- [ ] Resolved permissions из `caps_defaults` + `trust_profile` + `city.yaml` пишутся в run log с источником каждого значения.
- [ ] Pacing-контроль активирует conservation @ >1.1, emergency @ >1.25, hard cap @ >1.2 факта.
- [ ] Rollback требует двухступенчатого confirm; необратимые помечены `manual_only`.
- [ ] Quality metrics считаются в monthly (decision precision, rollback rate, coverage, telegram noise, pacing accuracy).
- [ ] Onboarding с нуля: пустой аккаунт → scan → proposal → drafts → активация → daily cycle, всё в `full_auto` без вмешательства.
- [ ] Mode switch `full_auto ↔ with_approvals ↔ read_only` корректно меняет поведение на следующем прогоне.

---

## 18. TODO специалисту перед стартом W1

- [ ] Создать Telegram bot, получить токен, создать чат для пилотного города, узнать `chat_id`.
- [ ] Заполнить `secrets.env` (`TELEGRAM_BOT_TOKEN`, `OPENROUTER_API_KEY`, прочие нужные).
- [ ] Когда определится пилотный город — создать `cities/<city>.yaml` по шаблону `_example.yaml`. До этого момента W1-W9 разрабатываются на mock/dev данных.
- [ ] Подтвердить `monthly_budget` и `target_cpa_*` для пилотных тематик.
- [ ] Подтвердить, что Claude Desktop routine может крутиться 24/7 (питание, сон Windows, обновления, пересоздание сессии).
- [ ] Подтвердить готовность дать боту `autonomy_mode: full_auto` на пилоте (понимание риска эксперимента).

---

## 19. Открытые вопросы (можно решать по ходу)

1. **HTML-рендер.** Pandoc локально или Python `markdown→html`? → решим в W2/W3.
2. **Графики в weekly/monthly отчётах.** matplotlib png inline vs ASCII? → решим в W7.
3. **`remove_labels` в MCP.** Нужно проверить, работает ли через `update_campaign` с пустым массивом. Если нет — fallback на `autopilot:released` (managed && !released). → решим в W6.
4. **Schema validation overhead.** Per-row для ledger, full-file periodic для остальных. Если станет медленно — оптимизация через batched write. → решим эмпирически в W2.
5. **Multi-account в одном login.** Сейчас 1:1 city↔login. Если в Этажах login покрывает несколько городов — обработать в W6.
6. **Время прогона.** Цель 5-15 мин/город. Измерить в W3 self-check, оптимизировать в W10.
7. **Watchdog.** Если 24ч без прогона — alert. Реализуется через отдельный routine или внешний scheduled-check. → решим в W10.
8. **`log_change_event` в Direct API.** Использовать как дополнительный audit trail (не как source of truth). → подтвердить в W5.

---

> **Memo для агента:** этот файл — итоговое ТЗ. После старта W1 любые изменения архитектуры — через PR + запись в `RECENT-CHANGES.md`. Все runtime-прогоны — в `autopilot/runtime/<city>/runs/`, не в корневом `RECENT-CHANGES.md`.
