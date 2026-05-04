# Signal Catalog

> Источники сигналов автопилота. Каждый signal имеет `signal_id`, expression, evidence-формат и mapped actions.

## 1. Бюджет / Pacing

### S-BUD-OVERRUN
- **Trigger:** `budget.spent_mtd > monthly_budget * (alert_budget_overrun_pct / 100)`
- **Severity:** medium
- **Maps to actions:** `budget.pause_due_to_overrun`
- **Evidence:** `{spent_mtd, monthly_budget, pacing_state}`

### S-BUD-PACING-CONS
- **Trigger:** `pacing_ratio > caps.pacing_conservation_ratio`
- **Severity:** medium
- **Maps to actions:** disable aggressive bid/budget increases (внутреннее, не отдельный action)
- **Evidence:** `{pacing_ratio, expected_spend_to_date, forecast_eom}`

### S-BUD-PACING-EMERG
- **Trigger:** `pacing_ratio > caps.pacing_emergency_ratio` OR `forecast_eom > monthly_budget * 1.2`
- **Severity:** high
- **Maps to actions:** `campaign.pause.budget_exhausted` (для перерасходующих)
- **Evidence:** same + per-topic shares

### S-BUD-HARDCAP
- **Trigger:** `spent_mtd > monthly_budget * (caps.budget_overrun_hard_stop_pct / 100)`
- **Severity:** critical
- **Maps to actions:** emergency pause всех managed campaigns + alert
- **Evidence:** same

## 2. CPA / Конверсии

### S-CPA-JUMP
- **Trigger:** CPA today vs rolling 7d > `caps.alert_cpa_jump_pct`%
- **Severity:** medium
- **Maps to actions:** `bid.decrease.within_cap`, `placement.block.low_performing` (РСЯ), детальный анализ
- **Evidence:** `{cpa_today, cpa_avg_7d, jump_pct, clicks, conversions, channel}`

### S-CPA-ABOVE-TARGET
- **Trigger:** `cpa_form_7d > target_cpa_form * 1.15` AND `clicks_7d >= channel.min_clicks_for_decision`
- **Severity:** medium
- **Maps to actions:** `bid.decrease.within_cap`, `negatives.add_from_search_queries`, `placement.block.low_performing`
- **Evidence:** `{cpa_form_7d, target_cpa_form, clicks_7d, conversions_7d}`

### S-CPA-WAY-BELOW-TARGET
- **Trigger:** `cpa_form_7d < target_cpa_form * 0.7` AND `clicks_7d >= min_clicks` AND есть запас бюджета
- **Severity:** low (opportunity)
- **Maps to actions:** `bid.increase.within_cap`, `budget.increase.within_cap`
- **Evidence:** same

### S-NO-CONVERSIONS
- **Trigger:** 0 leads_form за `caps.alert_no_conversions_days` дней при `clicks > min_clicks * 2`
- **Severity:** high
- **Maps to actions:** `campaign.pause.low_performance` (если 14+ дней), `keyword.pause.low_performance`, `negatives.add_from_search_queries`
- **Evidence:** `{spend_period, clicks_period, leads=0, days}`

## 3. CTR / Объявления

### S-CTR-LOW
- **Trigger:** `ctr_7d < tier_min_ctr` AND `impressions_7d > min_impressions`
- **Severity:** medium
- **Maps to actions:** `ad.pause.low_ctr`, `ad.add_new_variant`, `adgroup.pause.low_ctr`
- **Evidence:** `{ctr_7d, tier_min_ctr, impressions_7d}`

### S-IMPRESSIONS-DROP
- **Trigger:** impressions today vs avg 7d < (1 - `caps.alert_impressions_drop_pct`/100)
- **Severity:** high
- **Maps to actions:** диагностика — moderation rejection? bid слишком низкий? geo сжалось?
- **Evidence:** `{impressions_today, impressions_avg_7d, drop_pct}`

## 4. Площадки (РСЯ)

### S-PLACEMENT-BAD
- **Trigger:** placement spend > 500₽ AND 0 leads за период learning_period_days, OR cpa_per_placement > target * 3
- **Severity:** low
- **Maps to actions:** `placement.block.low_performing`
- **Evidence:** `{placement_id, spend, clicks, leads, cpa}`

## 5. Поисковые запросы

### S-SQ-TRASH
- **Trigger:** search query содержит phrase из `copy_blacklist.md` OR клики>3 без конверсий
- **Severity:** low
- **Maps to actions:** `negatives.add_from_search_queries`
- **Evidence:** `{query, clicks, leads, blacklist_match}`

## 6. Обучение стратегий

### S-LEARNING-IN-PROGRESS
- **Trigger:** campaign created < `learning_period_days` назад
- **Severity:** info
- **Maps to actions:** **disable** все ставочные правки до конца обучения. Не отдельный action — модификатор в decide.
- **Evidence:** `{campaign_id, created_at, days_learning}`

### S-LEARNING-ENDED
- **Trigger:** `learning_period_days` истекло, можно делать первое регулярное вмешательство
- **Severity:** info
- **Maps to actions:** unblock decision flow
- **Evidence:** same

## 7. Drift / Human override

### S-DRIFT-DETECTED
- **Trigger:** API state ≠ expected before_state (по ledger)
- **Severity:** medium
- **Maps to actions:** check `get_change_history` → human_override OR unexplained drift alert
- **Evidence:** `{entity, expected, actual, history_records}`

### S-HUMAN-OVERRIDE
- **Trigger:** drift с признаками ручного изменения (manual user_login)
- **Severity:** info
- **Maps to actions:** mark `human_override_until = now + 72h`, заморозить planned actions для entity
- **Evidence:** same + user_login

## 8. Onboarding (только для новых городов)

### S-NEW-TOPIC-ENABLED
- **Trigger:** в `city.yaml` появилась `status: active` тема, для которой нет campaigns в state
- **Severity:** info
- **Maps to actions:** `campaign.create_draft.in_new_topic` (per канал из `allowed_channels`)
- **Evidence:** `{topic, channels, monthly_budget}`

### S-EXPERIMENTAL-TOPIC
- **Trigger:** `status: experimental` topic
- **Severity:** info
- **Maps to actions:** `campaign.create_draft.in_new_topic` с уменьшенным budget cap
- **Evidence:** same

### S-CANDIDATE-TOPIC
- **Trigger:** `status: candidate` topic
- **Severity:** info
- **Maps to actions:** **никаких apply**, только демо-предложение в launch_proposal
- **Evidence:** same + demand_analysis

## 9. Конкурентные / SERP (опционально, через serp-monitor)

### S-COMPETITOR-PRICE-DROP
- **Trigger:** drop в конкурентных аукционных метриках
- **Severity:** low (информационный)
- **Maps to actions:** опциональный анализ — не для daily

## Структура signal entry в `runs/<run_id>.md`

```yaml
- signal_id: S-CPA-ABOVE-TARGET
  severity: medium
  entity_type: campaign
  entity_id: 8765432
  topic: vtorichka
  channel: search
  evidence:
    cpa_form_7d: 4900
    target_cpa_form: 3500
    clicks_7d: 84
    conversions_7d: 6
  proposed_actions:
    - bid.decrease.within_cap
    - negatives.add_from_search_queries
```

## Routing signals → actions

См. `references/decision_priorities.md` для приоритезации при множественных сигналах.
