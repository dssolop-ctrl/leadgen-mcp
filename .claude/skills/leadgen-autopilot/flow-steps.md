# Flow Steps — anchors шагов автопилота

> Стабильные anchors для cross-references из branches. Не меняй текст после anchor без обновления ссылок.

## Daily (D-шаги)

### D1: Bootstrap
- Прочитать `city.yaml`, `caps_defaults.yaml`, `trust_profiles/<profile>.yaml`.
- Проверить `autopilot/HALT.flag` и `autopilot/runtime/<city>/HALT.flag`.
- Acquire per-city lock.
- Сгенерировать `run_id`.
- Создать запись в `runs/<YYYY-MM>/<run_id>.md` с initial context.
- Phase: `loaded_context`.

### D2: Approval pull (только при autonomy=with_approvals)
- Запустить `lib/telegram_check_replies.sh`.
- Считать `pending_approvals.yaml`.
- Применить approve/reject/defer.
- Удалить expired (по `expires_at`).
- Phase: `approval_checked`.

### D3: Fetch metrics
- LYDC через `metrika_get_direct_report` для goals из `city.metrika.goals`.
- `get_campaign_stats`, `get_search_queries`, `get_change_history`, `get_blocked_placements`.
- Полные дни only (yesterday + rolling 3/7d).
- Записать строки в `metrics_snapshots.jsonl`.
- Phase: `fetched_metrics`.

### D4: Extract signals
- По `signal_catalog.md`: cpa_jump, budget_overrun, no_conversions, learning_ended, drift_detected, и т.п.
- Сигналы записать в `runs/<run_id>.md` с evidence.

### D5: Reconcile config↔state↔API
- См. `branches/reconcile_config.md`.
- Дельты config↔state и state↔API.
- При drift — human_override flow.

### D6: Build action plan
- Каждый сигнал/дельта → action(s) по `action_catalog.md`.
- Для каждого action: `signal_id, reason_code, confidence, evidence, risk_class, idempotency_key, permission_resolved, permission_source`.
- Применить caps + cooldowns + idempotency check.
- Phase: `planned_actions`.

### D7: Apply
- См. `branches/apply.md`.
- По autonomy_mode:
  - `read_only` → skip apply, всё остаётся в plan.
  - `with_approvals` → auto/auto_with_notify применяются, review_queue идут в pending_approvals.yaml.
  - `full_auto` → auto, auto_with_notify применяются (auto_with_notify шлёт отдельный TG), review_queue downgrade до auto_with_notify, block НЕ выполняется.
- Phase: `applying` → `applied_partial` → ... → конечный.

### D8: Memory + report + notify
- Регенерация `STATE.md`, `CURSOR.md`, `SUMMARY.md` из state.yaml.
- HTML report через `lib/render_html.sh`.
- Telegram daily summary + html attach.
- Phase: `memory_written` → `notified` → `succeeded`.
- Release lock.

---

## Weekly (W-шаги)

### W1: Daily safety check (always)
- HALT, lock recovery, drift detection, hard pacing cap, expired approvals.
- Никаких tactical daily-decide в weekly-день.

### W2: Weekly metrics
- Week-over-week по тематикам/каналам.
- Decision precision за прошлую неделю.

### W3: Update CURSOR with weekly tactics
- План на следующую неделю (приоритеты, гипотезы).

### W4: Weekly HTML report
- Графики недели.

### W5: SUMMARY compression
- Записи >30 дней → недельные строки.

---

## Monthly (M-шаги)

### M1: Daily safety check (always)
- То же, что W1.

### M2: Monthly metrics + quality
- Decision precision, rollback rate, approval health, coverage, telegram noise, pacing accuracy.

### M3: Budget plan для следующего месяца
- По тематикам, с обоснованием.

### M4: Learnings monthly digest
- proposed → review предложения специалисту.

### M5: Holdout comparison (если включён)
- managed vs holdout CPA/leads.

### M6: Monthly HTML report + Telegram push
- Запись в `narrative/decisions/monthly-<YYYY-MM>.md`.

---

## Финиш (всегда)
- Update `state.yaml.last_run.phase = succeeded | failed | halted`.
- Atomic write narrative md.
- Release lock.
- Exit code 0.
