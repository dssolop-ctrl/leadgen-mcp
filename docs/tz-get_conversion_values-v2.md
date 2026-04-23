# ТЗ: доработка `get_conversion_values` — точный и устойчивый расчёт CPA-бенчмарка

**Статус:** черновик
**Автор:** leadgen skill audit 2026-04-23
**Файл реализации:** `server/platform/direct/benchmarks.go`
**MCP-инструмент:** `get_conversion_values`
**Документация:** `.claude/skills/leadgen/mcp/goals.md`

---

## 1. Контекст и проблема

Сейчас инструмент выполняет двойную функцию:
- **Возвращает бенчмарк CPA** — используется агентом в аудитах как «нужная CPA по цели для города».
- **Генерирует `value` для `priority_goals`** — используется стратегией автобиддинга.

Расчёт: взвешенное среднее `Σ cost / Σ conversions` за 30 дней по всем ON-кампаниям города с атрибуцией LYDC, с фолбэком на 1500₽/3000₽ «средних по сети» при отсутствии данных.

### Известные дефекты

| # | Дефект | Последствие |
|---|---|---|
| 1 | В числитель идёт **полный `Cost` кампании**, а не стоимость кликов по конкретной цели. Кампания с 1 form и 100 call кинет весь свой cost в числитель form_cpa. | Form-CPA сильно переоценён в кампаниях с перекосом целей. |
| 2 | **Нет защиты от выбросов.** Одна кампания с расходом 200 000₽ и 3 конверсиями перетянет средневзвешенное на себя. | Бенчмарк нерепрезентативен. |
| 3 | **Нет фильтра по объёму данных.** Кампания с 1 конверсией за 30 дней участвует в расчёте наравне с кампанией со 150. | Шумовой бенчмарк. |
| 4 | **Все ON-кампании города мешаются в один расчёт** — вторичка + новостройки + аренда + HR дают усреднённую «среднюю по больнице». | Для конкретной тематики бенчмарк не подходит. |
| 5 | **Fallback 1500₽/3000₽ хардкод** — не зависит от тематики, тира города, сезона. | В тонких нишах ошибка в 2-3 раза. |
| 6 | **Окно 30 дней фиксировано.** В малых городах за 30 дней может быть 0-2 конверсии. | Либо отваливается на network_average, либо считает по 1 точке. |
| 7 | **Нет метрики уверенности.** Caller получает одну цифру и не знает, основана она на 3 или на 300 конверсиях. | Агент не может корректно интерпретировать расхождение CPA кампании vs бенчмарк. |
| 8 | **Смешаны бенчмарк и value для стратегии.** Value стратегии — это бизнес-решение (сколько мы готовы платить), а не факт (сколько стоит сейчас). | Невозможно задать таргет-CPA, отличный от текущей средневзвешенной. |

---

## 2. Цели доработки

1. **Точность.** Бенчмарк отражает реальную CPA по цели, а не артефакт арифметики.
2. **Сегментация.** Бенчмарк считается в разрезе тематики и тира города.
3. **Робастность.** Выбросы и кампании с малыми данными не искажают результат.
4. **Прозрачность.** Caller получает не только число, но и доверительный интервал, распределение, причину выбранного значения.
5. **Разделение ролей.** `benchmark_cpa` (факт) и `target_cpa` (что хотим) — разные сущности.
6. **Обратная совместимость.** Существующий `priority_goals_json` продолжает работать.

---

## 3. Функциональные требования

### FR-1. Per-goal cost attribution (корректный числитель)

**Проблема:** `Cost` кампании не принадлежит одной цели.

**Решение:** использовать `CostPerConversion` из отчёта Яндекс Директа (уже поле API) вместо деления `Cost / Conversions` самим. Для каждой цели в отчёте `CAMPAIGN_PERFORMANCE_REPORT` со списком `Goals` в запросе Яндекс возвращает отдельную колонку `CostPerConversion_<goal_id>` — это более корректная атрибуция, которую считает Яндекс внутри.

**Если такой колонки нет** (не все ReportType её возвращают): оставить текущую формулу, но добавить флаг `cost_attribution: "full_campaign_cost"` в output для прозрачности.

**Альтернатива:** `CAMPAIGN_ADGROUP_AD_CRITERION_PERFORMANCE_REPORT` с тем же фильтром — позволяет получить cost в разрезе групп, что даёт более точную картину, если кампания организована по тематическим группам.

### FR-2. Фильтр по минимальному объёму данных

Кампания включается в расчёт цели, только если:
- `conversions_for_goal >= MIN_CONV` (по умолчанию `5`)
- `clicks >= MIN_CLICKS` (по умолчанию `100`)
- `cost >= MIN_COST` (по умолчанию `1000₽`)

