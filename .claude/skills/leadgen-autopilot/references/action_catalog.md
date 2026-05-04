# Action Catalog

> Полный каталог действий автопилота. Каждое действие имеет:
> - `risk_class` — low/medium/high/critical
> - `min_evidence` — минимальные требования к данным
> - `cooldown_hours` (channel-specific overrides возможны)
> - `snapshot_required` — нужен ли before_snapshot
> - `idempotency_window` — единица decision_window для idempotency_key
> - `mcp_tools` — какие MCP-вызовы используются для apply
> - `playbook_ref` — ссылка на shared playbook (при необходимости)
>
> **Permissions** живут в `trust_profiles/<profile>.yaml`, не здесь.

## Формат entry

```yaml
- action_type: bid.increase.within_cap
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24                       # default; channel override применяется автоматически
  idempotency_window: YYYY-MM-DD
  min_evidence:
    clicks_7d: 30                           # для search; для rsya — 100 (channel override)
    conversions_7d: 3                       # для search; для rsya — 5
    spend_period_rub: null
    days_since_last_change: 1
  mcp_tools: [set_keyword_bids, set_audience_target_bids, set_dynamic_ad_target_bids, set_smart_ad_target_bids]
  playbook_ref: null
  expected_effect_template: "Lower CPC by ~{value}%"
  guard_metric: "spend_without_conversions_3d"
  rollback_trigger: null
```

## 1. Бюджет

```yaml
- action_type: budget.increase.within_cap
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24
  idempotency_window: YYYY-MM-DD
  min_evidence: {clicks_7d: 30, conversions_7d: 3}
  mcp_tools: [update_campaign]              # set DailyBudget
  expected_effect_template: "Increase daily delivery by ~{value}%"

- action_type: budget.increase.above_cap
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 48
  idempotency_window: YYYY-MM
  min_evidence: {clicks_7d: 100, conversions_7d: 10, days_since_last_change: 3}
  mcp_tools: [update_campaign]
  expected_effect_template: "Increase budget beyond cap by {value}%"
  guard_metric: "spend_to_conversions_ratio_jump"

- action_type: budget.decrease.within_cap
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24
  idempotency_window: YYYY-MM-DD
  min_evidence: {days_since_last_change: 1}
  mcp_tools: [update_campaign]

- action_type: budget.decrease.above_cap
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 24
  idempotency_window: YYYY-MM
  min_evidence: {days_since_last_change: 3}
  mcp_tools: [update_campaign]

- action_type: budget.set_total_monthly
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 168                       # неделя
  idempotency_window: YYYY-MM
  mcp_tools: [update_campaign]              # batch для всех campaigns в city

- action_type: budget.pause_due_to_overrun
  risk_class: high
  snapshot_required: true
  cooldown_hours: 0                         # экстренное
  idempotency_window: YYYY-MM-DD
  mcp_tools: [suspend_campaign]
  expected_effect_template: "Halt overrunning spend"
  guard_metric: "rebound_spend_within_24h"
```

## 2. Ставки

```yaml
- action_type: bid.increase.within_cap
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24                        # search; rsya: 48
  idempotency_window: YYYY-MM-DD
  min_evidence: {clicks_7d: 30, conversions_7d: 3}
  mcp_tools: [set_keyword_bids, set_audience_target_bids, set_dynamic_ad_target_bids, set_smart_ad_target_bids, set_bids]
  expected_effect_template: "Higher position / more impressions"

- action_type: bid.decrease.within_cap
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24
  idempotency_window: YYYY-MM-DD
  min_evidence: {clicks_7d: 30}
  mcp_tools: [set_keyword_bids, set_bids, set_audience_target_bids, ...]

- action_type: bid.change.above_cap
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 48
  idempotency_window: YYYY-MM-DD
  min_evidence: {clicks_7d: 100, conversions_7d: 10, days_since_last_change: 3}
  mcp_tools: [set_keyword_bids, set_bids]

- action_type: bid.adjust_strategy_target_cpa.within_cap
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 168                       # 7 дней — переучка стратегии
  idempotency_window: YYYY-WNN
  min_evidence: {conversions_period: 10, days_since_strategy_change: 7}
  mcp_tools: [update_strategy, update_campaign]

- action_type: bid.adjust_strategy_target_cpa.above_cap
  risk_class: high
  snapshot_required: true
  cooldown_hours: 336                       # 14 дней
  idempotency_window: YYYY-WNN
  mcp_tools: [update_strategy]
  rollback_trigger: "cpa_jumps_above_target_x2_for_5d"
```

