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

> 🚨 **Источник правды — `state.yaml`, не метки в Директе.** Internal-метки автопилота (`autopilot:managed`, `autopilot:holdout`, `autopilot:released`, `city:<city>`, `topic:<topic>`, `channel:<channel>`) **больше не ставятся** — они засоряют каталог Этажей и ломают агрегацию статистики. На existing-кампаниях такие метки снимаются однократным cleanup-прогоном.

Резолюция ownership (reconcile, decide, apply — везде одинаково):

| Где числится | Ownership |
|---|---|
| `state.yaml.campaigns[<id>].ownership == "managed"` | `managed` — бот свободно действует |
| `state.yaml.campaigns[<id>].ownership == "released"` | `released` — бот не действует, метаданные хранятся для истории |
| `city.yaml.holdout.campaign_ids[]` содержит `<id>` | `holdout` — read-only forever |
| API возвращает кампанию, нет ни в state, ни в holdout | `foreign` — read-only |

**Adoption:** добавление в `state.yaml.campaigns` через action `campaign.adopt_existing` (operational write, без вызова `add_labels` на стороне Директа). Запись содержит `topic`, `channel`, `client_login`, `ownership: managed`, `adopted_at`.

**Release:** меняем `state.yaml.campaigns[<id>].ownership = released` (или удаляем запись — равноценно для предиката ownership; запись лучше оставлять для аудита).

**Где живут topic / channel:** в `state.yaml.campaigns[<id>].{topic,channel}`. Если кампания adopted и значения неизвестны — выводятся: (1) channel из `BiddingStrategy` (Search/Network OFF), (2) topic из имени кампании по конвенции `<Город> | <Канал> | <Тематика> | <Детали> | [<посадка>]` через mapping в `.claude/skills/leadgen/config/labels.md` (business label → internal topic). Метки `topic:*` / `channel:*` для этого больше не нужны.

**Бизнес-метки Этажей (`Лидген`, `Вторичка`, `Покупатель`, `Поиск`, `РСЯ`, …) — НЕ трогаем.** Их ставит человек по правилам `leadgen/config/labels.md`. Автопилот их **только читает** для mapping, никогда не модифицирует и никогда не создаёт через `add_labels`/`set_banner_labels`.

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

## 5. Smoke / debug режим (опциональный)

Полный контур (W1-W10) **реализован** — см. RECENT-CHANGES.md #27 (2026-05-04). Все branches в `branches/` — рабочие, не заглушки.

Для отладки можно сделать «hello-only» прогон без onboarding/analyze/apply, передав в команде флаг `mode=hello`. Поведение:
- прочитать city config;
- проверить HALT;
- взять lock;
- отправить `Hello from autopilot, city=<city>, run_id=<run_id>` через `autopilot/lib/telegram_send.sh`;
- release lock; exit 0.

Без явного `mode=hello` запуск проходит **полный цикл** по таблице в секции 2 (роутер режим + onboarding если `state.yaml` пуст, analyze, decide, apply, memory_write, notify).

**Защита первого прогона:** новый город по умолчанию запускается с `baseline_mode: true` в `city.yaml` — `branches/onboarding.md` тогда собирает baseline state + формирует `launch_proposal.{md,yaml}` + шлёт в Telegram, **без создания кампаний**. После ревью оператора и снятия `baseline_mode` следующий прогон создаёт DRAFT-кампании и активирует их (если профиль это разрешает).

---

## 6. Ссылки

- Корневой роутер автопилота (контекст, ключевые правила): `autopilot/CLAUDE.md`.
- Архитектура: `PLAN.md`.
- Контракт shared playbooks: `references/playbook_contract.md`.
- Каталог сигналов: `references/signal_catalog.md`.
- Каталог действий с risk/min_evidence: `references/action_catalog.md`.
