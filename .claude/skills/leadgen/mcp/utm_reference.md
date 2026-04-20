# Справочник UTM-параметров — Этажи

Справочные таблицы для формирования `utm_content` при создании кампаний.
Используется в `tracking_params` на уровне группы объявлений.

> Формат utm_content: `campn:{campaign_name}|gid:{gbid}|adid:{ad_id}|pid:{phrase_id}|pos:{position_type}_{position}|device:{device_type}|city:{Город}|city_id:{ID}|type:{тип}|type_id:{ID}|direction:{направление}`

---

## 1. type_id — Типы заявок (direction)

Определяет тематику/направление кампании в системе аналитики.

| type_id | Название | Когда использовать |
|---|---|---|
| **3** | **Вторичная (покупатель)** | Покупка вторичного жилья |
| **2** | **Вторичная (продавец)** | Продажа вторичного жилья |
| 3;2 | Вторичка (общая) | Общие кампании вторичка |
| **11** | **Загородная (покупатель)** | Покупка дома/дачи/участка |
| **10** | **Загородная (продавец)** | Продажа загородной |
| 11;10 | Загородная (общая) | Общие кампании загородка |
| **68;69;7** | **Новостройки** | Все кампании по новостройкам |
| **70;71;6** | **Ипотека** | Все ипотечные кампании |
| **5** | **Аренда (покупатель)** | Снять квартиру |
| **4** | **Аренда (продавец)** | Сдать квартиру |
| 5;4 | Аренда (общая) | Общие кампании аренда |
| **14** | **Коммерческая (покупатель)** | Купить/снять коммерческую |
| **15** | **Коммерческая (продавец)** | Продать/сдать коммерческую |
| 14;15 | Коммерческая (общая) | Общие кампании коммерческая |
| **59;61** | **Агентства** | Брендовые кампании |
| **63** | **Бренд** | Имиджевые кампании |
| **25** | **HR** | Рекрутинг |
| **667** | **Межрегиональные** | Межрег. кампании |
| **7777** | **Партнерские** | Партнёрские кампании |
| **126;8;9** | **Юридические** | Юридические услуги |
| **36;216** | **Ремонт, оценка** | Оценка имущества, ремонт |
| **185;186;187;188** | **Страхование** | Страховые продукты |
| — | **Не определено** | Если тематика не подходит |
| — | **Услуги** | Общие услуги |
| — | **Финансовые** | Финансовые продукты |

### Маппинг тематика кампании → type_id

| Тематика кампании | type_id для utm_content | direction для utm_content |
|---|---|---|
| Вторичка (покупатель) | 3 | vtorichka |
| Вторичка (продавец) | 2 | vtorichka_seller |
| Загородка (покупатель) | 11 | zagorodka |
| Загородка (продавец) | 10 | zagorodka_seller |
| Новостройки | 68 или 69 или 7 | novostroyki |
| Ипотека | 70 или 71 или 6 | ipoteka |
| Аренда (арендатор) | 5 | arenda |
| Аренда (арендодатель) | 4 | arenda_seller |
| Коммерческая (покупатель) | 14 | commerce |
| Коммерческая (продавец) | 15 | commerce_seller |
| Агентство / Бренд | 59 или 63 | agency_obshie |
| HR | 25 | hr |
| Межрегиональные | 667 | mezhreg |
| Страхование | 185 | strahovanie |

---

## 2. city_id — Города

### Города присутствия Этажи с Яндекс Директом (основные)

