# leadgen-autopilot — dev-инструкции скилла

> **Этот файл — для разработчика скилла.** Не грузить при runtime-прогоне (skill.md это делает сам).

## Назначение

Скилл `leadgen-autopilot` — параллельная ветка к `leadgen`. Запускается слэш-командой `/leadgen-autopilot` через Claude Desktop routine. Реализует **policy/orchestration layer** автономного агента: анализ, память, permissions, reconcile, apply, отчёты.

## Codex-зеркало

**Это исключение из общего правила проекта.** На пилоте `leadgen-autopilot` существует **только в `.claude/`**. Зеркало в `.codex/skills/leadgen-autopilot-codex/` **не создаётся**.

Причина: пилотный эксперимент, разрабатывается под Claude. Если решим расширить на Codex — синхронизация двойного дерева вернётся как обязательная.

**Что всё равно синхронизируется в Codex:** общие shared-файлы `.claude/skills/leadgen/{branches,references,library}` — они обслуживают и автопилот через playbook contract, и Codex-зеркало `leadgen-codex` остаётся as is.

## Связь с `leadgen`

- `leadgen` (вызов `/leadgen`) и `leadgen-autopilot` (вызов `/leadgen-autopilot`) — **параллельны и независимы в runtime**.
- Автопилот **читает** определённые `leadgen/branches/{create,optimize}-{search,rsya}.md` как playbook (стабильные anchor'ы). Маппинг — в `references/playbook_contract.md`.
- Автопилот **никогда не вызывает** `/leadgen` в runtime.
- Автопилот **никогда не пишет** в файлы `leadgen/` (только в `autopilot/learnings/proposed`).

## Структура скилла

```
.claude/skills/leadgen-autopilot/
├── CLAUDE.md                этот файл
├── skill.md                 роутер: парсит команду, выбирает режим, грузит branches
├── flow-steps.md            anchors шагов (D1..D8 daily, W1..W5 weekly, M1..M6 monthly)
├── branches/
│   ├── analyze.md            daily/weekly/monthly анализ
│   ├── reconcile_config.md   diff config↔state↔API + drift
│   ├── decide.md             signal → action + caps/perms/cooldown
│   ├── apply.md              apply через MCP + idempotency + ledger
│   ├── memory_write.md       operational → narrative
│   ├── notify.md             отчёт + Telegram
│   ├── approval.md           только при autonomy=with_approvals
│   ├── learnings.md          proposed → digest
│   ├── safety.md             safety_check, kill-switch, hard caps
│   └── onboarding.md         baseline scan + launch_proposal + draft creation
└── references/
    ├── playbook_contract.md
    ├── signal_catalog.md
    ├── action_catalog.md
    ├── decision_priorities.md
    └── shared_refs.md
```

## Lazy-load policy

`skill.md` грузится при `/leadgen-autopilot`. Branches — лениво по режиму прогона. References — лениво по нужности (не предзагружать).

## Где runtime-данные

Runtime-данные **не лежат в скилле**. Они в `autopilot/runtime/<city>/`. Это правильное разделение: код vs данные.

## Перед коммитом

- Проверить, что `skill.md` читается за один проход без перекрёстных ссылок на runtime.
- Если поменял anchors в `flow-steps.md` — обновить ссылки в branches.
- Если поменял каталог действий — обновить `references/action_catalog.md` и проверить `playbook_contract.md`.

## RECENT-CHANGES

Изменения архитектуры скилла → запись в **корневой** `RECENT-CHANGES.md`. Runtime-прогоны бота туда **не пишутся** (для них — `autopilot/runtime/<city>/runs/`).
