# Direct MCP & Leadgen Skill — инструкции агентам

> Тонкий указатель для Codex и других агентов. Полные правила — в `CLAUDE.md` (root) и подчинённых `CLAUDE.md` (`server/`, `.claude/skills/leadgen/`).
> **Не дублируй сюда содержимое CLAUDE.md** — один и тот же контекст не должен оплачиваться дважды на разных агентах.

## Режимы работы

- **DEV (по умолчанию)** — мы разрабатываем MCP-сервер и скиллы. Скилл-правила не применяются автоматически.
- **RUNTIME (только по слэш-команде)** — `/leadgen`, `/yandex-direct`, `/yandex-metrika`, `/vk-ads`, `/demand-research`, `/serp-monitor`, `/seo`. Тогда грузится содержимое скилла.

Если пользователь просит рекламную задачу без слэш-команды — уточнить, нужен ли runtime-режим.

## Куда смотреть

| Что нужно | Файл |
|---|---|
| Журнал изменений (читать первым в новой сессии — последние 1–2 пункта) | `RECENT-CHANGES.md` |
| Dev-режим, маршрутизация, lazy-load правила | `CLAUDE.md` |
| Dev MCP-сервера (Go, build, packages) | `server/CLAUDE.md` |
| Dev скилла leadgen (двойное дерево, структура) | `.claude/skills/leadgen/CLAUDE.md` |
| Технические детали, API quirks, отладка (по требованию) | `TECHNICAL.md` |
| Скилл leadgen для Codex (runtime) | `.codex/skills/leadgen-codex/SKILL.md` |
| Скилл leadgen для Claude Code (runtime) | `.claude/skills/leadgen/skill.md` |
| Yandex Direct API правила (runtime, по требованию) | `METRIKA-ADS-RULES.md` |
| VK Ads API правила (runtime, по требованию) | `VK-ADS-RULES.md` |
| Юридическая проверка контента (runtime, по требованию) | `LEGAL.md` |
| Бизнес-настройки, цели, CPA-пороги (runtime, по требованию) | `PROJECTS.md` |

## Ключевые правила (полные — в `CLAUDE.md` и `TECHNICAL.md`)

- **DRAFT-only:** новые кампании после `add_campaign` остаются в DRAFT/SUSPENDED. Активирует пользователь.
- **Бюджет** — недельный, в рублях (число), без умножения на 1 000 000.
- **Атрибуция отчётов** — всегда `LYDC`.
- **UTM** — через `tracking_params` на уровне группы.
- **Цели и счётчики** — называй по-человечески, не голые ID.
- **Двойное дерево скиллов** (Claude + Codex). Любая правка флоу/референсов/библиотек синхронизируется в обе ветки. Расхождения помечаются `Codex-only` / `Claude-only` + запись в `lessons_registry.md`.
- **После значимых изменений** — записать новый пункт сверху в `RECENT-CHANGES.md`.

Если правишь правила — делай это в `CLAUDE.md` / `server/CLAUDE.md` / `TECHNICAL.md`, а сюда добавляй только ссылку.
