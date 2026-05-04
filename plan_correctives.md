# Корректировки к PLAN.md по leadgen-autopilot

Источник анализа: `C:/Users/animo/OneDrive/Desktop/PLAN.md`, драфт ТЗ `Autopilot — План разработки автономного агента leadgen`.

## Короткий вывод

План в целом правильно задаёт продуктовую идею: отдельный `/leadgen-autopilot`, config-driven управление городом, слой памяти, read-only пилот, caps/permissions, отчётность и постепенный рост автономности. Это хорошая рамка для пилота на 1-2 города.

Главная слабость плана: он описывает автономного агента в основном как набор markdown-инструкций и файлов памяти, но недостаточно фиксирует машинно-проверяемые контракты, идемпотентность, блокировки, источники правды, rollback и критерии принятия решений. Для ручного скилла это терпимо. Для 24/7-автопилота по рекламному бюджету это станет источником дорогих ошибок.

Ниже корректировки, которые стоит внести в итоговое ТЗ до разработки полного варианта и финального end-to-end теста.

---

## 1. Критичные корректировки архитектуры

### 1.1. Убрать противоречие «параллельный скилл, но использует branches leadgen»

В плане одновременно сказано:

- `leadgen-autopilot` параллельный и не оркестратор поверх `leadgen`;
- при этом он читает `leadgen/branches/{create-search,create-rsya,optimize-*}.md` как playbook.

Это фактически зависимость от ручного скилла. Она допустима, но её нужно назвать честно и оформить как контракт.

**Рекомендуемое решение:**

- `leadgen-autopilot` = policy/orchestration layer: анализ, память, caps, permissions, reconcile, отчёты.
- `leadgen` branches = shared operation playbooks: конкретные шаги создания/оптимизации.
- Ввести файл-контракт `leadgen-autopilot/references/playbook_contract.md`: какие playbooks можно читать, какие anchors стабильны, какие входы/выходы ожидаются, что делать при изменении anchor-структуры.
- Лучше в перспективе вынести общие playbooks в shared-слой, чтобы `leadgen` и `leadgen-autopilot` читали один источник, а не чтобы автопилот зависел от внутренностей ручного скилла.

### 1.2. Не оставлять `Codex-зеркало` просто «вне скоупа» без явной оговорки

В проектных инструкциях есть обязательное двойное дерево `.claude/skills/leadgen/` ↔ `.codex/skills/leadgen-codex/` для правок флоу/референсов/библиотек. План говорит, что Codex-зеркало автопилота вне скоупа.

Это можно оставить, но нужно явно оформить исключение:

- автопилот на пилоте `Claude-only`;
- в `.claude/skills/leadgen-autopilot/CLAUDE.md` и в `RECENT-CHANGES.md` зафиксировать причину;
- любые изменения shared-файлов `leadgen/references`, `leadgen/library`, `leadgen/branches` всё равно синхронизировать в Codex-зеркало;
- автопилот не должен править общие playbooks без обновления Codex-ветки.

### 1.3. Markdown-память нужна, но нужен машинный ledger

`STATE.md`, `CURSOR.md`, `SUMMARY.md`, `runs/*.md` удобны для чтения человеком и агентом. Но для автономной работы этого мало: markdown сложно валидировать, сложно безопасно обновлять, сложно проверять идемпотентность.

**Добавить рядом с markdown:**

- `STATE.json` или `state.yaml` как машинный источник текущего состояния.
- `action_ledger.jsonl` как неизменяемый журнал действий.
- `metrics_snapshots.jsonl` для ежедневных срезов показателей.
- `pending_approvals.yaml` вместо или рядом с `pending_approvals.md`.
- JSON Schema/YAML schema для city config, state, action ledger, approval queue.

Markdown можно генерировать из этих структур или использовать как human-readable summary. Источник правды для apply-фазы должен быть структурным.

### 1.4. Разделить versioned config и runtime data

План кладёт `autopilot/memory`, `reports`, `secrets.env` и learnings в репозиторий. Для пилота это удобно, но создаст шум в git, риск утечки токенов и тяжёлые отчёты при масштабировании.