| Город | city_id | Поддомен | client_login Директа |
|---|---|---|---|
| Тюмень | 23 | www.etagi.com | porg-hftcfrrz |
| Москва | 155 | msk.etagi.com | porg-2w2kj6z6 |
| Санкт-Петербург | 242 | spb.etagi.com | porg-m3foz3ju |
| Екатеринбург | 45 | ekb.etagi.com | porg-ntnwy7i7 |
| Новосибирск | 135 | novosibirsk.etagi.com | porg-25knnffr |
| Челябинск | 91 | chel.etagi.com | porg-jcjf5iew |
| Краснодар | 223 | krasnodar.etagi.com | porg-hratjvxn |
| Омск | 85 | omsk.etagi.com | porg-cyjuzztm |
| Пермь | 238 | perm.etagi.com | — |
| Самара | 301 | samara.etagi.com | porg-7ho2evfz |
| Сургут | 74 | surgut.etagi.com | porg-scxkyykb |
| Ростов-на-Дону | 190 | rostov-na-donu.etagi.com | porg-653l43h7 |
| Нижний Новгород | 2832 | nn.etagi.com | porg-jcceglb3 |
| Казань | 180 | kazan.etagi.com | porg-ljycd6no |
| Красноярск | 224 | kras.etagi.com | porg-z5qhml3t |
| Уфа | 254 | ufa.etagi.com | porg-uvfvezxc |
| Набережные Челны | 230 | chelny.etagi.com | porg-kadk7de3 |
| Нижний Тагил | 147 | tagil.etagi.com | porg-dfdfpb3d |
| Курган | 151 | kurgan.etagi.com | porg-gwr73sm7 |
| Новый Уренгой | 86 | n-urengoy.etagi.com | porg-hvalirls |
| Тобольск | 28 | tobolsk.etagi.com | porg-aiahey2h |
| Ишим | 27 | ishim.etagi.com | porg-675e5ogx |
| Ханты-Мансийск | 47 | khm.etagi.com | porg-dgmcwugn |
| Хабаровск | 255 | khabarovsk.etagi.com | porg-7gyixlh4 |
| Владивосток | 211 | vl.etagi.com | porg-vnsatkuz |
| Тула | 178 | tula.etagi.com | porg-m6y7ddob |
| Стерлитамак | 246 | sterlitamak.etagi.com | — |
| Тамбов | 248 | tambov.etagi.com | porg-4fm7mfgg |
| Саранск | 243 | saransk.etagi.com | porg-vncz2b2b |
| Нальчик | 2592 | nalchik.etagi.com | porg-pbjdkab2 |
| Якутск | 258 | yakutsk.etagi.com | porg-5bbyxu75 |
| Ялта | 775 | yalta.etagi.com | porg-uz5qwfj2 |
| Дмитров | 2040 | dmitrov.etagi.com | porg-5sd57v7i |
| Обнинск | 990 | obninsk.etagi.com | porg-t5zoyujy |

### Полный справочник city_id (все города Этажи)