## 3. Минусы

```yaml
- action_type: negatives.add_from_search_queries
  risk_class: low
  snapshot_required: false                  # additive single-field, ledger row хватает
  cooldown_hours: 0                         # можно много раз в день
  idempotency_window: YYYY-MM-DD
  min_evidence: {queries_with_clicks_no_conversions: 1}
  mcp_tools: [update_campaign]              # negativeKeywords field
  playbook_ref: "leadgen/branches/optimize-search.md#minusovka"

- action_type: negatives.add_global_set
  risk_class: low
  snapshot_required: false
  cooldown_hours: 0
  idempotency_window: YYYY-MM-DD
  mcp_tools: [add_negative_keyword_set, update_negative_keyword_set]

- action_type: negatives.remove
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 24
  idempotency_window: YYYY-MM-DD
  mcp_tools: [update_campaign]
```

## 4. Кампании

```yaml
- action_type: campaign.create_draft.in_existing_topic
  risk_class: medium
  snapshot_required: false                   # creation, before — нет такой кампании
  cooldown_hours: 0
  idempotency_window: YYYY-Qn
  min_evidence: {topic_status_active: true, demand_volume: ">100/мес"}
  mcp_tools: [add_campaign, add_adgroup, add_ad, add_keywords]
  playbook_ref: "leadgen/branches/create-{search,rsya}.md"
  # ВСЕГДА создаём в SUSPENDED (DRAFT). Активация — отдельным action.
  notes: "DRAFT-only. Шаг 11 playbook обязателен."

- action_type: campaign.create_draft.in_new_topic
  risk_class: medium
  snapshot_required: false
  cooldown_hours: 0
  idempotency_window: YYYY-Qn
  min_evidence: {topic_status_active: true, launch_proposal_approved: true}
  mcp_tools: [add_campaign, add_adgroup, add_ad, add_keywords, add_labels]
  playbook_ref: "leadgen/branches/create-{search,rsya}.md"

- action_type: campaign.create_draft.outside_config_topics
  risk_class: critical
  notes: "ВСЕГДА block. Любая попытка — alert + skipped."

- action_type: campaign.activate_existing_draft
  risk_class: high
  snapshot_required: true
  cooldown_hours: 0
  idempotency_window: YYYY-MM-DD
  min_evidence: {moderation_status: ACCEPTED, draft_age_hours: 1}
  mcp_tools: [resume_campaign]              # State: SUSPENDED → ON
  expected_effect_template: "Start ad delivery"
  guard_metric: "spend_zero_after_24h"
  rollback_trigger: "high_cpa_within_48h"
  notes: "В pilot_full_auto: auto_with_notify. В others: review_queue."

- action_type: campaign.pause.low_performance
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 0
  idempotency_window: YYYY-MM-DD
  min_evidence: {days_with_high_cpa: 14, cpa_vs_target_ratio: 2.0}
  mcp_tools: [suspend_campaign]

- action_type: campaign.pause.budget_exhausted
  risk_class: low
  snapshot_required: false
  cooldown_hours: 0
  idempotency_window: YYYY-MM-DD
  min_evidence: {daily_budget_spent_pct: 100}
  mcp_tools: [suspend_campaign]

- action_type: campaign.resume
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 0
  idempotency_window: YYYY-MM-DD
  mcp_tools: [resume_campaign]

- action_type: campaign.archive.no_traffic_30days
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 0
  idempotency_window: YYYY-MM
  min_evidence: {days_no_impressions: 30}
  mcp_tools: [archive_campaign]

- action_type: campaign.adopt_existing
  risk_class: high
  snapshot_required: true
  cooldown_hours: 0
  idempotency_window: YYYY-Qn
  min_evidence: {launch_proposal_recommendation: adopt, owner_approved: true}
  mcp_tools: [add_labels]
  notes: "Метим autopilot:managed, city:<>, topic:<>, channel:<>"

- action_type: campaign.release
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 0
  mcp_tools: [add_labels]                   # autopilot:released; remove_labels если поддерживается
  notes: "Ownership predicate: managed && !released"

- action_type: campaign.delete
  risk_class: critical
  notes: "ВСЕГДА block. Используй archive."
```

