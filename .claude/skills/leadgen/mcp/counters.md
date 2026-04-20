# Счётчики Яндекс Метрики — Этажи

Аккаунт Метрики: `etagi.agent` (467 счётчиков, 368 на доменах Этажи).
Аккаунт Директа: `etagi click` (76 клиентских аккаунтов / городов).

> При работе с Метрикой всегда указывай `account="etagi.agent"`.
> При работе с Директом — без account (default = "etagi click").

## Как найти счётчик города

```
1. Вызови: metrika_get_counters(account="etagi.agent")
2. Фильтруй по домену: {city_subdomain}.etagi.com
3. Проверь activity_status = "high" (счётчик активен)
```

Домен Тюмени — `www.etagi.com` (или `etagi.com`), остальные города — `{city}.etagi.com`.

## Основные городские счётчики (activity_status = high)

| Город | Counter ID | Домен | Клиент Директа |
|---|---|---|---|
| Тюмень | 942898 | etagi.com (www) | porg-hftcfrrz |
| Москва | 19308763 | msk.etagi.com | porg-2w2kj6z6 |
| Санкт-Петербург | 38553395 | spb.etagi.com | porg-m3foz3ju |
| Екатеринбург | 12032014 | ekb.etagi.com | porg-ntnwy7i7 |
| Новосибирск | 18476308 | novosibirsk.etagi.com | porg-25knnffr |
| Челябинск | 23226073 | chel.etagi.com | porg-jcjf5iew |
| Краснодар | — | — | porg-hratjvxn |
| Омск | 22325545 | omsk.etagi.com | porg-cyjuzztm |
| Пермь | 25857344 | perm.etagi.com | — |
| Самара | 24841433 | samara.etagi.com | porg-7ho2evfz |
| Сургут | 1868509 | surgut.etagi.com | porg-scxkyykb |
| Ростов-на-Дону | 39714890 | rostov-na-donu.etagi.com | porg-653l43h7 |
| Нижний Новгород | 43505874 | nn.etagi.com | porg-jcceglb3 |
| Казань | — | — | porg-ljycd6no |
| Красноярск | — | — | porg-z5qhml3t |
| Уфа | — | — | porg-uvfvezxc |
| Набережные Челны | 31448038 | chelny.etagi.com | porg-kadk7de3 |
| Нижний Тагил | 23226067 | tagil.etagi.com | porg-dfdfpb3d |
| Курган | 45771603 | kurgan.etagi.com | porg-gwr73sm7 |
| Новый Уренгой | 10984753 | n-urengoy.etagi.com | porg-hvalirls |
| Тобольск | 16440742 | tobolsk.etagi.com | porg-aiahey2h |
| Ишим | 12031981 | ishim.etagi.com | porg-675e5ogx |
| Ханты-Мансийск | 26209575 | khm.etagi.com | porg-dgmcwugn |
| Хабаровск | — | — | porg-7gyixlh4 |
| Владивосток | — | — | porg-vnsatkuz |
| Тула | — | — | porg-m6y7ddob |
| Стерлитамак | 31447638 | sterlitamak.etagi.com | — |
| Тамбов | 56857531 | tambov.etagi.com | porg-4fm7mfgg |
| Саранск | 69271498 | saransk.etagi.com | porg-vncz2b2b |
| Нальчик | 84576808 | nalchik.etagi.com | porg-pbjdkab2 |
| Якутск | 40955514 | yakutsk.etagi.com | porg-5bbyxu75 |
| Ялта | 49089640 | yalta.etagi.com | porg-uz5qwfj2 |
| Дмитров | 72200521 | dmitrov.etagi.com | porg-5sd57v7i |
| Обнинск | 84396994 | obninsk.etagi.com | porg-t5zoyujy |

> **34 города** = пересечение клиентов Директа (76) и счётчиков Метрики (467).
> Города с прочерком в Counter ID — не найден активный счётчик при сопоставлении. Проверь через `metrika_get_counters(account="etagi.agent")` при необходимости.
> Города без client_login Директа — нет рекламного аккаунта (Пермь, Стерлитамак). Проверь через `get_agency_clients`.

## Общий счётчик (для сводной аналитики)

| Название | Counter ID | Домен |
|---|---|---|
| Этажи общий (все сайты) | 44267379 | etagi.com |

## Лендинги (etagi.net)

| Название | Counter ID | Домен | Статус |
|---|---|---|---|
| Тильда / Все сайты | 88635132 | lp.etagi.net | high |
| Все города / Тильда | 57160711 | all.etagi.net | high |
| Уфа / Тильда | 65741812 | ufa.etagi.net | high |
| Тюмень Tilda | 42715354 | tyumen.etagi.net | low |
| Санкт-Петербург Тильда | 48983717 | spb.etagi.net | low |
| Челябинск Тильда | 55962415 | chelyabinsk.etagi.net | low |
| Ростов / Тильда | 87353877 | rostov.etagi.net | low |
| Казань / Тильда | 52364596 | kazan.etagi.net | low |

## Этажи Прайм (etagiprime.com)

| Название | Counter ID | Домен | Статус |
|---|---|---|---|
| Москва Этажи Прайм Prime | 106892351 | etagiprime.com | high |
| etagiprime Тюмень | 89261268 | tmn.etagiprime.com | low |

## Правила

1. При создании кампании: `counter_ids` = счётчик города из таблицы выше
2. Если город не в списке — запросить `metrika_get_counters(account="etagi.agent")` и найти по домену
3. Если счётчика нет — спросить у пользователя, НЕ угадывать
4. Для лендингов ([LP]) — использовать счётчик etagi.net соответствующего города
5. Для сводной аналитики по всем городам — использовать общий счётчик 44267379
