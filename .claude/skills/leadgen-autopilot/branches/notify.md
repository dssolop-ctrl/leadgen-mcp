# Branch: notify — отчёты + Telegram

> Грузится в фазе D8 (после memory_write). Формирует HTML отчёт и compact Telegram summary.

## 1. HTML отчёт

```bash
# Сгенерить markdown summary прогона
md_path="runtime/<city>/narrative/runs/<YYYY-MM>/<HHMM>.md"
html_path="reports/<city>/<YYYY-MM>/<HHMM>-daily.html"

bash autopilot/lib/render_html.sh "$md_path" "$html_path" --title "Daily — <city> $(date)"
```

Для weekly/monthly — отдельный md формируется branch'ем analyze (W4 / M6), тот же render_html.

## 2. Telegram daily summary (compact)

```text
🟢 [<city>] <DD MMM HH:MM> · <autonomy_mode>
Бюджет день: <X> / <Y> ₽ (<%>)
Лиды: <N> · CPA <Z> ₽ (цель <T>)
Действия: <total> (auto: <a>, auto_with_notify: <an>, skipped: <s>)
  · <topic-channel>: <action_summary>
  · <topic-channel>: <action_summary>
[⚠ <flag>] · [⚠ <flag>]

[detail.html прикреплён]
```

Эмодзи в зависимости от состояния:
- 🟢 — normal pacing, нет critical signals.
- 🟡 — conservation pacing OR medium-severity signals.
- 🔴 — emergency pacing OR high-severity drift / failed actions.
- 🛑 — HALT.

## 3. Per-action notify (auto_with_notify)

Отправляется отдельным сообщением **ДО** или сразу после apply (см. `branches/apply.md` шаг 7):

```text
🟡 [<city>] <action_type>
Entity: <type> <id> '<name>'
Reason: <reason_code>
Evidence: <key=val,...>
Result: applied successfully
```

## 4. Approval-уведомления (`with_approvals`)

При создании нового pending entry:

```text
🔔 [<city>] Approval needed #<id>
Action: <action_type>
Entity: <type> <id> '<name>'
Reason: <reason_code>
Evidence: <key=val,...>
Risk: <risk_class>
Expires: <ISO>

Reply:
- approve <id>
- reject <id>
- defer <id> 3d
```

`message_id` от ответа Telegram — сохраняется в `pending_approvals.yaml.entry.telegram.message_id`.

## 5. Алерты (incidents)

Отдельными сообщениями (high priority):

```text
🚨 [<city>] <incident_type>
<details>

Action required: <suggested_step>
```

Типы: `HALT`, `Stale lock`, `Drift unexplained`, `Apply failure (high risk)`, `Hard cap budget overrun`, `Telegram polling failed`.

## 6. Защиты (см. lib/telegram_send.sh)

- HTML/Markdown escaping динамических полей (если parse_mode используется).
- Сообщение >3900 символов → fallback на `sendDocument`.
- Retry 3 раза с exponential backoff.
- Маскирование токена в логах.
- Allowlist chat_id перед каждой отправкой.

## 7. Failure handling

Если `telegram_send.sh` упал 3 раза подряд:
- Записать в `runs/<run_id>.md` секция "Errors": `notify_failed`.
- Сохранить unsent message в `runtime/<city>/notify_queue.txt`.
- На следующем прогоне retry (в начале daily_safety_check).
- Если после 3 прогонов всё ещё не отправлено — пытаться через alternate channel (если есть в `secrets.env.TELEGRAM_FALLBACK_*`).

## Self-check

Mock сценарий:
1. Сформировать compact daily summary string.
2. Вызов `telegram_send.sh` с реальным токеном (если есть в secrets) или mock (если нет).
3. Сформировать HTML через `render_html.sh`.
4. Mock approval entry → форматирование approval-сообщения.
