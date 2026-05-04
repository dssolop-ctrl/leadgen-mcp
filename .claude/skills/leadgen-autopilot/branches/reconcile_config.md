# Branch: reconcile_config — diff config↔state↔API

> Грузится в фазе D5. Производит синхронизацию факта (API) с целевым состоянием (config) с учётом промежуточного кеша (state.yaml).

## Этап 1. config↔state дельты

Сравнение `cities/<city>.yaml` vs `state.yaml`:

| Что изменилось в config | Action |
|---|---|
| Появилась `topic.<t>.status: active`, нет в state | enqueue `campaign.create_draft.in_new_topic` для каждого `allowed_channels` |
| Появилась `topic.<t>.status: experimental` | то же, но с уменьшенным `monthly_budget` (×0.5) и cooldown_until=14d |
| `topic.<t>.status: active → paused` | enqueue `campaign.pause.low_performance` для всех campaigns этой темы |
| `topic.<t>.status: paused → blocked` | то же; campaigns не возобновятся, даже если их обновят permissions |
| `topic.<t>.monthly_budget` изменён | enqueue `budget.set_total_monthly` для campaigns темы |
| `topic.<t>.target_cpa_form|call|qualified` изменён | enqueue `bid.adjust_strategy_target_cpa.*` |
| `budget.total_monthly_limit` изменён | enqueue `budget.set_total_monthly` для всех managed campaigns |

## Этап 2. state↔API дельты (drift detection)

Для каждой managed campaign:

```python
expected = state.campaigns[id]            # последний known state
actual = api.get_campaign(id)             # current truth
if differ(expected, actual):
    history = api.get_change_history(id, since=last_run.ended_at)
    if history.is_human_change():
        # S-HUMAN-OVERRIDE
        state.campaigns[id].human_override_until = now + caps.cooldown_hours_after_human_change h
        emit_signal("S-HUMAN-OVERRIDE", entity_id=id, evidence={"changed_by": user, "field": <field>})
    else:
        # S-DRIFT-DETECTED — unexplained
        emit_signal("S-DRIFT-DETECTED", entity_id=id, evidence={"expected": expected, "actual": actual})
        send_alert("Unexplained drift on campaign <id>")
```

## Этап 3. Pacing recalc

```python
elapsed_share = days_passed_in_month / days_in_month
spent_mtd = api.get_account_stats(<>).spend_mtd
managed_spend_mtd = sum(api.get_campaign_stats(c).spend_mtd for c in managed)
unmanaged_spend_mtd = spent_mtd - managed_spend_mtd

expected_spend = budget.total_monthly_limit * elapsed_share
pacing_ratio = spent_mtd / max(expected_spend, 1)

avg_daily_recent = average(metrics_snapshots[-7:].spend)
forecast_eom = avg_daily_recent * days_in_month

if spent_mtd > budget.total_monthly_limit * 1.20:           # hard cap
    pacing_state = "hard_cap"
elif pacing_ratio > caps.pacing_emergency_ratio or forecast_eom > budget.total_monthly_limit * 1.20:
    pacing_state = "emergency"
elif pacing_ratio > caps.pacing_conservation_ratio or forecast_eom > budget.total_monthly_limit * 1.10:
    pacing_state = "conservation"
else:
    pacing_state = "normal"
```

`pacing_state` сохраняется в `state.budget.pacing_state`. Используется в decide для гейтинга.

## Этап 4. Topic status validations

- Если в `city.topics.<t>.status == active|experimental` но `t` не в `leadgen/references/site_structure.md` — записать warning, **не** создавать кампанию (топик не валиден).
- Если у тематики `monthly_budget == 0` при `status: active` — warning, treat as paused.

## Этап 5. Output

- Новые сигналы добавляются в общий список (для decide).
- Обновляется `state.yaml.budget.pacing_state` и `last_run.phase = "reconcile_done"`.
- Reconcile-события логируются в `runs/<run_id>.md` секция "Reconcile".

## Self-check

1. Mock `state.yaml` с `topic.zagorodka.status: paused`.
2. Изменить в `city.yaml` на `status: active`.
3. Run reconcile → должен enqueue `campaign.create_draft.in_new_topic` для search и rsya.
4. Mock API: campaign 8765432 paused (но в state ON) → должен emit `S-DRIFT-DETECTED` или `S-HUMAN-OVERRIDE`.
5. Mock spent_mtd = budget × 1.15, elapsed = 0.5 → `pacing_state = conservation`.
