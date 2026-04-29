package direct

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// negativeKeywordBlocks — справочник минус-слов по блокам.
// Источник: .claude/skills/leadgen/mcp/negative_keywords.md (синхронизирован с веткой Codex).

// =============================================================================
// БЛОКИ ГОРОДОВ — для исключения чужих регионов из региональных кампаний
// =============================================================================

var citiesMoscowMO = []string{
	"москва", "мск", "московская", "подмосковье", "подмосковный", "зеленоград",
	"мытищи", "химки", "балашиха", "королев", "одинцово", "домодедово",
	"красногорск", "серпухов", "люберцы", "подольск", "реутов", "дмитров",
	"сергиев посад", "раменское", "электросталь", "орехово-зуево", "клин",
	"воскресенск", "коломна", "долгопрудный",
}

var citiesSPbLO = []string{
	"санкт-петербург", "санкт петербург", "спб", "питер", "петербург",
	"ленинградская", "гатчина", "всеволожск", "пушкин", "колпино", "выборг",
	"сосновый бор", "кингисепп", "кировск", "волхов", "кириши", "луга", "тихвин",
}

var citiesMillionnikiAndCenters = []string{
	"новосибирск", "нск", "екатеринбург", "екб", "казань", "нижний новгород",
	"челябинск", "самара", "ростов-на-дону", "ростов на дону", "уфа", "красноярск",
	"пермь", "воронеж", "волгоград", "краснодар", "саратов", "тюмень", "тольятти",
	"ижевск", "барнаул", "ульяновск", "иркутск", "хабаровск", "ярославль",
	"владивосток", "махачкала", "томск", "оренбург", "кемерово", "новокузнецк",
	"рязань", "астрахань", "набережные челны", "пенза", "липецк", "тула", "киров",
	"чебоксары", "калининград", "брянск", "курск", "иваново", "магнитогорск",
	"тверь", "ставрополь", "симферополь", "белгород", "архангельск", "владимир",
	"сочи", "калуга", "курган", "орёл", "орел", "смоленск", "череповец",
	"волжский", "саранск", "мурманск", "сургут", "омск", "нижневартовск",
	"нефтеюганск", "ханты-мансийск", "хмао", "янао", "новый уренгой", "ноябрьск",
	"нягань", "салехард", "надым", "тамбов", "нальчик", "якутск", "ялта",
	"обнинск",
}

var citiesInternationalCIS = []string{
	"москва сити", "минск", "алматы", "астана", "нур-султан", "бишкек",
	"ташкент", "ереван", "баку", "тбилиси", "душанбе", "киев", "харьков",
	"одесса", "дубай", "анталия", "стамбул", "белград", "пхукет", "паттайя",
	"северный кипр",
}

var citiesAbroadRealty = []string{
	"за рубежом", "за границей", "заграницей", "зарубежом", "турция", "оаэ",
	"эмираты", "дубай", "таиланд", "кипр", "болгария", "сербия", "казахстан",
	"грузия", "узбекистан", "беларусь", "армения", "азербайджан", "таджикистан",
	"туркменистан", "киргизия", "молдова", "украина",
}

// =============================================================================
// УНИВЕРСАЛЬНЫЕ БЛОКИ — для всех кампаний недвижимости
// =============================================================================

var blockInformational = []string{
	"что такое", "как выбрать", "как купить", "какая", "какой", "какие", "какую",
	"какое", "где купить", "сколько стоит", "сколько лет", "в чём разница",
	"что лучше", "что выбрать", "советы покупателю", "советы продавцу", "рейтинг",
	"топ", "топ-10", "топ 10", "обзор", "обзоры", "сравнение", "плюсы и минусы",
	"отзывы покупателей", "форум", "обсуждение", "комментарии", "мнения",
	"википедия", "wiki", "определение", "значение", "смысл", "виды", "типы",
	"классификация", "история", "схема", "чертёж", "план", "инструкция",
	"принцип работы", "устройство", "своими руками", "самостоятельно", "проект",
}

