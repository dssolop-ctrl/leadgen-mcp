# Autopilot — План разработки автономного агента leadgen

> **Статус.** Драфт ТЗ для проработки. После заполнения раздела «Уровни доверия» (✏ TODO) и подтверждения дефолтов из раздела «Caps» начинаем W1.
> **Цель проекта.** Автономный агент `/leadgen-autopilot`, который ведёт рекламный аккаунт города 24/7 без участия специалиста: ежедневная операционка, недельные итоги, месячная стратегия. Триггер — Claude Desktop routine на одной локальной машине.
> **Пилот.** 1–2 города. Масштабирование до ~60 — после стабилизации пилота.

---

## 1. Объём и границы

**В скоупе.**
- Самостоятельный скилл `.claude/skills/leadgen-autopilot/` — **параллельный** к существующему `leadgen`, не оркестратор поверх него. Запускается своей слэш-командой `/leadgen-autopilot`, живёт независимо. Скилл `leadgen` используется специалистом для ручной работы через `/leadgen`. `/leadgen-autopilot` использует бранчи `leadgen`, чтобы не дублировать логику. Основные изменения по инструментам рабоыт с кампаниями будут производиться на уровне скилла `/leadgen`, а `/leadgen-autopilot` - это обертка автоматизации.
- **Shared references** — оба скилла читают общие справочники из репо: `.claude/skills/leadgen/references/{city_benchmarks,lessons_registry,copy_blacklist,rsya_defaults,...}.md` и `library/{titles,texts,extensions,...}.md`. Это не вызовы branches — это чтение md-файлов как источника правил/данных. Дублировать их в скилле автопилота не нужно.
- **Shared playbooks (создание/оптимизация).** Когда автопилот в результате reconcile решает создать кампанию в новой тематике или провести оптимизацию, он читает `.claude/skills/leadgen/branches/{create-search,create-rsya,optimize-search,optimize-rsya}.md` как playbook — источник конкретных шагов. Это всё ещё не оркестрация: автопилот сам интерпретирует шаги в своём контексте, не запускает скилл `leadgen` в runtime. Маппинг «какой playbook на что» — в `leadgen-autopilot/references/shared_refs.md`.
- Локальная папка `autopilot/` в этом же репо: конфиги, память, отчёты, learnings.
- Telegram-уведомления через `curl` без отдельного сервиса.
- Reconciliation-модель: конфиг описывает целевое состояние, бот сравнивает с фактом и приводит к плану.

**Вне скоупа.**
- Codex-зеркало (только Claude — пользователь подтвердил).
- Webhook-сервер для inline-кнопок Telegram (на пилоте — push + ответ текстом).
- Распределённый запуск (всё на одной машине, по одной routine на город).
- Автоматическое редактирование файлов скилла `leadgen` или `leadgen-autopilot` (только предложения через `learnings/`).
- Изменение существующего скилла `leadgen` под автопилот — он остаётся таким, как сегодня, для специалиста.

**Ключевое ограничение.** Не оптимизировать токены/контекст в ущерб качеству решения — пользователь готов платить за полную загрузку нужных данных.

---

## 2. Архитектура: где что лежит

```
leadgen-mcp/
├── autopilot/                              ← НОВАЯ папка, всё runtime-состояние агента
│   ├── PLAN.md                              (этот файл)
│   ├── README.md                            быстрый старт для специалиста
│   ├── CLAUDE.md                            корневой роутер автопилота, грузится при /leadgen-autopilot
│   ├── HALT.flag                            если есть — глобальный стоп всех городов
│   ├── config/
│   │   ├── caps_defaults.yaml               глобальные дефолты caps и permissions
│   │   ├── secrets.env.example              шаблон (TELEGRAM_BOT_TOKEN, ...)
│   │   ├── secrets.env                      .gitignored, реальные токены
│   │   └── cities/
│   │       ├── _example.yaml                эталон с комментариями
│   │       ├── omsk.yaml
│   │       └── kemerovo.yaml
│   ├── memory/
│   │   └── <city>/
│   │       ├── HALT.flag                    per-city стоп
│   │       ├── STATE.md                     [eager] текущее состояние
│   │       ├── CURSOR.md                    [eager] план / отложено / ждёт approve
│   │       ├── SUMMARY.md                   [eager] хронология 30 дней (компрессия далее)
│   │       ├── pending_approvals.md         [eager] очередь review
│   │       ├── campaigns/
│   │       │   └── <campaign_id>.md         [lazy] история действий по кампании
│   │       ├── runs/
│   │       │   └── <YYYY-MM>/<DD-HHMM>.md   [lazy] полный лог запуска
│   │       └── decisions/
│   │           └── <topic>-<slug>.md        [lazy] нестандартные кейсы
│   ├── reports/
│   │   └── <city>/<YYYY-MM>/
│   │       ├── <DD-HHMM>-daily.html
│   │       ├── week-<NN>.html
│   │       └── month-<MM>.html
│   ├── learnings/
│   │   ├── proposed/<id>.md                 гипотеза из наблюдения
│   │   ├── validated/<id>.md                подтверждена ≥3 повторами и 14 днями без отката
│   │   └── rejected/<id>.md                 отклонена специалистом
│   └── lib/
│       ├── telegram_send.sh                 curl wrapper для sendMessage
│       ├── telegram_send_doc.sh             curl для sendDocument (html-отчёт)
│       ├── telegram_check_replies.sh        getUpdates → парсинг ответов
│       └── render_html.sh                   md→html для отчётов
│
├── .claude/skills/leadgen/                  ← СУЩЕСТВУЮЩИЙ скилл, не трогаем
│   └── (как сегодня, для ручной работы специалиста через /leadgen)
│
└── .claude/skills/leadgen-autopilot/         ← НОВЫЙ скилл, параллельный leadgen
    ├── skill.md                              роутер скилла, грузится по /leadgen-autopilot
    ├── flow-steps.md                         анкоры шагов (D1..D8 daily, W1..W5 weekly, M1..M6 monthly)
    ├── branches/
    │   ├── analyze.md                        цикл daily/weekly/monthly
    │   ├── reconcile_config.md               diff config vs STATE → план действий
    │   ├── decide.md                         signal → action + проверка caps/permissions/cooldown
    │   ├── apply.md                          выполнение через MCP + log_change_event
    │   ├── memory_write.md                   правила обновления eager + lazy
    │   ├── notify.md                         формирование отчёта + Telegram
    │   ├── learnings.md                      hypothesis lifecycle
    │   └── safety.md                         kill-switch, idempotency, rollback
    └── references/                           специфичные для автопилота правила
        ├── signal_catalog.md                 каталог сигналов (CPA jump, overrun, no conv, ...)
        ├── action_catalog.md                 каталог 60+ действий + дефолтные cooldowns
        ├── decision_priorities.md            приоритезация при множественных сигналах
        └── shared_refs.md                    маппинг: какие references из leadgen используем
```

