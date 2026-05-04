# Branch: memory_write — operational + narrative

> Грузится в фазе D8. Источник правды для apply — operational. Narrative md — производное, для агента.

## 1. Operational layer (machine source of truth)

### 1.1. `state.yaml`

Регенерируется в конце прогона из:
- API: `get_campaigns` (with labels filter), `get_campaign_stats`, account spend.
- Ledger: `action_ledger.jsonl` (последние applied для last_action timestamps).
- Config: `cities/<city>.yaml` для нормировки.

**Записывается через `bash autopilot/lib/atomic_write.sh runtime/<city>/state.yaml -`.**

Schema: `autopilot/schemas/state.schema.json` (`schema_version: 1`).

### 1.2. `action_ledger.jsonl`

Append-only. Каждая попытка действия (включая planned/skipped) — отдельная строка JSON по schema `ledger_entry.schema.json`.

Запись: `printf '%s\n' "$json" >> runtime/<city>/action_ledger.jsonl` (атомарно по строкам).

Никогда не редактировать прошлые строки. Если действие отменено/откатано — добавить новую строку с `status: rolled_back` и ссылкой на прошлую через `idempotency_key`.

### 1.3. `metrics_snapshots.jsonl`

Append-only. Каждый прогон дописывает per-scope срезы (account/topic/channel/campaign).
Schema: `metrics_snapshot.schema.json`.

### 1.4. `pending_approvals.yaml`

Только при `autonomy_mode == with_approvals`. Список approval entries по schema `approval.schema.json`.

Запись через `atomic_write.sh`.

### 1.5. `before_snapshots/<action_id>.json`

Создаётся **до** apply для `risk_class >= medium` или bulk/replace/shared-setting.
Содержит структурный snapshot всего entity.

## 2. Narrative layer (для агента)

### 2.1. Регенерация после operational

Narrative md **никогда не редактируется в обход operational**. Каждое значение в narrative должно быть восстанавливаемо из `state.yaml` + `action_ledger.jsonl`.

Запись через `atomic_write.sh`.

### 2.2. Шаблоны

См. `autopilot/runtime/_templates/`.

#### `STATE.md`

```markdown
# STATE — <city> (last update <ISO>)

## Метаданные
- run_id: <run_id>
- autopilot_version: <ver>
- autonomy_mode: <mode>
- trust_profile: <profile>

## Бюджет
- monthly_total_limit: <X> ₽
- spent_mtd: <Y> ₽ (<%>)
  - managed: <X> ₽
  - unmanaged: <Y> ₽
- pacing_ratio: <r>
- forecast_month_end: <F> ₽
- pacing_state: <normal|conservation|emergency|hard_cap>

## Активные тематики
| topic | status | budget_used | budget_limit | active_campaigns | trend_cpa_7d |

## Активные кампании (managed)
| campaign_id | label | status | last_action | next_action | flags |

## Указатели в lazy memory
- кампании, изменённые сегодня: campaigns/<id>.md, ...
- последний run: runs/<YYYY-MM>/<HHMM>.md
- открытые decisions: decisions/<...>.md

## Флаги внимания
- ...
```

#### `CURSOR.md`

```markdown
# CURSOR — <city>

## Сделано в последнем прогоне (<ISO>)
- <action 1>
- <action 2>

## Отложено (cooldown / next run)
- <pending tactical move> (cooldown until <date>)

## В очереди review (только при with_approvals)
1. <approval_id>: <action description>

## План на неделю (от weekly rollup <date>)
- <plan item>
```

#### `SUMMARY.md`

```markdown
# SUMMARY — <city>

## Последние 30 дней (по дням)
- <YYYY-MM-DD>: <summary line>

## Недели 30-90 дней (компрессия 1 строка)
- W<NN>: <summary>

## Месяцы 3-12 мес (1-2 строки)
- <YYYY-MM>: <summary>
```

#### `runs/<YYYY-MM>/<HHMM>.md`

