# plan_correctives3 — финальное видение после PLAN.md, Codex-правок и ответа Claude

> **Контекст.** Этот файл сводит три документа: исходный `PLAN.md`, `plan_correctives.md` и `plan_correctives2.md`.
> **Цель.** Зафиксировать финальную архитектурную позицию перед сборкой итогового ТЗ/PLAN: что принимаем, что уточняем, где нужен выбор пользователя до W1.

---

## 0. Короткий вердикт

`plan_correctives2.md` в основном усиливает правильные места. Я принимаю почти все уточнения Claude:

- двухслойная память `operational + narrative`;
- trust profiles как отдельный слой между defaults и city overrides;
- weekly/monthly priority через `daily_safety_check`, без полного daily-decide;
- onboarding не только нового, но и существующего аккаунта через `adoptable_campaigns`;
- каталог целей Метрики в `city.yaml`;
- разделение form leads и qualified leads;
- разные defaults для search и RSYA;
- `metrics_snapshots.jsonl` для pacing, decision precision и evidence;
- runtime в `autopilot/runtime/` с `.gitignore`;
- ownership через labels;
- holdout как опциональная фича.

По трём спорным пунктам мой итоговый вердикт:

1. `expected_effect`, `guard_metric`, `rollback_trigger` не надо требовать для каждого действия на первом полном релизе. Минимум для всех действий: `signal_id`, `reason_code`, `confidence`, `evidence`, `risk_class`. Для applied medium/high действий добавлять `expected_effect` и `guard_metric`. `rollback_trigger` обязателен только для high/critical или для действий, где есть понятный автоматический критерий вреда.
2. `before_snapshot` не нужен как отдельный файл для каждого low-risk действия. Но ledger row должен содержать достаточный `before_value/after_value`, чтобы понять и вручную откатить действие. Snapshot обязателен для `risk_class >= medium`, bulk/replace операций, действий без простого обратного значения и любых изменений shared-настроек.
3. `risk_class` нужен сразу, но авто-rollback policies откладываем. На первом релизе rollback только manual/dry-run/confirm. Risk class используется для evidence thresholds, snapshot policy, approval expiry, notification level и permissions resolution.

---

## 1. Финальная архитектурная модель

### 1.1. Роль автопилота

`leadgen-autopilot` — это не копия ручного `leadgen` и не полностью независимый второй скилл. Его корректнее описать так:

- `leadgen-autopilot` = policy/orchestration layer: анализ, память, permissions, caps, planning, approval, apply, rollback, отчёты.
- `leadgen` branches/references/library = shared operation playbooks and rules: конкретные шаги создания/оптимизации кампаний.

Зависимость от `leadgen/branches` допустима, но должна быть оформлена как контракт:

- создать `.claude/skills/leadgen-autopilot/references/playbook_contract.md`;
- перечислить стабильные playbooks и anchors;
- описать ожидаемые входы/выходы;
- указать, что изменение shared playbooks требует проверки автопилота;
- любые правки `.claude/skills/leadgen/{branches,references,library}` продолжают синхронизироваться в `.codex/skills/leadgen-codex/`.

Автопилот на пилоте остаётся `Claude-only`, но это исключение надо явно записать в `.claude/skills/leadgen-autopilot/CLAUDE.md` и `RECENT-CHANGES.md`.

### 1.2. Рабочая директория routine

Для Claude Code routine рабочая папка:

```text
C:\git\leadgen-mcp
```

Не `autopilot/`, потому что автопилоту нужны:

- `.claude/skills/leadgen/` shared playbooks;
- MCP/server контекст;
- project-level docs and config;
- единые относительные пути.

Prompt:

```text
/leadgen-autopilot city=omsk
```

или:

```text
Запусти leadgen-autopilot для города omsk. Используй autopilot/config/cities/omsk.yaml.
```

### 1.3. Runtime и git

Финальная позиция:

- `autopilot/runtime/` — runtime state, gitignored;
- `autopilot/reports/` — отчёты, gitignored;
- `autopilot/config/secrets.env` — gitignored;
- `autopilot/learnings/` — не gitignore, потому что proposed/validated learnings нужны для review/PR;
- code/config/schemas/templates/scripts — в git.

