# Branch: analyze — daily / weekly / monthly анализ

> Грузится после safety_check. Содержит логику для всех трёх режимов; режим определяется в `skill.md` Роутере.

## Daily mode

### D3: Fetch metrics

Источники (вызываются через MCP-сервер):

```
# Direct stats per managed campaign (LYDC, last 7 full days)
get_campaign_stats(client_login=<>, campaign_ids=<managed_ids>, date_from=YYYY-MM-DD, date_to=YYYY-MM-DD)

# Metrika direct report with goals from city.metrika.goals
metrika_get_direct_report(counter_id=<>, goal_ids=[lead_form, call, qualified_lead], attribution=LYDC, date_from=, date_to=)

# Search queries для минусования
get_search_queries(client_login=<>, campaign_ids=<>, date_from=YYYY-MM-DD)

# Drift detection
get_change_history(client_login=<>, campaign_ids=<>, date_from=<last_run>)

# RSYA placements
get_blocked_placements(client_login=<>, campaign_ids=<rsya_ids>)
get_criteria_stats(...)  # для placement performance
```

**Freshness rules:**
- Использовать `yesterday` + rolling `last_3_full_days` / `last_7_full_days` для daily decisions.
- `today` — только для emergency safety check (overrun, drift).
- Qualified leads (CRM) — НЕ использовать для daily (CRM-задержка 5-7 дней). Только в weekly/monthly.

### D3.1: Запись metrics_snapshots

Каждый прогон append'ит per-scope srez в `metrics_snapshots.jsonl`:

```json
{"date": "2026-05-04", "city": "<city>", "scope": {"level": "account"}, "spend": 12000, "clicks": 240, "impressions": 4500, "leads_form": 8, "leads_call": 3, "cpa_form": 1500, "cpa_call": 4000, "data_window": "2026-05-04", "attribution": "LYDC"}
{"date": "2026-05-04", "city": "<city>", "scope": {"level": "topic", "topic": "vtorichka"}, ...}
{"date": "2026-05-04", "city": "<city>", "scope": {"level": "campaign", "campaign_id": 8765432}, ...}
```

### D4: Extract signals

Применить правила из `references/signal_catalog.md`. Для каждой entity (account / topic / campaign / adgroup / placement) — пройти по всем релевантным `S-*` правилам.

Каждый сработавший signal — записать в `runs/<run_id>.md` секция «Сигналы» с:
- `signal_id`, `severity`, `entity_type`, `entity_id`, `topic`, `channel`, `evidence`, `proposed_actions`.

### D5: Reconcile config↔state↔API

См. `branches/reconcile_config.md`.

### D6: Build action plan

См. `branches/decide.md` (ленивая загрузка только если есть signals → actions).

### D7-D8: см. `branches/apply.md` и `branches/notify.md`, `branches/memory_write.md`.

---

## Daily dynamics table — единый формат

> Применяется ко всем режимам (daily/weekly/monthly) и ко всем артефактам (`runs/<HHMM>.md`, weekly/monthly HTML, Telegram-вложения).

**Правило:** «Дневная динамика» = **одна строка на день**, агрегация по всем managed-кампаниям. **НЕ разбивать на search/rsya** (поиск и РСЯ суммируются в общие колонки). Разбивка по каналам/кампаниям живёт в отдельной таблице «за период» (см. выше «Метрики недели»), не дублируется в дневной.

Колонки таблицы дневной динамики (минимальный набор):

```markdown
| Дата | Imp | Clicks | Spend ₽ | CTR | CPC ₽ | Form | Calls | CPA_form | CPA_call |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 2026-05-15 | 4 232 | 188 | 764 | 4.44% | 4.06 | 0 | 0 | — | — |
| 2026-05-16 | 6 099 | 385 | 2 128 | 6.31% | 5.53 | 0 | 4 | — | 532 |
| 2026-05-17 | 10 778 | 519 | 2 541 | 4.82% | 4.90 | 1 | 0 | 2 541 | — |
| **Итого** | 21 109 | 1 092 | 5 433 | 5.17% | 4.98 | 1 | 4 | 5 433 | 1 358 |
```

Допустимые расширения: колонка `Note` для краткого комментария по дню (старт, инциденты, операторские правки). НЕ добавлять колонки `Канал` / `Кампания` — это ломает «один день — одна строка».

Если managed-кампаний нет вовсе (baseline_mode или новый город) — таблица не выводится; вместо неё текст-абзац «Дневная динамика недоступна — нет управляемых кампаний».

---

## Weekly mode

### W2: Weekly metrics

Сводка по тематикам и каналам за полную предыдущую неделю:
- spend, leads_form, leads_call, leads_qualified, CPA per goal.
- Сравнение неделя/неделя (`current vs previous_week`).
- Top-3 campaigns по росту/падению CPA.
- Top-5 search queries без конверсий (для добавления в global negatives).