**Принцип.** Скилл `.claude/skills/leadgen-autopilot/` — это **код автопилота** (правила, флоу). Папка `autopilot/` — **runtime-данные** (конфиги, память, отчёты). Обновление скилла = обновление логики (ручное, через PR). Обновление `autopilot/` — runtime, делается агентом.

**Связь с существующим скиллом leadgen.** Скилл `leadgen` (вызов `/leadgen`) и `leadgen-autopilot` (вызов `/leadgen-autopilot`) — **независимые параллельные ветки**. Ни один не вызывает другой в runtime. Общая часть — справочники в `.claude/skills/leadgen/references/` и `library/`: оба скилла читают их как md-файлы. Если правило одинаково для обеих веток (например, `lessons_registry.md`, `copy_blacklist.md`, `city_benchmarks.md`) — оно живёт в `leadgen/references/`, и оба скилла на него ссылаются. Если правило только для автопилота (например, формула приоритезации действий по сигналам) — оно в `leadgen-autopilot/branches/` или `references/`. Дублирования между скиллами быть не должно.

---

## 3. Конфиг города (формат YAML)

Файл `autopilot/config/cities/<city>.yaml`. Всё, что бот разрешён/запрещён делать — описано здесь.

```yaml
# === Идентификация ===
city: omsk
client_login: ethaji-omsk            # из MCP get_city_config
counter_id: 12345678                 # Yandex Metrika
geo_region_id: 65                    # Yandex Direct geo
domain: omsk.etagi.com
tier: 2                              # 1/2/3 — для подбора benchmarks

# === Активные тематики ===
# БОТ работает ТОЛЬКО с теми, у которых enabled: true.
# При появлении новой enabled-тематики (которой не было в STATE) — reconcile создаст кампанию.
# При выключении (enabled: false) — кампании ставятся на паузу (НЕ удаляются).
topics:
  vtorichka:
    enabled: true
    monthly_budget: 60000            # ₽
    target_cpa_form: 3500
    target_cpa_call: 2800
    channels: [search, rsya]
    notes: "Приоритет — поиск, РСЯ — добивка"
  novostroyki:
    enabled: true
    monthly_budget: 80000
    target_cpa_form: 5000
    target_cpa_call: 4000
    channels: [search, rsya]
  zagorodka:
    enabled: false
    monthly_budget: 0
    channels: []

# === Глобальный бюджет ===
budget:
  total_monthly_limit: 200000        # суммарный кап (даже если sum(topics) > этого — режется)
  daily_pacing: linear               # linear | front_loaded | back_loaded
  weekend_modifier: 0.8              # ставки в выходные ×0.8 (опционально)
  reserve_pct_for_month_end: 10      # 10% от плана зарезервировано на последние 5 дней

# === Расписание ===
# Реальный triggering — через Claude Desktop routine. Тут — целевая частота для self-check.
schedule:
  daily_runs_per_day: 1              # 1 | 2
  preferred_hours_msk: [10]          # для 1: [10]; для 2: [10, 22]
  weekly_rollup_dow: monday          # день недели для еженедельного прогона
  monthly_rollup_dom: 1              # день месяца для ежемесячного прогона

# === Caps ===
# Любое поле = null → берётся из autopilot/config/caps_defaults.yaml.
# Для города можно ужесточить или ослабить.
caps:
  max_daily_bid_change_pct: null
  max_daily_budget_change_pct: null
  cooldown_hours_after_create: null
  max_actions_per_run: null

# === Permissions: уровни доверия для действий ===
# Значения: auto | auto_with_notify | review_queue | block
# null = взять из caps_defaults.yaml.permissions
permissions:
  campaign.create.in_existing_topic: null
  campaign.create.in_new_topic: null
  bid.change.above_cap: null
  # ... полный список — см. раздел 5

# === Уведомления ===
notify:
  telegram_chat_id: -1001234567890   # один чат на этот город
  daily_summary: true
  weekly_report: true
  monthly_report: true
  alert_thresholds:
    cpa_jump_pct: 50                 # рост CPA на 50% за день
    budget_overrun_pct: 110          # перерасход >10% от плана
    no_conversions_days: 3           # 3 дня без конверсий
    impressions_drop_pct: 50         # падение показов >50%

# === Кастомные правила ===
# Свободный текст — бот учитывает при анализе и решениях.
custom_rules:
  - "Не повышать ставки на vtorichka выше 80₽ на клик (конкуренция в Омске низкая, выше — слив)"
  - "Не запускать рекламу новостроек до 1 числа месяца (бюджеты ЖК уточняются)"

# === Холодный старт ===
# Если true — первый прогон только собирает STATE без действий.
# Снимается специалистом вручную после ревью первого отчёта.
baseline_mode: false
```

