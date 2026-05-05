# Autopilot — инструкция для оператора

> Полное руководство по запуску нового города в leadgen-autopilot.
> Архитектура — `../PLAN.md`. Quick-start — `README.md`.

---

## 0. Что такое автопилот

`/leadgen-autopilot` — автономный агент. Запускается через Claude Desktop routine, ведёт рекламу города 24/7, шлёт отчёты в Telegram. На пилоте действует с **максимальной автономией** (`autonomy_mode: full_auto`, `trust_profile: pilot_full_auto`) — бот сам создаёт DRAFT-кампании, активирует их, оптимизирует ставки/бюджеты и площадки.

Гарантии безопасности:
- Hard-block actions нерушимы: `legal.*`, `account.*`, `campaign.delete` — не выполняются никогда.
- Бот трогает **только** кампании с label `autopilot:managed`.
- Идемпотентность: повторный run на тех же данных не дублирует действия.
- Drift detection: ручные правки специалиста не перетираются (24-72ч freeze).
- Rollback: команда `rollback <run_id>` через Telegram → dry-run → confirm.
- Pacing: hard cap на 120% MTD, emergency на 125% pacing_ratio.

---

## 1. Требования

### Софт

- Windows 10/11 с Claude Desktop.
- Git Bash (`C:\Program Files\Git\bin\bash.exe`) — обычно ставится с git.
- Python 3 (для schema validation; уже стоит, если работает MCP).
- Опционально: pandoc или `pip install markdown` (для красивых HTML-отчётов; без них работает trivial fallback).

### Доступы

- Yandex Direct API: рабочий MCP-сервер `leadgen-mcp` (см. `server/README.md`).
- Yandex Metrika: counter_id, OAuth-токен (через MCP).
- Telegram: bot token, chat_id для уведомлений.

### Машина 24/7

Claude Desktop должен быть открыт круглосуточно. Настройки Windows:
- Питание: **Никогда** не выключать дисплей / не уходить в сон (Settings → System → Power).
- Обновления: отложить автоматический рестарт на удобное время.
- Routine не зависает при блокировке экрана.

---

## 2. Подготовка Telegram

### 2.1. Создать бота

1. В Telegram пишем `@BotFather` → `/newbot`.
2. Имя: например `etagi_autopilot_bot`. Username должен оканчиваться на `bot`.
3. Получаем токен формата `123456789:ABCdefGhIJ...`.
4. Сохраняем токен — понадобится в `secrets.env`.

### 2.2. Создать чат для города

Один чат на каждый город. На пилоте — один чат для одного пилотного города.

1. Создать новую группу в Telegram: `Autopilot — <city>`.
2. Добавить бота в группу.
3. Дать боту админ-права (опционально, для возможности pin важных сообщений).
4. Узнать `chat_id`:
   - Способ 1: через бот `@userinfobot` или `@RawDataBot` — добавить в группу.
   - Способ 2: написать любое сообщение в группе → `https://api.telegram.org/bot<TOKEN>/getUpdates` → найти `chat.id` (для групп — отрицательное число формата `-1001234567890`).

### 2.3. Allowlist

В `secrets.env` укажем `TELEGRAM_ALLOWLIST_CHAT_IDS=-1001234567890` (через запятую, если городов несколько).

---

## 3. Настройка `secrets.env`

```bash
cd C:\git\leadgen-mcp\autopilot\config
cp secrets.env.example secrets.env
```

Открыть `secrets.env` в редакторе, заполнить:

```env
TELEGRAM_BOT_TOKEN=123456789:ABCdefGhIJ...
TELEGRAM_ALLOWLIST_CHAT_IDS=-1001234567890
TELEGRAM_HTTP_TIMEOUT=15
```

**Файл gitignored.** Никогда не коммитим.

---

## 4. Конфиг города `<city>.yaml`

```bash
cd C:\git\leadgen-mcp\autopilot\config\cities
cp _example.yaml <city>.yaml
```

Заполнить ключевые поля:

### 4.1. Идентификация