**Рекомендуемое разделение:**

- В репозитории версионировать: `autopilot/README.md`, `CLAUDE.md`, `config/caps_defaults.yaml`, `config/cities/_example.yaml`, schemas, templates, scripts.
- Runtime хранить в `autopilot/runtime/` или `autopilot/data/`, добавить в `.gitignore` по умолчанию.
- Для переносимой памяти можно отдельно делать snapshot/export, но не смешивать постоянные runtime-логи с кодовой базой.
- `secrets.env` обязательно в `.gitignore`; в README добавить проверку, что токены не попали в git.

### 1.5. Рабочая директория для Claude Code routine — корень `leadgen-mcp`

Для routine в Claude Code рабочей папкой нужно указывать общий корень проекта:

```text
C:\git\leadgen-mcp
```

Не папку `autopilot/`. Причины:

- автопилоту нужны shared playbooks и references из `.claude/skills/leadgen/`;
- MCP-сервер, конфиги, документация и git-контекст живут на уровне всего репозитория;
- runtime-данные автопилота — только часть проекта, а не самостоятельный проект;
- из корня проще держать единые относительные пути: `autopilot/config/cities/<city>.yaml`, `autopilot/runtime/<city>/...`, `.claude/skills/...`.

Prompt routine должен задавать город явно, например:

```text
/leadgen-autopilot city=omsk
```

или:

```text
Запусти leadgen-autopilot для города omsk. Работай из config autopilot/config/cities/omsk.yaml.
```

---

## 2. Идемпотентность, блокировки и crash recovery

### 2.1. Текущий `correlation_key` не идемпотентен

План предлагает `correlation_key = <city>-<run_id>-<action_slug>`. При повторном запуске после падения будет новый `run_id`, значит действие может выполниться второй раз.

**Нужен deterministic idempotency key:**

```text
<city>|<account>|<entity_type>|<entity_id>|<action_type>|<desired_value>|<decision_window>
```

Примеры:

- `omsk|ethaji-omsk|campaign|8765432|budget.set|65000|2026-05`
- `omsk|ethaji-omsk|keyword|123|pause.low_performance|true|2026-W18`
- `omsk|ethaji-omsk|placement|campaign876|block|example.com|2026-04-30`

Перед apply проверять не только `get_change_history`, но и локальный `action_ledger.jsonl`.

### 2.2. Нужен per-city lock

Claude Desktop routine может запуститься повторно, подвиснуть, пересечься с weekly/monthly, либо пользователь может вручную запустить тот же город.

Добавить:

- `autopilot/runtime/<city>/RUNNING.lock`;
- TTL lock, например 2 часа;
- запись `run_id`, `started_at`, `pid/session`, текущая фаза;
- если lock свежий — новый запуск выходит с уведомлением;
- если lock протух — запуск пишет recovery-событие и продолжает только после сверки ledger/state/API.

### 2.3. Run должен иметь статусы фаз

Сейчас `runs/<id>.md` описан как полный лог. Нужен структурный жизненный цикл:

- `started`;
- `loaded_context`;
- `fetched_metrics`;
- `planned_actions`;
- `approval_checked`;
- `applying`;
- `applied_partial`;
- `memory_written`;
- `notified`;
- `succeeded` / `failed` / `halted`.

Если агент упал между apply и memory_write, следующий прогон должен понять, что API уже изменён, а память отстала.

### 2.4. Запись файлов должна быть атомарной

Для `STATE`, `CURSOR`, `SUMMARY`, `pending_approvals`:

- писать во временный файл;
- валидировать;
- делать atomic rename;
- хранить `.bak` предыдущей версии;
- при битом файле восстанавливаться из `.bak` и API.

---

## 3. Источник правды и защита от конфликта с человеком

### 3.1. API должен быть источником факта, STATE — кешем

В reconcile нужно явно прописать: если `STATE` расходится с Direct/Metrika, факт из API важнее. STATE обновляется после диагностики drift.

Добавить drift-режим:

- `STATE says active`, API says paused → проверить `get_change_history`;
- если пауза сделана человеком, записать `human_override`, включить cooldown на N часов;
- если причина неизвестна, не откатывать автоматически, отправить alert.

### 3.2. Управлять только помеченными кампаниями

Автопилот не должен трогать все кампании аккаунта по умолчанию.

Добавить ownership-модель:

- управляемые кампании имеют label, naming marker или tracking marker: например `autopilot:managed`, `city:omsk`, `topic:vtorichka`;
- ручные кампании без метки read-only;
- переход существующей кампании под управление автопилота — отдельное действие `campaign.adopt_existing`, default `review_queue`;
- снятие с управления — `campaign.release`, default `review_queue`.

Это особенно важно при масштабировании и при параллельной работе специалиста.

### 3.3. После human changes нужен cooldown

Если специалист вручную поменял ставку/бюджет/статус кампании, автопилот не должен в тот же день «исправить» это обратно по своему STATE.

Добавить правило:

- human change detected → freeze affected entity на 24-72 часа;
- записать в `CURSOR`;
- в отчёте показать, что автопилот не вмешивался из-за ручного изменения.

---

## 4. Бюджетная безопасность

### 4.1. Hard stop на 120% MTD слишком поздний

`spent_mtd > budget * 120%` уже означает существенный перерасход. Нужна не только месячная отсечка, но и pacing-контроль.

Добавить расчёты:

- `expected_spend_to_date = monthly_budget * elapsed_month_share`;
- `pacing_ratio = spent_mtd / expected_spend_to_date`;
- `forecast_month_end_spend = avg_daily_spend_recent * days_in_month`;
- отдельные лимиты по topic/channel/campaign;
- режим conservation при `pacing_ratio > 1.1`;
- emergency при `pacing_ratio > 1.25` или `forecast > monthly_budget * 1.2`.

### 4.2. Развести budget cap и actual account spend

`topics.monthly_budget` и `budget.total_monthly_limit` — план. Фактические расходы могут идти по кампаниям вне управления автопилота. Нужно считать:

- spend managed campaigns;
- spend unmanaged campaigns;
- total account spend;
- долю управляемого расхода.

Иначе автопилот может принять неверное решение по бюджету города.

### 4.3. DRAFT-only для новых кампаний

В корневых правилах проекта зафиксировано: новые кампании после `add_campaign` остаются DRAFT/SUSPENDED, активирует пользователь.

В плане `campaign.create.in_new_topic` и `campaign.create.in_existing_topic` стоят как `auto_with_notify`. Это допустимо только если создание означает **создать DRAFT/SUSPENDED без старта показов**.

Добавить отдельные действия:

- `campaign.create_draft.in_existing_topic`;
- `campaign.create_draft.in_new_topic`;
- `campaign.activate_existing_draft`.

Для пилота:

- create draft: `auto_with_notify` или `review_queue`;
- activation: `review_queue` или `block`, пока специалист явно не разрешит.

---

## 5. Контрольные точки: daily/weekly/monthly

План говорит: daily всегда выполняется, weekly/monthly в дополнение. В исходной постановке было: месячное подведение итогов и планирование следующего месяца **без ежедневной и недельной рутины**.

Нужно выбрать явно.

**Рекомендация:** ввести приоритет циклов:

- обычный день: `daily`;
- weekly day: `daily_light_safety_check` + `weekly`;
- monthly day: `daily_light_safety_check` + `monthly`, weekly либо пропускается, либо включается только если monthly не совпал с weekly;
- emergency checks выполняются всегда: overrun, HALT, pending approvals, critical API drift.

Так автопилот не будет в один запуск менять тактику daily, затем weekly, затем monthly с противоречащими решениями.

---

## 5A. Паттерн запуска нового города

Бот не должен сам «с нуля» решать, какие направления запускать на реальные деньги. Правильная граница автономии: бот может сам анализировать аккаунт, спрос, сайт, Метрику и текущие кампании, но запускать только то, что разрешено конфигом города и уровнями доверия.

### Рекомендуемый onboarding-паттерн

**Шаг 1. Минимальный city config от специалиста**