`.gitignore`:

```gitignore
autopilot/runtime/
autopilot/reports/
autopilot/config/secrets.env
```

Корневой `RECENT-CHANGES.md` — для dev/architecture/code changes. Автономные прогоны бота туда не пишутся. Их место — `autopilot/runtime/<city>/runs/` и отчёты.

---

## 2. Память: operational + narrative

Принимаем двухслойную модель из `plan_correctives2.md`.

### 2.1. Operational layer — источник правды для apply

Файлы:

- `autopilot/runtime/<city>/state.yaml`;
- `autopilot/runtime/<city>/action_ledger.jsonl`;
- `autopilot/runtime/<city>/metrics_snapshots.jsonl`;
- `autopilot/runtime/<city>/pending_approvals.yaml`;
- `autopilot/runtime/<city>/before_snapshots/<action_id>.json`;
- `autopilot/runtime/<city>/locks/RUNNING.lock`.

Operational layer читает apply engine, idempotency, rollback, recovery и decision engine.

`state.yaml` — кеш текущего состояния, но факт сверяется с API. API остаётся source of truth для реального состояния аккаунта.

`action_ledger.jsonl` — append-only. Никакого редактирования прошлых строк. Для integrity можно добавлять `prev_hash`/`row_hash` позже, но это не блокер W1.

`metrics_snapshots.jsonl` — ежедневные срезы по city/topic/channel/campaign для:

- pacing;
- decision precision;
- weekly/monthly;
- evidence для learnings;
- comparison managed vs holdout.

### 2.2. Narrative layer — память для агента и lazy-load

Файлы:

- `STATE.md`;
- `CURSOR.md`;
- `SUMMARY.md`;
- `pending_approvals.md`;
- `runs/<YYYY-MM>/<run_id>.md`;
- `campaigns/<campaign_id>.md`;
- `decisions/<topic>-<slug>.md`.

Narrative layer нужен, потому что агенту удобнее читать контекст в markdown, а пользователю — ревьюить ход рассуждений. Но markdown не является источником правды для apply.

Если `STATE.md` расходится с `state.yaml`, markdown регенерируется. Если `state.yaml` расходится с API, запускается drift flow.

### 2.3. Schemas и atomic writes

Schemas:

```text
autopilot/schemas/city_config.schema.json
autopilot/schemas/state.schema.json
autopilot/schemas/action.schema.json
autopilot/schemas/approval.schema.json
autopilot/schemas/ledger_entry.schema.json
autopilot/schemas/metrics_snapshot.schema.json
autopilot/schemas/launch_proposal.schema.json
```

Validation policy:

- `city_config`, `state`, `pending_approvals`, `launch_proposal` — validate on every write;
- `action_ledger.jsonl` — validate each row before append, не перечитывая весь файл каждый раз;
- full ledger integrity check — periodic или перед финальным тестом;
- writes через tmp → validate → rename → `.bak`.

---

## 3. City config и уровни доверия

### 3.1. Trust resolution

Финальная иерархия:

```text
1. autopilot/config/caps_defaults.yaml
2. autopilot/config/trust_profiles/<profile>.yaml
3. autopilot/config/cities/<city>.yaml permissions
4. autopilot/config/cities/<city>.yaml topics.<topic>.permissions
5. runtime approval override with expires_at
```

`trust_profile`:

```yaml
trust_profile: conservative | balanced | aggressive | custom
```

- `conservative` — для первого боевого города и критичных аккаунтов.
- `balanced` — для стабильных городов после первичной проверки.
- `aggressive` — только после накопленной истории и отдельного approval.
- `custom` — если город полностью задаёт permissions вручную.

В каждом run log писать resolved permissions с источником значения:

```yaml
permissions_resolution:
  bid.increase.within_cap:
    value: auto
    source: trust_profiles/balanced.yaml
  campaign.create_draft.in_new_topic:
    value: review_queue
    source: cities/omsk.yaml
```