var blockFreeDownloadEducation = []string{
	"бесплатно", "скачать", "торрент", "без регистрации", "онлайн смотреть",
	"курс", "курсы", "обучение", "тренинг", "мастер-класс", "вебинар", "школа",
	"учебник", "как стать", "как научиться", "как делать", "реферат", "курсовая",
	"диплом", "эссе", "сочинение", "доклад", "презентация", "задача", "задание",
	"пример",
}

var blockIrrelevantServices = []string{
	"ремонт", "отделка", "дизайн интерьера", "уборка", "клининг", "переезд",
	"грузчики", "мебель", "техника", "обои", "плитка", "ламинат", "сантехника",
	"электрика", "стиральная машина", "холодильник", "диван", "кровать", "шкаф",
	"гарнитур", "строительство", "фундамент", "кровля", "стройматериалы",
	"кирпич", "блок", "сруб",
}

var blockJobs = []string{
	"вакансии", "работа", "зарплата", "оклад", "резюме", "трудоустройство",
	"риэлтор вакансия", "карьера", "устроиться", "подработка", "удалёнка",
	"удалённая",
}

var blockLegalBureaucracy = []string{
	"закон", "кодекс", "статья", "фз", "судебная практика", "наследство",
	"приватизация", "прописка", "регистрация", "выписка", "документы для",
	"оформление прав", "налог", "налоговый вычет", "декларация", "госуслуги",
	"мфц", "росреестр", "кадастр", "кадастровый", "егрн", "справка",
	"выписка из",
}

var blockMedicalFamily = []string{
	"роддом", "поликлиника", "больница", "школа", "детский сад", "детсад",
	"садик", "кружок", "секция", "репетитор", "армия", "военкомат", "тюрьма",
	"колония", "реабилитационный", "хоспис", "кладбище", "похороны",
	"ритуальный", "ритуал",
}

var blockNegativeComplaints = []string{
	"обман", "обманывают", "мошенники", "мошенничество", "развод", "кинули",
	"плохие отзывы", "жалоба", "претензия", "суд на", "подача иска", "жалоба в",
}

// =============================================================================
// ТЕМАТИЧЕСКИЕ БЛОКИ
// =============================================================================

var blockThemeVtorichka = []string{
	"новостройка", "от застройщика", "долевое", "ДДУ", "эскроу", "жк",
	"коммерческая", "офис", "склад", "торговое", "производственное", "гараж",
	"парковка", "машиноместо", "земельный участок", "дом", "дача", "коттедж",
	"сдать", "сдаю", "аренда", "снять", "посуточно", "обменять", "обмен",
	"дарение",
}

var blockThemeZagorodka = []string{
	"квартира", "однокомнатная", "двухкомнатная", "студия", "новостройка",
	"коммерческая", "офис", "склад", "аренда",
	"проект дома", "чертёж", "строительство", "фундамент", "кровля",
	"стройматериалы", "кирпич", "блок", "сруб", "каркасный", "брус",
	"сип панель", "модульный дом",
}

var blockThemeNovostroyki = []string{
	"вторичка", "вторичное", "вторичный рынок", "без посредников",
	"от собственника", "хрущёвка", "сталинка", "панельный", "кирпичный",
	"сдать", "снять", "аренда", "посуточно", "обмен", "коммерческая",
}

var blockThemeArenda = []string{
	"купить", "продать", "продажа", "ипотека", "новостройка", "от застройщика",
	"коммерческая", "посуточно", "командировка", "хостел", "гостиница", "отель",
}

var blockThemeIpoteka = []string{
	"рефинансирование", "реструктуризация", "банкротство", "просрочка",
	"военная ипотека", "сельская ипотека", "дальневосточная ипотека",
}

var blockThemeKommercheskaya = []string{
	"квартира", "однокомнатная", "двухкомнатная", "студия", "жилая",
	"новостройка вторичка", "ипотечная", "семейная", "детская",
}

// =============================================================================
// КОНКУРЕНТЫ
// =============================================================================

