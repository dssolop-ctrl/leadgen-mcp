# Цели конверсии Этажи

## Основные цели (для оптимизации стратегии)

Эти цели одинаковы по названиям на всех счётчиках городов, но **ID различаются**.
При создании кампании — получи цели через `metrika_get_goals(counter_id, account="etagi.agent")` и найди нужную по имени.

### Приоритет выбора цели для стратегии

**Всегда ставим 2 цели на кампанию:**

1. **Общая (все лиды)** (условие: `form_sum_leads`) — ВСЕГДА ставим. Сумма всех форм заявок. **НЕ включает звонки.**
2. **Звонки** — одна из двух (взаимоисключающие):
   - **Коллтрекинг, реальный звонок** (условие: `received_real_calls`) — если есть на счётчике. Подтверждённый звонок. Есть НЕ на всех счётчиках.
   - **Общая / Все звонки** (условие: `all_calls`) — если коллтрекинга НЕТ на счётчике. Все звонки (включая короткие).

### Взаимосвязь целей

```
Коллтрекинг, реальный звонок  ← подтверждённые звонки (НЕ входит в «Общая все лиды»)
Общая (все лиды)               ← СУММА всех форм заявок:
  ├── Заявка на ипотеку / общая
  ├── Заявка по новостройкам
  ├── Заявка по вторичному жилью
  ├── Заявка на покупку загородной недвижимости
  ├── Заявка на аренду вторичного жилья
  ├── Заявка на покупку коммерческой недвижимости
  ├── Заявка на продажу недвижимости
  └── ... (другие формы)
Общая / Все звонки              ← все звонки (НЕ входит в «Общая все лиды»)
```

> **Важно:** Для полной картины конверсий нужно смотреть и «Общая (все лиды)» И «Коллтрекинг» отдельно. Они НЕ пересекаются.

> НИКОГДА не ставить микроцели (просмотр страниц, клик по телефону, скролл) в goal_id стратегии.

### Таблица основных целей

| Название цели | Тип | Приоритет | Описание |
|---|---|---|---|
| Общая (все лиды) | action (`form_sum_leads`) | ВСЕГДА | Сумма всех форм заявок. НЕ включает звонки |
| Коллтрекинг, реальный звонок | action (`received_real_calls`) | Если есть | Подтверждённый звонок. Есть НЕ на всех счётчиках |
| Общая / Все звонки | action (`all_calls`) | Если нет коллтрекинга | Все звонки (включая короткие). Fallback для звонков |

### Специализированные цели (для отчётности, не для стратегии)

| Название цели | Тип | Описание |
|---|---|---|
| Заявка по новостройкам | action | Форма на странице новостроек |
| Заявка на ипотеку / общая | action | Форма на странице ипотеки |
| Общая / Заявка на продажу | action | Форма продажи квартиры |
| Заявка на покупку коммерческой недвижимости | action | Коммерческая недвижимость |
| Заявка на аренду вторичного жилья (снять) | action | Аренда — покупатель |
| Заявка на покупку гаража | action | Гараж |
| Личный кабинет (общая) | action | Регистрация/вход в ЛК |

### Цели коллтрекинга (автоматические)

| Название цели | Тип | Описание |
|---|---|---|
| Уникальный звонок | conditional_call | Первый звонок с номера |
| Уникально-целевой звонок | conditional_call | Первый звонок > 30 сек |
| Целевой звонок | conditional_call | Любой звонок > 30 сек |

## Как найти цели и ценности при создании кампании

**Используй `get_conversion_values` — он делает всё автоматически:**

```
1. Определи counter_id и client_login города из config/counters.md
2. Вызови: get_conversion_values(client_login=<login>, counter_id=<id>)
   → Инструмент сам:
     a) Найдёт цели form_sum_leads + received_real_calls/all_calls
     b) Проанализирует CPA по действующим кампаниям города (30 дней)
     c) Рассчитает ценности конверсий на основе реальных данных
     d) Вернёт готовый JSON для priority_goals
3. Используй priority_goals_json из ответа в add_campaign / update_campaign
```

**Ручной вариант (если нужно только найти цели без ценностей):**
```
metrika_get_goals(counter_id=<id>, account="etagi.agent", conditions="form_sum_leads,received_real_calls,all_calls")
```

## Ценность конверсий (value в priority_goals)

**Не ставь value=0 или произвольные числа.** Используй `get_conversion_values` для расчёта на основе реальных данных.

### Параметры