Кампании, не прошедшие фильтр, возвращаются в поле `excluded_campaigns` с причиной — чтобы caller видел, почему их нет в расчёте.

Параметризовать через input:

```
min_conversions: int (default 5)
min_clicks: int (default 100)
min_cost: float (default 1000)
```

### FR-3. Робастная статистика (медиана + IQR)

Помимо взвешенного среднего считать:

- `p50_form_cpa` — медиана CPA по включённым кампаниям
- `p25_form_cpa` / `p75_form_cpa` — квартили
- `iqr_form_cpa = p75 - p25`
- Отсекать выбросы по правилу Тьюки: `[p25 - 1.5*IQR, p75 + 1.5*IQR]`
- `robust_mean_form_cpa` — среднее после отсечения выбросов

Рекомендуемое value для priority_goals брать из `robust_mean`, а не сырого среднего.

### FR-4. Сегментация по тематике

Добавить параметр:

```
theme: "vtorichka" | "zagorodka" | "novostroyki" | "ipoteka" | "brand" | "arenda" | "commerce" | null
```

Если `theme != null`:
- Парсить имена ON-кампаний и фильтровать только по совпадающей тематике (логика из `.claude/skills/leadgen/config/campaign_naming.md`).
- Возвращать per-theme бенчмарк.

Если `theme == null`:
- Возвращать общий бенчмарк + разбивку по тематикам в поле `breakdown_by_theme`.

### FR-5. Динамическое окно

Добавить параметр `auto_window: bool` (default `true`).

Алгоритм:
- Старт с `days=30`
- Если после FR-2 фильтра суммарных конверсий < 20: расширить окно до 60 дней
- Если < 10 после 60: расширить до 90
- Если > 150 в 30: сузить до 14 (более свежий сигнал)
- Возвращать `actual_days_used` и `why` в output

### FR-6. Confidence score

Добавить в output:

```json
{
  "confidence": "high | medium | low",
  "confidence_reason": "<human-readable>",
  "total_conversions": 87,
  "campaigns_included": 6,
  "campaigns_excluded": 3
}
```

Правила:
- `high`: ≥ 50 конверсий и ≥ 3 кампании
- `medium`: ≥ 15 конверсий и ≥ 2 кампании
- `low`: всё остальное, **включая случай, когда сработал network_average**

### FR-7. Разделение benchmark vs target

Output возвращает **две** CPA:

```json
{
  "benchmark_form_cpa": 3301,
  "target_form_cpa": 3301,
  ...
}
```

Добавить параметр `target_cpa_override: {form: int, call: int}` — если задан, `target_*_cpa` = override, иначе = benchmark. `priority_goals_json` всегда строится из `target_*_cpa`.

### FR-8. Улучшенный network fallback

Ввести таблицу network-бенчмарков: `library/network_benchmarks.json`, обновляемую еженедельным cron-job по всем 70 городам:

```json
{
  "updated_at": "2026-04-20",
  "benchmarks": {
    "vtorichka": {"tier_1": {"form": 2800, "call": 1200}, "tier_2": {"form": 1800, "call": 800}},
    "novostroyki": {"tier_1": {"form": 4500, "call": 2000}}
  }
}
```

Тир города — из `config/counters.md` или по численности населения (уже часть `get_city_config`).

Fallback-логика:
1. Нет данных по городу → смотрим `theme + tier` в таблице
2. Нет тематики в таблице → смотрим `tier` (без темы, усреднение по темам)
3. Нет тира → хардкод 1500/3000 (как сейчас)

### FR-9. Тренд

Считать CPA за `[actual_days_used]` и за последние 7 дней отдельно. Возвращать:

```json
{
  "cpa_trend_7d": {
    "form_cpa_7d": 3100,
    "form_cpa_vs_period": "-6%",
    "call_cpa_7d": 1500,
    "call_cpa_vs_period": "+4%",
    "signal": "improving | stable | degrading"
  }
}
```

`signal` = `degrading`, если CPA 7d выше периодной на > 15% при сопоставимом объёме.

### FR-10. Исключение кампаний в обучении

Кампания исключается из расчёта, если:
- `start_date > today - 14 days` (свежий запуск)
- Была смена стратегии за последние 14 дней (`get_change_history`)
- `StatusClarification != "NONE"` (есть замечания модерации)

---

## 4. Изменения в API инструмента

### Input (добавить):

| Параметр | Тип | Default | Описание |
|---|---|---|---|
| `theme` | string? | null | Фильтр тематики (vtorichka/novostroyki/...) |
| `min_conversions` | int | 5 | Минимум конверсий на кампанию для включения |
| `min_clicks` | int | 100 | Минимум кликов на кампанию |
| `min_cost` | float | 1000 | Минимум расхода на кампанию |
| `auto_window` | bool | true | Автоматический выбор окна |
| `days` | int | 30 | Жёстко зафиксировать окно (игнорируется при auto_window) |
| `target_cpa_override` | object? | null | `{form: int, call: int}` — явный таргет для стратегии |
| `exclude_learning` | bool | true | Не включать кампании первых 14 дней после запуска/смены стратегии |

