# Branch: apply — выполнение через MCP + ledger + idempotency

> Грузится в фазе D7. На вход — ranked action plan из `decide.md`. На выход — applied + ledger rows + state.yaml updates.

## Per-action поток

Для каждого action со `status: planned` или `queued_approval` (с approved):

### 1. Pre-apply checks

```python
# Re-check idempotency (ledger мог обновиться в текущем прогоне)
if ledger.find(idempotency_key=action.idempotency_key, status="applied"):
    action.status = "skipped_idempotent"; return

# Re-check ownership
if state.campaigns[action.entity_id].ownership != "managed":
    action.status = "skipped_block"; action.skip_reason = "not_managed"; return

# Re-check human_override
if state.campaigns[action.entity_id].human_override_until > now:
    action.status = "skipped_human_override"; return

# Re-check API state vs expected before_state (drift guard)
actual = api.get_<entity>(action.entity_id)
if actual != action.before_value:
    history = api.get_change_history(action.entity_id, since=last_run)
    if history.has_human_change():
        mark_human_override(action.entity_id, 72h)
        action.status = "skipped_drift"; action.skip_reason = "human_override_freeze_set"; return
    else:
        action.status = "skipped_drift"; action.skip_reason = "unexplained"; return
```

### 2. Snapshot (для risk>=medium)

```python
if action.risk_class in ("medium", "high") or action.snapshot_required:
    snapshot = api.get_full_entity(action.entity_id)  # campaign with all fields
    snapshot_path = f"runtime/<city>/before_snapshots/{action.action_id}.json"
    write_atomic(snapshot_path, json.dumps(snapshot))
    action.before_snapshot_ref = snapshot_path
```

### 3. Set phase = applying

```bash
bash autopilot/lib/lock.sh update_phase <city> <run_id> applying
```

Append ledger row с `status: applying`:
```json
{"ts": "<now>", "run_id": "<>", "city": "<>", "action_id": "<>",
 "idempotency_key": "<>", "action_type": "<>", "risk_class": "<>",
 "entity_type": "<>", "entity_id": "<>", "before_value": <>, "after_value": <>,
 "before_snapshot_ref": "<>", "permission_resolved": "<>", "permission_source": "<>",
 "status": "applying"}
```

### 4. MCP call

По `action_catalog[action_type].mcp_tools`:

```python
match action.action_type:
    case "negatives.add_from_search_queries":
        result = mcp.update_campaign(campaign_id=action.entity_id, NegativeKeywords=new_list)
    case "bid.increase.within_cap":
        result = mcp.set_keyword_bids(...) или set_bids(...) в зависимости от entity_type
    case "campaign.create_draft.in_existing_topic":
        # Применить playbook leadgen/branches/create-{search,rsya}.md
        # 1. add_campaign(...) → SUSPENDED state
        # 2. add_adgroup
        # 3. add_keywords
        # 4. add_ad
        # 5. add_labels(business_labels)  # ТОЛЬКО бизнес-метки из leadgen/config/labels.md
        #    Например ["Лидген", "Вторичка", "Покупатель", "РСЯ"].
        #
        #    ⚠️ HARD RULE (см. lesson #32 + lesson о state-based ownership):
        #    Автопилот НЕ ставит и НЕ читает internal-метки в Директе:
        #    - autopilot:managed / autopilot:holdout / autopilot:released
        #    - city:<city>
        #    - topic:<topic>, channel:<channel>
        #    Они засоряют каталог Этажей и ломают агрегацию статистики.
        #    Ownership резолвится через state.yaml.campaigns[<id>] и
        #    city.yaml.holdout.campaign_ids (см. skill.md §3.4).
        #
        # 6. state.yaml.campaigns.append({
        #        campaign_id: <new_id>, ownership: "managed",
        #        topic: <topic>, channel: <channel>,
        #        client_login: <login>, created_by_autopilot: true,
        #        adopted_at: <ISO>
        #    })
        result = ...
    case "campaign.activate_existing_draft":
        result = mcp.resume_campaign(campaign_id=action.entity_id)
    case "placement.block.low_performing":
        result = mcp.apply_blocked_placements(...)
    # ... etc.
```

### 5. log_change_event (опциональный audit trail)

```python
mcp.log_change_event(
    type="autopilot.action_applied",
    payload={
        "run_id": run_id,
        "action_id": action.action_id,
        "action_type": action.action_type,
        "entity_id": action.entity_id,
        "idempotency_key": action.idempotency_key,
    }
)
```

### 6. Post-apply ledger update

Append ledger row с `status: applied`:
```json
{...same fields..., "status": "applied", "log_change_event_id": <id>}
```

### 7. Send Telegram (для auto_with_notify)

Если `action.permission_resolved == "auto_with_notify"`:
```bash
bash autopilot/lib/telegram_send.sh <chat_id> "
🟡 [<city>] <action_type>
Entity: <entity_type> <entity_id> '<name>'
Reason: <reason_code>
Evidence: <key=val ...>
Result: applied successfully
"
```

### 8. On error

```python
try:
    result = mcp_call(...)
except Exception as e:
    append_ledger({...same..., "status": "failed", "error": str(e)})
    action.status = "failed"
    if action.risk_class in ("high", "critical"):
        send_telegram_alert(f"Action failed: {action_type} on {entity_id}: {e}")
```

Failed actions retry **в следующем прогоне** (не в том же, чтобы избежать каскадных сбоев).

## Per-run summary

После всех actions:
- Update `state.yaml.last_run.phase = applied_partial | applied`.
- Update `state.campaigns[<id>].last_action`, `last_action_at`, `next_action`.
- Подсчёт metrics: `actions_applied`, `actions_skipped`, `actions_failed`.

## Rollback (вызов из Telegram)

См. `branches/safety.md` секция «rollback». Здесь — только применение dry-run plan / confirm:

```python
def rollback_run(run_id, dry_run=True):
    rows = ledger.filter(run_id=run_id, status="applied")
    plan = []
    for row in rows:
        if action_catalog[row.action_type].rollback_policy == "manual_only":
            plan.append({"action_id": row.action_id, "rollback": "manual_only", "note": "..."})
            continue
        # Восстановить before_value
        if row.before_snapshot_ref:
            target_state = json.loads(open(row.before_snapshot_ref).read())
        else:
            target_state = row.before_value
        plan.append({"action_id": row.action_id, "target_state": target_state, "mcp_tool": <inferred>})
    if dry_run:
        return plan  # → шлём в Telegram для confirm
    else:
        # Confirm — применить
        for p in plan:
            if p.get("rollback") == "manual_only": continue
            mcp_call_inverse(p)
            append_ledger({..., "status": "rolled_back", "idempotency_key": <ref to original>})
```

## Self-check

1. Mock action: `negatives.add_from_search_queries` для campaign с ownership=`managed`.
2. Apply → ledger row applied.
3. Repeat → ledger row skipped_idempotent.
4. Mock action для campaign с ownership=`foreign` → skipped_block.
5. Mock drift: API state ≠ before_value → skipped_drift.
6. Mock medium-risk action → создан snapshot файл.