Специалист создаёт `autopilot/config/cities/<city>.yaml` с обязательным минимумом:

- `city`, `client_login`, `counter_id`, `geo_region_id`, `domain`;
- месячный бюджет или бюджетный потолок;
- список разрешённых тематик, хотя бы в статусе `candidate`;
- целевые CPA/ориентиры или пометка `use_benchmark`;
- `trust_profile`;
- Telegram chat.

**Шаг 2. Baseline scan**

Первый запуск нового города работает без apply:

- инвентаризация существующих кампаний;
- проверка Метрики, целей, UTM, связки Direct↔Metrika;
- анализ текущих расходов/лидов/CPA;
- проверка сайта/посадочных;
- анализ спроса по разрешённым или candidate-тематикам;
- поиск дыр: тематики есть в спросе, но нет кампаний; кампании есть, но нет конверсий; есть расход вне управляемого контура.

**Шаг 3. Launch proposal**

Бот формирует `autopilot/runtime/<city>/onboarding/launch_proposal.md` и структурный `launch_proposal.yaml`:

- какие тематики рекомендует запускать;
- какие каналы: search/rsya;
- какой бюджет и target CPA;
- какие кампании взять под управление;
- какие кампании оставить read-only;
- какие риски и зависимости.

**Шаг 4. Human approval / config update**

Специалист утверждает план одним из двух способов:

- редактирует `city.yaml`: переводит нужные темы в `enabled: true`;
- либо отвечает в Telegram/Claude, после чего бот создаёт draft config diff, но не меняет бизнес-решения молча.

**Шаг 5. Draft-only creation**

После approval бот создаёт только draft/suspended кампании. Активация показов — отдельное действие `campaign.activate_existing_draft`, по baseline `review_queue`.

### Статусы тематик

Вместо простого `enabled: true/false` лучше использовать:

```yaml
topics:
  vtorichka:
    status: active        # candidate | active | paused | blocked
    allowed_channels: [search, rsya]
```

- `candidate` — можно анализировать и предлагать, нельзя запускать.
- `active` — можно создавать draft/управлять в рамках permissions.
- `paused` — не создавать новое, существующее можно поддерживать/останавливать по правилам.
- `blocked` — не трогать и не предлагать без явного изменения конфига.

Так бот остаётся автономным в анализе, но не становится самовольным в бизнес-решениях.

---

## 6. Permissions: индивидуальные уровни доверия по городу

Уровни доверия не должны быть глобальным поведением для всех аккаунтов. Правильная модель: базовые значения живут в `autopilot/config/caps_defaults.yaml`, а каждый город переопределяет их в `autopilot/config/cities/<city>.yaml`.

Иерархия должна быть такой:

- `caps_defaults.yaml` — безопасный baseline и полный список permissions;
- `city.yaml.permissions` — индивидуальные значения конкретного города/аккаунта;
- `city.yaml.topics.<topic>.permissions` — опционально, если одна тематика рискованнее другой;
- `runtime override` через approval — временное разрешение на конкретное действие с expiry.

Если в city config поле `null` или отсутствует, используется baseline из `caps_defaults.yaml`. Это соответствует исходному ТЗ: базовые значения можно взять из таблицы, но город имеет право быть строже или свободнее.

### 6.1. Базовый baseline сделать консервативным

Мои рекомендации ниже относятся именно к базовым дефолтам, а не к обязательному поведению каждого города. Для города с высокой степенью доверия можно переопределить в `city.yaml`.

Рекомендованные изменения baseline:

- `campaign.create.in_new_topic` → `review_queue` или create draft only.
- `campaign.archive.no_traffic_14days/30days` → `review_queue`.
- `campaign.resume` → `review_queue` по baseline; для зрелого города можно `auto_with_notify`.
- `adgroup.split_by_intent` → `review_queue` (в таблице сейчас конфликт: рекомендую `review_queue`, дефолт `auto`).
- `ad.update_text` → `review_queue` или `auto_with_notify`, но не `auto`, пока нет legal/content validation.
- `keyword.add_new_group_in_existing_topic` → `auto_with_notify` или `review_queue`.
- `placement.unblock` → `review_queue`, не `auto`.
- `audience.create` и `retargeting.create_list` → `review_queue`.
- `strategy.change_type` → `review_queue`.
- `creative.generate_new_image` → `review_queue` до стабильной проверки OpenRouter, юридики и качества.