```markdown
---
run_id: <id>
started: <ISO>
ended: <ISO>
phase: succeeded|failed|halted
mode: daily|weekly|monthly
autonomy: full_auto|with_approvals|read_only
---

# Run <run_id>

## Контекст
- city: <city>
- trust_profile: <profile>
- pacing_state: <state>

## Метрики
- ...

## Сигналы
- <signal_id>: <evidence>

## План действий
- <action 1>: status=<applied|skipped_*>, reason=<reason_code>, confidence=<level>

## Применённые действия
- <action_id>: <action_type>, before=<v>, after=<v>

## Решения
- <reason for non-trivial choice>

## Изменения в памяти
- updated state.yaml: ...
- ledger rows: <count>

## Errors / warnings
- ...
```

#### `campaigns/<campaign_id>.md`

```markdown
---
tags: [campaign:<id>] [topic:<t>] [channel:<c>] [city:<city>]
last_action: <ISO>
status: <status>
---

# <campaign_name>

## История изменений (обратный хронологический)

### <run_id> @ <ISO>
- action: <type>
- before → after
- reason: <reason_code>
- evidence: <summary>
- ledger_ref: <action_id>
```

#### `decisions/<topic>-<slug>.md`

Нестандартные кейсы / гипотезы / эксперименты.

```markdown
---
tags: [topic:<t>] [city:<city>] [type:experiment|incident|hypothesis]
created: <ISO>
status: open|closed
---

# <title>

## Контекст
...

## Решение
...

## Результат (заполняется позже)
...
```

## 3. SUMMARY компрессия (раз в неделю в weekly rollup)

```python
threshold_compress = caps.summary_compress_after_days   # 30
threshold_collapse = caps.summary_collapse_after_days   # 90
threshold_archive = caps.summary_archive_after_days     # 365

for entry in summary.entries:
    age_days = (today - entry.date).days
    if age_days > threshold_archive:
        archive_to(f"decisions/historical-{entry.year}.md", entry)
        delete_from_summary(entry)
    elif age_days > threshold_collapse:
        merge_into_monthly_line(entry)
    elif age_days > threshold_compress:
        merge_into_weekly_line(entry)
```

## 4. Crash recovery (при stale lock / interrupted run)

При `lock.sh acquire` exit code 2 (stale recovered):

1. Прочитать прошлый `state.yaml.last_run`:
   - phase ∈ `started|loaded_context|fetched_metrics|approval_checked|planned_actions` → ничего не применилось, продолжаем штатно.
   - phase ∈ `applying|applied_partial` → вызвать **drift check** для всех entities из ledger, где status = `applying` (не `applied`).
   - phase = `memory_written` или `notified` → штатно, можно начинать новый прогон.

2. **Drift check для applying-actions:**
   - Для каждой строки ledger где status=`applying` без последующего `applied|failed|rolled_back`:
     - Прочитать API (get_<entity>).
     - Сравнить с `before_value` и `after_value`:
       - API == `after_value` → apply прошёл, ledger не обновился. Дописать `status: applied, recovered_from_ledger`.
       - API == `before_value` → apply не прошёл. Дописать `status: failed, error: lost_during_crash`.
       - Иначе → drift. `status: failed, error: drift_detected, actual: <value>`.

3. Записать recovery-event в `runs/<new_run_id>.md`.

4. Telegram alert: `Autopilot recovered from stale lock — N actions reconciled`.

## 5. Atomic writes — обязательно

Любое обновление eager-файла (`state.yaml`, narrative md, `pending_approvals.yaml`) — через `bash autopilot/lib/atomic_write.sh <target> -`. Никаких прямых `>` redirect'ов.

Append-only ledger/snapshots — допустим прямой append, но при ошибке записи строку повторить (ledger знает идемпотентность через `idempotency_key`).

## 6. Помощник lookup

`bash autopilot/lib/memory_lookup.sh <city> <tag_query>` — grep по тегам в narrative md.
Примеры:
- `memory_lookup.sh omsk "campaign:8765432"` → найти все упоминания кампании.
- `memory_lookup.sh omsk "topic:vtorichka"` → все файлы по теме.
- `memory_lookup.sh omsk "type:incident"` → все decisions с инцидентами.