```yaml
city: omsk                              # latin slug, используется в путях
client_login: ethaji-omsk               # из MCP get_city_config / get_client
counter_id: 12345678                    # из Yandex Metrika
geo_region_id: 65                       # 65=Омск, 66=Екатеринбург, 213=Москва
domain: omsk.etagi.com
tier: 2                                 # 1/2/3 — для подбора city_benchmarks
timezone: Asia/Omsk                     # IANA TZ; влияет на freshness
```

Полный список city_login и counter_id — через MCP-команду `get_businesses` или из админки Etagi.

### 4.2. Autonomy и trust

```yaml
autonomy_mode: full_auto                # экспериментальный режим пилота
trust_profile: pilot_full_auto
```

Альтернативы (можно поменять в любой момент, изменения подхватятся следующим прогоном):
- `autonomy_mode: with_approvals` — все medium+ через Telegram approve.
- `autonomy_mode: read_only` — только наблюдение и план.
- `trust_profile: conservative` — большинство medium+ в review_queue.
- `trust_profile: balanced` — auto_with_notify для medium.
- `trust_profile: aggressive` — auto для medium.

### 4.3. Метрика и цели

```yaml
metrika:
  counter_id: 12345678
  goals:
    lead_form: { id: 100001, name: "Заявка форма", attribution: LYDC, value_type: leads }
    call: { id: 100002, name: "Звонок", attribution: LYDC, value_type: leads }
    qualified_lead: { id: 100004, name: "Квал.лид CRM", attribution: LYDC, value_type: qualified_leads }
  primary_conversion_goal: lead_form
  secondary_goals: [call]
  qualified_goal: qualified_lead         # для weekly/monthly
```

`id` целей — из Yandex Metrika → Цели. `attribution: LYDC` — последний значимый источник, стандарт проекта.

### 4.4. Тематики

```yaml
topics:
  vtorichka:
    status: active                       # candidate | experimental | active | paused | blocked
    allowed_channels: [search, rsya]
    monthly_budget: 60000                # ₽
    target_cpa_form: 3500
    target_cpa_call: 2800
    target_cpa_qualified: 12000
    notes: "Приоритет — поиск"
  novostroyki:
    status: active
    allowed_channels: [search, rsya]
    monthly_budget: 80000
    target_cpa_form: 5000
    target_cpa_call: 4000
    target_cpa_qualified: 15000
  zagorodka:
    status: candidate                    # анализ есть, запуска нет
    allowed_channels: [search, rsya]
    monthly_budget: 0
```

**Что значат статусы:**

| status | Что бот делает |
|---|---|
| `candidate` | Только анализ спроса, никаких действий |
| `experimental` | Создаёт DRAFT с уменьшенным бюджетом (×0.5), активация — review_queue даже в full_auto |
| `active` | Полный цикл: создать, активировать, оптимизировать |
| `paused` | Существующие campaigns ставит на паузу, новые не создаёт |
| `blocked` | Полный запрет; не трогает существующие, не предлагает |

**Валидные топики** — см. `.claude/skills/leadgen/references/site_structure.md`.

### 4.5. Бюджет

```yaml
budget:
  total_monthly_limit: 200000            # суммарный кап
  daily_pacing: linear                   # linear | front_loaded | back_loaded
  weekend_modifier: 0.8                  # ставки в выходные ×0.8
  reserve_pct_for_month_end: 10          # 10% на последние 5 дней
```

Если `sum(topics.monthly_budget) > total_monthly_limit` — `total_monthly_limit` побеждает.

### 4.6. Расписание

```yaml
schedule:
  daily_runs_per_day: 1                  # 1 | 2 (для 2: преф. часы [10, 22])
  preferred_hours_msk: [10]
  weekly_rollup_dow: monday
  monthly_rollup_dom: 1
```

Реальное triggering — через Claude Desktop routine (см. шаг 6). Эти поля — целевая частота.

### 4.7. Уведомления