### 3.2. Рекомендация по первому городу

Для первого реального города я бы ставил `trust_profile: conservative`.

Причина простая: мы тестируем не только качество решений, но и саму инфраструктуру автономного исполнения. `balanced` можно использовать в mock/dev-сценариях или перевести город в `balanced` после успешного финального теста и нескольких штатных прогонов без incident.

### 3.3. Status тематик

Принимаем расширенный статус:

```yaml
topics:
  vtorichka:
    status: active # candidate | experimental | active | paused | blocked
    allowed_channels: [search, rsya]
```

Семантика:

- `candidate` — можно анализировать и предлагать, нельзя создавать кампании.
- `experimental` — можно создать draft с жёсткими caps; activation только `review_queue`; бюджет/срок ограничены.
- `active` — можно управлять в рамках permissions.
- `paused` — не создавать новое, существующее можно поддерживать/останавливать по правилам.
- `blocked` — не трогать и не предлагать без изменения config.

`experimental` стоит оставить: он нужен для реального бизнеса, где направление хочется попробовать, но не переводить сразу в полноценный `active`.

---

## 4. Onboarding нового и существующего города

### 4.1. Бот анализирует сам, но не запускает самовольно

Граница автономии:

- бот сам делает baseline scan;
- сам находит возможности, проблемы и дыры;
- сам формирует launch/adoption proposal;
- но запускает/принимает под управление только то, что разрешено config и approval.

### 4.2. Baseline scan

Для каждого города первый сценарий:

1. Проверить `city.yaml`.
2. Проверить Метрику, цели, UTM, связку Direct↔Metrika.
3. Инвентаризировать кампании, группы, объявления, ключи, аудитории.
4. Разделить кампании на:
   - `owned`: есть `autopilot:managed`;
   - `adoptable`: нет label, но тема/город совпадают с разрешённым config;
   - `foreign`: не в разрешённом scope, read-only;
   - `holdout`: label `autopilot:holdout`, используется для сравнения, не управляется.
5. Собрать метрики last 30 full days.
6. Сформировать `launch_proposal.md` и `launch_proposal.yaml`.

### 4.3. Adoption existing campaigns

Для реальных 60 городов adoption важнее, чем чистый запуск с нуля.

`launch_proposal.yaml` должен иметь блок:

```yaml
adoptable_campaigns:
  - campaign_id: 8765432
    name: "Омск Вторичка Поиск"
    inferred_topic: vtorichka
    inferred_channel: search
    last_30d:
      spend: 120000
      leads_form: 34
      cpa_form: 3529
    recommendation: adopt
    reason: "Совпадает с active topic vtorichka/search, стабильная история, UTM корректны"
```

Adoption — только через approval. После approval бот ставит labels:

```text
autopilot:managed
city:omsk
topic:vtorichka
channel:search
```

Release: проверить поддержку удаления labels в MCP/Yandex Direct. Если прямого remove нет, fallback:

- добавить `autopilot:released`;
- ownership check требует `autopilot:managed` и отсутствие `autopilot:released`;
- позже добавить MCP-инструмент remove/update labels.

### 4.4. DRAFT-only

Все новые кампании:

- `campaign.create_draft.in_existing_topic`;
- `campaign.create_draft.in_new_topic`;

создаются DRAFT/SUSPENDED.

Активация:

- `campaign.activate_existing_draft`;
- baseline `review_queue`;
- в первом реальном городе не `auto`.

Это не рекомендация, а обязательное правило проекта.

---

## 5. Метрики, атрибуция и evidence

### 5.1. Цели Метрики в city.yaml

Принимаем:

```yaml
metrika:
  counter_id: 12345678
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
```

Агент в отчётах использует alias/name, не голые ID.

### 5.2. Двойная атрибуция

Daily decisions:

- `target_cpa_form`;
- `target_cpa_call`;
- full days only: yesterday + rolling 3/7 days;
- today только для emergency.

Weekly/monthly:

- `target_cpa_qualified`;
- CRM/qualified leads с задержкой;
- last 7/30 full days.