**Почему в YAML, а не в роутере:** валидируется схемой, версионируется, читается отдельно от логики, легко добавить/выключить тематику без правки скилла.

---

## 4. Контрольные точки (3 цикла)

| Цикл | Триггер | Что делает | Артефакт |
|---|---|---|---|
| **Daily** | routine 1–2 раза/сутки | reconcile config↔state, fetch metrics, decide+apply, обновить eager memory, отправить summary | `runs/<DD-HHMM>.md`, `reports/.../daily.html`, Telegram push |
| **Weekly** | routine в `weekly_rollup_dow` | сравнение неделя/неделя по тематикам, итоги + правки тактики на след. неделю, обновить CURSOR | `reports/.../week-NN.html`, обновлённый CURSOR.md, Telegram push с приложенным html |
| **Monthly** | routine в `monthly_rollup_dom` | стратегические выводы, дайджест learnings (proposed→validated), план бюджета на след. месяц | `reports/.../month-MM.html`, дайджест предложений в `learnings/validated/`, Telegram push |

**Важно.** Daily всегда выполняется. Weekly и Monthly — **в дополнение** к daily, не вместо. То есть в понедельник 1-го числа в 10:00 пройдут все три (daily → weekly → monthly), последовательно в одном запуске routine.

**Решение о цикле принимает скилл.** Routine просто пишет в prompt `Запусти leadgen-autopilot для города omsk`. Бот сам по дате/времени решает, какие циклы активны (daily обязательно, weekly если сегодня monday, monthly если сегодня 1-е).

---

## 5. Уровни доверия и каталог действий

**Значения (4 уровня):**

| Уровень | Поведение |
|---|---|
| `auto` | Выполнить молча. Запись в memory, без Telegram-уведомления (попадёт только в daily summary). |
| `auto_with_notify` | Выполнить и отправить отдельное Telegram-сообщение сразу. |
| `review_queue` | НЕ выполнять. Положить в `pending_approvals.md` с описанием. Уведомить. Ждать ответа. |
| `block` | НЕ выполнять никогда. При попытке — лог + alert. |

**Каталог действий (для проставления значений):**

> ✏️ **TODO специалисту.** Проставь желаемый уровень в столбце «Дефолт». Мои предложения — в столбце «Рекомендую» (консервативный пилот).

### 5.1 Бюджет
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `budget.increase.within_cap` | +N% бюджета в пределах `max_daily_budget_change_pct` | auto | auto |
| `budget.increase.above_cap` | +N% бюджета сверх капа | review_queue | review_queue |
| `budget.decrease.within_cap` | −N% в пределах капа | auto | auto |
| `budget.decrease.above_cap` | −N% сверх капа | review_queue | review_queue |
| `budget.set_total_monthly` | переопределить месячный лимит | review_queue | review_queue |
| `budget.pause_due_to_overrun` | выкл при перерасходе >`alert.budget_overrun_pct` | auto_with_notify | auto_with_notify |

### 5.2 Ставки
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `bid.increase.within_cap` | +N% ставки в пределах капа | auto | auto |
| `bid.decrease.within_cap` | −N% ставки в пределах капа | auto | auto |
| `bid.change.above_cap` | сверх капа | auto_with_notify | auto_with_notify |
| `bid.adjust_strategy_target_cpa.within_cap` | сдвиг target CPA автостратегии в пределах ±15% | auto_with_notify | auto_with_notify |
| `bid.adjust_strategy_target_cpa.above_cap` | сдвиг сверх ±15% | review_queue | review_queue |

### 5.3 Минус-слова
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `negatives.add_from_search_queries` | минусовка по отчёту поисковых запросов | auto | auto |
| `negatives.add_global_set` | пополнение общегородского набора | auto | auto |
| `negatives.remove` | удаление минус-слова (редко, ошибочно сминусованного) | review_queue | review_queue |

### 5.4 Кампании
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `campaign.pause.low_performance` | пауза при CPA × 2 от target за 14+ дней | auto_with_notify | auto_with_notify |
| `campaign.pause.budget_exhausted` | пауза при исчерпании дневного бюджета | auto | auto |
| `campaign.resume` | возобновление | auto_with_notify | auto_with_notify |
| `campaign.create.in_existing_topic` | новая кампания в уже работающей тематике | auto_with_notify | auto_with_notify |
| `campaign.create.in_new_topic` | новая тематика появилась в config (reconcile) | auto_with_notify | auto_with_notify |
| `campaign.create.outside_config_topics` | тематика не разрешена в config | **block** | |
| `campaign.archive.no_traffic_14days` | архивация молчуна | auto_with_notify | auto_with_notify |
| `campaign.delete` | физическое удаление | block | |