```yaml
notify:
  telegram_chat_id: -1001234567890       # из шага 2.2
  daily_summary: true
  weekly_report: true
  monthly_report: true
  alert_thresholds:
    cpa_jump_pct: 50
    budget_overrun_pct: 110
    no_conversions_days: 3
    impressions_drop_pct: 50
```

### 4.8. Кастомные правила (опционально)

```yaml
custom_rules:
  - "Не повышать ставки на vtorichka выше 80₽ на клик"
  - "Не запускать рекламу новостроек до 1 числа месяца"
```

Свободный текст. Бот учитывает при анализе.

### 4.9. Holdout (опционально)

```yaml
holdout:
  enabled: false                         # для пилота с нуля выключено
  campaign_ids: []                       # если есть — пометит autopilot:holdout, не управляет
```

### 4.10. Holds — опционально, не обязательно

Заполнили? Сохраните файл. Можно опционально провалидировать схему:

```bash
python -c "import yaml; yaml.safe_load(open('autopilot/config/cities/<city>.yaml', encoding='utf-8'))"
```

---

## 5. Первый ручной запуск (smoke test)

Перед routine — запустите вручную, чтобы убедиться, что всё работает.

В Claude Code из корня репо:

```
/leadgen-autopilot city=<city>
```

Что произойдёт (на пилоте `full_auto` + `pilot_full_auto`):

1. Bot читает `cities/<city>.yaml`, проверяет HALT, берёт lock.
2. **Onboarding** (так как это первый запуск):
   - Inventory: проверяет Метрику, цели, существующие кампании (на новом аккаунте — пусто).
   - Demand analysis для активных тем.
   - Формирует `runtime/<city>/onboarding/launch_proposal.{md,yaml}`.
   - **Сразу** создаёт DRAFT-кампании для каждого `proposed_launch` (ownership labels `autopilot:managed`, `city:<city>`, `topic:<>`, `channel:<>`).
   - Дожидается модерации DRAFT.
   - **Активирует** кампании (`auto_with_notify` в pilot_full_auto).
3. Записывает `state.yaml`, narrative md, ledger.
4. Шлёт Telegram daily summary с ссылкой на launch_proposal.html и список созданных/активированных кампаний.
5. Releases lock, exit 0.

**Что проверить после первого запуска:**

```bash
# Operational state
cat autopilot/runtime/<city>/state.yaml
cat autopilot/runtime/<city>/action_ledger.jsonl | tail -10

# Narrative
cat autopilot/runtime/<city>/narrative/STATE.md
cat autopilot/runtime/<city>/narrative/runs/$(date +%Y-%m)/*.md | head -50

# Onboarding
cat autopilot/runtime/<city>/onboarding/launch_proposal.md

# Reports
ls autopilot/reports/<city>/$(date +%Y-%m)/

# Lock освобождён
test -f autopilot/runtime/<city>/locks/RUNNING.lock && echo "WARN: lock not released!" || echo "OK: lock released"
```

В Yandex Direct (UI): должны появиться кампании с лейблами `autopilot:managed`, `city:<>`, `topic:<>`, `channel:<>`. Они в статусе ON (активированы), но первые часы — обучение стратегии (показов мало).

В Telegram: пришло 1 общее summary + по одному `auto_with_notify` сообщению на каждое создание/активацию кампании.

---

## 6. Настройка автоматического запуска

После успешного ручного прогона — ставим автоматику. **Два варианта**, оба рабочих:

| Вариант | Когда выбирать |
|---|---|
| **6A. Claude Desktop Routines** | Простой single-city, операторы привыкли к Desktop UI. Минус: зависит от того, что Desktop запущен. |
| **6B. Claude Code CLI + планировщик** ⭐ | Production-режим, multi-city, нужны логи/observability/robustness. Не зависит от UI-сессии. |

Качество выполнения работы автопилотом **в обоих вариантах идентичное** (одна модель, тот же контекст, те же MCP/skills/hooks). Разница только в обвязке запуска.

### 6A. Claude Desktop routine

#### 6A.1. Создать routine