Qualified lead не должен агрессивно двигать daily ставки из-за задержки атрибуции.

### 5.3. Search vs RSYA defaults

Принимаем разные defaults:

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

Плюс per-action overrides в `action_catalog.md`.

### 5.4. Action evidence

Для всех proposed actions обязательно:

```yaml
signal_id: cpa_jump
reason_code: cpa_above_target_with_enough_clicks
confidence: medium
risk_class: medium
evidence:
  window: last_7_full_days
  clicks: 84
  conversions: 6
  spend: 4200
  cpa_form: 700
  target_cpa_form: 800
```

Для applied medium/high дополнительно:

```yaml
expected_effect: "CPA form -5..10% over 7 full days"
guard_metric: "spend without conversions for 3 full days"
```

`rollback_trigger`:

- required for high/critical;
- optional for low/medium unless action_catalog defines a reliable trigger.

---

## 6. Idempotency, locks, drift and rollback

### 6.1. Idempotency key

Использовать deterministic key:

```text
<city>|<account>|<entity_type>|<entity_id>|<action_type>|<desired_value>|<decision_window>
```

Проверки перед apply:

- local `action_ledger.jsonl`;
- API/change history where available;
- current API state vs expected before state.

### 6.2. Per-city lock

`RUNNING.lock`:

```yaml
city: omsk
run_id: omsk-2026-04-30-1030
started_at: 2026-04-30T10:30:00+05:00
phase: applying
ttl_minutes: 120
```

Если lock свежий — новый run выходит с уведомлением. Если lock протух:

- первый раз recovery + alert;
- если lock протух дважды подряд — alert уровня incident.

### 6.3. Drift/human override

STATE не должен перетирать ручные изменения.

Если API отличается от ожидаемого:

- проверить `get_change_history`;
- если это human change — записать `human_override`;
- freeze entity на 24-72 часа;
- не возвращать значение обратно автоматически;
- показать в отчёте.

### 6.4. Snapshot policy

Финальная политика:

- `risk_class >= medium` → structured `before_snapshot`;
- low-risk single-field additive action → ledger row with `before_value`/`after_value` enough;
- bulk/replace/shared-setting action → snapshot независимо от risk class;
- irreversible action → `rollback: manual_only`.

Rollback:

- только dry-run → show plan → `confirm rollback <id>`;
- no automatic rollback на первом релизе;
- failed rollback логируется и уходит в Telegram alert.

---

## 7. Telegram and approvals

Финальная модель:

- один Telegram bot token;
- один чат на город;
- `notify.telegram_chat_id` в `city.yaml`;
- allowlist chat_id;
- persisted `update_offset`;
- escaping HTML/Markdown;
- retry/backoff;
- length limit, fallback to file;
- token masking in logs.

Commands:

```text
approve <id>
reject <id>
defer <id> <Nd>
rollback <run_id>
confirm rollback <id>
```

Approval expiry:

- 24h для bid/budget;
- 72h для draft creation/adoption;
- after expiry action recalculates, not blindly applies.

Approval polling:

- в начале run;
- после отчёта сохранять `message_id` у новых pending actions.

---

## 8. Cycle priority

Принимаем схему Claude:

| День | Выполняется |
|---|---|
| Обычный день | full `daily` |
| Weekly day | `daily_safety_check` + `weekly` |
| Monthly day | `daily_safety_check` + `monthly` |
| Weekly + monthly совпали | `daily_safety_check` + `monthly`, weekly merged into monthly |

`daily_safety_check`:

- HALT;
- lock/recovery;
- critical drift;
- hard pacing cap;
- expired approvals;
- notify failure alerts.

Никаких tactical daily-decide в weekly/monthly день.

---

## 9. Learnings

Принимаем осторожную формулировку: автопилот накапливает гипотезы и evidence, но не меняет сам правила скилла.

Lifecycle:

- `proposed` — обнаружен паттерн;
- evidence includes positive and negative cases;
- monthly digest предлагает специалисту;
- specialist принимает через PR/правку `lessons_registry.md`;
- `rejected` больше не предлагается;
- expiry/recheck каждые 60 дней.