### 5.5 Объявления и группы
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `adgroup.pause.low_ctr` | пауза группы при CTR ниже tier_min | auto_with_notify | auto_with_notify |
| `adgroup.split_by_intent` | разделить группу по интенту | review_queue | auto |
| `ad.add_new_variant` | добавить новый вариант объявления | auto | auto |
| `ad.pause.low_ctr` | выкл слабого варианта | auto | auto |
| `ad.update_text` | правка текста | auto_with_notify | auto |
| `ad.update_image_creative` | замена картинки | auto_with_notify | auto_with_notify |

### 5.6 Ключевые слова
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `keyword.pause.low_performance` | пауза не отрабатывающего ключа | auto | auto  |
| `keyword.add_in_existing_group` | добавить ключ в существующую группу | auto | auto |
| `keyword.add_new_group_in_existing_topic` | новая группа внутри текущей тематики | auto_with_notify | auto |
| `keyword.remove` | удалить ключ | auto_with_notify | auto_with_notify |

### 5.7 Стратегии
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `strategy.change_type` | смена типа (manual → autostrategy и т.п.) | review_queue | auto_with_notify |
| `strategy.adjust_constraint` | правка ограничений внутри текущей стратегии | auto_with_notify | auto_with_notify |

### 5.8 Площадки (РСЯ)
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `placement.block.low_performing` | блокировка слабой площадки | auto | auto |
| `placement.block.from_blacklist` | применение общегородского чёрного списка | auto | auto |
| `placement.unblock` | разблокировка | review_queue | auto |

### 5.9 Аудитории / ретаргетинг
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `audience.create` | новая аудитория | auto_with_notify | auto |
| `audience.adjust_bid_modifier` | модификатор ставки на аудиторию | auto | auto |
| `retargeting.create_list` | новый список ретаргета | auto_with_notify | auto |

### 5.10 Региональные модификаторы
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `region.adjust_bid_modifier` | сдвиг ставки на регион | auto | auto |
| `region.add_to_targeting` | добавить регион | review_queue | review_queue |
| `region.remove_from_targeting` | убрать регион | review_queue | review_queue |

### 5.11 Расширения и vcard
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `extension.add_sitelinks` | добавить быстрые ссылки | auto | auto |
| `extension.update_sitelinks` | обновить | auto | auto |
| `extension.update_vcard` | правка визитки | review_queue | review_queue |

### 5.12 Креативы
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `creative.generate_new_image` | сгенерить новую картинку через OpenRouter | auto_with_notify | auto_with_notify |
| `creative.update_text_variant` | новый вариант текста | auto | auto |

### 5.13 Юр-чувствительное
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `legal.update_disclaimer` | правка юр.дисклеймера | block | block |
| `legal.change_landing_url` | смена посадочной | review_queue | review_queue |
| `legal.change_company_info` | смена реквизитов | block | block |

### 5.14 Аккаунт
| Действие | Описание | Рекомендую | Дефолт ✏ |
|---|---|---|---|
| `account.change_settings` | глобальные настройки аккаунта | block | block |
| `account.change_billing` | биллинг | block | block |
| `account.close` | закрытие аккаунта | block | block |

---

## 6. Дефолты caps (`autopilot/config/caps_defaults.yaml`)

> Подобраны на основе общеотраслевых рекомендаций по контекстной рекламе (Яндекс/Google) и привязаны к специфике автостратегий Direct, которые обучаются 7–14 дней. Все значения — переопределяемы в `cities/<city>.yaml`.