1. Claude Desktop → Settings → Routines (или аналог; зависит от версии).
2. Создать новую routine:
   - **Name:** `Autopilot — <city>`
   - **Working directory:** `C:\git\leadgen-mcp` (корень репо, **не** autopilot/)
   - **Schedule:** ежедневно в `<preferred_hours_msk>` (например 10:00 МСК = 07:00 UTC).
   - **Prompt:** `Запусти leadgen-autopilot для города <city>` или `/leadgen-autopilot city=<city>`

3. Если городов несколько — создать отдельную routine для каждого. Staggered times (omsk в 10:00, kemerovo в 10:30, ...).

#### 6A.2. Watchdog

Чтобы знать, если routine не отработала:
- Создать вторую routine на 18:00 МСК: `Если за сегодня нет run для <city>, прислать alert в Telegram`.
- Или добавить cron-проверку через scheduled-tasks MCP, если установлен.

### 6B. Claude Code CLI + Windows Task Scheduler ⭐

Headless-mode CLI работает идентично Desktop по качеству, но независим от UI и даёт structured logs. Рекомендуется для production.

#### 6B.1. Команда headless-запуска

```bash
claude -p "/leadgen-autopilot city=<city>" \
  --dangerously-skip-permissions \
  --output-format json \
  --max-turns 80 \
  --max-budget-usd 5
```

Что важно:
- **`-p`** (print mode) — запускает один промпт, выходит после завершения. Auto-discover скиллов (`.claude/skills/leadgen-autopilot/`), MCP (`.mcp.json`), hooks (`.claude/settings.json`).
- **НЕ используйте `--bare`** — он отключает auto-discovery, нам нужны skills и MCP.
- **`--dangerously-skip-permissions`** обязателен для unattended: без него `-p` зависнет на любом permission prompt (например, при первом вызове bash). Альтернатива — заполнить `permissions.allow` в `.claude/settings.json` со whitelist всех MCP tools и Bash, но это десятки строк.
- **`--output-format json`** — structured лог (`{run_id, total_cost_usd, num_turns, ...}`) для парсинга / observability.
- **`--max-turns N`** — кап на число итераций (защита от зацикливания).
- **`--max-budget-usd N`** — кап на стоимость одного прогона.

#### 6B.2. Batch-обёртка

Создать `C:\scripts\run-autopilot-<city>.bat`:

```batch
@echo off
REM Batch-обёртка для unattended-запуска leadgen-autopilot.
REM Логи: autopilot/runtime/_global/cli-runs.log

cd /d C:\git\leadgen-mcp

set LOG=C:\git\leadgen-mcp\autopilot\runtime\_global\cli-runs.log
if not exist "C:\git\leadgen-mcp\autopilot\runtime\_global" mkdir "C:\git\leadgen-mcp\autopilot\runtime\_global"

echo. >> "%LOG%"
echo === %DATE% %TIME% — start city=<city> === >> "%LOG%"

REM Optional: pre-flight check that MCP server is alive
curl -sf http://localhost:8080/sse -o NUL --max-time 5
if errorlevel 1 (
  echo [ERROR] MCP server localhost:8080 not reachable, aborting >> "%LOG%"
  exit /b 1
)

claude -p "/leadgen-autopilot city=<city>" ^
  --dangerously-skip-permissions ^
  --output-format json ^
  --max-turns 80 ^
  --max-budget-usd 5 ^
  >> "%LOG%" 2>&1

set RC=%errorlevel%
echo === %DATE% %TIME% — end rc=%RC% === >> "%LOG%"
exit /b %RC%
```

Заменить `<city>` на пилотный город (пример: `omsk`).

#### 6B.3. Windows Task Scheduler

1. `Win+R` → `taskschd.msc` → Create Task (не Create Basic Task — нужны полные настройки).
2. **General:**
   - Name: `Autopilot — <city>`
   - "Run whether user is logged on or not" ✓
   - "Hidden" ✓ (без всплывающего окна)
   - "Configure for: Windows 10/11"