Validated learning не становится auto-behavior без явного approval и city permissions.

---

## 10. Holdout

Holdout полезен, но не должен блокировать W1-W6.

Решение:

- label `autopilot:holdout` поддержать в schema/ownership уже сейчас;
- в apply engine holdout всегда read-only;
- сравнение managed vs holdout добавить в W7 monthly/weekly reports;
- если holdout не задан, отчёт просто пропускает этот блок.

---

## 11. Волны разработки как рабочие пакеты

Разрабатываем полный контур, не запускаем пользовательский пилот на W3. Self-checks внутри волн остаются обязательными, чтобы финальный e2e тест не стал первой проверкой базовой инфраструктуры.

### W1 — Каркас, schemas, gitignore, lock, Telegram hello

- `autopilot/{config,runtime,reports,learnings,lib,schemas}/`;
- `.gitignore`;
- schemas;
- `.claude/skills/leadgen-autopilot/skill.md`;
- `autopilot/CLAUDE.md`;
- `caps_defaults.yaml`;
- `trust_profiles/{conservative,balanced,aggressive}.yaml`;
- `cities/_example.yaml`;
- `secrets.env.example`;
- `telegram_send`, `telegram_send_doc`, `atomic_write`, `lock`;
- global/per-city HALT;
- per-city `RUNNING.lock`;
- cwd root `C:\git\leadgen-mcp`.

Self-check: hello run берет lock, проверяет HALT, шлёт Telegram, отпускает lock.

### W2 — Память operational + narrative + recovery

- templates markdown;
- `state.yaml`;
- `action_ledger.jsonl`;
- `metrics_snapshots.jsonl`;
- `pending_approvals.yaml`;
- `before_snapshots/`;
- narrative generation;
- memory lookup;
- SUMMARY compression;
- crash recovery.

Self-check: create/read/recover corrupt state.

### W3 — Analytics, signals, dry-run plan, onboarding scan

- Metrika/Direct reports with LYDC;
- goals from `city.yaml`;
- freshness checks;
- `signal_catalog.md`;
- `action_catalog.md`;
- action plan with reason/evidence/confidence/risk;
- onboarding scan;
- `launch_proposal.{md,yaml}`;
- HTML + Telegram summary.

Self-check: dev city generates state and proposal without apply.

### W4 — Approval queue and Telegram replies

- `pending_approvals.yaml`;
- `telegram_check_replies` with `update_offset`;
- commands approve/reject/defer/rollback/confirm;
- expiry;
- message_id binding.

Self-check: mock approval cycle end-to-end.

### W5 — Apply engine, ownership, low-risk auto

- idempotency key;
- ownership label check;
- drift check before apply;
- snapshot policy;
- local ledger + `log_change_event`;
- low-risk actions only:
  - clear negatives;
  - blacklist placements;
  - small bid/budget inside caps with high confidence.

Self-check: safe apply once, repeat run no duplicate.

### W6 — Reconciliation, adoption, DRAFT-only

- config↔state↔API diff;
- topic status transitions;
- launch proposal/draft creation;
- adoption approval + labels;
- DRAFT-only creation;
- activation separate;
- human override cooldown;
- pacing control.

Self-check: active topic creates draft, not live campaign.

### W7 — Weekly/monthly rollups, compression, quality metrics

- weekly report;
- monthly report;
- budget plan;
- SUMMARY compression;
- decision precision;
- rollback rate;
- approval queue health;
- coverage;
- Telegram noise;
- optional holdout comparison.

Self-check: mock weekly/monthly dates.

### W8 — Medium/high-risk actions

- ad variants;
- keyword/group expansion;
- pause/resume;
- creative generation;
- audiences/retargeting;
- snapshots for risk>=medium;
- strict min_evidence.

Self-check: high CPA scenario produces full evidence trail and requires proper permission.

### W9 — Learnings

- proposed;
- negative evidence;
- monthly digest;
- rejected;
- expiry/recheck;
- no automatic behavior change.