```yaml
# === Лимиты темпа изменений ===
max_daily_bid_change_pct: 20             # за прогон не двигать ставку >±20%; иначе ломаем обучение автостратегии
max_daily_budget_change_pct: 30          # бюджет инкрементальнее ставок, +30% безопасно
max_daily_target_cpa_change_pct: 15      # target CPA — самый чувствительный, минимальный шаг

# === Cooldown'ы (часы) ===
cooldown_hours_after_create: 72          # 3 дня после старта — не трогать (первичный набор статистики)
cooldown_hours_after_bid_change: 24      # сутки на эффект изменения ставки
cooldown_hours_after_strategy_change: 168 # 7 дней — переучка стратегии
cooldown_hours_after_budget_change: 24

# === Пороги принятия решений (стат. значимость) ===
min_clicks_for_decision: 50              # отраслевой минимум для решения по ставке/CPA
min_conversions_for_decision: 5          # минимум для решения по target CPA
min_impressions_for_decision: 1000       # для решения по CTR/тексту объявления
learning_period_days: 14                 # первые 14 дней — только мелкие правки

# === Лимиты на прогон ===
max_actions_per_run: 10                  # защита от взбесившегося агента (тут не понятно какие именно действия, если имеются ввиду любые изменения, то лимит должен быть гораздо больше, чтобы не терять в качестве принимаемых решений из за лимита)
max_new_campaigns_per_run: 2             # две кампании за прогон — иначе сложно отслеживать обучение
max_pauses_per_run: 5                    # не паузить больше 5 кампаний/групп за раз

# === Бюджетные защитные правила ===
budget_overrun_hard_stop_pct: 120        # при перерасходе 120% — экстренная пауза
min_remaining_budget_pct_for_aggressive_actions: 25  # если остаток <25% — режим консервации

# === Пороги уведомлений (можно переопределить в city.notify.alert_thresholds) ===
alert_cpa_jump_pct: 50
alert_budget_overrun_pct: 110
alert_no_conversions_days: 3
alert_impressions_drop_pct: 50

# === Permissions defaults ===
# Применяются если в city.yaml поле = null
permissions:
  budget.increase.within_cap: auto
  budget.increase.above_cap: review_queue
  budget.decrease.within_cap: auto
  budget.decrease.above_cap: review_queue
  bid.increase.within_cap: auto
  bid.decrease.within_cap: auto
  bid.change.above_cap: review_queue
  negatives.add_from_search_queries: auto
  negatives.add_global_set: auto
  negatives.remove: review_queue
  campaign.pause.low_performance: auto_with_notify
  campaign.pause.budget_exhausted: auto
  campaign.resume: auto_with_notify
  campaign.create.in_existing_topic: auto_with_notify
  campaign.create.in_new_topic: auto_with_notify
  campaign.create.outside_config_topics: block
  campaign.archive.no_traffic_30days: auto_with_notify
  campaign.delete: block
  adgroup.pause.low_ctr: auto_with_notify
  adgroup.split_by_intent: review_queue
  ad.add_new_variant: auto
  ad.pause.low_ctr: auto
  ad.update_text: auto_with_notify
  ad.update_image_creative: auto_with_notify
  keyword.pause.low_performance: auto
  keyword.add_in_existing_group: auto
  keyword.add_new_group_in_existing_topic: auto_with_notify
  keyword.remove: auto_with_notify
  strategy.change_type: review_queue
  strategy.adjust_constraint: auto_with_notify
  placement.block.low_performing: auto
  placement.block.from_blacklist: auto
  placement.unblock: review_queue
  audience.create: auto_with_notify
  audience.adjust_bid_modifier: auto
  retargeting.create_list: auto_with_notify
  region.adjust_bid_modifier: auto
  region.add_to_targeting: review_queue
  region.remove_from_targeting: review_queue
  extension.add_sitelinks: auto
  extension.update_sitelinks: auto
  extension.update_vcard: review_queue
  creative.generate_new_image: auto_with_notify
  creative.update_text_variant: auto
  legal.update_disclaimer: block
  legal.change_landing_url: review_queue
  legal.change_company_info: block
  account.change_settings: block
  account.change_billing: block
  account.close: block
```

---

## 7. Слой памяти

### 7.1 Eager (всегда грузится в начале прогона)

**`STATE.md`** — текущее состояние города. ≤200 строк. Перезаписывается в конце каждого прогона.

```markdown
# STATE — Омск (last update 2026-04-29 14:30)

## Метаданные
- last_skill_version: <git-sha leadgen>
- last_autopilot_version: <git-sha autopilot>
- last_run_id: omsk-2026-04-29-1430
- baseline_mode: false

## Бюджет
- monthly_total_limit: 200 000 ₽
- spent_mtd: 134 200 ₽ (67%)
- pace_vs_linear: −3% (опережение/отставание от линейного)

## Активные тематики
| topic | enabled | budget_used | budget_limit | active_campaigns | trend_cpa_7d |
|---|---|---|---|---|---|
| vtorichka | yes | 42 100 | 60 000 | 3 (search:1, rsya:2) | ↘ −8% |
| novostroyki | yes | 92 100 | 80 000 | ⚠ overrun! 4 (s:2, r:2) | → flat |
| zagorodka | no | — | — | — | — |

## Активные кампании (краткая сводка)
| campaign_id | label | status | last_action | next_action | flag |
|---|---|---|---|---|---|
| 8765432 | Омск-Вторичка-Поиск | running | 2026-04-28 +bid 10% | watch | — |
| 8765433 | Омск-Вторичка-РСЯ | learning (D5) | 2026-04-24 created | wait cooldown | learning |
| 8765499 | Омск-Новостройки-РСЯ | running | 2026-04-29 split groups | watch | ⚠ CPA 1450, target 1000 |

## Указатели в lazy memory
- кампании, изменённые сегодня: campaigns/8765432.md, campaigns/8765499.md
- последний run: runs/2026-04/29-1430.md
- открытые decisions: decisions/novostroyki-cpa-spike.md

## Флаги внимания
- ⚠ novostroyki: CPA 1450 vs target 1000, день 7 обучения, сглаживается
- ⚠ pending_approvals: 1 запрос (см. pending_approvals.md)

## Последний weekly rollup: 2026-04-22
## Последний monthly rollup: 2026-04-01
```

**`CURSOR.md`** — план: что делал последний раз, что отложено, что ждёт approve. ≤80 строк.

```markdown
# CURSOR — Омск

## Сделано в последнем прогоне (2026-04-29 14:30)
- vtorichka-search: +12 минусов из search queries
- vtorichka-search: +bid 10% на группе "вторичка-1комн" (CPA опережал target)
- vtorichka-rsya: применил blocked_placements (+34 хоста)

## Отложено (сделать на следующем прогоне)
- novostroyki-rsya: дать ещё 3 дня обучения, потом решать (cooldown until 2026-05-02)
- проверить эффект split_by_intent на vtorichka-search-2 через 14 дней

## В очереди review (см. pending_approvals.md)
1. campaign.create.in_new_topic: zagorodka — пользователь enabled tomorrow?

## План на неделю (от weekly rollup 2026-04-22)
- вырастить долю РСЯ в vtorichka до 40% от лидов
- проверить гипотезу: вечерние ставки ×1.2 на новостройках
```