| Параметр | Default | Описание |
|---|---|---|
| `client_login` | — | Логин клиента города (обяз.) |
| `counter_id` | — | ID счётчика Метрики (обяз.) |
| `theme` | null | Фильтр тематики: `vtorichka`, `novostroyki`, `zagorodka`, `ipoteka`, `arenda`, `commerce`, `agency`, `imidzh`, `hr`. Если задан — расчёт только по совпадающим кампаниям; иначе общий + `breakdown_by_theme`. |
| `days` | 30 | Стартовый период (игнорируется при `auto_window=true` если данных мало или много) |
| `auto_window` | true | Авто-окно: 30→60→90 при <20 conv; 30→14 при >150 conv |
| `min_conversions` | 5 | Минимум конверсий per-goal на кампанию |
| `min_clicks` | 100 | Минимум кликов на кампанию |
| `min_cost` | 1000 | Минимум расхода на кампанию (₽) |
| `exclude_learning` | true | Исключать кампании <14 дней + с реальными «красными флагами» модерации (`отклон`, `закончил`, `запрещ`, `не принят`, `ошибк`, `приостановл`) |
| `target_form_cpa_override` | 0 | Явный таргет формы для `priority_goals_json` |
| `target_call_cpa_override` | 0 | Явный таргет звонка |

### Алгоритм

1. **Resolve goals** — `form_sum_leads` + `received_real_calls`/`all_calls`.
2. **Get active campaigns** (state=ON) с полями `StartDate`, `StatusClarification`, `Name`.
3. **Learning/moderation filter** (если `exclude_learning=true`):
   - запуск <14 дней → `learning_period: запущена N дн. назад`
   - `StatusClarification` содержит «красный флаг» (`отклон|закончил|запрещ|не принят|ошибк|приостановл`) → `moderation: …`
   - **Важно:** «Идут показы» и подобные — это нормальное состояние, НЕ исключаем
4. **Theme filter** (если задан `theme`):
   - Парсим тематику из имени кампании (3-й pipe-сегмент: `Город | Тип | Тематика | …`)
   - Кампании с другой тематикой → `theme_mismatch: parsed=X, requested=Y`
5. **Auto-window loop** (если `auto_window=true`):
   - Старт 30d → если `total_conv > 150` → 14d (более свежий сигнал)
   - Если `total_conv < 20` → 60d → 90d
   - Возвращает `actual_days_used`, `window_reason: "settled at Nd: total_conv=X (attempts: [30 60])"`
6. **Fetch report** `CAMPAIGN_PERFORMANCE_REPORT` с `AttributionModels=LYDC`, `Goals=[form, call]`, `FieldNames=[CampaignId, Cost, Clicks, Conversions, CostPerConversion]`.
   - Парсим per-goal колонки (`Conversions (X)`, `CostPerConversion (X)`)
   - Если Yandex вернул положительный `CostPerConversion` — используем его, флаг `cost_attribution: "yandex_per_goal_cost_per_conversion"`
   - Иначе считаем `cost/conv`, флаг `cost_attribution: "full_campaign_cost"`
   - **Замечание:** на практике Yandex `CostPerConversion` = `Cost/ConversionsForGoal` — та же атрибуция, что у нас, просто с его стороны. Реальной per-goal атрибуции в API нет. Флаг сохраняем для прозрачности.
7. **Campaign-level filter** — отсекаем по `min_clicks`, `min_cost`.
8. **Per-goal stats** для каждой цели (form, call):
   - `included = [b for b in campaignLevelKept if conv_for_goal >= min_conv]`
   - `weighted_mean = Σ cost / Σ conv`
   - `p25, p50, p75, IQR` распределения per-campaign CPA
   - **При n ≥ 5** — IQR-отсечка Тьюки `[p25 − 1.5·IQR, p75 + 1.5·IQR]`, `robust_mean` без выбросов
   - **При n < 5** — IQR пропускается, `robust_mean = weighted_mean`, `outlier_note: "n=X < 5, IQR пропущена"`
   - Если **обе** цели пусты после фильтра — авто-релакс `min_conversions → 1`, флаг `filter_relaxed=true`
9. **Benchmark CPA** (`benchmark_form_cpa`, `benchmark_call_cpa`) = `robust_mean`.
10. **Cross-goal estimation** (если одна из целей пуста):
    - Только call → `benchmark_form = 2 × benchmark_call` (внутригородское соотношение, сильнее network avg)
    - Только form → `benchmark_call = 0.5 × benchmark_form`
    - `source = "mixed"`, `recommended_value.source = "robust_with_2x_heuristic"`
11. **Network fallback** (если ни одной кампании не прошло фильтр):
    - Тир города из `cityConfig.Tier` (`tier_1` 1M+, `tier_2` 300K-1M, `tier_3` <300K)
    - Lookup в `server/data/network_benchmarks.json` по `(theme, tier)`, иначе avg по тиру, иначе default
    - `source = "network_average"`, `confidence = "low"`