| Город | city_id | Поддомен |
|---|---|---|
| Alanya (закрыт) | 2815 | alanya.etagi.com |
| Antalya | 2974 | antalya.etagi.com |
| Belgrade | 2927 | belgrade.etagi.com |
| Dubai | 2863 | dubai.etagi.com |
| Northern Cyprus | 2835 | northcyprus.etagi.com |
| Pattaya | 2943 | pattaya.etagi.com |
| Phuket | 3002 | phuket.etagi.com |
| Абакан | 200 | abakan.etagi.com |
| Азербайджан | 3098 | az.etagi.com |
| Актау | 2195 | aktau.etagi.com |
| Актобе | 615 | aktobe.etagi.com |
| Алапаевск | 152 | alapaevsk.etagi.com |
| Алексеевка | 1968 | alekseevka.etagi.com |
| Алматы | 201 | almaty.etagi.com |
| Алушта | 1419 | alushta.etagi.com |
| Альметьевск | 283 | almetevsk.etagi.com |
| Анапа | 748 | anapa.etagi.com |
| Ангарск | 489 | angarsk.etagi.com |
| Анжеро-Судженск | 542 | anzhero-sudzhensk.etagi.com |
| Армения | 3097 | am.etagi.com |
| Арсеньев | 2439 | arsenev.etagi.com |
| Артем | 1338 | artem.etagi.com |
| Архангельск | 202 | arhangelsk.etagi.com |
| Астана | 577 | astana.etagi.com |
| Астрахань | 203 | astrakhan.etagi.com |
| Атырау | 2793 | atyrau.etagi.com |
| Ачинск | 573 | ach.etagi.com |
| Аша | 2615 | asha.etagi.com |
| Байкалово с. | 3141 | baykalovo.etagi.com |
| Баку | 2862 | baku.etagi.com |
| Балашов | 2171 | balashov.etagi.com |
| Бахчисарай | 2481 | bahchisaray.etagi.com |
| Беларусь | 3099 | belarus.etagi.com |
| Белгород | 205 | bel.etagi.com |
| Белогорск | 1299 | belogorsk.etagi.com |
| Бердск | 973 | berdsk.etagi.com |
| Березники | 826 | berezniki.etagi.com |
| Березовский | 2955 | berezovskiy.etagi.com |
| Бийск | 206 | biysk.etagi.com |
| Бирск | 1116 | birsk.etagi.com |
| Бирюч | 2445 | biryuch.etagi.com |
| Бишкек | 736 | bishkek.etagi.com |
| Благовещенск | 207 | blag.etagi.com |
| Богандинский р.п. | 1890 | boganda.etagi.com |
| Болгария | 3100 | bulgaria.etagi.com |
| Борисоглебск | 3076 | borisoglebsk.etagi.com |
| Брестская область | 3212 | ust-katav.etagi.com |
| Брянск | 209 | bryansk.etagi.com |
| Бургас | 2928 | burgas.etagi.com |
| Валуйки | 1993 | val.etagi.com |
| Вейделевка пгт. | 2173 | veydelevka.etagi.com |
| Верхняя Салда | 555 | vsalda.etagi.com |
| Вилючинск | 2358 | vilyuchinsk.etagi.com |
| Владивосток | 211 | vl.etagi.com |
| Владикавказ | 2755 | vladikavkaz.etagi.com |
| Владимир | 212 | vladimir.etagi.com |
| Волжск | 3024 | volzhsk.etagi.com |
| Вологда | 387 | vologda.etagi.com |
| Волоконовка пгт. | 2137 | volokonovka.etagi.com |
| Волхов | 2512 | volhov.etagi.com |
| Воронеж | 213 | voronezh.etagi.com |
| Воскресенск | 316 | voskresensk.etagi.com |
| Всеволожск | 1094 | vsevolozhsk.etagi.com |
| Выкса | 2426 | vyksa.etagi.com |
| Геленджик | 429 | gelendzhik.etagi.com |
| Глазов | 1529 | glazov.etagi.com |
| Голышманово р.п. | 2278 | golyshmanovo.etagi.com |
| Горно-Алтайск | 214 | gorno-altaysk.etagi.com |
| Горячий Ключ | 607 | gk.etagi.com |
| Губкин | 2659 | gubkin.etagi.com |
| Далматово | 325 | dalmatovo.etagi.com |
| Дмитров | 2040 | dmitrov.etagi.com |
| Домодедово | 2111 | domodedovo.etagi.com |
| Дубна | 2622 | dubna.etagi.com |
| Душанбе | 1026 | dushanbe.etagi.com |
| Евпатория | 1420 | evpatoria.etagi.com |
| Ейск | 1193 | eysk.etagi.com |
| Екатеринбург | 45 | ekb.etagi.com |
| Елизово | 2357 | elizovo.etagi.com |
| Ереван | 1489 | yerevan.etagi.com |
| Железноводск | 2655 | zheleznovodsk.etagi.com |
| Заводоуковск | 24 | zavodoukovsk.etagi.com |
| Златоуст | 1061 | zlatoust.etagi.com |
| Ижевск | 189 | izh.etagi.com |
| Излучинск пгт. | 3149 | izluchinsk.etagi.com |
| Ирбит | 341 | irbit.etagi.com |
| Иркутск | 184 | irk.etagi.com |
| Исетское с. | 3112 | isetskoe.etagi.com |
| Ишим | 27 | ishim.etagi.com |
| Йошкар-Ола | 217 | yoshkar-ola.etagi.com |
| Казань | 180 | kazan.etagi.com |
| Казахстан | 3071 | kz.etagi.com |
| Калининград | 218 | kaliningrad.etagi.com |
| Калуга | 219 | kaluga.etagi.com |
| Каменск-Уральский | 1995 | kamensk-uralskiy.etagi.com |
| Камышлов | 305 | kamyshlov.etagi.com |
| Карабулак | 2521 | karabulak.etagi.com |
| Кемерово | 220 | kem.etagi.com |
| Кириши | 2514 | kirishi.etagi.com |
| Киров | 221 | kirov.etagi.com |
| Кисловодск | 2650 | kislovodsk.etagi.com |
| Клин | 2324 | klin.etagi.com |
| Ковров | 2817 | kovrov.etagi.com |
| Кокшетау | 605 | kokshetau.etagi.com |
| Коломна | 2390 | kolomna.etagi.com |
| Комсомольск-на-Амуре | 1113 | kna.etagi.com |
| Короча | 2443 | korocha.etagi.com |
| Корсаков | 2225 | korsakov.etagi.com |
| Коряжма | 2851 | koryazhma.etagi.com |
| Костанай | 2256 | kostanay.etagi.com |
| Кострома | 222 | kostroma.etagi.com |
| Котлас | 2074 | kotlas.etagi.com |
| Краснодар | 223 | krasnodar.etagi.com |
| Красноярск | 224 | kras.etagi.com |
| Кумертау | 1393 | kumertau.etagi.com |
| Курган | 151 | kurgan.etagi.com |
| Курск | 225 | kursk.etagi.com |
| Куса | 2534 | kusa.etagi.com |
| Кызыл | 1084 | kyzyl.etagi.com |
| Кыргызстан | 3101 | kg.etagi.com |
| Кыштым | 315 | kyshtym.etagi.com |
| Лангепас | 308 | langepas.etagi.com |
| Ленинск-Кузнецкий | 226 | leninsk-kuznetskiy.etagi.com |
| Лермонтов | 2654 | lermontov.etagi.com |
| Липецк | 227 | lipetsk.etagi.com |
| Луховицы | 2963 | luhovitsy.etagi.com |
| Лянтор | 326 | lyantor.etagi.com |
| Майкоп | 2638 | maykop.etagi.com |
| Махачкала | 395 | mahachkala.etagi.com |
| Мегион | 76 | megion.etagi.com |
| Междуреченск | 386 | mezhdurechensk.etagi.com |
| Мелеуз | 1478 | meleuz.etagi.com |
| Миасс | 694 | miass.etagi.com |
| Минск | 1066 | minsk.etagi.com |
| Мичуринск | 2796 | michurinsk.etagi.com |
| Можга | 2625 | mozhga.etagi.com |
| Москва | 155 | msk.etagi.com |
| Муравленко | 310 | muravlenko.etagi.com |
| Мурманск | 1611 | murmansk.etagi.com |
| Муром | 832 | murom.etagi.com |
| Набережные Челны | 230 | chelny.etagi.com |
| Надым | 80 | nadym.etagi.com |
| Назрань | 2437 | nazran.etagi.com |
| Нальчик | 2592 | nalchik.etagi.com |
| Невинномысск | 1115 | nevinnomyssk.etagi.com |
| Нерчинск | 2523 | nerchinsk.etagi.com |
| Нефтекамск | 2837 | neftekamsk.etagi.com |
| Нефтеюганск | 95 | ugansk.etagi.com |
| Нижневартовск | 66 | vartovsk.etagi.com |
| Нижний Новгород | 2832 | nn.etagi.com |
| Нижний Тагил | 147 | tagil.etagi.com |
| Новодвинск | 2388 | novodvinsk.etagi.com |
| Новороссийск | 234 | novoros.etagi.com |
| Новосибирск | 135 | novosibirsk.etagi.com |
| Новый Оскол | 1971 | novyy-oskol.etagi.com |
| Новый Уренгой | 86 | n-urengoy.etagi.com |
| Норильск | 235 | norilsk.etagi.com |
| Ноябрьск | 69 | noyabrsk.etagi.com |
| Нягань | 84 | nyagan.etagi.com |
| ОАЭ | 3072 | uae.etagi.com |
| Обнинск | 990 | obninsk.etagi.com |
| Озерск | 985 | ozersk.etagi.com |
| Октябрьский (Башкортостан) | 950 | oktyabrskiy.etagi.com |
| Октябрьский (Пермский край) | 2938 | oktyabrskii.etagi.com |
| Омск | 85 | omsk.etagi.com |
| Орёл | 176 | orel.etagi.com |
| Оренбург | 187 | orenburg.etagi.com |
| Орехово-Зуево | 2475 | orehovo-zuevo.etagi.com |
| Остров | 2996 | ostrov.etagi.com |
| Павлодар | 499 | pavlodar.etagi.com |
| Пенза | 237 | pnz.etagi.com |
| Первоуральск | 145 | pervouralsk.etagi.com |
| Пермь | 238 | perm.etagi.com |
| Петрозаводск | 239 | ptz.etagi.com |
| Петропавловск | 598 | petropavlovsk.etagi.com |
| Петропавловск-Камчатский | 339 | petropavlovsk-kamchatskiy.etagi.com |
| Петушки | 2268 | petushki.etagi.com |
| Печоры | 2994 | pechory.etagi.com |
| Подольск | 1899 | podolsk.etagi.com |
| Пойковский пгт. | 117 | poykovskiy.etagi.com |
| Покров | 2261 | pokrov.etagi.com |
| Прокопьевск | 286 | prokopevsk.etagi.com |
| Прохладный | 2398 | prohladnyy.etagi.com |
| Псков | 1693 | pskov.etagi.com |
| Пыть-Ях | 73 | pyt-yah.etagi.com |
| Пятигорск | 819 | pyatigorsk.etagi.com |
| Раменское | 620 | ramenskoe.etagi.com |
| Ростов-на-Дону | 190 | rostov-na-donu.etagi.com |
| Рудный | 358 | rudnyy.etagi.com |
| Саки | 2428 | saki.etagi.com |
| Салават | 1487 | salavat.etagi.com |
| Салехард | 83 | salehard.etagi.com |
| Самара | 301 | samara.etagi.com |
| Санкт-Петербург | 242 | spb.etagi.com |
| Саранск | 243 | saransk.etagi.com |
| Сарапул | 1527 | sarapul.etagi.com |
| Саратов | 140 | saratov.etagi.com |
| Сатка | 2539 | satka.etagi.com |
| Свободный | 940 | svob.etagi.com |
| Севастополь | 327 | sevastopol.etagi.com |
| Северодвинск | 1797 | sev.etagi.com |
| Северск | 403 | seversk.etagi.com |
| Семей | 501 | semey.etagi.com |
| Сербия | 3105 | serbia.etagi.com |
| Сергиев Посад | 761 | sergiev-posad.etagi.com |
| Серпухов | 2847 | serpuhov.etagi.com |
| Симферополь | 774 | simf.etagi.com |
| Славянск-на-Кубани | 2636 | slavyansk-na-kubani.etagi.com |
| Смоленск | 244 | smolensk.etagi.com |
| Сочи | 199 | sochi.etagi.com |
| Ставрополь | 141 | stav.etagi.com |
| Старый Оскол | 245 | staryy-oskol.etagi.com |
| Стерлитамак | 246 | sterlitamak.etagi.com |
| Судак | 1702 | sudak.etagi.com |
| Сургут | 74 | surgut.etagi.com |
| Сыктывкар | 247 | sykt.etagi.com |
| Таджикистан | 3102 | tj.etagi.com |
| Таиланд | 3103 | thailand.etagi.com |
| Талица | 59 | talica.etagi.com |
| Тамбов | 248 | tambov.etagi.com |
| Тарко-Сале | 314 | tarko-sale.etagi.com |
| Ташкент | 2549 | tashkent.etagi.com |
| Тверь | 249 | tver.etagi.com |
| Темрюк | 2394 | temryuk.etagi.com |
| Тихорецк | 2899 | tihoretsk.etagi.com |
| Тобольск | 28 | tobolsk.etagi.com |
| Тольятти | 250 | tolyatti.etagi.com |
| Томск | 251 | tomsk.etagi.com |
| Троицк | 1102 | troitsk.etagi.com |
| Туапсе | 2765 | tuapse.etagi.com |
| Туймазы | 792 | tmz.etagi.com |
| Тула | 178 | tula.etagi.com |
| Туринск | 3001 | turinsk.etagi.com |
| Турция | 3070 | turkey.etagi.com |
| Тюмень | 23 | www.etagi.com |
| Узбекистан | 3104 | uz.etagi.com |
| Уйское с. | 710 | uyskoe.etagi.com |
| Улан-Удэ | 519 | ulan-ude.etagi.com |
| Ульяновск | 252 | ul.etagi.com |
| Уральск | 2542 | uralsk.etagi.com |
| Усинск | 330 | usinsk.etagi.com |
| Усть-Каменогорск | 407 | ust-kamenogorsk.etagi.com |
| Уфа | 254 | ufa.etagi.com |
| Ухта | 1927 | ukhta.etagi.com |
| Феодосия | 1709 | feodosia.etagi.com |
| Хабаровск | 255 | khabarovsk.etagi.com |
| Ханты-Мансийск | 47 | khm.etagi.com |
| Чебаркуль | 717 | chebarkul.etagi.com |
| Чебоксары | 256 | cheboksary.etagi.com |
| Челябинск | 91 | chel.etagi.com |
| Череповец | 1312 | cher.etagi.com |
| Черкесск | 2797 | cherkessk.etagi.com |
| Черногорск | 1563 | chernogorsk.etagi.com |
| Черноморское пгт. | 3222 | uray.etagi.com |
| Чернушка | 3209 | chernyshevsk.etagi.com |
| Чита | 257 | chita.etagi.com |
| Чолпон-Ата | 3108 | cholpon-ata.etagi.com |
| Шадринск | 132 | shadrinsk.etagi.com |
| Шексна | 1928 | sheksna.etagi.com |
| Шымкент | 2392 | shymkent.etagi.com |
| Щёлкино | 2458 | schelkino.etagi.com |
| Экибастуз | 572 | ekibastuz.etagi.com |
| Электросталь | 1824 | elektrostal.etagi.com |
| Элиста | 2427 | elista.etagi.com |
| Энгельс | 1105 | engels.etagi.com |
| Южно-Сахалинск | 192 | sakhalin.etagi.com |
| Южноуральск | 816 | yuzhnouralsk.etagi.com |
| Юрга | 383 | urga.etagi.com |
| Якутск | 258 | yakutsk.etagi.com |
| Ялта | 775 | yalta.etagi.com |
| Ялуторовск | 26 | yalutorovsk.etagi.com |
| Ярославль | 259 | yar.etagi.com |
| Ярцево | 2889 | yartsevo.etagi.com |

---

## 3. type — Тип размещения (латиница для utm_content)

| Тип кампании | type для UTM |
|---|---|
| Поиск | poisk |
| РСЯ | rsya |
| Мастер кампаний | master |
| ЕПК | epk |
| Динамические | dynamic |
| Медийная | mediynaya |
| Товарная | tovarnaya |

---

## 4. Примеры готовых utm_content

**Тюмень, Поиск, Вторичка (покупатель):**
```
city:Тюмень|city_id:23|type:poisk|type_id:3|direction:vtorichka
```

**Краснодар, РСЯ, Загородка (покупатель):**
```
city:Краснодар|city_id:223|type:rsya|type_id:11|direction:zagorodka
```

**Омск, Мастер, Новостройки:**
```
city:Омск|city_id:85|type:master|type_id:68|direction:novostroyki
```

**Челябинск, Поиск, Ипотека:**
```
city:Челябинск|city_id:91|type:poisk|type_id:70|direction:ipoteka
```

**Сургут, РСЯ, Аренда (арендатор):**
```
city:Сургут|city_id:74|type:rsya|type_id:5|direction:arenda
```