## 5. Группы и объявления

```yaml
- action_type: adgroup.pause.low_ctr
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 24
  min_evidence: {ctr_7d_below_tier_min: true, impressions_7d: 1000}
  mcp_tools: [manage_ads]                   # paused state? или suspend на уровне adgroup

- action_type: adgroup.split_by_intent
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 168
  mcp_tools: [add_adgroup, manage_keywords] # перенос ключей в новую группу
  playbook_ref: "leadgen/branches/optimize-search.md#O3.6"

- action_type: ad.add_new_variant
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24
  mcp_tools: [add_ad]
  playbook_ref: "leadgen/library/{titles,texts,banner_titles,banner_texts}.md"

- action_type: ad.pause.low_ctr
  risk_class: low
  snapshot_required: false
  mcp_tools: [manage_ads]

- action_type: ad.update_text
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 48
  mcp_tools: [update_ad]
  playbook_ref: "leadgen/references/copy_blacklist.md"   # обязательная фильтрация

- action_type: ad.update_image_creative
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 168
  mcp_tools: [update_ad, add_ad_image, delete_ad_images]
```

## 6. Ключевые

```yaml
- action_type: keyword.pause.low_performance
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24
  min_evidence: {clicks_7d: 50, conversions_7d: 0, spend_period_rub: 1000}
  mcp_tools: [manage_keywords]

- action_type: keyword.add_in_existing_group
  risk_class: low
  snapshot_required: false
  mcp_tools: [add_keywords]

- action_type: keyword.add_new_group_in_existing_topic
  risk_class: medium
  snapshot_required: true
  mcp_tools: [add_adgroup, add_keywords, add_ad]

- action_type: keyword.remove
  risk_class: medium
  snapshot_required: true
  mcp_tools: [manage_keywords]
```

## 7. Стратегии

```yaml
- action_type: strategy.change_type
  risk_class: high
  snapshot_required: true
  cooldown_hours: 336                       # 14 дней
  idempotency_window: YYYY-WNN
  min_evidence: {conversions_period: 30, current_strategy_age_days: 30}
  mcp_tools: [update_strategy, update_campaign]
  rollback_trigger: "cpa_jumps_x2_for_7d_after_change"

- action_type: strategy.adjust_constraint
  risk_class: medium
  snapshot_required: true
  cooldown_hours: 168
  mcp_tools: [update_strategy]
```

## 8. Площадки РСЯ

```yaml
- action_type: placement.block.low_performing
  risk_class: low
  snapshot_required: false
  cooldown_hours: 0
  idempotency_window: YYYY-MM-DD
  min_evidence: {placement_spend_rub: 500, placement_leads: 0}
  mcp_tools: [apply_blocked_placements, set_excluded_sites]
  playbook_ref: "leadgen/branches/optimize-rsya.md#placements"

- action_type: placement.block.from_blacklist
  risk_class: low
  snapshot_required: false
  mcp_tools: [apply_blocked_placements]

- action_type: placement.unblock
  risk_class: medium
  snapshot_required: true
  mcp_tools: [apply_blocked_placements]     # новый список без unblocked
```

