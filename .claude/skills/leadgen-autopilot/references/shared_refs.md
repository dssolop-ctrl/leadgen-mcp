# Shared References Map

> Какие файлы из `.claude/skills/leadgen/` автопилот читает как источник правил.

| Источник | Используется для | Когда грузится |
|---|---|---|
| `leadgen/references/city_benchmarks.md` | tier-based budget/CPA пороги, min_ctr per tier | при analyze (пороги для S-CTR-LOW и др.), reconcile |
| `leadgen/references/lessons_registry.md` | агрегированные уроки специалиста | в decide для коррекции confidence/обоснования |
| `leadgen/references/copy_blacklist.md` | запрещённые слова | при formировании ad.update_text, ad.add_new_variant |
| `leadgen/references/rsya_defaults.md` | дефолты РСЯ (формат бюджета, audience targeting baseline) | при create_draft (rsya), при analyze placement |
| `leadgen/references/site_structure.md` | валидные топики, URL-схема | при reconcile (валидация topic в city.yaml) |
| `leadgen/references/ui_naming.md` | naming convention для кампаний | при `campaign.create_draft.*` (формирование Name) |
| `leadgen/references/image_prompts.md` | промпты для генерации картинок | при `creative.generate_new_image` |
| `leadgen/library/banner_titles.md` | пул заголовков баннеров | при создании RSYA-объявлений |
| `leadgen/library/banner_texts.md` | пул текстов баннеров | то же |
| `leadgen/library/titles.md` | пул заголовков для search | при `ad.add_new_variant` (search) |
| `leadgen/library/texts.md` | пул текстов для search | то же |
| `leadgen/library/extensions.md` | шаблоны быстрых ссылок и уточнений | при `extension.add_sitelinks` |
| `leadgen/library/display_urls.md` | display URLs | при создании ad |
| `leadgen/library/selling_modifiers.md` | модификаторы для текстов | при `creative.update_text_variant` |

## Правила использования

1. **Только чтение.** Автопилот никогда не пишет в `.claude/skills/leadgen/`.
2. **Версионная зависимость.** Если значение из `city_benchmarks.md` изменилось — пересчитать пороги при следующем прогоне.
3. **Lazy-load.** Грузить только при необходимости конкретного действия (по таблице выше). Не предзагружать все 13 файлов в каждом прогоне.
4. **Кэширование.** В рамках одного прогона можно держать прочитанные файлы в контексте без повторного чтения.

## Расхождение и контракт

См. `playbook_contract.md` для anchors и стабильности структуры.

Если правки в `leadgen/references/*.md` или `leadgen/library/*.md` ломают ожидания автопилота (поменялся формат таблицы, удалены секции) — обновить:
- ссылки в `signal_catalog.md` / `action_catalog.md`;
- этот файл;
- запись в `RECENT-CHANGES.md`.
