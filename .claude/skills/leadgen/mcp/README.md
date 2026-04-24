# `mcp/` — библиотеки, не покрытые MCP-инструментами

Эта папка хранит **только те справочники, которые не возвращаются MCP-инструментами**. Всё, что есть в API (`get_*`), вынесено из папки — оно дублировало контекст.

## Что осталось

- **[`negative_keywords.md`](negative_keywords.md)** — типовые минус-слова по тематикам, блок «Чужие города РФ», шаблоны сборки (~150+ слов на кампанию). Используется на шагах C4/R4 и во всех O3.1/O3.6 при расширении минусов.
- **[`blocked_placements.md`](blocked_placements.md)** — стандартный чёрный список площадок РСЯ Этажей (400+). Используется на OR.1 как база для `add_blocked_placements`, а также при создании новой РСЯ как стартовый набор.

## Что удалено (апр 2026) и чем заменено

| Был файл | Источник теперь |
|---|---|
| `counters.md` | MCP `get_city_config(city)` → counter_id, domain, geo_region_id |
| `goals.md` | MCP `get_conversion_values(client_login, counter_id, theme)` → goal_id, target CPA, priority_goals_json |
| `utm_reference.md` | MCP `get_utm_config(city, theme, placement)` → готовый `tracking_params` |
| `semantic_clusters.md` | MCP `get_semantic_cluster(theme)` → модификаторы, объекты, минус-слова |
| `sitelinks.md` | MCP `get_sitelink_templates(theme, city, placement)` → наборы быстрых ссылок |
| `site_filters.md` | MCP `build_landing_url(…)` и `get_site_filters(…)` (SQLite-backed) |

**Правило:** сначала MCP — не читай файл скилла, если нужная информация есть в API. MCP не тратит контекст, файл — тратит.