12. **Target CPA** = `override ?? benchmark`. `priority_goals_json` строится из `target`.
13. **Breakdown by theme** (если не задан input theme): группируем included по theme, считаем `weighted_mean` per group. Возвращаем `breakdown_by_theme: {theme: {form_cpa, call_cpa, n_campaigns, conv}}`.
14. **Trend 7d** (всегда): отдельный fetch отчёта за 7d, `weighted_mean`, сравнение с периодом:
    - >+15% к периодному CPA → `degrading`
    - <−15% → `improving`
    - в пределах ±15% → `stable`
    - <3 conv в 7d по обеим целям → `insufficient_data`
    - Combined signal: degrading если хоть одна degrading; improving если обе improving; иначе stable

### Confidence (per-goal, общий = min)

- `high` — total_conv ≥ 50 **и** included_count ≥ 3 кампаний
- `medium` — ≥ 15 conv **и** ≥ 2 кампании
- `low` — всё остальное, включая `network_average`

### Прозрачность в выдаче

| Поле | Что показывает |
|---|---|
| `actual_days_used` + `window_reason` | Какое окно использовалось и почему |
| `cost_attribution` | Метод атрибуции расхода на цели |
| `campaigns_included` / `campaigns_excluded` | Кампании в расчёте + причины отсечки |
| `statistics.form` / `.call` | Полная статистика per-goal: weighted, robust, квартили, outliers, confidence |
| `filter_applied` / `filter_relaxed` / `relaxed_reason` | Фактические пороги |
| `breakdown_by_theme` | Per-theme weighted CPA (когда не задан input theme) |
| `trend_7d` | 7d vs период, signal |
| `recommended_value.source` | `robust_mean` / `robust_with_2x_heuristic` / `network_average` / `target_override` |

### Backward compatibility

- `avg_form_cpa` / `avg_call_cpa` — **weighted mean** по всем fetched-кампаниям с конверсиями (старая семантика).
- `priority_goals_json` теперь строится из `target_*_cpa` (= `robust_mean` по умолчанию). Может отличаться от старого `avg_*_cpa` за счёт фильтра + отсечки выбросов. Это и есть цель доработки.

Типичное соотношение form_cpa / call_cpa: **2-3×** (заявка дороже звонка).

### Когда использовать `theme`

- **Конкретная стратегия для конкретной кампании** → передавай `theme` совпадающий с тематикой этой кампании. Получишь точный таргет, не размытый по другим тематикам города.
- **Общий обзор города** → без `theme`. Получишь общий benchmark + `breakdown_by_theme` для понимания, какая тематика на каком уровне.

### Network benchmarks table

Файл: `server/data/network_benchmarks.json` (монтируется в Docker). Структура:

```json
{
  "updated_at": "2026-04-23",
  "benchmarks": {
    "vtorichka": {"tier_1": {"form": 3500, "call": 1500}, "tier_2": ..., "tier_3": ...},
    "novostroyki": {...},
    ...
  },
  "default": {"form": 3000, "call": 1500}
}
```

Текущие значения — образованные оценки по рынку недвижимости. Phase 4 (cron) обновит их еженедельным агрегатом по 70 городам.

### Бенчмарки по городам (обновлено 2026-04-23, live прогон 24 городов)

#### Общий benchmark города (без theme)

| Город | Tier | Period | Source | Confidence | Form CPA | Call CPA | Ratio |
|---|---|---|---|---|---|---|---|
| Санкт-Петербург | tier_1 | 90d | campaigns | low | 14430 | 3330 | 4.3 |
| Москва | tier_1 | — | network | low | 3722 | 1644 | 2.3 |
| Екатеринбург | tier_1 | — | network | low | 3722 | 1644 | 2.3 |
| Новосибирск | tier_1 | — | network | low | 3722 | 1644 | 2.3 |
| Челябинск | tier_1 | 30d | campaigns | low | 3943 | 1850 | 2.1 |
| Омск | tier_1 | 30d | campaigns | **medium** | 2858 | 1457 | 2.0 |
| Самара | tier_1 | — | network | low | 3722 | 1644 | 2.3 |
| Ростов-на-Дону | tier_1 | 90d | mixed (2×) | low | 2892 | 1446 | 2.0 |
| Нижний Новгород | tier_1 | — | network | low | 3722 | 1644 | 2.3 |
| Тюмень | tier_2 | **14d** (narrow) | campaigns | low | 6721 | 1767 | 3.8 |
| Сургут | tier_2 | 30d | campaigns | low | 4392 | 1536 | 2.9 |
| Набережные Челны | tier_2 | — | network | low | 2689 | 1183 | 2.3 |
| Нижний Тагил | tier_3 | — | network | low | 1889 | 822 | 2.3 |
| Курган | tier_3 | — | network | low | 1889 | 822 | 2.3 |
| Новый Уренгой | tier_3 | 90d | mixed (2×) | low | 7534 | 3767 | 2.0 |
| Тобольск | tier_3 | — | network | low | 1889 | 822 | 2.3 |
| Ишим | tier_3 | 60d | campaigns | low | 2101 | 1051 | 2.0 |
| Ханты-Мансийск | tier_3 | — | network | low | 1889 | 822 | 2.3 |
| Тамбов | tier_3 | 60d | mixed (2×) | **medium** | 8252 | 4126 | 2.0 |
| Саранск | tier_3 | 90d | mixed (2×) | low | 3432 | 1716 | 2.0 |
| Нальчик | tier_3 | — | network | low | 1889 | 822 | 2.3 |
| Якутск | tier_3 | — | network | low | 1889 | 822 | 2.3 |
| Ялта | tier_3 | — | network | low | 1889 | 822 | 2.3 |
| Дмитров | tier_3 | — | network | low | 1889 | 822 | 2.3 |
| Обнинск | tier_3 | 30d | mixed (2×) | **high** | 610 | 305 | 2.0 |