### W3: Tactical план на след. неделю → CURSOR

В narrative `CURSOR.md` секция "План на неделю":
- Какие ставки/бюджеты планируется тестировать.
- Какие гипотезы открыты в `decisions/`.
- Какие cooldowns истекут (можно действовать).

### W4: Weekly HTML report

`reports/<city>/<YYYY-MM>/week-<NN>.html`. Через `lib/render_html.sh` — обязательно с YAML frontmatter по конвенции из `branches/notify.md` § 1.1.

Структура md:
```yaml
---
title: "Weekly — <city>, неделя <NN>"
date: "<YYYY-MM-DD конца недели>"
city: <city>
mode: weekly
status: { label: "<тренд недели в 2-3 словах>", kind: success|warn|danger }
kpi:
  cards:
    - { label: "Spend неделя", value: "<X>", suffix: "/ <Y> ₽", foot: "<Δ vs prev_week>" }
    - { label: "Leads (form+call)", value: "<N>", foot: "<Δ vs prev_week>" }
    - { label: "CPA неделя", value: "<C>", suffix: "₽", foot: "target <T> ₽", kind: warn|alert? }
    - { label: "Decision precision", value: "<%>", foot: "<applied non-rollback>" }
    - { label: "Open decisions", value: "<N>", foot: "<context>" }
---
```

Содержание body:
- `## TL;DR` — 3-4 строки.
- Таблица KPI per topic/channel.
- **Таблица дневной динамики** по правилу выше («Daily dynamics table — единый формат»): одна строка на день, агрегация всех РК, без колонок «Канал»/«Кампания».
- Список применённых actions за неделю.
- Сравнение с прошлой неделей.
- Открытые вопросы.

### W5: SUMMARY компрессия

См. `branches/memory_write.md` секция 3.

## Monthly mode

### M2: Monthly metrics + quality

- KPI по месяцу: spend, leads (form/call/qualified), CPA, decision precision, rollback rate.
- Per-topic ROI и pacing accuracy (`forecast_eom_actual / forecast_eom_predicted`).
- Coverage: % дней без HALT/падения.
- Telegram noise: `auto_with_notify` сообщений в день.

### M3: Budget plan для следующего месяца

```yaml
# Записывается в runs/<run_id>.md секция "Monthly budget plan"
proposed_budget_<YYYY-MM>:
  vtorichka:
    monthly_budget: 65000   # +5k vs прошлый
    rationale: "Лиды растут, CPA стабилен на target"
  novostroyki:
    monthly_budget: 70000   # -10k vs прошлый
    rationale: "CPA выше target × 1.3, режем"
```

Если есть delta vs текущий `city.yaml` — формируется как **предложение специалисту** (через monthly digest), а не апплаится автоматически (изменение `city.yaml` — ручное).

### M4: Learnings monthly digest

См. `branches/learnings.md`. Сводка proposed → suggest validated.

### M5: Holdout comparison (если включён)

Сравнить managed vs holdout campaigns:
- avg CPA managed vs holdout.
- Если managed >= holdout — alert ("автопилот не лучше базы за месяц").
- Если managed < holdout × 0.85 — заметка "автопилот даёт значимый прирост, рассмотреть aggressive profile".

### M6: Monthly HTML report + Telegram push

`reports/<city>/<YYYY-MM>/month-<MM>.html` — обязательно с YAML frontmatter по конвенции из `branches/notify.md` § 1.1.

Структура frontmatter:
```yaml
---
title: "Monthly — <city>, <YYYY-MM>"
date: "<YYYY-MM-DD конца месяца>"
city: <city>
mode: monthly
status: { label: "<резюме месяца>", kind: success|warn|danger }
kpi:
  cards:
    - { label: "Spend месяц", value: "<X>", suffix: "/ <Y> ₽", foot: "<Δ vs prev_month>" }
    - { label: "Qualified leads", value: "<N>", foot: "<Δ vs prev_month>" }
    - { label: "CPA топ-темы", value: "<C>", suffix: "₽", foot: "target <T> ₽" }
    - { label: "Rollback rate", value: "<%>", foot: "<count> / <total>", kind: alert? }
    - { label: "Pacing accuracy", value: "<%>", foot: "forecast_eom vs actual" }
    - { label: "Holdout Δ", value: "<+X%>", foot: "managed vs holdout CPA" }
---
```

Содержание body:
- `## TL;DR` — 3-4 строки.
- KPI месяца + сравнение с предыдущим.
- **Таблица дневной динамики** по правилу из daily-секции («Daily dynamics table — единый формат»): одна строка на день за весь месяц, без разбивки на каналы/РК.
- Quality metrics автопилота.
- Budget plan на следующий месяц (proposed).
- Learnings digest.
- Holdout comparison (если есть).
- Открытые вопросы и риски.

В narrative — `decisions/monthly-<YYYY-MM>.md` с архивом размышлений.