## 9. Аудитории / Ретаргетинг

```yaml
- action_type: audience.create
  risk_class: low                           # creation, не активирует показы
  snapshot_required: false
  mcp_tools: [add_retargeting_list, add_audience_targets]

- action_type: audience.adjust_bid_modifier
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24
  mcp_tools: [set_bid_modifiers, set_audience_target_bids]

- action_type: retargeting.create_list
  risk_class: low
  snapshot_required: false
  mcp_tools: [add_retargeting_list]
```

## 10. Регионы

```yaml
- action_type: region.adjust_bid_modifier
  risk_class: low
  snapshot_required: false
  cooldown_hours: 24
  mcp_tools: [set_bid_modifiers]

- action_type: region.add_to_targeting
  risk_class: high
  snapshot_required: true
  cooldown_hours: 168
  mcp_tools: [update_campaign]              # GeoIDs
  rollback_trigger: "spend_jump_x2_in_added_region_for_3d"

- action_type: region.remove_from_targeting
  risk_class: high
  snapshot_required: true
  cooldown_hours: 168
  mcp_tools: [update_campaign]
```

## 11. Расширения / vCard

```yaml
- action_type: extension.add_sitelinks
  risk_class: low
  snapshot_required: false
  mcp_tools: [add_sitelinks, add_ad_extension]

- action_type: extension.update_sitelinks
  risk_class: low
  snapshot_required: false
  mcp_tools: [update_ad]

- action_type: extension.update_vcard
  risk_class: medium
  snapshot_required: true
  mcp_tools: [add_vcard, delete_vcards]
```

## 12. Креативы

```yaml
- action_type: creative.generate_new_image
  risk_class: medium
  snapshot_required: true                   # сохраняем prompt + image refs до генерации
  cooldown_hours: 168
  mcp_tools: [generate_image, add_ad_image]
  playbook_ref: "leadgen/references/image_prompts.md"
  notes: "Создаёт DRAFT-объявление; активация специалиста."

- action_type: creative.update_text_variant
  risk_class: low
  snapshot_required: false
  mcp_tools: [add_ad]                       # новый вариант ad с обновлённым текстом
```

## 13. Юр-чувствительное (HARD BLOCKS)

```yaml
- action_type: legal.update_disclaimer
  risk_class: critical
  notes: "ВСЕГДА block. Никогда не выполняется в любом профиле."

- action_type: legal.change_landing_url
  risk_class: high
  snapshot_required: true
  cooldown_hours: 168
  mcp_tools: [update_ad]
  notes: "Только проверенные посадочные."

- action_type: legal.change_company_info
  risk_class: critical
  notes: "ВСЕГДА block."
```

## 14. Аккаунт (HARD BLOCKS)

```yaml
- action_type: account.change_settings
  risk_class: critical
  notes: "ВСЕГДА block."

- action_type: account.change_billing
  risk_class: critical
  notes: "ВСЕГДА block."

- action_type: account.close
  risk_class: critical
  notes: "ВСЕГДА block."
```

## Channel-specific overrides

В `caps_defaults.yaml.channels.{search|rsya}`:
- `min_clicks_for_decision`
- `min_conversions_for_decision`
- `cooldown_hours_after_bid_change`
- `learning_period_days`

При формировании `min_evidence` для action автопилот применяет channel override автоматически по `entity.channel`.

## Risk class semantics

| risk | snapshot | evidence | approval (in pilot_full_auto) | example |
|---|---|---|---|---|
| low | ledger row | minimal | auto | минусование, blacklist |
| medium | structured JSON | per-action min_evidence | auto_with_notify | bid changes, draft creation, pause |
| high | structured + history | strict | auto_with_notify | activation, strategy change, region |
| critical | n/a | n/a | block | account, legal, delete |