**`SUMMARY.md`** — хронология. Свежие 30 дней — построчно, далее компрессия.

```markdown
# SUMMARY — Омск

## Последние 30 дней (по дням)
- 2026-04-29: ⬇ CPA vtorichka, ⚠ novostroyki overrun. Действий: 4. Лидов: 12.
- 2026-04-28: ровно. Действий: 1. Лидов: 9.
- 2026-04-27: weekly rollup. Лидов: 11.
...

## Недели 30–90 дней (компрессия 1 строка)
- W17: средний CPA 720, лидов 76, без существенных правок
- W16: эксперимент с расширением региона — отказ, CPA вырос 30%

## Месяцы 3–12 мес (1–2 строки)
- 2026-03: запуск загородки, отключили после 2 недель (CPA × 2.5)
- 2026-02: пилот РСЯ — успешно
```

### 7.2 Lazy (грузится по тегам через grep)

**Шаблон тегов в шапке любого lazy-файла:**
```markdown
---
tags: [campaign:8765432] [topic:vtorichka] [channel:rsya] [city:omsk]
last_action: 2026-04-29
status: running
---
# История кампании 8765432 — Омск-Вторичка-РСЯ
...
```

Поиск: `grep -l "\[campaign:8765432\]" memory/omsk/campaigns/`. Без RAG, без векторов — для md-файлов это избыточно.

**Структура `campaigns/<id>.md`:** список изменений в обратном хронологическом, каждое — ссылка на runs/...md где было принято решение.

**Структура `runs/<YYYY-MM>/<DD-HHMM>.md`:** полный лог прогона — какой контекст загрузил, что увидел в метриках, какие сигналы выделил, какие решения принял (с обоснованием), какие действия выполнил, что записал в память.

**Структура `decisions/<topic>-<slug>.md`:** нестандартные кейсы, которые не вписываются в обычные действия. Например, гипотеза «вечерние ставки ×1.2», эксперимент с novostroyki в субботу, и т.п. Сюда же — отчёт о результате эксперимента.

### 7.3 Компрессия SUMMARY (раз в неделю в weekly rollup)

- Записи >30 дней → схлопнуть в недельные строки.
- Записи >90 дней → схлопнуть в месячные.
- Записи >365 дней → перевести в `decisions/historical-<year>.md` и удалить из SUMMARY.

Бот отвечает за компрессию сам в ходе weekly rollup.

---

## 8. Reconciliation: config-driven управление тематиками

**Принцип.** Конфиг описывает «как должно быть», STATE — «как есть». Бот сравнивает и приводит факт к плану.

**Алгоритм reconcile (на каждом daily-прогоне):**

1. Прочитать `config/cities/<city>.yaml`.
2. Прочитать `STATE.md` → `topics_in_state`.
3. Дельты:
   - **Появилась новая enabled-тематика** (нет в STATE): запланировать `campaign.create.in_new_topic` для каждого канала из `topics.<t>.channels`.
   - **Тематика стала disabled** (есть в STATE с активными кампаниями): запланировать `campaign.pause.low_performance` (или `campaign.archive.no_traffic_30days` если давно простаивает).
   - **Изменился `monthly_budget`**: запланировать `budget.set_total_monthly` (через permissions проверить уровень).
   - **Изменился `target_cpa_form/call`**: проверить delta vs текущий tCPA автостратегий → `bid.adjust_strategy_target_cpa.*`.
4. Каждое запланированное действие проходит общий decision-фильтр (caps + permissions + cooldown).

**Критично.** Бот **никогда** не запускает кампании в тематиках, которых нет в config (даже в `enabled: false`). Соответствующее действие `campaign.create.outside_config_topics` имеет дефолт `block`.

---

## 9. Авто-обучение

**Жизненный цикл гипотезы:**

1. **Naissance.** В ходе анализа бот замечает паттерн: «после повышения ставок на vtorichka в Омске +15% CPA снижался на 20% в течение 7 дней». Создаёт `learnings/proposed/<id>.md`:
   ```markdown
   id: omsk-vtorichka-bid-uplift-2026-04
   created: 2026-04-29
   confidence: low
   observed_count: 1
   pattern: "+15% bid on vtorichka_search → −20% CPA over 7d"
   evidence:
     - run omsk-2026-04-22-1030, action bid.increase.within_cap, result +18% conv at −22% CPA
   needs_repeats: 2
   action_when_validated: "В analyze-ветке при tCPA > target × 1.15 на vtorichka — приоритет +bid 15% (до bid_increase_within_cap проверки)"
   ```
2. **Validation.** При повторении паттерна (≥3 раза) и без откатов 14 дней → `learnings/validated/<id>.md`. Бот **начинает учитывать** правило в decision-фазе автоматически (загружает validated learnings в контекст при анализе).
3. **Monthly digest.** В ежемесячном rollup бот шлёт специалисту список validated learnings. Специалист может **закодировать** их в `references/lessons_registry.md` скилла `leadgen` — это уже ручная задача через PR.
4. **Rejection.** Если специалист отвечает «не использовать» → бот переносит в `learnings/rejected/<id>.md` и больше не предлагает повтор.

**Граница:** автопилот никогда сам не пишет в файлы скилла `leadgen` или `leadgen-autopilot`. Только в `autopilot/learnings/`.

