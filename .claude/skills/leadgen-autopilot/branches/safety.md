# Branch: safety — kill-switch, idempotency, hard caps, lock recovery

> Этот branch грузится **всегда** в начале прогона + при weekly/monthly как `daily_safety_check`.

## 1. HALT-проверка

```bash
test -f autopilot/HALT.flag && halt_global=true
test -f autopilot/runtime/<city>/HALT.flag && halt_city=true
```

Если `halt_global` или `halt_city`:
1. Прочитать причину (содержимое HALT.flag — свободный текст; "manual stop" по умолчанию).
2. Записать в `runs/<run_id>.md`: `phase: halted`, `reason: <halt reason>`.
3. Послать Telegram alert: `🛑 Autopilot HALT — <city> — reason: <reason>`.
4. Release lock (если успели взять).
5. Exit 0.

## 2. Lock acquire

```bash
exit_code=$(bash autopilot/lib/lock.sh acquire <city> <run_id> started; echo $?)
```

- `0` → продолжаем.
- `1` → активный прогон есть. Telegram alert + exit 0 (нет ошибки, просто пересечение).
- `2` → recovery. Прочитать `state.yaml.last_run.phase` и `runs/` последнего прогона:
  - Если phase ∈ `applying|applied_partial` → drift check всех entities с открытыми actions в ledger.
  - Если phase ∈ `started|loaded_context|fetched_metrics|planned_actions|approval_checked` → можно продолжить штатно (ничего не применилось).
  - Записать recovery-event в `runs/<run_id>.md`.

Двойной stale lock подряд → incident alert.

## 3. Hard pacing cap

В каждом прогоне после `D3 fetch metrics`:
```python
elapsed_share = days_passed_in_month / days_in_month
expected = monthly_budget * elapsed_share
spent_mtd = managed_spend_mtd + unmanaged_spend_mtd
pacing_ratio = spent_mtd / expected
forecast_eom = avg_daily_spend_recent * days_in_month

if spent_mtd > monthly_budget * (caps.budget_overrun_hard_stop_pct / 100):
    pacing_state = hard_cap
    # экстренная пауза всех managed campaigns + alert
elif pacing_ratio > caps.pacing_emergency_ratio or forecast_eom > monthly_budget * (caps.forecast_emergency_pct / 100):
    pacing_state = emergency
    # пауза aggressive actions, разрешены только защитные (минусы, blacklist)
elif pacing_ratio > caps.pacing_conservation_ratio:
    pacing_state = conservation
    # отключить инкремент бюджета и расширения, разрешены контрол-actions
else:
    pacing_state = normal
```

Записать `pacing_state` в `state.yaml.budget.pacing_state` и в Telegram daily summary.

## 4. Hard blocks (никогда не выполняются)

```yaml
hard_blocked_actions:
  - legal.update_disclaimer
  - legal.change_company_info
  - account.change_settings
  - account.change_billing
  - account.close
  - campaign.delete
  - campaign.create_draft.outside_config_topics
```

В `branches/decide.md` при формировании плана — отбрасывать с `status: skipped_block` и логировать.

## 5. Hello-mode (W1 self-check)

Пока другие branches не реализованы (W1):
- Прочитать city.yaml, проверить HALT, взять lock.
- Сформировать сообщение:
  ```
  Hello from autopilot
  city: <city>
  run_id: <run_id>
  trust_profile: <profile>
  autonomy_mode: <mode>
  topics_active: <count>
  ```
- Отправить через `bash autopilot/lib/telegram_send.sh <chat_id> "<text>"`.
- Записать `runs/<run_id>.md` с phase: started → notified → succeeded.
- Release lock.

После W2-W10 этот hello-mode заменяется полным циклом.