### Output (новые поля):

```json
{
  "city": "Омск",
  "period": "2026-03-24 — 2026-04-22",
  "actual_days_used": 30,
  "window_reason": "30d sufficient: 87 conversions collected",
  "source": "campaigns | network_average | mixed",

  "benchmark_form_cpa": 3301,
  "benchmark_call_cpa": 1447,
  "target_form_cpa": 3301,
  "target_call_cpa": 1447,
  "ratio": 2.3,

  "statistics": {
    "form": {
      "weighted_mean": 3301,
      "robust_mean": 3180,
      "p25": 2450, "p50": 3100, "p75": 4200,
      "iqr": 1750,
      "outliers_removed": 1,
      "total_conversions": 87,
      "total_cost": 287187
    },
    "call": {}
  },

  "confidence": "high",
  "confidence_reason": "87 form + 124 call conv across 6 campaigns",

  "trend_7d": {
    "form_cpa_7d": 3100,
    "form_cpa_vs_period": "-6%",
    "signal": "stable"
  },

  "campaigns_included": [
    {"id": 123, "name": "...", "form_cpa": 3100, "call_cpa": 1400, "form_conv": 12, "call_conv": 25}
  ],
  "campaigns_excluded": [
    {"id": 456, "name": "...", "reason": "conversions=2 < min_conversions=5"}
  ],

  "breakdown_by_theme": {
    "vtorichka": {"form_cpa": 3180, "call_cpa": 1400, "n_campaigns": 2},
    "novostroyki": {"form_cpa": 4500, "call_cpa": 1900, "n_campaigns": 1}
  },

  "recommended_value": {
    "form_value_rubles": 3301,
    "call_value_rubles": 1447,
    "source": "robust_mean | target_override | network_average"
  },
  "priority_goals_json": "[{\"goal_id\":123,\"value\":3301}]",

  "form_goal_id": 123,
  "form_goal_name": "Общая (все лиды)",
  "call_goal_id": 456,
  "call_goal_name": "Коллтрекинг, реальный звонок",
  "call_goal_type": "received_real_calls"
}
```

---

## 5. Алгоритм (pseudo)

```
1. resolve_goals(counter_id) → form_goal_id, call_goal_id
2. campaigns = get_active_campaigns(client_login)
   if theme: campaigns = filter_by_theme(campaigns, theme)
   if exclude_learning: campaigns -= get_learning_campaigns(campaigns)

3. days = 30
   loop (до 3 раз):
       stats = get_report(campaigns, days, goals, LYDC)
       benchmarks = parse(stats, min_conv, min_clicks, min_cost)
       if auto_window and total_conv(benchmarks) < 20 and days < 90:
           days *= 2
           continue
       break

4. Для каждой цели:
   included = [b for b in benchmarks if b.conv_for_goal >= min_conv]
   weighted_mean = Σ cost / Σ conv
   p25, p50, p75 = quantiles([b.cpa for b in included])
   iqr = p75 - p25
   robust = [b for b in included if p25 - 1.5*iqr <= b.cpa <= p75 + 1.5*iqr]
   robust_mean = Σ robust.cost / Σ robust.conv

5. trend_7d:
   stats_7d = get_report(campaigns, 7, goals, LYDC)
   compute cpa_7d, compare vs period

6. confidence:
   если included >= 3 и total_conv >= 50 → high
   если included >= 2 и total_conv >= 15 → medium
   иначе → low

7. source:
   если все цели имеют robust_mean → "campaigns"
   если часть из network_benchmarks.json → "mixed"
   если всё из network → "network_average"

8. target_cpa = target_cpa_override ?? robust_mean ?? network_benchmark
9. priority_goals_json = build from target_cpa
10. возврат полного output
```

---

## 6. Фазы внедрения

### Phase 1 — MVP (1-2 дня)

- FR-2 (min-thresholds)
- FR-3 (медиана + IQR outliers)
- FR-6 (confidence)
- FR-7 (разделение benchmark/target)
- FR-10 (исключение learning)

Не трогаем: FR-1 (per-goal cost), FR-4 (theme), FR-5 (auto-window), FR-8 (network table), FR-9 (trend).

Критерий готовности: текущий аудит Омска возвращает confidence=high, робастное среднее в пределах ±10% от текущего weighted_mean, outliers и excluded_campaigns прозрачно видны в output.

### Phase 2 — сегментация (2-3 дня)