---

## 10. Уведомления (Telegram)

**Транспорт.** `curl` к Bot API, токен из `autopilot/config/secrets.env`. Никакого отдельного процесса.

**Формат daily-уведомления (compact):**
```
🟢 Омск · 29 апр 14:30
Бюджет день: 8 420 / 10 000 ₽ (84%)
Лиды: 12 · CPA 702 (цель 800)
Действия: 4
  · vtorichka: +12 минусов, +bid 10%
  · vtorichka-rsya: +34 заблоч.площадки
⚠ novostroyki: CPA 1 450, день 7, target 1 000
[прикреплён detail.html]
```

**HTML-отчёт.** Markdown→HTML через `lib/render_html.sh` (pandoc или встроенный конвертер). Складывается в `reports/<city>/<YYYY-MM>/<run-id>.html`. Не удаляется.

**Approval polling (W6).** Простая реализация: в начале каждого прогона `lib/telegram_check_replies.sh` через `getUpdates` забирает все ответы пользователя на ранее отправленные сообщения с pending. Парсит ключевые слова (`approve N`, `reject N`, `defer N 3d`) и обновляет `pending_approvals.md`.

**Per-city чат.** Один Telegram-чат на город — `notify.telegram_chat_id` в `city.yaml`. Никаких alias/mux.

---

## 11. Безопасность

| Механизм | Реализация |
|---|---|
| **Глобальный kill-switch** | `autopilot/HALT.flag` — если есть, бот при старте логирует и останавливается. |
| **Per-city kill-switch** | `autopilot/memory/<city>/HALT.flag` |
| **Idempotency** | Каждое действие имеет `correlation_key = <city>-<run_id>-<action_slug>`. Перед apply — `get_change_history` за 24ч; если correlation_key уже есть — skip. |
| **Кап действий** | `max_actions_per_run` — после превышения бот фиксирует «не успели в этом прогоне», шлёт alert, остаток выполнит в следующем. |
| **Обязательный baseline-режим** | Первый прогон нового города — `baseline_mode: true`, бот только собирает STATE без действий. Снимается специалистом. |
| **Rollback** | Каждое действие в `runs/<id>.md` пишет «обратное действие» (например, +bid 10% → откат: −9.09%). Специалист командой `rollback <run_id>` в Telegram может откатить весь прогон. |
| **Hard cap при overrun** | Если `spent_mtd > budget * (budget_overrun_hard_stop_pct/100)` — экстренная пауза всех кампаний, alert, баста до ручного резюма. |

---

## 12. Метрики качества автопилота

В каждом monthly-rollup:

- **Decision precision.** Доля действий, которые через 14 дней дали улучшение (CPA −X% или conv +Y%). Цель: >60%.
- **Rollback rate.** Доля действий, откатанных специалистом. Цель: <10%.
- **Approval queue health.** Среднее время в `pending_approvals.md` и сколько устарело >7 дней. Цель: <72ч.
- **Coverage.** % дней, когда бот отработал без падений / без halt'ов.
- **Telegram noise.** Среднее число `auto_with_notify` сообщений в день. Если >5 — пересмотреть permissions (слишком шумно).

---

## 13. Волны разработки

> Порядок строгий. Нельзя начинать W4 без работающего W3 и т.д. После W3 пилот можно держать в read-only режиме столько, сколько нужно для доверия.

### W1 — Каркас (1–2 дня)
- Создать структуру каталогов `autopilot/{config,memory,reports,learnings,lib}`.
- Скилл `.claude/skills/leadgen-autopilot/skill.md` минимальный — роутер шагов, без branches пока.
- `autopilot/CLAUDE.md` — корневой роутер, грузится при `/leadgen-autopilot`.
- `config/caps_defaults.yaml` с дефолтами из раздела 6.
- `config/cities/_example.yaml` с комментариями.
- `config/secrets.env.example`.
- `lib/telegram_send.sh` (curl wrapper).
- `HALT.flag` логика.
- README с быстрым стартом для специалиста (как добавить город, как настроить routine).
- **Acceptance:** `/leadgen-autopilot` запускается, читает HALT, шлёт `Hello from autopilot, city=omsk` в Telegram, выходит.

### W2 — Память: структура и компрессия (1–2 дня)
- Шаблоны `STATE.md`, `CURSOR.md`, `SUMMARY.md`, `pending_approvals.md` — генератор для нового города.
- Правила тегов (раздел 7.2).
- Branch `branches/memory_write.md` — алгоритм обновления eager/lazy.
- Алгоритм компрессии SUMMARY (>30/>90/>365 дней).
- Скрипты grep по тегам (`lib/memory_lookup.sh`).
- **Acceptance:** ручной прогон создаёт корректные файлы; повторный запуск читает их; компрессия запускается из weekly-ветки (mock).

### W3 — Read-only daily (3–5 дней) — **ПИЛОТ ЗАПУСКАЕТСЯ ЗДЕСЬ**
- Branch `branches/analyze.md` — daily-цикл без действий.
- Сбор метрик: `get_campaign_stats`, `metrika_get_direct_report`, `get_search_queries`, `get_change_history`.
- Выявление сигналов (CPA jump, budget overrun, no conversions, learning ended, и т.п.).
- Запись в `runs/<id>.md` и обновление STATE/CURSOR/SUMMARY.
- HTML-отчёт через `lib/render_html.sh`.
- Telegram daily summary.
- **НЕТ apply-фазы** — только наблюдение и предложения «что бы я сделал, если бы мог».
- **Acceptance:** routine 1×/день запускается, после 7 дней бот предлагает 5+ корректных действий (специалист валидирует руками), отчёты приходят без падений.