var blockCompetitorAggregators = []string{
	"циан", "cian", "домклик", "domclick", "авито", "avito", "юла",
	"яндекс недвижимость", "yandex realty", "m2", "mir kvartir",
	"мир квартир", "n1", "домофонд", "сбер недвижимость", "сбердомклик",
	"дом рф", "домрф", "квадратный метр", "kvadratnyy metr", "bnmap", "irr",
}

var blockCompetitorAgencies = []string{
	"инком", "incom", "миэль", "миелькомфорт", "метры", "метражи",
	"метр квадратный", "риэлторский центр", "агентство бест",
	"бест недвижимость", "новая волна", "аверс", "центральное агентство",
}

// =============================================================================
// РСЯ-СПЕЦИФИЧНЫЕ (с восклицательным знаком — точная форма)
// =============================================================================

var blockRSYACommon = []string{
	"!отделка", "!ремонт", "!своими руками", "!как выбрать", "!как купить",
	"!что это", "!значение", "!определение", "!википедия", "!форум",
	"!отзывы", "!бесплатно", "!реферат", "!скачать", "!образец",
	"!скачать договор",
}

// =============================================================================
// СОБСТВЕННЫЙ ГОРОД — словоформы для НЕ-исключения
// =============================================================================

// ownCityForms возвращает словоформы города, которые НЕ нужно минусовать.
// Маппинг базовых словоформ — частые случаи. Для остальных городов добавляется
// просто базовая форма + общий список ниже исключается.
var ownCityForms = map[string][]string{
	"москва":          {"москва", "московский", "московская", "московское", "московские", "мск"},
	"санкт-петербург": {"санкт-петербург", "санкт петербург", "спб", "питер", "петербург", "петербургский"},
	"екатеринбург":    {"екатеринбург", "екб", "екатеринбургский"},
	"казань":          {"казань", "казанский", "казанская"},
	"новосибирск":     {"новосибирск", "нск", "новосибирский"},
	"нижний новгород": {"нижний новгород", "нижегородский", "нижегородская", "нн"},
	"омск":            {"омск", "омский", "омская", "омское", "в омске"},
	"тюмень":          {"тюмень", "тюменский", "тюменская", "в тюмени"},
	"краснодар":       {"краснодар", "краснодарский", "краснодарская"},
	"ростов-на-дону":  {"ростов-на-дону", "ростов на дону", "ростовский", "ростовская"},
	"челябинск":       {"челябинск", "челябинский", "челябинская"},
	"уфа":             {"уфа", "уфимский", "уфимская"},
	"красноярск":      {"красноярск", "красноярский", "красноярская"},
	"пермь":           {"пермь", "пермский", "пермская"},
	"самара":          {"самара", "самарский", "самарская"},
	"сургут":          {"сургут", "сургутский", "сургутская"},
	"курган":          {"курган", "курганский", "курганская"},
	"набережные челны": {"набережные челны", "челны"},
}

// =============================================================================
// ТЕМАТИКИ — какие блоки подмешивать
// =============================================================================

var themeBlockMap = map[string][]string{
	"вторичка":     {"theme_vtorichka"},
	"новостройки":  {"theme_novostroyki"},
	"загородка":    {"theme_zagorodka"},
	"аренда":       {"theme_arenda"},
	"ипотека":      {"theme_ipoteka"},
	"коммерческая": {"theme_kommercheskaya"},
	"агентство":    {},
	"бренд":        {},
	"hr":           {},
}

var blocksByName = map[string][]string{
	"informational":           blockInformational,
	"free_download_education": blockFreeDownloadEducation,
	"irrelevant_services":     blockIrrelevantServices,
	"jobs":                    blockJobs,
	"legal_bureaucracy":       blockLegalBureaucracy,
	"medical_family":          blockMedicalFamily,
	"negative_complaints":     blockNegativeComplaints,
	"theme_vtorichka":         blockThemeVtorichka,
	"theme_zagorodka":         blockThemeZagorodka,
	"theme_novostroyki":       blockThemeNovostroyki,
	"theme_arenda":            blockThemeArenda,
	"theme_ipoteka":           blockThemeIpoteka,
	"theme_kommercheskaya":    blockThemeKommercheskaya,
	"competitor_aggregators":  blockCompetitorAggregators,
	"competitor_agencies":     blockCompetitorAgencies,
	"international_cis":       citiesInternationalCIS,
	"abroad_realty":           citiesAbroadRealty,
	"rsya_common":             blockRSYACommon,
}

