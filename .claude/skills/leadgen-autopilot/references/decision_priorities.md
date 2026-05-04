# Decision Priorities

> Когда несколько signals сработали одновременно — порядок приоритетов и стратегия слияния.

## 1. Жёсткий приоритет (по убыванию)

1. **Safety / hard caps** — `S-BUD-HARDCAP`, hard pacing emergency, drift с unexplained причиной → перебивают всё. Действия типа `budget.pause_due_to_overrun`.
2. **Drift detected** — entity отличается от ожиданий API → freeze planned actions для этой entity, не пытаться "поправить".
3. **Human override** — есть `human_override_until` в state → skip все actions для entity до истечения cooldown.
4. **Block actions** — никогда не выполняются (legal/account/billing/delete). Логируются как `skipped_block`.
5. **Idempotency / cooldown** — действие уже выполнено в decision_window OR cooldown не истёк → skip.
6. **Pacing-driven** — если `pacing_state >= conservation`, отключить агрессивные actions (только защитные).
7. **Каскадная защита**: если для entity сработал `S-NO-CONVERSIONS`, не повышать ставки/бюджет, даже если другие сигналы (например, `S-CPA-WAY-BELOW-TARGET`) предлагают.
8. **Reconcile config** — изменения в `city.yaml` (новая active тема, изменение target_cpa) обрабатываются перед tactical actions.
9. **Tactical actions** — обычные сигналы → actions по action_catalog.
10. **Opportunity actions** — `S-CPA-WAY-BELOW-TARGET` и т.п. → исполнять последними.

## 2. Слияние множественных signals на одну entity

| Сигналы | Решение |
|---|---|
| `S-CPA-ABOVE-TARGET` + `S-NO-CONVERSIONS` | действовать по более тяжёлому: `campaign.pause.low_performance` если 14+ дней без, иначе `bid.decrease` + `negatives.add` |
| `S-CTR-LOW` + `S-IMPRESSIONS-DROP` | сначала диагностика (moderation? bid?), потом `ad.add_new_variant` если CTR проблема |
| `S-CPA-JUMP` + `S-LEARNING-IN-PROGRESS` | **не** действовать ставочно (учится), только `negatives.add` если есть мусор в search queries |
| `S-PLACEMENT-BAD` (РСЯ) + `S-CPA-ABOVE-TARGET` | `placement.block` сначала, потом подождать рефреш — не двигать ставки в тот же прогон |

## 3. Per-run caps

- Не больше `caps.max_actions_per_run` всего (default 50).
- Не больше `caps.max_new_campaigns_per_run` (default 4).
- Не больше `caps.max_pauses_per_run` (default 10).
- Не больше `caps.max_creative_generations_per_run` (default 5).

При превышении — отбрасывать low-priority actions сначала (opportunity), затем medium-priority. Hard caps (safety) — никогда не отбрасывать.

## 4. Tie-breaker

Если несколько actions имеют одинаковый приоритет:
- Действия с `confidence: high` приоритетнее `medium` приоритетнее `low`.
- При равной confidence — действие с более ранним применимым `cooldown_until`.
- При равенстве — alphabetical по `action_type` (для детерминизма).
