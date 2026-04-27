# Промпты для генерации изображений РСЯ

> Инструкция для шага **R6.5** (генерация креативов) ветки R.
> Используется MCP-инструментами `generate_image` и `generate_banner_set`.
> Токен OpenRouter — в `server/config.yaml` секция `openrouter` или env `OPENROUTER_API_KEY`.

---

## Принципы генерации

### 1. Жёсткие правила (нерушимые)

1. **NO TEXT на картинке.** Текст задаётся заголовком/текстом самого объявления Директа. Картинка — только визуал.
2. **NO people.** Людей не генерируем — сложно проходят модерацию Яндекса + часто получаются с артефактами (лишние пальцы, странные лица).
3. **NO логотипы/брендинг.** Без лого Этажей или конкурентов.
4. **NO искусственные элементы.** Никаких watermark, рамок, надписей «реклама», плашек скидок.
5. **Только реализм.** Никаких иллюстраций, мультяшности, анимешных стилей, cartoon.
6. **Лимит 20 генераций на кампанию** (включая реджект+регенерации).

### 2. Переиспользование картинок между группами

Одна картинка (AdImageHash) может быть привязана к N группам. Стратегия:

```
кампания = 1 стилевой набор (4–6 базовых визуалов)
          × переиспользование в группах через AdImageHash
          + A/B варианты для оптимизации (OR.2)
```

**НЕ генерировать одну и ту же сцену дважды.** Если V3 (фасад ЖК) подходит и для группы «Общие-купить», и для «LAL» — одна картинка на обе.

### 3. Модели OpenRouter

| Модель | ID | Сильные стороны | Слабые стороны | Цена (~оценка) |
|---|---|---|---|---|
| **Flux 2 Pro** | `black-forest-labs/flux.2-pro` | Фотореализм интерьеров/фасадов, чёткая геометрия | Дороже | $0.03/МП (~$0.035 за 1080×1080) |
| **Gemini 3 Pro Image** | `google/gemini-3-pro-image-preview` | Контекстное понимание, композиция | Средний реализм, может добавлять текст | $2/M input + $12/M output (~$0.015/изображение) |
| **Gemini 3.1 Flash Image** | `google/gemini-3.1-flash-image-preview` | Быстро, экстремальные aspect ratios | Preview, нестабильно | Дешевле Pro |
| **Flux 2 Flex** | `black-forest-labs/flux.2-flex` | Бюджетный фолбэк | Качество ниже Pro | ~60% цены Pro |

**Дефолты:**
- V1 (интерьеры), V3 (фасады), V4 (частные дома) → **Flux 2 Pro** (фотореализм критичен).
- V2 (3D-рендер планировки), V6 (коммерческое) → **Gemini 3 Pro Image** (нужен контекст).
- Фолбэк при реджекте — вторая модель.

---

## Структура промпта

Каждый промпт собирается из 4 блоков:

```
[1] PHOTO_STYLE  — маркеры фотореализма (константа для V1/V3/V4)
[2] SCENE        — описание сцены из V1–V7 + подвариант
[3] CONTEXT      — инъекция: город, тематика, сегмент, детали
[4] NEGATIVE     — жёсткий негатив (константа)
```

---

## Константы

### PHOTO_STYLE_BASE (для реалистичных фото — V1, V3, V4, V5)

```
shot on Sony A7 IV, 35mm lens, natural window light,
RAW photo, professional real estate photography,
sharp focus, 8k resolution, editorial quality,
neutral white balance, no HDR, no oversaturation
```

### RENDER_STYLE_BASE (для 3D-рендеров — V2, V6)

```
professional architectural 3D render, isometric view,
realistic materials and textures, soft ambient lighting,
neutral color palette, clean composition, no artifacts
```

### NEGATIVE_HARD_BAN (константа ко всем промптам)

```
NO text, NO captions, NO words, NO letters, NO numbers,
NO logos, NO watermarks, NO labels, NO signs, NO frames,
NO people, NO human figures, NO faces, NO hands,
NO distorted furniture, NO impossible geometry,
NO floating objects, NO melting edges,
NO neon colors, NO oversaturation, NO HDR effect,
NO cartoon, NO illustration, NO anime, NO painting,
NO fish-eye distortion, NO vignette
```

### CITY_ARCHITECTURE_HINTS (инъекция по городу)

