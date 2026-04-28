# Ветка A — Анализ кампаний

> Вспомогательный файл скилла `leadgen-codex`. Загружается, когда пользователь запросил анализ/аудит/статистику кампании (Роутер 1 → АНАЛИЗ).
>
> Связанные: [`../SKILL.md`](../SKILL.md), [`optimize-search.md`](optimize-search.md), [`optimize-rsya.md`](optimize-rsya.md).

---

## ВЕТКА 2: АНАЛИЗ кампаний
<!-- A — ветка анализа -->

> **Статус:** Полностью рабочая ветка. Внешний дашборд (MCP #2) пока не подключен — используем только Директ API + Метрику.

### Источники данных

| Источник | Инструмент | Статус |
|---|---|---|
| **Яндекс Директ API** | `get_campaign_stats` | ✅ Работает |
| **Яндекс Метрика API** | `metrika_get_direct_report` | ✅ Работает |
| **Внешний дашборд** | _(будущий MCP #2)_ | ⏳ Не подключен |

### Flow

```
// A1 — определение скоупа
1. ОПРЕДЕЛИТЬ СКОУП
   - Одна кампания (по ID или имени) → детальный аудит
   - Все кампании города → дашборд
   - Все города → сводка

// A2 — сбор статистики
2. СБОР СТАТИСТИКИ
   Для одной кампании — `summarize_campaign_snapshot` (один вызов вместо
   четырёх) даёт базовые поля + 7-дневные метрики:
   summarize_campaign_snapshot(
     client_login=<login>, campaign_id=<id>,
     last_n_days=7, goal_ids=<основные цели>, attribution="LYDC"
   )

   Для дашборда по нескольким кампаниям — `get_campaign_stats`:
   get_campaign_stats(
     client_login=<login>,
     campaign_ids=<id1,id2,...>,  // или без → все кампании
     date_from=<7/30 дней назад>,
     date_to=<вчера>,
     field_names="CampaignName,Impressions,Clicks,Cost,Ctr,AvgCpc,Conversions,CostPerConversion",
     goal_ids=<goal_id>,
     attribution="LYDC"
   )

// A3 — данные метрики
3. ДАННЫЕ МЕТРИКИ (для детального аудита)
   metrika_get_direct_report(
     counter_id=<id>,
     account="etagi.agent",
     date1=<7 дней назад>,
     date2=<вчера>,
     utm_campaign=<campaign_id>
   )

// A4 — оценка здоровья
4. ОЦЕНКА ЗДОРОВЬЯ
   | Метрика | GOOD | ATTENTION | CRITICAL |
   |---------|------|-----------|----------|
   | CPA | < порог из benchmarks | < порог × 1.5 | > порог × 1.5 |
   | CTR (поиск) | > 5% | 3-5% | < 3% |
   | CTR (РСЯ) | > 0.5% | 0.3-0.5% | < 0.3% |
   | Отказы | < 30% | 30-50% | > 50% |

   // Триггер SERP (опционально, если tokens.env настроен):
   // если CPA = ATTENTION/CRITICAL ИЛИ CTR упал ≥20% vs прошлая неделя —
   // запустить serp-monitor/scripts/track_position.sh по топ-3 ключам
   // группы, чтобы проверить: просадка от падения позиции или от самой группы.
   // Результат добавить в секцию «Позиции в выдаче» файла campaigns/<slug>.md.

// A5 — диагностика проблемных
5. ДИАГНОСТИКА ПРОБЛЕМНЫХ
   - Поисковые запросы: `summarize_search_queries(goal_ids=<цели>, waste_min_clicks=5)`
     → блоки `waste` (расход без эффекта) и `top_by_conversions` (где работает).
     Полный TSV — get_search_queries, только если summary не хватает.
   - CTR объявлений: `summarize_ads_performance(top_n=10)` →
     блок `low_ctr_candidates_for_ab` = что заменить (A/B).
   - Ключевые: get_criteria_stats → неэффективные фразы (для них summary пока нет —
     если станет узким местом, добавим summarize_criteria).

   // Если из A4 пришёл SERP-триггер: прогнать track_position.sh по 3-5 ключам
   // из get_criteria_stats с низким CTR и сопоставить позицию с показами.
   // Частый паттерн: ключ выпал из Premium (#1-4 → #7+) → показов стало в 5-10× меньше
   // → CPA растёт не из-за семантики, а из-за ставки. Это меняет рекомендацию
   // с «убрать ключ» на «поднять ставку / пересмотреть стратегию».

// A6 — отчёт
6. ОТЧЁТ
   - Сводная таблица с светофором
   - Конкретные рекомендации (входные данные для ОПТИМИЗАЦИИ)
   - Обновление campaigns/<utm>.md
```

---


