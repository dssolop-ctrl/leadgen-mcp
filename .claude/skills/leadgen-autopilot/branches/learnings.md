# Branch: learnings — proposed → digest

> Грузится в weekly/monthly. Накапливает гипотезы, **не** меняет поведение автоматически.

## Lifecycle

```
proposed (≥1 наблюдение)
  ↓ (≥3 повтора + 14 дней без отката + scope-стабильно)
validated
  ↓ (предложение в monthly digest)
specialist review → PR в leadgen/references/lessons_registry.md
  ↓ (поведение меняется через код-правку скилла)

   ↘ rejected (специалист отказал)
   ↘ expired (>60 дней без подтверждения, re-check)
```

## 1. Naissance: detection of patterns

В weekly/monthly анализе:

```python
# Пример паттерна: "+15% bid → -20% CPA через 7d на vtorichka_search в Омске"
candidates = []
for action in ledger.filter(action_type="bid.increase.within_cap", ts > 30d_ago):
    delta_bid = action.after_value - action.before_value
    snapshot_after_7d = metrics_snapshots.find(
        date=action.ts + 7d,
        scope={"campaign_id": action.entity_id}
    )
    snapshot_before = metrics_snapshots.find(
        date=action.ts - 1d,
        scope={"campaign_id": action.entity_id}
    )
    if snapshot_after_7d.cpa_form < snapshot_before.cpa_form * 0.85:
        candidates.append({
            "pattern": f"+{delta_bid}% bid on {topic}_{channel} → -{delta_pct}% CPA over 7d",
            "evidence": {"run_id": action.run_id, "before": snapshot_before, "after": snapshot_after_7d},
        })
```

Сгруппировать candidates по `(topic, channel, action_type, evidence_signature)`. Если у группы ≥1 наблюдение → создать или обновить `learnings/proposed/<id>.md`:

```markdown
---
id: omsk-vtorichka-bid-uplift-2026-04
created: 2026-04-29
last_observed: 2026-04-29
confidence: low
observed_count: 1
scope:
  city: omsk
  topic: vtorichka
  channel: search
needs_repeats: 2
status: proposed
---

# Pattern: +15% bid on vtorichka_search → -20% CPA over 7d

## Pattern
+15% bid increase on vtorichka_search → CPA decrease ~20% within 7 days.

## Positive evidence
- run omsk-2026-04-22-1030, action bid.increase.within_cap on campaign 8765432:
  - before: cpa_form=850, clicks_7d=80
  - after_7d: cpa_form=680, clicks_7d=110
  - delta: -20% CPA, +37% clicks

## Negative evidence
(пока нет — записывается, если паттерн НЕ срабатывал)

## Action_when_validated
"В analyze: при cpa_form > target × 1.15 на vtorichka_search → приоритезировать `bid.increase.within_cap` 15% (если within cap)."
```

## 2. Validation: 3 повтора + 14 дней + no rollback

После каждого weekly/monthly прохода:

```python
for proposed in load_all("learnings/proposed/*.md"):
    new_evidence = scan_recent_runs(proposed.scope, proposed.pattern_signature)
    proposed.observed_count += len(new_evidence.positive)
    proposed.negative_evidence.extend(new_evidence.negative)
    if (proposed.observed_count >= 3
        and (now - proposed.created) >= 14_days
        and len(proposed.negative_evidence) == 0):
        promote_to_validated(proposed)
```

Validated learning → `learnings/validated/<id>.md`. **Никаких изменений в behavior** — только запись.

## 3. Monthly digest

В `branches/analyze.md` секция M4:

```markdown
# Monthly digest <YYYY-MM> — learnings ready for review

## Validated this month
- omsk-vtorichka-bid-uplift-2026-04 (status: validated)
  - Pattern: +15% bid → -20% CPA on vtorichka_search
  - Confidence: medium (4 повтора за 30 дней)
  - Suggested action: добавить в lessons_registry.md
  - Next: создать PR в .claude/skills/leadgen/references/lessons_registry.md

## Aged out (60+ days, re-check needed)
- ...

## Rejected this month
- ...
```

Это содержание копируется в `reports/<city>/<YYYY-MM>/month-<MM>.html` секция «Learnings digest» и отправляется в Telegram.

## 4. Rejection / expiry

Через Telegram `reject learning <id>` (или ручной merge `learnings/proposed/<id>.md` → `learnings/rejected/<id>.md`):
- Файл переезжает в `rejected/`.
- В future weekly не предлагается.

Expiry:
- `proposed`: если 60 дней без новых evidence → re-check trigger (вернуть в anaylze для проверки). Если повторно не подтвердилось → `rejected/<id>-expired.md`.
- `validated`: если нет PR в lessons_registry за 60 дней — re-check (паттерн может устареть).

## 5. Scope ограничения

- Learning привязан к `scope: {city, topic, channel}`.
- Не переносится между городами автоматически. Если паттерн повторился в другом city — отдельный proposed.
- Если scope меняется (например, target_cpa изменился) — learning expired.

## 6. Граница

- Автопилот **никогда** не пишет в `.claude/skills/leadgen/`.
- Validated learnings — только в `autopilot/learnings/validated/` (артефакт ревью).
- Перенос в `lessons_registry.md` — ручная работа специалиста через PR.

## Self-check

Mock сценарий:
1. 1 ledger row с `action: bid.increase.within_cap`, snapshot до и после 7d с -20% CPA.
2. Прогон learnings → создаётся `proposed/<id>.md`.
3. Симуляция ещё 2 повторов в течение 14 дней.
4. Re-run → promote в `validated/<id>.md`.
5. Monthly digest содержит этот learning.
6. Reject через Telegram `reject learning <id>` → переносится в `rejected/<id>.md`.