- FR-4 (тематики)
- FR-5 (динамическое окно)
- FR-8 (network_benchmarks.json + seed из прогона по 70 городам)

Критерий готовности: `get_conversion_values(city=Тамбов, theme=vtorichka)` возвращает отдельный бенчмарк, а не общий по городу. При 0 данных по теме — используется network-медиана по тиру.

### Phase 3 — продвинутая атрибуция и тренд (1-2 дня)

- FR-1 (per-goal cost via `CostPerConversion_<goal>` column)
- FR-9 (trend_7d)

Критерий готовности: trend_7d показывает signal=degrading на синтетическом тесте кампании с 2× ухудшением за неделю.

### Phase 4 — cron + кеш (1 день)

- Cron-job `refresh_network_benchmarks` раз в неделю: проходит по всем 70 городам, считает `get_conversion_values` без фильтра по городу, аггрегирует в `network_benchmarks.json`.
- TTL-кеш на уровень инструмента: результат для `(city, theme, days)` кешируется на 1 час (CPA за прошлые периоды не меняется).

---

## 7. Обратная совместимость

- Существующие caller'ы используют только `priority_goals_json`, `form_goal_id`, `call_goal_id`, `avg_form_cpa`, `avg_call_cpa` → все эти поля остаются в output.
- Поля `avg_form_cpa` и `avg_call_cpa` продолжают возвращать **weighted mean** (не robust_mean), чтобы не ломать существующие отчёты.
- Новые поля `benchmark_*_cpa`, `target_*_cpa`, `statistics.*` — дополнительные.
- Параметр `days` продолжает работать для явного указания окна (при `auto_window=false`).

---

## 8. Тесты

### Unit-тесты (`benchmarks_test.go`)

1. **TestQuantiles** — корректный p25/p50/p75 на наборах из 1, 3, 5, 10 кампаний.
2. **TestOutlierRemoval** — IQR-отсечение: кампания с CPA = 10× медианы исключается.
3. **TestMinConversionFilter** — кампания с conv=4 и min_conv=5 попадает в excluded.
4. **TestNetworkFallbackByTier** — при `source=network_average` для tier_1/theme=vtorichka берётся значение из таблицы, а не 1500/3000.
5. **TestAutoWindow** — при total_conv=10 в 30d окно расширяется до 60d.
6. **TestLearningExclusion** — кампания со StartDate < today-14d исключается с reason="learning_period".
7. **TestTargetOverride** — при задании target_cpa_override=1000 priority_goals_json содержит value=1000, а benchmark остаётся исходным.

### Интеграционные (на живых данных Омска)

1. Прогон `get_conversion_values(city=Омск, theme=vtorichka)` с Phase 1:
   - `confidence == "high"`
   - `benchmark_form_cpa` в диапазоне [2800, 3800]
   - `robust_mean_form_cpa` ≈ `weighted_mean_form_cpa` ±10%
   - `campaigns_included` ≥ 1, `campaigns_excluded` содержит причины

2. Прогон для малого города (например, Тамбов) → `confidence ∈ {medium, low}`, source может быть `mixed`.

3. Прогон без `theme` → `breakdown_by_theme` содержит вторичку, новостройки, ипотеку отдельно.

### Регрессионные

- До/после прогон на последнем аудите кампании 706224151: сравнить старый `avg_form_cpa` (3301₽) с новым `benchmark_form_cpa`. Разница > 15% — требует ручного ревью.

---

## 9. Документация

После Phase 1 обновить:
- `.claude/skills/leadgen/mcp/goals.md` — раздел «Ценность конверсий» → описать robust_mean, confidence, разделение benchmark/target.
- `METRIKA-ADS-RULES.md` — в правила по бенчмаркам добавить ссылку на новый output format.
- Примеры использования в CLAUDE.md / SKILL.md — показать `theme` и `target_cpa_override`.

---

## 10. Риски

1. **`CostPerConversion_<goal>` может возвращать NULL** для кампаний, где цель сработала, но атрибуция не определена. Нужен fallback на старую формулу с флагом в output.
2. **Парсинг тематики из имени кампании** хрупок. Нужен whitelist паттернов + явный fallback «theme: unknown» без отбраковки кампании, если пользователь не передал `theme`.
3. **Network benchmarks table** требует отдельного процесса поддержки. Без cron-job быстро устареет.
4. **Confidence=low в малых городах** может сломать существующие автоматические сценарии, которые ожидают число. Нужно в Phase 1 не менять fallback-поведение — только добавить метрику.

---

**Итог:** Phase 1 (MVP) закрывает 80% проблем аудита (выбросы, прозрачность, confidence) при минимальных изменениях кода. Phase 2-4 — для полноценной сегментации и автообновляемой нормативной базы.