3. **Triggers:** New → Daily → 10:00 (МСК), Recur every 1 day.
4. **Actions:** New → Start a program:
   - Program: `C:\scripts\run-autopilot-<city>.bat`
   - Start in: `C:\git\leadgen-mcp` (это важно — иначе bash скрипты могут не найти относительные пути)
5. **Settings:**
   - "Allow task to be run on demand" ✓
   - "Run task as soon as possible after a scheduled start is missed" ✓
   - "If the running task does not end when requested, force it to stop" ✓
   - "If the task is already running, then the following rule applies: **Do not start a new instance**" ✓ (двойная защита поверх нашего `RUNNING.lock`)

При создании задачи Windows запросит пароль учётки — нужно ввести (для запуска без login session).

#### 6B.4. Альтернативные планировщики (с UI)

Windows Task Scheduler работает, но UI древний и неудобный для нескольких задач. Альтернативы:

| Сервис | Плюсы | Минусы |
|---|---|---|
| **[Cronicle](https://github.com/jhuckaby/Cronicle)** ⭐ | Open-source, web UI на `localhost:3012`, dashboard со всеми задачами, history, retry policy, email/Slack уведомления, plugin system, multi-server scaling. Работает на Node.js. | Нужно развернуть Node.js |
| **[System Scheduler (Splinterware)](https://www.splinterware.com/products/scheduler.html)** | Бесплатный для personal use, простой Windows UI, лог-консоль на каждую задачу. | Только Windows, нет web UI |
| **[Z-Cron](https://www.z-cron.com/)** | Бесплатный, классический Windows UI, cron-syntax. | UI устаревший |
| **[VisualCron](https://www.visualcron.com/)** | Мощный workflow engine, free trial, корпоративный уровень. | Платный после trial |
| **[n8n](https://n8n.io/) (self-hosted)** | Low-code, web UI, можно строить целые pipeline вокруг прогона (pre-checks → запуск → post-processing). | Overkill для одной задачи |
| **WSL + cron** | Линуксовский cron, минималистичный, надёжный. | Нет UI, нужен WSL |

**Рекомендация для пилота на 1 город:** Windows Task Scheduler (встроенный) или System Scheduler (если хочется UI попроще).
**Для масштабирования на 5+ городов:** Cronicle — даёт dashboard со всеми job-ами, history по каждому прогону, легко добавлять/удалять города.

#### 6B.5. Мини-инструкция: Cronicle на Windows

```powershell
# 1. Установить Node.js LTS https://nodejs.org/
# 2. Установить Cronicle:
npm install -g cronicle
cronicle --setup
cronicle start

# 3. Открыть http://localhost:3012 (login admin/admin при первом входе, сменить пароль).
# 4. Schedule → Add Event:
#    Title: Autopilot — omsk
#    Category: Production
#    Plugin: Shell Script
#    Schedule: 0 7 * * * (UTC, 10:00 МСК)
#    Script: C:\scripts\run-autopilot-omsk.bat
#    Resource Limits: CPU/Memory caps (опционально)
#    Notifications: email/Slack/Telegram при failure
```

Cronicle сохраняет stdout/stderr каждого run-а в свою БД, показывает в UI, поддерживает retry policy и chained jobs.

#### 6B.6. Watchdog для unattended

Task Scheduler / Cronicle сами отслеживают exit code. Дополнительная страховка через Telegram:

Создать вторую задачу на 18:00:
```batch
@echo off
REM Watchdog: проверить, был ли успешный run за последние 26 часов.
REM Если нет — отправить alert.
cd /d C:\git\leadgen-mcp
powershell -NoProfile -Command "
  $log = 'autopilot\runtime\_global\cli-runs.log';
  $cutoff = (Get-Date).AddHours(-26);
  $lastSuccess = Select-String -Path $log -Pattern 'rc=0' -SimpleMatch | Select-Object -Last 1;
  if ($null -eq $lastSuccess -or (Get-Item $log).LastWriteTime -lt $cutoff) {
    'No successful autopilot run in 26h' | bash autopilot/lib/telegram_send.sh -1003957360112 -;
  }
"
```

---

## 7. Управление автопилотом

### 7.1. Глобальный стоп

```bash
echo "manual stop $(date)" > autopilot/HALT.flag
```

Все routine следующего цикла прочитают HALT, отправят alert и выйдут.

Снять:
```bash
rm autopilot/HALT.flag
```

### 7.2. Стоп одного города

```bash
echo "test pause" > autopilot/runtime/<city>/HALT.flag
```

Снять: `rm` тот же файл.

### 7.3. Поменять режим автономии

Открыть `autopilot/config/cities/<city>.yaml`, изменить `autonomy_mode`:
- `full_auto` → бот сам всё делает.
- `with_approvals` → medium+ через Telegram approve.
- `read_only` → ничего не применяет, только plan + report.

Изменения подхватятся следующим routine-прогоном.

### 7.4. Поменять trust profile

Аналогично — `trust_profile` в `cities/<city>.yaml`.

### 7.5. Добавить тематику

```yaml
topics:
  zagorodka:
    status: active                       # было candidate
    allowed_channels: [search, rsya]
    monthly_budget: 30000
    target_cpa_form: 6000
```

Следующий run заметит дельту (reconcile_config) → создаст DRAFT → активирует (в full_auto).

### 7.6. Поставить тематику на паузу

```yaml
topics:
  novostroyki:
    status: paused
```

Следующий run поставит все managed campaigns этой темы на паузу.

### 7.7. Telegram-команды (только при `autonomy_mode: with_approvals`)

| Команда | Эффект |
|---|---|
| `approve <id>` | Одобрить pending action. Применится в следующем run. |
| `reject <id>` | Отклонить навсегда. |
| `defer <id> 3d` | Отложить на 3 дня (бот пересчитает evidence к моменту истечения). |
| `rollback <run_id>` | Запросить dry-run отката всего прогона. Бот пришлёт обратный план. |
| `confirm rollback <id>` | Подтвердить откат. Бот применит обратные действия. |

В `full_auto` команды `approve/reject/defer` бесполезны (нет pending), но `rollback`/`confirm rollback` работают.

### 7.8. Откатить кампанию вручную в UI Direct

Если поменяли ставку/бюджет/статус **в Yandex Direct UI** — следующий run обнаружит drift через `get_change_history` → пометит entity как `human_override` на 72 часа → не вернёт обратно. В отчёте увидите: `⚠ human_override on campaign <id> until <date>`.

### 7.9. Управление holdout

В `cities/<city>.yaml` добавить:
```yaml
holdout:
  enabled: true
  campaign_ids: [8765432]
```

Следующий run пометит campaign 8765432 label `autopilot:holdout`, перестанет управлять, в monthly будет сравнивать managed vs holdout.

---

## 8. Чтение отчётов

### 8.1. Telegram daily summary

```
🟢 [omsk] 04 May 10:30 · full_auto
Бюджет день: 8 420 / 10 000 ₽ (84%)
Лиды: 12 · CPA 702 ₽ (цель 800)
Действия: 4 (auto: 3, auto_with_notify: 1, skipped: 0)
  · vtorichka-search: +12 минусов, +bid 10%
  · vtorichka-rsya: +34 заблоч.площадки
⚠ novostroyki: CPA 1 450, день 7 обучения
[detail.html прикреплён]
```

Эмодзи:
- 🟢 normal pacing, всё в порядке.
- 🟡 conservation pacing OR medium-severity сигналы.
- 🔴 emergency OR high-severity drift / failed actions.
- 🛑 HALT.

### 8.2. HTML отчёты

`autopilot/reports/<city>/<YYYY-MM>/`:
- `<DD-HHMM>-daily.html` — каждый прогон.
- `week-<NN>.html` — еженедельно.
- `month-<MM>.html` — ежемесячно.

Открыть в браузере. Если вид простой (preformatted block) — установите `pandoc` или `pip install markdown` для красивого рендера.

### 8.3. Operational data

| Файл | Что внутри |
|---|---|
| `runtime/<city>/state.yaml` | Текущее состояние (бюджет, тематики, кампании, последний run) |
| `runtime/<city>/action_ledger.jsonl` | Append-only журнал всех action-попыток |
| `runtime/<city>/metrics_snapshots.jsonl` | Ежедневные срезы метрик |
| `runtime/<city>/pending_approvals.yaml` | Очередь approvals (только в `with_approvals`) |
| `runtime/<city>/before_snapshots/<id>.json` | Snapshots для rollback (medium+ actions) |

### 8.4. Narrative (для агента и для чтения человеком)

| Файл | Что внутри |
|---|---|
| `runtime/<city>/narrative/STATE.md` | Текущее состояние (читаемая версия state.yaml) |
| `runtime/<city>/narrative/CURSOR.md` | План: сделано / отложено / pending |
| `runtime/<city>/narrative/SUMMARY.md` | Хронология 30/90/365 дней |
| `runtime/<city>/narrative/runs/<YYYY-MM>/<HHMM>.md` | Полный лог одного прогона |
| `runtime/<city>/narrative/campaigns/<id>.md` | История изменений по кампании |
| `runtime/<city>/narrative/decisions/<topic>-<slug>.md` | Нестандартные кейсы / эксперименты |

### 8.5. Learnings

| Файл | Что внутри |
|---|---|
| `learnings/proposed/<id>.md` | Гипотезы, накопленные ботом |
| `learnings/validated/<id>.md` | Подтверждённые (≥3 повтора, 14 дней, нет отката) |
| `learnings/rejected/<id>.md` | Отклонённые специалистом |

В monthly digest `validated` идут как **предложения** для PR в `.claude/skills/leadgen/references/lessons_registry.md`. **Бот сам ничего не пишет в скилл.**

---

## 9. Чек-лист troubleshooting

| Симптом | Что проверить |
|---|---|
| Бот не запустился | `cat autopilot/HALT.flag` — есть глобальный halt? `cat autopilot/runtime/<city>/HALT.flag` — per-city halt? |
| "active lock" ошибка | `bash autopilot/lib/lock.sh inspect <city>` — есть ли активный lock от другого процесса? Если процесс зависший: `bash autopilot/lib/lock.sh release <city>` |
| Telegram не приходит | Проверить `secrets.env`, токен, allowlist. Тест: `echo "test" \| bash autopilot/lib/telegram_send.sh -1001234567890 -` |
| Бот не трогает кампанию | У неё есть label `autopilot:managed`? Проверить через MCP `get_campaigns` с filter по labels. Если нет — adoption не выполнялось. |
| Drift alert | Смотреть в `runs/<id>.md` секция «Сигналы». `get_change_history` покажет ручные правки. Если не было — это unexplained drift, разбираться. |
| Pacing emergency | Смотреть `state.yaml.budget.{spent_mtd,pacing_ratio,forecast_month_end,pacing_state}`. Возможно, плохо настроены `target_cpa_*` или `monthly_budget`. |
| Кампания не активировалась | Проверить `state.campaigns[<id>].status_status` — может быть на модерации (DRAFT/MODERATION). Активация происходит, только когда `ACCEPTED`. |
| HTML отчёт без форматирования | Установить `pandoc` (Windows: choco/scoop) или `pip install markdown`. |
| **CLI зависает на permission prompt** | В Task Scheduler `.bat` забыт `--dangerously-skip-permissions`. Без него `claude -p` ждёт ввода stdin вечно. |
| **Task Scheduler exit code 0, но run не отработал** | Открыть `autopilot/runtime/_global/cli-runs.log` — там stdout от `--output-format json`. Возможно MCP server упал (`curl localhost:8080/sse`). |
| **Двойной запуск через CLI** | Task Scheduler настройка "Do not start a new instance" + наш `RUNNING.lock` дают двойную защиту. Если оба обходятся — проверить TTL lock в `caps_defaults.yaml.running_lock_ttl_minutes`. |
| **CLI не видит скилл `/leadgen-autopilot`** | Запуск из неправильного cwd. В Task Scheduler "Start in:" должен быть `C:\git\leadgen-mcp`. Проверка: `cd C:\git\leadgen-mcp && claude -p "list available skills"`. |
| **MCP не подключается** | Pre-flight check в `.bat`: `curl -sf http://localhost:8080/sse`. Если падает — Docker container `leadgen-mcp` не запущен или нет network. |

### 9.1. Логи

**Все шаги прогона** — в `runtime/<city>/narrative/runs/<YYYY-MM>/<HHMM>.md`. Phases lifecycle: `started → loaded_context → fetched_metrics → planned_actions → applying → applied_partial → memory_written → notified → succeeded`.

Если phase застрял в `applying` — был crash. Recovery — в следующем прогоне через ledger.

**При CLI-режиме** дополнительные логи:
- `autopilot/runtime/_global/cli-runs.log` — stdout/stderr Task Scheduler (включая `--output-format json` ответ).
- `~/.claude/projects/<hashed-path>/transcripts/` — полные транскрипты сессий Claude Code (если не отключено `--no-session-persistence`).
- Cronicle UI (если используется) — таб History с filter по job-у, retention 30 дней по умолчанию.

---

## 10. Откат / экстренная остановка

### 10.1. Откатить один прогон

В Telegram:
```
rollback omsk-2026-05-04-1030
```

Бот пришлёт dry-run plan обратных действий. Если устраивает:
```
confirm rollback omsk-2026-05-04-1030
```

Необратимые actions (creative_generation, после долгого обучения стратегии) помечены `rollback: manual_only` — бот покажет, что нужно сделать руками.

### 10.2. Полный stop + retreat

1. `echo "incident $(date)" > autopilot/HALT.flag` — глобальная остановка.
2. В UI Direct: вручную поставить на паузу проблемные кампании.
3. После выяснения причины: убрать HALT.flag, изменить `autonomy_mode: read_only` для дальнейшего наблюдения без действий.

### 10.3. Полный wipe (только в крайнем случае)

```bash
# Освободить кампании от ownership
# (bot не сможет ими управлять, но они продолжат работать)
# В UI Direct: убрать labels autopilot:managed.
# ИЛИ через MCP в ручном режиме.

# Удалить runtime данные
rm -rf autopilot/runtime/<city>
rm -rf autopilot/reports/<city>
```

При следующем запуске бот переинициализирует state с пустого листа (re-onboarding).

---

## 11. Масштабирование на N городов

Когда пилотный город отработал 2-4 недели без инцидентов:

1. **Создать profile для зрелых городов:** в `cities/<city2>.yaml`:
   ```yaml
   trust_profile: balanced       # или aggressive после ещё 4 недель
   autonomy_mode: full_auto      # или with_approvals для безопасности
   ```
2. **Новый routine** в Claude Desktop: staggered start (omsk 10:00, kemerovo 10:30, ekat 11:00, ...).
3. **Per-city Telegram chat** (`telegram_chat_id` уникальный для каждого города).
4. Проверить scale-метрики (decision precision, rollback rate) в monthly после 30 дней.

---

## 12. Поддержка и обновления

- Архитектура: `PLAN.md`.
- Бизнес-логика рекламы (валидные топики, бенчмарки): `.claude/skills/leadgen/references/`.
- Action catalog: `.claude/skills/leadgen-autopilot/references/action_catalog.md`.
- Signal catalog: `.claude/skills/leadgen-autopilot/references/signal_catalog.md`.
- Trust profiles: `autopilot/config/trust_profiles/*.yaml`.
- Caps: `autopilot/config/caps_defaults.yaml`.

Изменения в скилле — через PR в репо. Изменения в city config — точечная правка yaml, подхватится следующим прогоном.

Журнал архитектурных решений — `RECENT-CHANGES.md` в корне.

---

**Удачного запуска!** Если что-то идёт не по плану — начните с раздела 9 (troubleshooting), затем 10 (откат). Любые архитектурные сомнения — `PLAN.md`.