### 6.2. Разделить action risk и permission

Сейчас permission — это только уровень доверия. Нужен ещё risk class:

- `low`: минусовка явного мусора, применение blacklist, read-only отчёты;
- `medium`: ставки/target CPA в пределах cap, новые объявления;
- `high`: pause/resume, новые группы, аудитории, изменение стратегии;
- `critical`: activation, billing, account settings, legal/company info.

Для каждой risk class задать минимальный уровень доказательств, cooldown и rollback policy.

### 6.3. В city config добавить профиль доверия

Чтобы не заполнять десятки permissions вручную для каждого города, добавить поле:

```yaml
trust_profile: conservative | balanced | aggressive | custom
```

- `conservative` — почти всё high-risk через `review_queue`.
- `balanced` — low/medium можно auto, high через notify/review.
- `aggressive` — для зрелых городов после стабильной работы.
- `custom` — все важные permissions явно указаны в `city.yaml`.

При этом итоговый resolved config должен записываться в run log: какие значения пришли из defaults, какие из city override.

---

## 7. Метрики и принятие решений

### 7.1. Зафиксировать атрибуцию и data freshness

В проектных правилах отчёты должны использовать `LYDC`. В плане это не закреплено.

Добавить в W3/analyze:

- все Metrika/Direct отчёты строить с `LYDC`;
- цели и счётчики писать человекочитаемо, не голыми ID;
- фиксировать timezone аккаунта и города;
- проверять задержку данных: не принимать сильные решения по сегодняшним неполным данным;
- для daily решений использовать yesterday / rolling 3-7 days, а today — только для emergency.

### 7.2. Одних глобальных min_clicks/min_conversions недостаточно

`min_clicks_for_decision: 50`, `min_conversions_for_decision: 5` слишком грубые. Для разных действий нужна разная доказательная база.

Примеры:

- блокировка площадки РСЯ: можно по spend/clicks без конверсий, но с учётом темы и периода;
- target CPA change: нужны конверсии и период обучения;
- pause keyword: нужны клики/расход/позиция/интент;
- ad text pause: нужны impressions + CTR + downstream conversion, а не только CTR.

Добавить в `action_catalog.md` поля:

```yaml
min_evidence:
  clicks: 50
  conversions: 5
  spend_rub: 3000
  days_since_last_change: 3
  data_window_days: 7
  compare_to: target | peer_group | previous_period
```

### 7.3. Нужен confidence score и reason codes

Каждое действие в плане должно иметь:

- signal_id;
- reason_code;
- confidence: low/medium/high;
- expected_effect;
- primary metric;
- guard metric;
- rollback trigger.

Это сильно упростит ревью специалистом и будущую авто-валидацию learnings.

---

## 8. Rollback: текущая модель слишком оптимистична

Не все действия обратимы простой формулой:

- генерация креативов не откатывает модерацию и историю;
- минус-слова можно удалить, но эффект мог уже повлиять на обучение;
- pause/resume меняет обучение стратегии;
- создание кампании не равно удалению;
- изменение текстов/ссылок требует сохранения предыдущего состояния.

**Рекомендуемое правило:**

- перед каждым apply сохранять `before_snapshot` структурно;
- `rollback_plan` должен быть dry-run сначала;
- `rollback <run_id>` через Telegram не должен сразу выполнять критичные откаты. Сначала: показать список обратных действий и ждать `confirm rollback <id>`;
- для необратимых действий писать `rollback: manual_only`.

---

## 9. Telegram и уведомления

### 9.1. `curl` подходит для пилота, но нужны технические защиты

Добавить в W1/W2:

- escaping HTML/Markdown в сообщениях;
- лимит длины Telegram-сообщения, fallback на файл;
- retry/backoff при сетевой ошибке;
- хранение `update_offset` для `getUpdates`, иначе ответы могут примениться повторно;
- allowlist chat_id;
- маскирование токена в логах;
- отдельное alert-сообщение при падении notify.

