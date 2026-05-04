# Playbook Contract: использование leadgen/branches как shared playbooks

> Этот файл — **контракт зависимости** автопилота от ручного скилла `leadgen`.
> Любые правки anchors / структуры в `leadgen/branches/` требуют проверки этого файла.

## Концепция

Автопилот = **policy/orchestration** (анализ, память, permissions, reconcile, apply).
`leadgen/branches/` = **shared operation playbooks** (конкретные шаги создания/оптимизации).

Когда автопилот в результате reconcile решает создать кампанию или провести оптимизацию — он **читает** соответствующий branch из `leadgen` как playbook (источник конкретных шагов через стабильные anchors). Это не runtime-вызов скилла `/leadgen`, а чтение md-файла как справочника.

## Маппинг

| Действие автопилота | Playbook | Anchor / шаг |
|---|---|---|
| `campaign.create_draft.in_existing_topic` (search) | `.claude/skills/leadgen/branches/create-search.md` | Шаги 1-10 (создание DRAFT). Шаг 11 (DRAFT-only финиш) — обязателен. |
| `campaign.create_draft.in_existing_topic` (rsya) | `.claude/skills/leadgen/branches/create-rsya.md` | Шаги 1-10. Финиш в DRAFT. |
| `campaign.create_draft.in_new_topic` (search/rsya) | те же | + проверка demand через wordstat в onboarding |
| `campaign.activate_existing_draft` | inline в `branches/apply.md` | manage_campaigns(state=ON), без обращения к leadgen |
| `keyword.add_in_existing_group` | `optimize-search.md` подветка O3.6 | поиск/расширение групп |
| `keyword.add_new_group_in_existing_topic` | `optimize-search.md` O3.6 | то же |
| `negatives.add_from_search_queries` | `optimize-search.md` секция «минусация» | автоматизировать |
| `placement.block.low_performing` | `optimize-rsya.md` секция «площадки» | использовать blocked_placements |
| `creative.generate_new_image` | `create-rsya.md` секция «картинки» + `references/image_prompts.md` | OpenRouter generation |
| `ad.add_new_variant` (search/rsya) | `library/{titles,texts,banner_titles,banner_texts}.md` | пул вариантов |
| `extension.add_sitelinks` | `library/extensions.md` | шаблоны |

## Стабильные anchors (чек-лист)

Когда правишь `leadgen/branches/`, сохраняй:

- `create-search.md`: «Шаг 1», «Шаг 7», «Шаг 11. Финальное состояние: DRAFT».
- `create-rsya.md`: те же шаги 1-11 + «РСЯ-специфика».
- `optimize-search.md`: подветка «O3.6 — расширение», секция «минусация».
- `optimize-rsya.md`: секция «площадки», секция «модераторы».

Если anchors сдвинулись — обнови этот файл и `references/action_catalog.md`.

## Shared references / library

Оба скилла читают как md-источники (без runtime-вызовов):

| Файл | Назначение |
|---|---|
| `leadgen/references/city_benchmarks.md` | tier-based budget/CPA пороги |
| `leadgen/references/lessons_registry.md` | агрегированные уроки (только чтение из автопилота) |
| `leadgen/references/copy_blacklist.md` | запрещённые слова в copy |
| `leadgen/references/rsya_defaults.md` | дефолты РСЯ |
| `leadgen/references/site_structure.md` | структура сайта Этажей и валидные топики |
| `leadgen/references/ui_naming.md` | naming convention для кампаний (УЧИТЫВАТЬ ownership labels отдельно) |
| `leadgen/references/image_prompts.md` | промпты для генерации картинок |
| `leadgen/library/banner_titles.md` | пул заголовков для банеров |
| `leadgen/library/banner_texts.md` | пул текстов |
| `leadgen/library/titles.md`, `texts.md` | пул для search |
| `leadgen/library/extensions.md` | шаблоны расширений |
| `leadgen/library/display_urls.md` | display URLs |
| `leadgen/library/selling_modifiers.md` | модификаторы для текстов |

## Контракт зависимости

- **Автопилот не вызывает `/leadgen` в runtime.**
- **Автопилот не редактирует файлы `leadgen/`.** Если автопилот накопил правило, которое стоит закодировать — оно идёт в `autopilot/learnings/proposed/`, потом в monthly digest предложением специалисту.
- **При изменении anchors в `leadgen/branches/`** — обновить этот файл и `references/action_catalog.md`.
- **Codex-зеркало.** `leadgen-codex` остаётся в синхроне с `leadgen` по правилам корневого `CLAUDE.md`. Автопилот в зеркало не уходит (пилотный эксперимент Claude-only).

## Версионирование контракта

Текущая версия: 1 (W1). Bump при:
- добавлении/удалении строк в маппинге;
- изменении anchors-чек-листа;
- появлении новой shared reference.