// buildOtherCitiesRF собирает список «чужие города РФ» с исключением собственного города и его словоформ.
func buildOtherCitiesRF(ownCity string) []string {
	all := []string{}
	all = append(all, citiesMoscowMO...)
	all = append(all, citiesSPbLO...)
	all = append(all, citiesMillionnikiAndCenters...)

	if ownCity == "" {
		return uniqueLowercase(all)
	}

	exclude := map[string]bool{}
	exclude[strings.ToLower(ownCity)] = true
	if forms, ok := ownCityForms[strings.ToLower(ownCity)]; ok {
		for _, f := range forms {
			exclude[strings.ToLower(f)] = true
		}
	}

	out := make([]string, 0, len(all))
	for _, w := range all {
		if !exclude[strings.ToLower(w)] {
			out = append(out, w)
		}
	}
	return uniqueLowercase(out)
}

func uniqueLowercase(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, w := range in {
		k := strings.ToLower(strings.TrimSpace(w))
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func registerGetNegativeKeywordGuidance(s *mcpserver.MCPServer) {
	tool := mcp.NewTool("get_negative_keyword_guidance",
		mcp.WithDescription(
			"Сводка минус-слов по блокам для региональной кампании недвижимости. "+
				"Возвращает: блоки слов (чужие города РФ с исключением собственного, информационные, бесплатно/обучение, "+
				"нерелевантные услуги, работа, юр-бюрократия, медицина/семья, негатив/жалобы, тематический блок, "+
				"конкуренты), готовую строку через запятую для negative_keywords, чеклист сборки и хард-гейт ≥150 слов. "+
				"Заменяет ручное чтение mcp/negative_keywords.md."),
		mcp.WithString("theme",
			mcp.Description("Тематика: вторичка | новостройки | загородка | аренда | ипотека | коммерческая | агентство | бренд | hr"),
			mcp.Required()),
		mcp.WithString("city",
			mcp.Description("Собственный город кампании (русский) — будет исключён из списка «чужих городов РФ» вместе со словоформами. Узнать через get_city_config().")),
		mcp.WithString("placement",
			mcp.Description("Канал: search (по умолчанию) | rsya. Для rsya добавляется блок РСЯ-специфичных минусов с восклицательным знаком.")),
		mcp.WithBoolean("include_competitors",
			mcp.Description("Подмешать блоки конкурентов-агрегаторов и агентств (по умолчанию true). Выключи если кампания брендовая/конкурентная.")),
		mcp.WithBoolean("include_jobs",
			mcp.Description("Включить блок «работа/вакансии» (по умолчанию true). Выключи для HR-кампаний.")),
		mcp.WithBoolean("include_legal",
			mcp.Description("Включить блок «юр./бюрократия» (по умолчанию true). Выключи для юридических кампаний.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		theme := strings.ToLower(strings.TrimSpace(common.GetString(req, "theme")))
		city := strings.ToLower(strings.TrimSpace(common.GetString(req, "city")))
		placement := strings.ToLower(strings.TrimSpace(common.GetString(req, "placement")))
		if placement == "" {
			placement = "search"
		}

		// Defaults: true unless explicitly false
		includeCompetitors := req.GetBool("include_competitors", true)
		includeJobs := req.GetBool("include_jobs", true)
		includeLegal := req.GetBool("include_legal", true)

		if theme == "" {
			return common.ErrorResult("Параметр theme обязателен. Допустимые: вторичка, новостройки, загородка, аренда, ипотека, коммерческая, агентство, бренд, hr."), nil
		}
		if _, ok := themeBlockMap[theme]; !ok {
			keys := make([]string, 0, len(themeBlockMap))
			for k := range themeBlockMap {
				keys = append(keys, k)
			}
			return common.ErrorResult(fmt.Sprintf("Тематика '%s' не поддерживается. Допустимые: %s.", theme, strings.Join(keys, ", "))), nil
		}

		// Auto-disable jobs for HR campaigns
		if theme == "hr" {
			includeJobs = false
		}

		blocks := map[string][]string{}

		// Universal city block — always
		blocks["other_cities_rf"] = buildOtherCitiesRF(city)
		blocks["international_cis"] = citiesInternationalCIS
		blocks["abroad_realty"] = citiesAbroadRealty

		blocks["informational"] = blockInformational
		blocks["free_download_education"] = blockFreeDownloadEducation
		blocks["irrelevant_services"] = blockIrrelevantServices
		blocks["medical_family"] = blockMedicalFamily
		blocks["negative_complaints"] = blockNegativeComplaints

		if includeJobs {
			blocks["jobs"] = blockJobs
		}
		if includeLegal {
			blocks["legal_bureaucracy"] = blockLegalBureaucracy
		}

		// Theme-specific block(s)
		for _, name := range themeBlockMap[theme] {
			if words, ok := blocksByName[name]; ok {
				blocks[name] = words
			}
		}

		if includeCompetitors {
			blocks["competitor_aggregators"] = blockCompetitorAggregators
			blocks["competitor_agencies"] = blockCompetitorAgencies
		}

		// RSYA-specific
		if placement == "rsya" {
			blocks["rsya_common"] = blockRSYACommon
		}

		// Build final flat list (deduplicated)
		seen := map[string]bool{}
		flat := []string{}
		blockNames := make([]string, 0, len(blocks))
		for k := range blocks {
			blockNames = append(blockNames, k)
		}
		sort.Strings(blockNames)
		for _, name := range blockNames {
			for _, w := range blocks[name] {
				k := strings.ToLower(strings.TrimSpace(w))
				if k == "" || seen[k] {
					continue
				}
				seen[k] = true
				flat = append(flat, w)
			}
		}

		// Сomma-separated string for direct use in negative_keywords parameter
		summaryString := strings.Join(flat, ", ")

		hardGate := "PASS"
		gateNote := fmt.Sprintf("Слов в наборе: %d (хард-гейт: ≥150).", len(flat))
		if len(flat) < 150 {
			hardGate = "FAIL"
			gateNote += " Добери блоки или проверь, не пропущен ли «other_cities_rf»."
		}

		out := map[string]any{
			"theme":               theme,
			"city":                city,
			"placement":           placement,
			"include_competitors": includeCompetitors,
			"include_jobs":        includeJobs,
			"include_legal":       includeLegal,
			"blocks":              blocks,
			"block_sizes":         blockSizes(blocks),
			"total_words":         len(flat),
			"summary_string":      summaryString,
			"hard_gate":           hardGate,
			"hard_gate_note":      gateNote,
			"rules": []string{
				"Минус-слова добавляются на уровень КАМПАНИИ при add_campaign через negative_keywords=\"...\".",
				"На уровне групп — только для кросс-минусации (см. шаг C8 скилла).",
				"Хард-гейт перед add_campaign: ≥150 слов. Если меньше — проверь блок other_cities_rf и informational.",
				"API-лимит Яндекс Директ v5: 4096 слов / 20 000 символов на кампанию.",
				"При вызове передавай в negative_keywords строку через запятую (готовая в summary_string).",
				"Собственный город НЕ удаляй из своего списка — он автоматически исключён из other_cities_rf по словоформам.",
				"РСЯ-минусы (rsya_common) идут с восклицательным знаком (точная форма) — это норма, не ошибка.",
			},
			"source": ".claude/skills/leadgen/mcp/negative_keywords.md (синхронизировано с .codex/skills/leadgen-codex/mcp/negative_keywords.md)",
		}

		return common.JSONResult(out), nil
	})
}

func blockSizes(blocks map[string][]string) map[string]int {
	out := make(map[string]int, len(blocks))
	for k, v := range blocks {
		out[k] = len(v)
	}
	return out
}
