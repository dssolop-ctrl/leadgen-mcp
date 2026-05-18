---
id: direct-api-suspend-before-archive-2026-05
created: 2026-05-18
last_observed: 2026-05-18
confidence: low
observed_count: 1
scope:
  city: novosibirsk-ai
  topic: any
  channel: any
needs_repeats: 2
status: proposed
---

# Pattern: Direct API manage_ads(action=archive) требует suspend до archive

## Pattern

`mcp__yandex-direct__manage_ads(action=archive, ad_ids=...)` отвергает ads с **State=ON** или **Status="Принято на модерации"** ошибкой Code 8300 («Неверный статус у объекта», «Объявление показывается и не может быть заархивировано»).

Корректный двухшаговый apply:
1. `manage_ads(action=suspend, ad_ids=...)` — переводит State→OFF/SUSPENDED.
2. `manage_ads(action=archive, ad_ids=...)` — теперь archive проходит.

Замечено даже на ads с State=OFF, но Status="Принято на модерации" (т.е. модерация одобрила, но ad не активирован) — Direct всё равно отвергает прямой archive.

## Positive evidence

- **run novosibirsk-ai-2026-05-18-0736**, action `ad.archive_stale_post_activation` on 6 ads of campaign 709915110:
  - Прямой `manage_ads(action=archive, ad_ids=[17718835281,17718835312,17718835319,17718835332,17718972606,17718972609])` → 6 ошибок Code 8300.
  - После `manage_ads(action=suspend, ad_ids=...)` → 6 OK.
  - Затем `manage_ads(action=archive, ad_ids=...)` → 6 OK, заархивировано.

## Negative evidence

(пока нет)

## Action_when_validated

В `branches/apply.md` для action_types `ad.archive_*` (включая `ad.archive_stale_post_activation`, `ad.pause.low_ctr` с last-step archive, и любые ad.archive.*):

```python
case "ad.archive_*":
    # ШАГ 1: suspend
    suspend_result = mcp.manage_ads(ad_ids=action.entity_ids, action="suspend")
    # ШАГ 2: archive
    archive_result = mcp.manage_ads(ad_ids=action.entity_ids, action="archive")
    result = archive_result
```

Также добавить в `references/action_catalog.md` notes для всех `ad.archive_*`:
> Direct API rejects archive on ads with State=ON or Status="Принято на модерации" (code 8300). Apply must be two-step suspend→archive.

## Cross-references

- Lesson #29 в `.claude/skills/leadgen/references/lessons_registry.md` упоминал, что `manage_ads` не работает для **DRAFT** (Code 8300). Это смежная грань той же проблемы: archive требует **OFF state** + **non-DRAFT status**.
- Нужно объединить лессоны: единое правило «`manage_ads(action=archive)` требует State=OFF AND Status ∈ {ACCEPTED но не "Принято на модерации"}, иначе сначала suspend».
