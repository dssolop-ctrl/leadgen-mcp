# Branch: decide — signal → action plan

> Грузится в фазе D6 (после signals + reconcile). Преобразует сигналы в ranked action plan.

## Алгоритм

1. **Собрать candidate actions.** Для каждого `signal` в `runs/<run_id>.md`:
   - Найти `proposed_actions` (из `signal_catalog.md`).
   - Для каждого action_type:
     - Создать черновой Action object: `{action_type, signal_id, reason_code, entity_*, evidence}`.
     - Нагенерить `action_id` (ULID).
     - Нагенерить `idempotency_key` по правилу `<city>|<account>|<entity_type>|<entity_id>|<action_type>|<desired_value>|<decision_window>`.

2. **Resolve permissions.** Для каждого action:
   - Lookup в `caps_defaults.permissions` → `trust_profiles/<profile>.permissions` → `cities/<city>.permissions` → `cities/<city>.topics_permissions.<topic>` → `runtime_override` (approval).
   - Записать `permission_resolved` и `permission_source`.

3. **Apply autonomy_mode.**
   - `read_only` → все actions помечаются `status: planned`, в apply не идут.
   - `with_approvals` → permission resolution используется как есть.
   - `full_auto`:
     - `auto` → выполнить.
     - `auto_with_notify` → выполнить + send Telegram.
     - `review_queue` → **downgrade до `auto_with_notify`**.
     - `block` → **остаётся `block`**, никогда не выполняется.

4. **Caps + cooldown filter.**
   - Проверить `caps.max_daily_*_change_pct` — если desired_value превышает кап, downgrade action_type до `*.above_cap` варианта (если есть в catalog).
   - Проверить `cooldown_hours_after_*` — если последний change на entity слишком свежий → `status: skipped_cooldown`.
   - Проверить channel-specific `min_evidence` — если недостаточно → `status: skipped_caps, skip_reason: insufficient_evidence`.

5. **Idempotency check.**
   - Скан `action_ledger.jsonl` для `idempotency_key` == текущему за `decision_window`.
   - Если найдено `applied|failed` → `status: skipped_idempotent`.

6. **Drift / human_override check.**
   - Если `state.campaigns[entity_id].human_override_until > now` → `status: skipped_human_override`.
   - Если drift detected по entity (S-DRIFT-DETECTED) → skip действия для этой entity.

7. **Hard blocks.**
   - Action_type ∈ `hard_blocked_actions` (см. `branches/safety.md`) → `status: skipped_block`.

8. **Pacing-driven gating.**
   - `pacing_state == hard_cap`: разрешены только `*.pause_due_to_overrun`.
   - `pacing_state == emergency`: разрешены защитные actions (pause, negatives, blacklist), запрещены увеличения бюджета/ставок.
   - `pacing_state == conservation`: запрещены агрессивные actions (`bid.increase.*`, `budget.increase.*`).
   - `pacing_state == normal`: всё открыто.

9. **Priority & merge** (см. `references/decision_priorities.md`):
   - Слияние множественных signals на одну entity.
   - Применение per-run caps (`max_actions_per_run`, `max_new_campaigns_per_run`, ...).
   - Tie-breaker: confidence → cooldown_until → action_type alphabetical.

10. **Output.**
    - Список Action objects с полным `status`.
    - Записать в `runs/<run_id>.md` секция «План действий».
    - Передать в `branches/apply.md`.

## Confidence-определение

```python
# Простое правило: confidence зависит от качества evidence.
def determine_confidence(action):
    ev = action.evidence
    minev = action_catalog[action_type].min_evidence
    score = 0
    if ev.clicks >= 2 * minev.clicks: score += 1
    if ev.conversions >= 2 * minev.conversions: score += 1
    if ev.window == "last_7_full_days": score += 1
    if ev.compare_to == "target": score += 1
    if score >= 3: return "high"
    if score >= 1: return "medium"
    return "low"
```

Confidence используется в:
- Tie-breaker.
- HTML-отчёте (для прозрачности специалисту).
- Learnings (proposed обогащается confidence).

## Expected effect / guard metric

Для applied medium/high — обязательно:

```python
expected_effect = action_catalog[action_type].expected_effect_template.format(value=desired_delta)
guard_metric = action_catalog[action_type].guard_metric  # например "spend_without_conversions_3d"
```

Записываются в Action object и в narrative `runs/<run_id>.md`.

## Self-check (mock сценарий)

1. Создать mock-signals: `S-CPA-ABOVE-TARGET` для campaign 8765432.
2. Запустить decide → план содержит `bid.decrease.within_cap` + `negatives.add_from_search_queries`.
3. Permissions: `pilot_full_auto` + `full_auto` → `auto`.
4. Idempotency check: ledger пуст → не skip.
5. Cooldown: last_action 5 дней назад → не skip.
6. Output: 2 actions со статусом `planned`, готовы к apply.