| Город | Архитектурная подсказка |
|---|---|
| Москва | modern urban architecture, high-end residential, glass facades |
| СПб | classic European style buildings, historic districts mixed with modern |
| Екатеринбург, Новосибирск | post-Soviet modernized residential, mid-rise mixed with high-rise |
| Омск, Тюмень, Курган | mid-rise brick and panel residential buildings, Russian regional style |
| Краснодар, Сочи | southern Russian architecture, light-colored facades, greenery |
| Казань | Volga region residential style, brick and concrete mid-rise |
| default (новый город) | contemporary Russian residential architecture |

---

## Визуальные направления (V1–V7)

### V1 — Фото интерьера квартиры

**Когда:** Вторичка, Новостройки (с отделкой), Аренда.

**Подтипы:**

#### V1.1 — Гостиная

```
modern bright living room interior, natural daylight through large window,
neutral tones (white, beige, light grey), minimal furniture: low sofa,
coffee table, indoor plants, hardwood floors, high ceiling
```

#### V1.2 — Кухня

```
modern kitchen interior, clean white cabinets, quartz countertop,
stainless steel appliances, natural daylight, kitchen island,
pendant lights, minimal styling, no dirty dishes
```

#### V1.3 — Спальня

```
modern bedroom interior, soft natural daylight, king-size bed with
crisp white linens, neutral wall colors, nightstand with lamp,
hardwood floor, indoor plants, minimal styling
```

#### V1.4 — Ванная

```
modern bathroom interior, white tiles, marble accents, freestanding
bathtub, glass shower cabin, natural light, minimalist styling,
chrome fixtures, clean lines
```

**Aspect ratios:** 1:1 (для 450×450 и 1080×1080), 16:9 (для 1920×1080).

---

### V2 — 3D-рендер планировки сверху

**Когда:** Новостройки, Вторичка (планы квартир).

```
top-down isometric 3D render of a {rooms}-bedroom apartment floor plan,
fully furnished: living room with sofa, kitchen with dining area,
bedroom(s) with beds, bathroom with fixtures,
realistic materials (hardwood floors, marble countertops, fabric sofas),
soft ambient lighting from above, no walls obstructing the view,
professional architectural visualization
```

Параметры инъекции:
- `{rooms}` — 1, 2, 3, 4
- дополнительно для премиум: «luxury finishing, designer furniture, walk-in closet»

**Aspect ratios:** 1:1, 4:3.

---

### V3 — Фасад ЖК / малоэтажки / новостройки

**Когда:** Новостройки, Вторичка (если важен вид дома).

```
residential building facade exterior, {city_hint},
well-maintained courtyard with benches and greenery,
modern playground visible, trees and flower beds,
clear sunny day, blue sky with soft clouds,
architectural photography style, wide angle, street level view
```

Параметры инъекции:
- `{city_hint}` — из CITY_ARCHITECTURE_HINTS
- высотность: mid-rise (5–9 этажей), high-rise (10+), low-rise (2–4)

**Aspect ratios:** 16:9 (главный), 3:2, 1:1.

---

### V4 — Частный дом

**Когда:** Загородка (покупатель).

```
modern private two-story house, {material} facade,
spacious plot with green lawn, landscaped garden,
paved driveway, clear summer day, blue sky,
architectural photography, wide angle, welcoming atmosphere,
no cars in frame
```

Параметры:
- `{material}` — brick / wooden cladding / stone / light-colored plaster
- стиль: modern minimalist / classic European / Scandinavian / traditional Russian

**Aspect ratios:** 16:9, 3:2.

---

### V5 — Земельный участок

**Когда:** Загородка (участки), Новостройки (на этапе котлована — с пометкой «perspective view»).

```
empty land plot for construction, green meadow,
young trees at the edges, country road leading to the plot,
summer day, blue sky with clouds, aerial or elevated view,
no buildings in frame, pristine countryside
```

**Aspect ratios:** 16:9, 3:2.

---

### V6 — Коммерческое помещение

**Когда:** Коммерческая недвижимость.

```
modern commercial space interior, open floor plan,
floor-to-ceiling windows, polished concrete floors,
exposed ceiling with pendant lights, neutral walls,
empty or minimal staging, professional commercial
real estate photography
```

Подтипы:
- `office` — office space with cubicles hint
- `retail` — retail storefront view with glass windows
- `warehouse` — warehouse interior with high ceiling, concrete floor

**Aspect ratios:** 16:9, 1:1.

---

### V7 — Символ сделки (ипотека, бренд)

**Когда:** Ипотека, Бренд (редко в РСЯ).

```
close-up macro photograph, single house key on a wooden desk,
blurred modern apartment interior in the background,
soft natural daylight, neutral tones, shallow depth of field,
no hands, no people, minimalist composition
```

