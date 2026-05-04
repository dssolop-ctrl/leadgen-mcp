---
name: leadgen-autopilot
description: "Автономный агент для управления рекламой Этажей в Яндекс Директе. Триггер: /leadgen-autopilot или 'запусти автопилот для города X'. Делает daily/weekly/monthly анализ, принимает решения, применяет действия (по уровню autonomy), формирует отчёт, шлёт в Telegram. Работает по конфигу autopilot/config/cities/<city>.yaml. ОТЛИЧИЕ от /leadgen: автопилот действует самостоятельно по расписанию, /leadgen — диалог со специалистом."
---

# Leadgen Autopilot — автономный агент управления рекламой

> **Это policy/orchestration скилл.** Используется для **автоматических** прогонов через Claude Desktop routine. Для ручной работы со специалистом — отдельный скилл `/leadgen` (не вызывать в runtime автопилота).

**MCP-сервер:** `leadgen-mcp` (Yandex Direct + Metrika + Wordstat + VK Ads + image gen).
**Конфиг:** `autopilot/config/cities/<city>.yaml`. Если не нашёл — остановись и попроси специалиста создать.
**Память:** двухслойная (operational + narrative) в `autopilot/runtime/<city>/`.
**Документация архитектуры:** `PLAN.md` в корне репо.

---

## 0. Старт прогона: парсинг команды

Команда пользователя (из routine): `/leadgen-autopilot city=<city>` или `Запусти автопилот для <city>`.

**Шаг 0.1.** Извлечь `<city>` из ввода. Если не указан — fail loud, попросить указать.

**Шаг 0.2.** Прочитать:
- `autopilot/HALT.flag` — если есть, лог + Telegram alert + exit.
- `autopilot/config/cities/<city>.yaml` — обязательно. Если нет — fail loud.
- `autopilot/runtime/<city>/HALT.flag` — если есть, exit.

**Шаг 0.3.** Acquire per-city lock через `bash autopilot/lib/lock.sh acquire <city> <run_id> started`.
- Exit 0 → продолжаем.
- Exit 1 → лог, exit (есть активный прогон).
- Exit 2 → recovery: читаем последний `runs/<id>.md` и `state.yaml.last_run.phase`, продолжаем с понимания, где упал прошлый прогон.

**Шаг 0.4.** Сгенерировать `run_id = <city>-<YYYY-MM-DD>-<HHMM>`. Запомнить.

---

## 1. Роутер: какой режим прогона активен сегодня

| Условие | Режим |
|---|---|
| `today.dom == city.schedule.monthly_rollup_dom` | `monthly` (включает daily_safety_check) |
| `today.dow == city.schedule.weekly_rollup_dow` | `weekly` (включает daily_safety_check) |
| иначе | `daily` |

**Совпадение weekly + monthly** → `monthly` (weekly merged).

---

## 2. Branches: что грузить по режиму

| Шаг | Файл | Когда грузить |
|---|---|---|
| Onboarding (первый прогон или пустой `state.yaml`) | `branches/onboarding.md` | Если `state.yaml` отсутствует или `state.campaigns == []` |
| Daily safety check | `branches/safety.md` (раздел "safety-check") | Всегда |
| Analyze daily | `branches/analyze.md` (секция "daily") | Если режим = daily |
| Analyze weekly | `branches/analyze.md` (секция "weekly") | Если режим = weekly или monthly |
| Analyze monthly | `branches/analyze.md` (секция "monthly") | Если режим = monthly |
| Reconcile | `branches/reconcile_config.md` | Всегда (после analyze) |
| Decide | `branches/decide.md` | Если режим = daily |
| Approval pull | `branches/approval.md` | Если `city.autonomy_mode == with_approvals` |
| Apply | `branches/apply.md` | Если есть запланированные действия и autonomy != read_only |
| Memory write | `branches/memory_write.md` | Всегда |
| Notify | `branches/notify.md` | Всегда |
| Learnings | `branches/learnings.md` | В weekly/monthly после anaylze |

---

## 3. Сквозные правила

### 3.1. Источник правды

- **API > STATE.** При любом расхождении — `branches/reconcile_config.md` секция "drift handling".
- `state.yaml` — кеш. Регенерируется в конце прогона из API + ledger.
- `state.yaml` обновляется только через `bash autopilot/lib/atomic_write.sh`.

### 3.2. Permissions resolution

Порядок: `caps_defaults.yaml` → `trust_profiles/<profile>.yaml` → `cities/<city>.yaml.permissions` → `cities/<city>.yaml.topics_permissions.<topic>` → runtime override (approval).

Записать resolved permissions в `runs/<run_id>.md` с источником каждого значения.

### 3.3. Idempotency

Каждое действие имеет `idempotency_key = <city>|<account>|<entity_type>|<entity_id>|<action_type>|<desired_value>|<decision_window>`.
Перед apply — проверка `action_ledger.jsonl` + `get_change_history`. Совпадение → skip с `status: skipped_idempotent`.

### 3.4. Ownership

Бот трогает только campaigns с label `autopilot:managed`. Foreign campaigns — read-only. Холдоут (`autopilot:holdout`) — read-only.

### 3.5. DRAFT-only

`campaign.create_draft.*` всегда создаёт SUSPENDED. Активация — отдельным `campaign.activate_existing_draft`. Это правило проекта (см. `.claude/skills/leadgen/branches/create-search.md` шаг 11).

### 3.6. Hard blocks

Никогда не выполняются: `legal.update_disclaimer`, `legal.change_company_info`, `account.change_settings`, `account.change_billing`, `account.close`, `campaign.delete`, `campaign.create_draft.outside_config_topics`. Действуют во всех профилях, включая `pilot_full_auto`.

### 3.7. Atomic writes

Все eager-файлы (`state.yaml`, `STATE.md`, `CURSOR.md`, `SUMMARY.md`, `pending_approvals.{yaml,md}`) — через `bash autopilot/lib/atomic_write.sh`.

### 3.8. Логирование прогона

`runs/<YYYY-MM>/<HHMM>.md` — полный лог. Структура: контекст → метрики → сигналы → план действий → apply → memory → notify. Phases пишутся в `state.yaml.last_run.phase` через `bash autopilot/lib/lock.sh update_phase`.

---

## 4. Финиш прогона

1. Обновить `state.yaml` с финальными значениями.
2. Сгенерировать narrative md из operational (`STATE.md`, `CURSOR.md`, `SUMMARY.md`).
3. Сформировать HTML report → `reports/<city>/<YYYY-MM>/<run_id>.html`.
4. Отправить Telegram daily summary (compact) + прикрепить html report.
5. Установить `state.yaml.last_run.phase = succeeded` (или `failed`/`halted`).
6. Release lock: `bash autopilot/lib/lock.sh release <city> <run_id>`.

---

## 5. Self-check (W1 hello)

В W1 (текущая фаза разработки) скилл умеет:
- прочитать city config;
- проверить HALT;
- взять lock;
- сформировать `Hello from autopilot, city=<city>, run_id=<run_id>` сообщение;
- отправить через `autopilot/lib/telegram_send.sh`;
- release lock;
- exit с кодом 0.

Все остальные branches — заглушки до соответствующих волн.

---

## 6. Ссылки

- Корневой роутер автопилота (контекст, ключевые правила): `autopilot/CLAUDE.md`.
- Архитектура: `PLAN.md`.
- Контракт shared playbooks: `references/playbook_contract.md`.
- Каталог сигналов: `references/signal_catalog.md`.
- Каталог действий с risk/min_evidence: `references/action_catalog.md`.
