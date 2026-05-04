# Branch: onboarding — baseline scan + launch_proposal + draft creation

> Грузится в первом прогоне города (когда `state.yaml` отсутствует ИЛИ `state.campaigns == []`).

## Шаги

### 1. Inventory

- `metrika_get_counter(counter_id)` — проверка доступа.
- `metrika_get_goals(counter_id)` — список целей; сверка с `city.metrika.goals`.
- `metrika_get_traffic_sources(...)` — наличие direct/utm трафика.
- `get_campaigns(client_login)` с filter по labels — отделить:
  - `owned`: имеют label `autopilot:managed` (для пилота с нуля — пусто).
  - `adoptable`: без label, в разрешённых тематиках (`active`/`experimental`/`candidate`).
  - `foreign`: вне разрешённых тематик (read-only forever).
  - `holdout`: имеют label `autopilot:holdout`.

### 2. Demand analysis (для `active` и `experimental` тем)

- Через `wordstat_*` или `check_search_volume` — индикативные объёмы по seed-фразам топика.
- Через `get_geo_regions` — корректность `geo_region_id`.

### 3. Generate launch_proposal

Файл: `runtime/<city>/onboarding/launch_proposal.yaml` + дополняющий `.md` для чтения.
Schema: `autopilot/schemas/launch_proposal.schema.json`.

```yaml
city: <city>
run_id: <run_id>
generated_at: <ISO>
autonomy_mode: full_auto
trust_profile: pilot_full_auto
account_state:
  existing_campaign_count: 0
  metrika_counter_ok: true
  goals_present: [lead_form, call, qualified_lead]
  utm_template_ok: true
  domain_reachable: true
  warnings: []
demand_analysis:
  - topic: vtorichka
    channel: search
    indicative_volume: 12500
    competitiveness_hint: medium
    comment: "Стабильный спрос, высокая конкурентность по 'купить квартиру вторичка'"
  - topic: vtorichka
    channel: rsya
    indicative_volume: 12500
    comment: "Подходит для добивки конверсий"
proposed_launches:
  - topic: vtorichka
    channel: search
    monthly_budget: 60000
    target_cpa_form: 3500
    target_cpa_call: 2800
    playbook_ref: "leadgen/branches/create-search.md"
    rationale: "Активная тема в city.yaml, спрос подтверждён, бенчмарки tier 2 укладываются в budget"
    risks: ["Холодный старт стратегии 7-14 дней", "Возможны колебания CPA в первую неделю"]
    auto_apply_in_full_auto: true
  - topic: vtorichka
    channel: rsya
    monthly_budget: 30000
    target_cpa_form: 4500
    playbook_ref: "leadgen/branches/create-rsya.md"
    rationale: "..."
    auto_apply_in_full_auto: true
adoptable_campaigns: []
foreign_campaign_count: 0
```

### 4. Approval gate

- `autonomy_mode == full_auto`:
  - launch_proposal сохраняется как audit trail.
  - Telegram сообщение: "Launch proposal generated, autonomy=full_auto. Proceeding to draft creation."
  - **Сразу** переходим к шагу 5 (draft creation) для каждого `auto_apply_in_full_auto: true`.
- `autonomy_mode == with_approvals`:
  - Каждый предложенный launch → отдельный entry в `pending_approvals.yaml`.
  - Telegram: ссылка на launch_proposal.html + список approvals.
  - Действия применятся в следующем прогоне после `approve <id>`.
- `autonomy_mode == read_only`:
  - launch_proposal сохранён, ничего не применяется.

### 5. Draft creation (если разрешено)

Для каждого `proposed_launches[*]`:
- Прочитать playbook `leadgen/branches/<playbook_ref>` (через `playbook_contract.md`).
- Применить шаги 1-10 playbook. **Шаг 11** (DRAFT-only финиш) обязателен.
- Получить `campaign_id` от `add_campaign`.
- Вызвать `add_labels(campaign_id, ["autopilot:managed", "city:<city>", "topic:<topic>", "channel:<channel>"])`.
- Записать в ledger строки для каждого MCP-вызова с `idempotency_key`.

### 6. Activation (если разрешено)

После draft creation:
- Дождаться `moderation_status: ACCEPTED` (на пилоте — обычно мгновенно для текстов из шаблонов).
- В `pilot_full_auto` + `full_auto`: `campaign.activate_existing_draft = auto_with_notify` → выполняется автоматически с TG-уведомлением.
- В `with_approvals`: ставится в pending_approvals.

### 7. Post-onboarding

- Записать `state.yaml` с новыми кампаниями.
- Создать narrative файлы: `STATE.md`, `campaigns/<id>.md` для каждой.
- Telegram summary: "Onboarding complete: created N drafts, activated M, awaiting K approvals."
- Дальнейшие прогоны — обычный daily cycle.

## Adoption (для существующего аккаунта — НЕ для пилота "с нуля")

Если `inventory` нашёл `adoptable_campaigns` — добавить в `launch_proposal.yaml.adoptable_campaigns`:

```yaml
adoptable_campaigns:
  - campaign_id: 8765499
    name: "Омск Вторичка Поиск (старая)"
    inferred_topic: vtorichka
    inferred_channel: search
    last_30d:
      spend: 120000
      leads_form: 34
      cpa_form: 3529
    recommendation: adopt
    reason: "Стабильная история, UTM корректные, тема в config"
```

При `recommendation: adopt`:
- `autonomy=full_auto` → выполнить `campaign.adopt_existing` (auto_with_notify).
- `autonomy=with_approvals` → review_queue.

Action `campaign.adopt_existing`:
- `add_labels(campaign_id, ["autopilot:managed", "city:<city>", "topic:<inferred>", "channel:<inferred>"])`.

Если в каталоге есть кампания, не вписывающаяся ни в одну `active`/`experimental` тематику — `recommendation: leave_readonly`. Никаких действий.

## Specials

- Если в `proposed_launches` или `adoptable_campaigns` topic = `experimental`:
  - `monthly_budget` режется на 50% от значения в `city.yaml.topics.<t>.monthly_budget`.
  - Период активного действия — 14 дней; через 14 дней без улучшений → авто-перевод в `paused` (logged как proposal в monthly digest).
- Если topic = `candidate`:
  - НЕ создавать draft. Только включить в `demand_analysis` и в monthly digest как "ready to enable".
