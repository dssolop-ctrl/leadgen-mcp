# Autopilot — корневой роутер

> **Этот файл — корневой роутер runtime-данных автопилота.** Грузится при `/leadgen-autopilot` после `skill.md`.
> **Для dev-инструкций скилла:** `.claude/skills/leadgen-autopilot/CLAUDE.md`.
> **Для архитектуры:** `PLAN.md` в корне репо.

## Что лежит в этой папке

| Путь | Содержимое | Жизненный цикл |
|---|---|---|
| `config/caps_defaults.yaml` | Глобальные numeric caps, thresholds, expiry | versioned, редко меняется |
| `config/trust_profiles/*.yaml` | Permissions baselines: conservative / balanced / aggressive / pilot_full_auto | versioned |
| `config/cities/*.yaml` | Per-city конфиги | versioned (кроме `*.local.yaml`) |
| `config/secrets.env` | Telegram token, API keys | **gitignored** |
| `schemas/*.schema.json` | JSON Schemas для всех артефактов | versioned |
| `runtime/<city>/` | Operational state, ledger, snapshots, narrative md | **gitignored** |
| `reports/<city>/<YYYY-MM>/` | HTML-отчёты | **gitignored** |
| `learnings/{proposed,validated,rejected}/` | Гипотезы агента | versioned (артефакт ревью) |
| `lib/*.sh` | Shell helpers (Telegram, atomic write, lock) | versioned |

## Старт прогона: что нужно агенту

Когда пользователь запускает `/leadgen-autopilot city=<city>`:

1. **Валидация ввода.** Если city не указан — fail, попросить.
2. **HALT check.** `autopilot/HALT.flag` (global) и `autopilot/runtime/<city>/HALT.flag` (per-city). Если есть — exit с alert.
3. **Config load.** `autopilot/config/cities/<city>.yaml`. Если нет — fail, попросить специалиста.
4. **Lock acquire.** `bash autopilot/lib/lock.sh acquire <city> <run_id> started`.
5. **Determine mode.** По дате: daily | weekly | monthly. См. `.claude/skills/leadgen-autopilot/skill.md` раздел "Роутер".
6. **Lazy load branches.** По режиму — нужные `branches/*.md`.
7. **Run cycle** по `flow-steps.md` anchors.
8. **Finish.** State + narrative regenerate, report, Telegram, release lock.

## Где найти конкретные правила

| Вопрос | Файл |
|---|---|
| Какие действия разрешены / какой profile | `config/trust_profiles/<profile>.yaml` |
| Какие caps для bid/budget changes | `config/caps_defaults.yaml` |
| Каталог сигналов и actions | `.claude/skills/leadgen-autopilot/references/{signal,action}_catalog.md` |
| Как читать `state.yaml` / `action_ledger.jsonl` | `schemas/<name>.schema.json` |
| Как создавать кампанию (playbook) | `.claude/skills/leadgen/branches/create-{search,rsya}.md` через `references/playbook_contract.md` |
| Архитектура | `../PLAN.md` |

## RECENT-CHANGES автопилота

**Не пишутся** в корневой `RECENT-CHANGES.md`. Каждый прогон создаёт `runtime/<city>/narrative/runs/<YYYY-MM>/<HHMM>.md` — это и есть журнал.

Архитектурные изменения скилла → корневой `RECENT-CHANGES.md` (через PR).

## Текущий статус разработки

См. `PLAN.md` раздел 16 (Волны разработки). Готовые волны помечены в `RECENT-CHANGES.md` корня репо.