Self-check: mock repeated pattern creates proposed; rejection suppresses it.

### W10 — Hardening, chaos, rollback, multi-city checklist

- MCP 5xx;
- broken config;
- state/API drift;
- stale locks;
- rollback confirm;
- emergency budget cap;
- multi-city scheduling;
- docs for specialist;
- scale checklist.

Self-check: synthetic chaos test passes without critical incident.

---

## 12. Финальный end-to-end тест

Тестируем итоговый вариант, не промежуточный W3.

Сценарий:

1. `city.yaml` создан для пилотного города.
2. Baseline scan existing account.
3. `launch_proposal.yaml` with adoptable campaigns.
4. Specialist approves adoption.
5. Bot labels adopted campaigns.
6. Daily run builds action plan and metrics snapshots.
7. Approval queue works via Telegram.
8. Low-risk action applies once.
9. Repeat run proves idempotency.
10. Simulated crash during apply recovers.
11. Manual change in Direct triggers drift + human_override freeze.
12. New active topic creates DRAFT/SUSPENDED campaign only.
13. Weekly rollup on test date.
14. Monthly rollup on test date.
15. Rollback dry-run + confirm works or marks manual_only.
16. HALT global and per-city stop the run.
17. Report includes managed/unmanaged spend and pacing.

Acceptance:

- no duplicate actions;
- no live campaign activation without explicit approval;
- no changes to foreign/holdout campaigns;
- every action has evidence and resolved permission source;
- ledger and API do not contradict each other, or contradiction is logged as drift/human_override;
- Telegram approvals cannot be applied twice;
- runtime files survive restart/crash.

---

## 13. Мои ответы на новые open questions Claude

1. **Один Telegram-бот или per-city?**
   Один бот, разные чаты per city. Так проще secrets, логирование и эксплуатация. `chat_id` всё равно city-specific.

2. **Holdout-кампании?**
   Да, поддержать label/schema сейчас, отчётность добавить в W7. Не блокирует W1-W6.

3. **Adoption на пилоте?**
   Да, иначе пилот будет мало связан с реальным бизнесом. Но adoptить не весь аккаунт, а ограниченный scope: один город, 1-2 темы, кампании после launch proposal и approval.

4. **`remove_labels` в MCP?**
   Не блокер W1. Проверить в W1/W6. Если remove невозможен сразу, использовать fallback `autopilot:released` и ownership predicate `managed && !released`.

5. **Schema validation overhead?**
   Валидировать каждую ledger row перед append, но не весь jsonl файл на каждый write. Полная проверка — periodic/e2e. State/approval/config валидировать всегда.

6. **`experimental` status?**
   Оставить. Он покрывает важный бизнес-сценарий пробного направления.

7. **Trust profile для пилотного Омска?**
   Для первого реального города — `conservative`. `balanced` можно включить после успешного e2e и нескольких штатных прогонов без incident.

8. **Routine per city или один на все города?**
   На старте — one routine per city, staggered по времени. Это проще дебажить и лучше сочетается с per-city lock. Orchestrator "all cities sequentially" можно добавить после 3+ стабильных городов.

9. **Куда писать RECENT-CHANGES от автопилота?**
   Dev/architecture changes — корневой `RECENT-CHANGES.md`. Runtime-прогоны — только `autopilot/runtime/<city>/runs/`. Не смешивать.

10. **Stale lock alerts?**
   Да. Первый stale lock → recovery alert. Повторный stale lock подряд → incident alert.

---

## 14. Что нужно решить пользователю до W1

Минимальный список решений:

- Подтвердить runtime location: `autopilot/runtime/` gitignored.
- Подтвердить первый город и `trust_profile` (`conservative` рекомендую).
- Подтвердить one Telegram bot + per-city chats.
- Подтвердить adoption на пилоте: какие темы/кампании рассматриваем.
- Подтвердить topic statuses включая `experimental`.
- Подтвердить routine model: one routine per city на старте.
- Подтвердить DRAFT-only + activation через `review_queue`.

После этого можно собирать финальный `PLAN.md` и начинать W1-W10 как один полный контур.