> Не тестировалось (counter_id или client_login пусты): Краснодар, Пермь, Казань, Красноярск, Уфа, Хабаровск, Владивосток, Тула, Стерлитамак.

#### Per-theme бенчмарки (breakdown из live данных)

| Город | Theme | Form CPA | Call CPA | N camp | form conv | call conv |
|---|---|---|---|---|---|---|
| Санкт-Петербург | zagorodka | 14430 | 3330 | 1 | 3 | 13 |
| Челябинск | imidzh | 47234/10768 (p50) | 1850 | 3 | 5-7 | 42 |
| Омск | agency | 2060 | 1236 | 1 | 18 | 30 |
| Омск | vtorichka | 5988 | 1871 | 2 | 5 | 16 |
| Сургут | vtorichka | 4392 | 1198 | 1 | 3 | 11 |
| Сургут | arenda | — | 1219 | 2 | 0 | 17 |
| Сургут | novostroyki | — | 4571 | 1 | 0 | 3 |
| Ростов-на-Дону | zagorodka | 5783 | 1446 | 1 | 1 | 4 |
| Тюмень | zagorodka | 6721 | 2385 | 2 | 11 | 31 |
| Тюмень | arenda | 12395 | 2066 | 1 | 1 | 6 |
| Тюмень | commerce | 4063 | 1354 | 1 | 1 | 3 |
| Тюмень | vtorichka | — | 1500 | 1 | 0 | 3 |
| Новый Уренгой | vtorichka | — | 3767 | 1 | 0 | 17 |
| Ишим | zagorodka | 2101 | 1051 | 1 | 6 | 12 |
| Тамбов | vtorichka | 23623 | 3634 | 1 | 2 | 13 |
| Тамбов | novostroyki | — | 4925 | 1 | 0 | 8 |
| Саранск | vtorichka | 24025 | 1716 | 1 | 2 | 28 |
| Обнинск | vtorichka | 23098 | 366 | 2 | 1 | 67 |
| Обнинск | zagorodka | 7855 | 201 | 1 | 1 | 39 |

#### Ключевые наблюдения

- **Form-конверсий КРАЙНЕ мало** в большинстве городов. Даже в tier_1 Омск/Челябинск все form_cpa вычислены по 1-5 конверсиям. Confidence по форме почти везде `low`.
- **Call-конверсии собираются на порядок лучше** (в Обнинске 106 call vs 2 form, в Саранске 28 call vs 2 form).
- **`mixed` source (heuristic 2× call)** применён в 5 городах — там form_cpa в `priority_goals` — это оценка, не факт.
- **`network_average`** у 14 городов — нет активных кампаний с данными. Значения берутся из `server/data/network_benchmarks.json` по тиру.
- **Соотношение form/call** в российской недвижимости у этажей в среднем **2.0-2.3×**. В Тюмени/zagorodka — 3.8×, в СПб/zagorodka — 4.3× (дорогой дом требует формальной заявки).
- **Аномалия Обнинск:** call_cpa=305₽ при 106 conv за 30d — это очень дёшево (tier_3 — вероятно low competition + хорошая работа с коллтрекингом). `confidence=high` подтверждает надёжность.
- **Тюмень auto-window=14d** (сузился с 30 из-за >150 conv) — свежий сигнал.
- **9 городов без counter_id/client_login** в `cities` map (Краснодар, Пермь, Казань, Красноярск, Уфа, Хабаровск, Владивосток, Тула, Стерлитамак) — либо агентством ещё не заведены, либо счётчик не привязан. Запрос ценностей для них вернёт ошибку.

Типичное соотношение form_cpa / call_cpa: **2-3×** (заявка дороже звонка).
