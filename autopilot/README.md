# Autopilot — Quick Start

> Автономный агент управления рекламой Этажей. Полное ТЗ — в `../PLAN.md`.

## Установка

1. **Telegram bot.**
   - Создай бота через @BotFather, получи токен.
   - Создай группу/канал для уведомлений пилотного города, добавь бота, получи `chat_id` (через @userinfobot или Bot API `getUpdates`).
2. **Secrets.**
   ```bash
   cp config/secrets.env.example config/secrets.env
   # отредактируй config/secrets.env, заполни TELEGRAM_BOT_TOKEN и TELEGRAM_ALLOWLIST_CHAT_IDS
   ```
3. **City config.**
   ```bash
   cp config/cities/_example.yaml config/cities/<your_city>.yaml
   # заполни city/login/counter/geo/topics/budgets/notify
   ```
4. **Trust profile.** Для пилота используй `trust_profile: pilot_full_auto` + `autonomy_mode: full_auto`. Это даёт максимум автономии при сохранении hard-block границ.

## Запуск (вручную, для проверки)

В Claude Code из корня репо:
```
/leadgen-autopilot city=<your_city>
```

## Запуск через Claude Desktop routine

См. `../docs/autopilot-routine-setup.md` (создаётся в W10 hardening). Базовый шаблон:
- Routine prompt: `Запусти leadgen-autopilot для города <your_city>`
- Working directory: `C:\git\leadgen-mcp` (корень репо, не autopilot/)
- Расписание: 1 раз/день в 10:00 МСК (как минимум).

## Управление

| Действие | Как сделать |
|---|---|
| Глобально остановить агента | Создать файл `autopilot/HALT.flag` |
| Остановить только один город | Создать `autopilot/runtime/<city>/HALT.flag` |
| Поменять режим автономии | Поправить `autonomy_mode` в `config/cities/<city>.yaml` |
| Поменять профиль доверия | Поправить `trust_profile` в `config/cities/<city>.yaml` |
| Добавить тематику | Добавить блок в `topics:` с `status: active` |
| Выключить тематику | Поменять `status: paused` |

## Что писать в Telegram (только при `autonomy_mode: with_approvals`)

| Команда | Что делает |
|---|---|
| `approve <id>` | Одобряет pending action |
| `reject <id>` | Отклоняет |
| `defer <id> 3d` | Отложить на 3 дня |
| `rollback <run_id>` | Запросить dry-run отката прогона |
| `confirm rollback <id>` | Подтвердить откат после dry-run |

## Где искать информацию

| Что нужно | Куда смотреть |
|---|---|
| Архитектура | `../PLAN.md` |
| Текущее состояние города | `runtime/<city>/state.yaml` или `runtime/<city>/narrative/STATE.md` |
| История прогонов | `runtime/<city>/narrative/runs/<YYYY-MM>/` |
| Журнал действий | `runtime/<city>/action_ledger.jsonl` |
| Отчёты | `reports/<city>/<YYYY-MM>/*.html` |
| Гипотезы агента | `learnings/proposed/`, `learnings/validated/` |

## FAQ

**Q: Бот сам активирует кампании?**
A: В `pilot_full_auto` + `full_auto` — да, активация кампаний через `campaign.activate_existing_draft = auto_with_notify`. В `conservative`/`balanced` — нет, нужно approve.

**Q: Что если бот сломал кампанию?**
A: В Telegram пиши `rollback <run_id>` (run_id есть в каждом отчёте). Бот покажет dry-run обратных действий. Если устраивает → `confirm rollback <id>`.

**Q: Бот может удалить кампанию?**
A: Нет. `campaign.delete` — hard-block во всех профилях. Только `archive` (для no_traffic_30days).

**Q: Почему бот не трогает кампанию X?**
A: Скорее всего у неё нет label `autopilot:managed`. Бот трогает только labeled-кампании. Adoption — отдельное действие через onboarding.