### W4 — Действия уровня `auto` (3–4 дня)
- Branch `branches/decide.md` — signal → action + проверка caps/permissions/cooldown/idempotency.
- Branch `branches/apply.md` — выполнение через MCP, обязательный `log_change_event`.
- Включить безопасные `auto`-действия: минусование, мелкие правки ставок в cap, blocked_placements, бюджетные правки в cap.
- Запись в lazy memory (`campaigns/<id>.md`, `decisions/...md`).
- **Acceptance:** за 7 дней пилота нет инцидентов overrun / неконтролируемых правок; rollback rate <5%.

### W5 — Действия уровня `auto_with_notify` (2–3 дня)
- Включить: pause/resume кампаний, `campaign.create.in_existing_topic`, `creative.generate_new_image`, аудитории.
- Усиленные уведомления — каждое действие отдельным сообщением.
- **Acceptance:** specialist не вмешивается в течение недели; отчёты совпадают с ожиданием.

### W6 — Review queue + approval polling (3–4 дня)
- Branch `branches/safety.md` — формат `pending_approvals.md`.
- `lib/telegram_check_replies.sh` — getUpdates polling, парсинг ответов.
- При следующем запуске бот применяет approved, отказывается от rejected, продлевает deferred.
- **Acceptance:** review-цикл работает end-to-end: бот → pending → ответ в TG → следующий run → apply.

### W7 — Reconciliation (2 дня)
- Branch `branches/reconcile_config.md` — diff config↔STATE.
- Сценарии: новая тематика, выключение тематики, изменение бюджета, изменение target CPA.
- **Acceptance:** включение новой тематики в `city.yaml` → следующий прогон создаёт кампанию (через approval-цепочку, если permissions = `auto_with_notify`/`review_queue`).

### W8 — Weekly rollup (2–3 дня)
- Branch расширения `analyze.md`: weekly-режим.
- Сравнение неделя/неделя по тематикам.
- Обновление CURSOR с планом на след. неделю.
- HTML-отчёт недели (с графиками — простая genchart через python).
- Компрессия SUMMARY.
- **Acceptance:** в monday отчёт уходит, CURSOR обновляется, daily-прогоны учитывают weekly-план.

### W9 — Авто-обучение: proposed→validated (3–4 дня)
- Branch `branches/learnings.md`.
- Naissance: наблюдение паттерна → `proposed/<id>.md`.
- Validation: автоматическая после ≥3 повторов и 14 дней без отката.
- Применение validated в decide-фазе.
- **Acceptance:** через 30+ дней работы бот формирует ≥1 validated learning, явно использует его в решении.

### W10 — Monthly rollup + learnings digest (2–3 дня)
- Monthly режим analyze.
- Дайджест validated → специалисту.
- План бюджета на след. месяц по тематикам (с обоснованием).
- Метрики качества автопилота (раздел 12).
- **Acceptance:** 1-го числа отчёт уходит, дайджест содержит конкретные предложения по `lessons_registry.md`.

### W11 — Hardening (2–3 дня)
- Стресс-тесты: что если MCP отдал ошибку 5xx, что если конфиг битый, что если STATE расходится с реальностью аккаунта.
- Rollback-команда из Telegram.
- Hard cap budget overrun.
- Документация для специалиста: как читать отчёты, как управлять approval, как читать learnings.
- Чеклист «что проверить перед масштабированием на 3+ городов».
- **Acceptance:** chaos-тест 3 дня без вмешательства; ни одного критичного инцидента.

---

## 14. Открытые вопросы (для решения по ходу)

1. **HTML-рендер.** Pandoc локально или python-скрипт `markdown→html`? У pandoc больше возможностей, но требует установки. → решим в W2.
2. **Графики в weekly/monthly отчётах.** matplotlib png + inline в html? Или текстовые ASCII-графики? → решим в W8.
3. **Approval-команды формат.** `approve 1`, `reject 1`, `defer 1 3d` — стандартизировать. → решим в W6.
4. **Версионирование learnings.** Если правило перестало работать через 60 дней — как откатить? Авто-инвалидация по rolling window? → решим в W9.
5. **Multi-account: один STATE на login или на city?** Сейчас 1:1, но в Этажах login может покрывать несколько городов. → проверить на onboarding'е.
6. **Время прогона.** Сколько занимает один daily run по факту (с учётом `get_search_queries` и т.п.)? → измерим в W3 пилоте.

---

## 15. TODO специалисту перед стартом W1

- [ ] Заполнить столбец «Дефолт ✏» в разделе 5 (уровни доверия для всех действий).
- [ ] Подтвердить дефолты caps в разделе 6 или скорректировать.
- [ ] Создать Telegram-бота, получить токен, создать чат для пилотного города, узнать `chat_id`.
- [ ] Назвать пилотный город (omsk? другой?).
- [ ] Подтвердить `monthly_budget` и `target_cpa_*` для пилотных тематик.
- [ ] Подтвердить `baseline_mode: true` для первого прогона.
- [ ] Подтвердить, что Claude Desktop routine может крутиться 24/7 (питание/сон/обновления).

После заполнения — открываем W1.