### 9.2. Approval polling нужно делать до анализа и после отчёта

В начале прогона:

- забрать approve/reject/defer;
- обновить queue;
- применить только те approvals, срок действия которых не истёк и контекст не изменился.

После отчёта:

- сохранить message_id для новых pending items;
- связать pending item с message_id, run_id и action_id.

### 9.3. Approval должен иметь срок годности

Например:

- approval valid 24h для ставок/бюджетов;
- 72h для создания draft;
- после истечения бот пересчитывает действие, а не применяет старую рекомендацию.

---

## 10. Auto-learning: не путать корреляцию и причинность

Текущий критерий `≥3 повтора и 14 дней без отката` слишком слабый. Паттерн может совпасть с сезонностью, акцией, изменением рынка, перераспределением бюджета или задержкой атрибуции.

Рекомендуемые изменения:

- validated learning не должен автоматически менять поведение в пилоте; сначала `review_queue` в monthly digest;
- хранить negative evidence: когда паттерн не сработал;
- иметь expiry/rolling validation, например re-check каждые 60 дней;
- указывать scope: city/topic/channel/device/audience;
- не переносить learning между городами без явного approval;
- для learnings, влияющих на ставки/бюджеты, задавать guardrails и rollback trigger.

Хорошая формулировка: автопилот накапливает **гипотезы и эвристики**, а не «самообучается» как модель без контроля.

---

## 11. Разработка всех волн сразу, тестирование итогового контура

С учётом уточнения: волны не нужно трактовать как последовательные продуктовые релизы, где W3 уже запускается в пилот. Правильнее считать их рабочими пакетами разработки полного контура. Тестировать будем итоговый вариант, когда собраны память, анализ, decisions, approvals, apply, отчёты, rollback и hardening.

При этом внутри разработки всё равно нужны технические самопроверки, но это не отдельный пользовательский пилот:

- schema validation для YAML/JSON;
- dry-run режим: собрать план действий без apply;
- mock fixtures для MCP ответов;
- golden run: одинаковый вход → одинаковый action plan;
- тест на idempotency: повторный запуск не дублирует действия;
- тест на broken STATE/config;
- тест на Telegram retry и duplicate updates.

Эти проверки нужны, чтобы финальный end-to-end тест не превратился в отладку базовой инфраструктуры.

---

## 12. Волны как рабочие пакеты полного релиза

Ниже не порядок запуска в прод, а структура разработки. Можно делать волны последовательно или частично параллельно, но итоговый тест начинается только после готовности всех критичных пакетов.

**W1 — Каркас + схемы + gitignore**

- структура каталогов;
- `.gitignore` для runtime/secrets;
- schemas для city config/state/action ledger;
- минимальный `/leadgen-autopilot`;
- запуск из корня `C:\git\leadgen-mcp`;
- Telegram hello;
- per-city lock;
- HALT.

**W2 — Память + ledger + atomic writes**

- markdown templates;
- JSON/YAML state;
- `action_ledger.jsonl`;
- atomic write helpers;
- memory lookup;
- backup/recovery.

**W3 — Аналитика и план действий**

- сбор метрик;
- LYDC;
- freshness checks;
- signal extraction;
- action plan с reason codes и confidence;
- onboarding scan для нового города;
- launch proposal.

**W4 — Approval queue**

- pending approvals;
- Telegram replies;
- expiry;
- apply approved actions;
- dry-run rollback;
- защита от повторного применения approve через `update_offset`.

**W5 — Apply engine и low-risk auto**

- deterministic idempotency key;
- применение только owned/managed campaigns;
- минусовка явного мусора;
- применение blacklist placements;
- мелкие изменения в пределах cap только при high confidence;
- no campaign activation по baseline.

**W6 — Reconciliation**

- diff config↔STATE↔API;
- новая тематика → launch proposal или draft only;
- отключение тематики → review_queue для pause;
- budget/target changes через permissions;
- drift detection и human override cooldown.

**W7 — Weekly/monthly rollups**

