# Branch: notify — отчёты + Telegram

> Грузится в фазе D8 (после memory_write). Формирует HTML отчёт и compact Telegram summary.

> 🇷🇺 **HARD RULE: язык Telegram-сообщений — только русский.**
>
> Любой текст, который уходит в Telegram (daily summary, per-action notify, approval-уведомления, алерты, incident-сообщения, ответы на pending), формируется **на русском**. Никаких автогенерируемых английских строк, англоязычных reason_code / status / error_message в видимой пользователю части — всё локализовать перед отправкой.
>
> Допустимые исключения внутри текста: идентификаторы (`run_id`, `campaign_id`, `idempotency_key`), названия MCP-инструментов, технические имена полей API (например `BiddingStrategy.State`), коды ошибок API (`8300`), URL'ы. Они остаются как есть.
>
> **Почему:** пользователь (оператор) работает с автопилотом по-русски, mix-of-languages снижает скорость восприятия и доверие. Все шаблоны в этом файле уже на русском — следить, чтобы рантайм не подставлял англоязычные fragments из MCP/ошибок без перевода.
>
> **Как применять:** перед каждым вызовом `telegram_send.sh` — sanity-check: текст читается полностью на русском? Если нет — переписать. Это правило перекрывает любые шаблонные тексты в этом и других branches.

## 1. HTML отчёт

```bash
# Сгенерить markdown summary прогона
md_path="runtime/<city>/narrative/runs/<YYYY-MM>/<HHMM>.md"
html_path="reports/<city>/<YYYY-MM>/<HHMM>-daily.html"

bash autopilot/lib/render_html.sh "$md_path" "$html_path" --title "Daily — <city> $(date)"
```

Для weekly/monthly — отдельный md формируется branch'ем analyze (W4 / M6), тот же render_html.

### 1.1. Структура md-отчёта (обязательная YAML frontmatter)

Любой md, который пойдёт в `render_html.sh` и потом в Telegram, **обязан** начинаться с YAML frontmatter — это контракт с pandoc-шаблоном `autopilot/lib/templates/report.html`. Хвост frontmatter'а — KPI-карточки сверху отчёта; они рендерятся в hero-strip автоматически.

```yaml
---
title: "<человекочитаемый заголовок>"        # обязательно
lang: ru                                     # по умолчанию ru, оставь
date: "YYYY-MM-DD"                           # дата прогона
city: <city-slug>
run_id: <run_id>
mode: daily | weekly | monthly
autonomy: full_auto | with_approvals | read_only
trust_profile: <profile-name>
pacing_state: normal | conservation | emergency | hard_cap
baseline_mode: true                          # ОПЦИОНАЛЬНО, только если активен
status:
  label: "<короткая фраза-резюме прогона>"   # появляется как badge в hero
  kind: info | success | warn | danger       # цвет badge
kpi:
  cards:
    - label: "Бюджет MTD"
      value: "12 340"                        # уже отформатированное (пробелы как разделители тысяч)
      suffix: "/ 100 000 ₽"                  # опционально
      foot: "12.3% spent · pacing normal"    # подзаголовок карточки, опционально
      kind: ""                               # пусто | warn | accent | alert — окрашивает border + value
    - label: "..."
      value: "..."
---

## TL;DR                                     # body начинается СРАЗУ с ## (template даёт h1 из title)

...
```

**Правила содержимого md под frontmatter'ом:**
- НЕ дублировать `# H1` в начале — шаблон сам выводит `<h1>` из `title`.
- Первая секция — `## TL;DR` (2-4 строки сути для оператора).
- Дальше — обычные `##`/`###` секции, таблицы, списки.
- Markdown-таблицы автоматически получают zebra-stripes из CSS шаблона; колонки с числами выравниваются по правому краю при `align="right"`.
- Длинные блоки кода/yaml — через ```fenced``` (шаблон стилизует pre/code).

**Что собирать в KPI-cards по режимам:**

| mode | Рекомендуемые 4-6 карточек |
|---|---|
| daily | Spend сегодня, Spend MTD/limit, Leads сегодня (form+call), CPA 7д vs target, Actions applied/skipped, Pacing |
| weekly | Spend неделя, Δ к прошлой, Leads неделя, CPA неделя, Decision precision %, Open decisions |
| monthly | Spend месяц / limit, Total leads (qualified), CPA по топ-теме, Rollback rate, Pacing accuracy %, Holdout Δ |
| onboarding (baseline) | Бюджет, Managed campaigns, Visits 14д baseline, Ad-source baseline, Действий, Top demand |

**`status.kind` логика:**
- `success` (зелёный) — всё штатно, pacing normal, нет инцидентов.
- `info` (синий) — наблюдение, baseline, нет действий.
- `warn` (жёлтый) — conservation pacing OR medium-severity signals.
- `danger` (красный) — emergency/hard_cap OR drift OR failed actions OR HALT.

**`kpi.cards[].kind`** окрашивает border карточки:
- `accent` — нейтрально-выделенная (для "Действий применено", "Decision precision" и т.п.).
- `warn` — карточка требует внимания (например, orphan campaigns, near-cap pacing).
- `alert` — критика (бюджет превышен, нет лидов 3+ дней).
- пусто — обычная.

### 1.2. Запасные пути рендера

`render_html.sh` имеет 3 fallback'а в порядке убывания качества:
1. **pandoc + наш template** (`autopilot/lib/templates/report.html`) — основной путь.
2. pandoc без template (standalone, дефолтная типографика) — если template-файла нет.
3. python -m markdown с минимальным CSS — если pandoc отсутствует.
4. Текстовый pre-wrap fallback — если ни pandoc, ни python-markdown нет.

Frontmatter работает только на путях 1-2; в python-markdown YAML просто отрисуется как обычный текст в начале (это ок, но KPI-strip недоступен — fallback показывает body without hero/cards).

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