Альтернатива: связка ключей / миниатюрный макет дома / подписанный документ (без читаемого текста).

**Aspect ratios:** 1:1, 16:9.

---

## Промпт-билдер (алгоритм для агента)

Вызывается на шаге R6.5. Псевдокод:

```python
def build_prompt(visual_type: str, context: dict) -> str:
    """
    visual_type: "V1.1", "V2", "V3", ...
    context: {
        "city": "омск",
        "theme": "вторичка",
        "segment": "двухкомнатные",     # опционально
        "rooms": 2,                      # для V2
        "style": "modern"                # опционально
    }
    """
    # 1. Выбираем стиль-базу
    if visual_type in ("V2", "V6"):
        style_base = RENDER_STYLE_BASE
    else:
        style_base = PHOTO_STYLE_BASE

    # 2. Берём сцену
    scene = SCENES[visual_type]  # из раздела V1–V7 выше

    # 3. Собираем контекст
    city_hint = CITY_ARCHITECTURE_HINTS.get(context["city"], CITY_ARCHITECTURE_HINTS["default"])
    context_str = f"Context: {context['theme']} in {context['city']}, {city_hint}"
    if context.get("segment"):
        context_str += f", segment: {context['segment']}"
    if context.get("rooms"):
        scene = scene.replace("{rooms}", str(context["rooms"]))

    # 4. Собираем полный промпт
    prompt = f"{style_base}. {scene}. {context_str}. {NEGATIVE_HARD_BAN}"
    return prompt
```

---

## Маппинг тематика → визуал (дефолт для агента)

Если пользователь не указал явно визуал — агент выбирает по тематике:

| Тематика | Визуал по умолчанию | Альтернатива |
|---|---|---|
| Вторичка, новостройки (покупатель) | V1 (интерьер) + V3 (фасад) | V2 (планировка) |
| Новостройки (на этапе котлована) | V3 (фасад-рендер) + V2 (план) | — |
| Загородка (дом) | V4 (частный дом) + V5 (участок) | — |
| Загородка (участок) | V5 (участок) + V4 | — |
| Аренда | V1 (интерьер) | — |
| Ипотека | V7 (ключ) + V1 (интерьер) | — |
| Коммерческая | V6 (офис/ритейл/склад) | — |
| Бренд, HR | V7 (символ сделки) | — |

---

## Распределение 20 генераций на кампанию

```
Базовый сет (12):
  3 визуала × 2 формата (1:1 + 16:9) × 2 вариации сцены = 12

A/B-пул для оптимизации (6):
  те же 3 визуала × 2 формата × 1 альтернативная сцена = 6

Резерв для реджектов (2):
  регенерация одной неудачной картинки

Всего: 20
```

**Правило:** если после 5 попыток regen на одну сцену нет приемлемого результата — фиксируем задачу на ручную подборку и идём дальше.

---

## Валидация результата (до `add_ad_image`)

### Автоматическая

1. **Размер файла** — должен быть ≤ 10 МБ (лимит Яндекса).
2. **Разрешение** — ≥ 450×450 для 1:1, ≥ 1080×607 для 16:9.
3. **Формат** — JPG/PNG. GIF без анимации допустим, но не генерируем.

### Полу-автоматическая (если есть OCR)

Опционально: прогнать картинку через tesseract/easyocr → если найден текст шириной > N пикселей → реджект. В MVP пропускаем, добавим позже.

### Ручная (в режиме `review`)

Картинка кладётся в `docs/campaign_previews/<slug>/`.
Пользователь смотрит → одобряет/реджектит по одной или пакетом.
После одобрения — `add_ad_image` + `add_ad`.

### В режиме `auto`

Пропускаем preview, льём сразу после автовалидации.
Риск — модерация Яндекса отклонит часть объявлений. Это нормально, доберём A/B в OR.2.

---

## Сохранение промпта в файл кампании

После генерации в `campaigns/<slug>.md` секция «Креативы» должна содержать:

```markdown
## Креативы

| # | Визуал | Формат | AdImageHash | Модель | Промпт (сокр.) | Группы |
|---|---|---|---|---|---|---|
| 1 | V1.1 гостиная | 1:1 | `abc123...` | flux.2-pro | `modern living room, Omsk...` | Общие-купить, LAL |
| 2 | V3 фасад | 16:9 | `def456...` | flux.2-pro | `mid-rise facade, Omsk...` | Общие-купить, 2-комн |
...
```

Полный промпт сохраняется в `campaigns/<slug>_prompts.json` (опционально, для последующей переработки).