- weekly planning;
- monthly strategy;
- compression;
- quality metrics;
- режим monthly без конфликтующего daily/weekly изменения тактики.

**W8 — Medium/high-risk actions**

- ad variants;
- keyword/group expansion;
- limited pause/resume;
- creative generation;
- audiences/retargeting только по city-specific permissions.

**W9 — Learnings**

- proposed;
- evidence tracking;
- negative evidence;
- monthly digest;
- promotion в поведение только если city permissions это разрешают.

**W10 — Hardening and scale checklist**

- chaos;
- multi-city runtime;
- performance;
- recovery;
- rollback confirmation;
- scale to 3+ cities.

**Финальный end-to-end тест**

Тестировать не W3 отдельно, а полный сценарий:

1. Новый город → baseline scan → launch proposal.
2. Approval/config update → draft campaign creation.
3. Daily analysis → action plan → low-risk auto/action approval.
4. Telegram report + approval polling.
5. Crash/retry/idempotency check.
6. Weekly/monthly rollup на тестовой дате.
7. Rollback dry-run и confirm flow.

---

## 13. Решённые уточнения и оставшиеся вопросы

### Зафиксировать в ТЗ как решение

- Уровни доверия индивидуальны для каждого города: defaults в `caps_defaults.yaml`, overrides в `cities/<city>.yaml`.
- Routine в Claude Code запускается из корня `C:\git\leadgen-mcp`, не из `autopilot/`.
- Новый город стартует через baseline scan и launch proposal. Бот анализирует и предлагает, но не запускает неизвестные тематики без разрешения в city config.
- Все волны разрабатываются как полный контур; пользовательский тест проводится на итоговом варианте, а не на промежуточном W3.
- Новые кампании создаются draft/suspended; активация показов — отдельное действие.

### Оставшиеся вопросы

1. Где физически живёт runtime-память: в gitignored `autopilot/runtime`, OneDrive, или отдельной локальной папке?
2. Какие существующие кампании можно принять под управление: только с label, через onboarding approval, или весь аккаунт города после ревизии?
3. В monthly day daily/weekly выполняются полностью или только safety-check?
4. Какой минимальный набор MCP-инструментов уже есть для rollback/snapshots? Если нет, какие нужно добавить в server.
5. Как фиксируется `log_change_event`: в API, локальном ledger или обоих местах?
6. Есть ли единый список целей Метрики с человеческими названиями для каждого города?
7. Как учитывать лиды/звонки с задержкой атрибуции?
8. Нужны ли разные rules/caps для search и RSYA?
9. Что является SLA routine: допустимое время прогона, retry, alert при пропуске дня?

---

## 14. Минимальный acceptance для финального end-to-end теста

Перед итоговым тестом полного варианта должны быть выполнены условия:

- Для каждого proposed action есть reason_code, confidence и evidence.
- Повторный запуск одного и того же input не дублирует action plan.
- Есть lock и recovery после interrupted run.
- Есть action ledger.
- Есть clear ownership: автопилот управляет только разрешёнными кампаниями.
- Все новые кампании создаются только draft/suspended.
- Telegram approval работает с expiry и offset.
- Есть manual HALT и per-city HALT.
- Есть отчёт о drift между STATE и API.
- Есть сценарий onboarding нового города: baseline scan → launch proposal → approval/config update → draft creation.
- City-specific permissions корректно наследуются из defaults и переопределяются в `city.yaml`.

---

## 15. Что в исходном плане оставить как сильные решения

- Отдельный `/leadgen-autopilot`, а не попытка встроить автономность в ручной `/leadgen`.
- Config-driven модель города в YAML.
- Разделение eager/lazy memory.
- Read-only/dry-run режим как часть итогового теста перед apply.
- Caps, cooldowns и permissions как отдельный слой.
- Telegram compact summary + HTML detail.
- Monthly digest learnings вместо автоматической правки скилла.
- Reconciliation по целевому состоянию, а не набор разрозненных команд.

Эти решения стоит сохранить, но усилить машинными контрактами, безопасностью исполнения и консервативными baseline-настройками.
