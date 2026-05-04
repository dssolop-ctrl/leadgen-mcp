# Branch: approval — pending queue + Telegram replies + expiry

> Грузится **только** при `city.autonomy_mode == with_approvals`.

## 1. Структура `pending_approvals.yaml`

Schema: `autopilot/schemas/approval.schema.json`. Файл — список approvals.

```yaml
- approval_id: "1"
  action:
    action_id: act-2026-05-04-001
    action_type: campaign.activate_existing_draft
    risk_class: high
    city: omsk
    entity_type: campaign
    entity_id: 8765432
    decision_window: 2026-05-04
    idempotency_key: "omsk|client|campaign|8765432|activate|true|2026-05-04"
    signal_id: S-NEW-TOPIC-ENABLED
    reason_code: draft_ready_for_activation
    confidence: high
    evidence: { moderation_status: ACCEPTED, draft_age_hours: 1 }
    permission_resolved: review_queue
    permission_source: trust_profiles/conservative.yaml
    status: planned
  created_at: 2026-05-04T07:30:30Z
  expires_at: 2026-05-07T07:30:30Z   # 72h для draft activation
  status: pending
  telegram:
    chat_id: -1001234567890
    message_id: 12345
    sent_at: 2026-05-04T07:30:30Z
```

## 2. Approval pull (D2 phase)

В начале каждого прогона при `with_approvals`:

```bash
# Загрузить новые ответы
bash autopilot/lib/telegram_check_replies.sh <city>
```

Скрипт:
1. Читает `runtime/_global/telegram_offset` (last_update_id).
2. `getUpdates?offset=<last>+1&allowed_updates=[message]`.
3. Для каждого message в allowlisted chat_id:
   - Парсит команды: `approve <id>`, `reject <id>`, `defer <id> <Nd>`, `rollback <run_id>`, `confirm rollback <id>`.
   - Применяет к `pending_approvals.yaml` (статус: approved/rejected/deferred).
4. Обновляет `telegram_offset` до max(update_id) + 1.

## 3. Expiry

```python
now = utcnow()
for entry in pending_approvals:
    if entry.status == "pending" and entry.expires_at < now:
        entry.status = "expired"
        # Не применять старое решение слепо.
        # На следующем прогоне signal извлечётся заново и сформируется новое предложение
        # с актуальной evidence.
```

Expiry windows:
- `bid.*` / `budget.*` → 24h.
- `campaign.create_draft.*` / `campaign.adopt_existing` / `campaign.activate_existing_draft` → 72h.
- Прочее → 48h (default).

## 4. Применение approved actions

После approval pull, перед reconcile/decide:

1. Найти все `entry.status == "approved"` с непротёкшим expiry.
2. Для каждого:
   - Сверить `idempotency_key` с ledger — не выполнено ли уже (защита от двойного approve).
   - Проверить, что evidence ещё актуален (например, draft не старше approval_age + 24h).
   - Передать action в apply pipeline (`branches/apply.md`) с `permission_source: runtime_override:approval:<id>`.
3. После apply — `entry.status = applied`, добавить `applied_run_id`.

## 5. Применение rejected / deferred

- `rejected` → action skip навсегда. В ledger: `status: skipped_block, skip_reason: rejected_by_user`.
- `deferred 3d` → `expires_at += 3d`, status сохраняется `pending`. Но **action не применяется**, ждёт нового approve.

## 6. Создание новых pending entries

В D7 apply: для actions с `permission_resolved == review_queue`:

```python
entry = {
    "approval_id": next_seq(),
    "action": <full Action obj>,
    "created_at": utcnow(),
    "expires_at": utcnow() + timedelta(hours=expiry_hours_for(action_type)),
    "status": "pending",
}
pending_approvals.append(entry)
```

После добавления — отправить в Telegram (`branches/notify.md` — секция "approvals"):

```
🔔 [omsk] Approval needed #1
Action: campaign.activate_existing_draft
Entity: campaign 8765432 "Омск-Вторичка-Поиск"
Reason: draft moderation passed, ready for activation
Evidence: moderation_status=ACCEPTED, draft_age=1h
Risk: high
Expires: 2026-05-07 07:30 UTC

Reply:
- approve 1
- reject 1
- defer 1 3d
```

`message_id` от ответа Telegram сохраняется в `entry.telegram.message_id`.

## 7. Защита от двойного применения

- `update_offset` персистится (защита от повторных обновлений после crash).
- `idempotency_key` проверяется перед apply (защита от двойного approve).
- `applied_run_id` фиксируется (для аудита).

## 8. Self-check (W4)

Mock-сценарий:
1. Создать pending entry в `pending_approvals.yaml` с истёкшим `expires_at`.
2. Запустить (имитация) approval pull → entry.status = "expired", не применяется.
3. Создать pending entry с актуальным expires_at.
4. Положить fake update в getUpdates response (mock) с `approve 1`.
5. Запустить approval pull → entry.status = "approved".
6. Запустить apply pipeline → action применяется, ledger row `status: applied`.
7. Повторный запуск с тем же idempotency_key → skip (`status: skipped_idempotent`).
