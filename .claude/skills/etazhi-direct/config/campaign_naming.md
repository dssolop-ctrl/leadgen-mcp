# Конвенция именования кампаний Этажи

## Формат имени кампании

```
{Город} | {Тип размещения} | {Тематика} ({уточнение}) | {Детализация/посыл} | [{посадка}]
```

### Примеры из кабинета

```
Тюмень | Поиск | Загородка | Купить дом коттедж дачу | [site]
Тюмень | РСЯ | Вторичка | Скидки | Комн [site]
Тюмень | Мастер кампаний | Новостройки (общие) | Рассрочка | [site]
Краснодар | Поиск | Аренда (жилая / покупатель) | [LP]
Краснодар | Медийная | Имидж | Безопасная сделка [site]
Санкт-Петербург | Медийная | Видеореклама - УТП | [site]
```

## Компоненты

### {Город}
Название города на русском. Примеры: Тюмень, Краснодар, Санкт-Петербург, Челябинск

### {Тип размещения}

| Значение | API тип | Описание |
|---|---|---|
| Поиск | TEXT_CAMPAIGN (network=OFF) | Поисковая реклама |
| РСЯ | TEXT_CAMPAIGN (search=OFF) | Рекламная сеть Яндекса |
| ЕПК | UNIFIED_CAMPAIGN | Единая перфоманс-кампания |
| Мастер кампаний | UNIFIED_CAMPAIGN | Автоматизированная кампания |
| Динамические объявления | DYNAMIC_TEXT_CAMPAIGN | Автогенерация из фида/сайта |
| Медийная | CPM_BANNER_CAMPAIGN | Охватная кампания с CPM |
| товарная кампания | SMART_CAMPAIGN | Товарные баннеры по фиду |

### {Тематика}

| Значение | Описание | Основная посадочная |
|---|---|---|
| Вторичка (покупатель) | Покупка вторичного жилья | /realty/ |
| Вторичка | Общее по вторичке | /realty/ |
| Загородка (покупатель) | Дома, коттеджи, дачи, участки | /realty_out/ |
| Новостройки (общие) | Все новостройки от застройщиков | /zastr/ |
| Новостройки (ЖК) | Конкретный ЖК | /zastr/{jk_slug}/ |
| Аренда (жилая / покупатель) | Аренда квартир | /arenda/ |
| Коммерческая (покупатель) | Коммерческая недвижимость | /commerce/ |
| Ипотека (общие) | Ипотечные программы | /ipoteka/ |
| Агентство недвижимости | Брендовые запросы | / (главная) |
| Имидж | Имиджевая/охватная | / или лендинг |
| HR | Рекрутинг | /career/ |

### {Детализация}
Свободное поле — конкретный посыл, акция, ЖК, комнатность, гео-расширение.
Примеры: `Купить дом коттедж дачу`, `Скидки`, `Рассрочка`, `Семейная ипотека`, `Однокомнатные`

### [{посадка}]

| Значение | Домен | Описание |
|---|---|---|
| [site] | {city}.etagi.com | Основной сайт города |
| [LP] | etagi.net или lp.etagi.net | Лендинг |

## UTM-шаблон

### Полный шаблон

```
utm_source=yandex&utm_medium=cpc&utm_campaign={campaign_id}&utm_content=campn:{campaign_name}|gid:{gbid}|adid:{ad_id}|pid:{phrase_id}|pos:{position_type}_{position}|device:{device_type}|city:{Город}|city_id:{ID_города}|type:{тип}|type_id:{ID_типа}|direction:{направление}&utm_term={keyword}&utm_pos={position_type}
```

### Параметры utm_content (pipe-separated)

| Ключ | Значение | Источник |
|---|---|---|
| `campn` | `{campaign_name}` | Автоподстановка Директ |
| `gid` | `{gbid}` | Автоподстановка — ID группы |
| `adid` | `{ad_id}` | Автоподстановка — ID объявления |
| `pid` | `{phrase_id}` | Автоподстановка — ID фразы |
| `pos` | `{position_type}_{position}` | Автоподстановка — позиция |
| `device` | `{device_type}` | Автоподстановка — тип устройства |
| `city` | Название города (кириллица) | Вручную: `Тюмень`, `Краснодар` |
| `city_id` | ID города в системе аналитики | Вручную: из шаблона аналогичной кампании |
| `type` | Тип размещения (латиница) | Вручную: `poisk`, `rsya`, `master`, `mediynaya` |
| `type_id` | ID типа в системе аналитики | Вручную: из шаблона аналогичной кампании |
| `direction` | Направление/тематика | Вручную: `agency_obshie`, `vtorichka`, `zagorodka`, `novostroyki`, `ipoteka`, `arenda` |

### Автоподстановки Директа (в `{}`)

```
{campaign_id}    — ID кампании
{campaign_name}  — название кампании
{gbid}           — ID группы объявлений
{ad_id}          — ID объявления
{phrase_id}      — ID ключевой фразы
{keyword}        — текст ключевой фразы
{position_type}  — тип блока (premium, other, none)
{position}       — позиция в блоке
{device_type}    — desktop, mobile, tablet
```

### Примеры

**Тюмень, Поиск, Вторичка:**
```
utm_source=yandex&utm_medium=cpc&utm_campaign={campaign_id}&utm_content=campn:{campaign_name}|gid:{gbid}|adid:{ad_id}|pid:{phrase_id}|pos:{position_type}_{position}|device:{device_type}|city:Тюмень|city_id:23|type:poisk|type_id:59|direction:vtorichka&utm_term={keyword}&utm_pos={position_type}
```

**Краснодар, РСЯ, Загородка:**
```
utm_source=yandex&utm_medium=cpc&utm_campaign={campaign_id}&utm_content=campn:{campaign_name}|gid:{gbid}|adid:{ad_id}|pid:{phrase_id}|pos:{position_type}_{position}|device:{device_type}|city:Краснодар|city_id:44|type:rsya|type_id:60|direction:zagorodka&utm_term={keyword}&utm_pos={position_type}
```

### Как определить city_id, type_id, direction

1. **Лучший способ:** скопировать из аналогичной кампании того же города/типа
2. Найди существующую кампанию: `get_campaigns(client_login=<login>, states="ON")`
3. Получи группу: `get_adgroups(client_login=<login>, campaign_ids=<id>)` → посмотри `tracking_params`
4. Если нет аналога — **спроси специалиста** для city_id и type_id

### Техническое

Устанавливается через `tracking_params` на уровне группы (`update_adgroup` или `add_adgroup`).

## Slug кампании для файла

Файл кампании: `campaigns/{slug}.md`

Формирование slug: `{город}_{тип}_{тематика}` в нижнем регистре, латиницей.
Примеры: `tyumen_poisk_zagorodka`, `krasnodar_rsya_novostroyki`
