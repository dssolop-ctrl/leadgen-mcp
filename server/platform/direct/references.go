package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// titleCase capitalizes first letter of each word (simple replacement for deprecated strings.Title).
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r, size := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToUpper(r)) + w[size:]
	}
	return strings.Join(words, " ")
}

// RegisterReferenceTools registers in-memory lookup tools that replace skill config files.
func RegisterReferenceTools(s *mcpserver.MCPServer) {
	registerGetCityConfig(s)
	registerGetUTMConfig(s)
	registerGetBlockedPlacements(s)
	registerGetSitelinkTemplates(s)
	registerGetSemanticCluster(s)
}

// ===== CITY CONFIG =====

type cityConfig struct {
	CityID      int    `json:"city_id"`
	Subdomain   string `json:"subdomain"`
	Domain      string `json:"domain"`
	CounterID   string `json:"counter_id,omitempty"`
	ClientLogin string `json:"client_login,omitempty"`
	GeoRegionID string `json:"geo_region_id,omitempty"`
	// Tier: "tier_1" (1M+ pop / federal), "tier_2" (300K-1M), "tier_3" (<300K).
	// Used by get_conversion_values for network-fallback CPA selection.
	Tier string `json:"tier,omitempty"`
}

var cities = map[string]cityConfig{
	"тюмень":             {23, "www", "etagi.com", "942898", "porg-hftcfrrz", "55", "tier_2"},
	"москва":             {155, "msk", "msk.etagi.com", "19308763", "porg-2w2kj6z6", "1", "tier_1"},
	"санкт-петербург":    {242, "spb", "spb.etagi.com", "38553395", "porg-m3foz3ju", "2", "tier_1"},
	"екатеринбург":       {45, "ekb", "ekb.etagi.com", "12032014", "porg-ntnwy7i7", "54", "tier_1"},
	"новосибирск":        {135, "novosibirsk", "novosibirsk.etagi.com", "18476308", "porg-25knnffr", "65", "tier_1"},
	"челябинск":          {91, "chel", "chel.etagi.com", "23226073", "porg-jcjf5iew", "56", "tier_1"},
	"краснодар":          {223, "krasnodar", "krasnodar.etagi.com", "", "porg-hratjvxn", "35", "tier_1"},
	"омск":               {85, "omsk", "omsk.etagi.com", "22325545", "porg-cyjuzztm", "66", "tier_1"},
	"пермь":              {238, "perm", "perm.etagi.com", "25857344", "", "50", "tier_1"},
	"самара":             {301, "samara", "samara.etagi.com", "24841433", "porg-7ho2evfz", "51", "tier_1"},
	"сургут":             {74, "surgut", "surgut.etagi.com", "1868509", "porg-scxkyykb", "973", "tier_2"},
	"ростов-на-дону":     {190, "rostov-na-donu", "rostov-na-donu.etagi.com", "39714890", "porg-653l43h7", "39", "tier_1"},
	"нижний новгород":    {2832, "nn", "nn.etagi.com", "43505874", "porg-jcceglb3", "47", "tier_1"},
	"казань":             {180, "kazan", "kazan.etagi.com", "", "porg-ljycd6no", "43", "tier_1"},
	"красноярск":         {224, "kras", "kras.etagi.com", "", "porg-z5qhml3t", "62", "tier_1"},
	"уфа":                {254, "ufa", "ufa.etagi.com", "", "porg-uvfvezxc", "5", "tier_1"},
	"набережные челны":   {230, "chelny", "chelny.etagi.com", "31448038", "porg-kadk7de3", "236", "tier_2"},
	"нижний тагил":       {147, "tagil", "tagil.etagi.com", "23226067", "porg-dfdfpb3d", "11171", "tier_3"},
	"курган":             {151, "kurgan", "kurgan.etagi.com", "45771603", "porg-gwr73sm7", "53", "tier_3"},
	"новый уренгой":      {86, "n-urengoy", "n-urengoy.etagi.com", "10984753", "porg-hvalirls", "103735", "tier_3"},
	"тобольск":           {28, "tobolsk", "tobolsk.etagi.com", "16440742", "porg-aiahey2h", "11168", "tier_3"},
	"ишим":               {27, "ishim", "ishim.etagi.com", "12031981", "porg-675e5ogx", "11169", "tier_3"},
	"ханты-мансийск":     {47, "khm", "khm.etagi.com", "26209575", "porg-dgmcwugn", "572", "tier_3"},
	"хабаровск":          {255, "khabarovsk", "khabarovsk.etagi.com", "", "porg-7gyixlh4", "76", "tier_2"},
	"владивосток":        {211, "vl", "vl.etagi.com", "", "porg-vnsatkuz", "75", "tier_2"},
	"тула":               {178, "tula", "tula.etagi.com", "", "porg-m6y7ddob", "15", "tier_2"},
	"стерлитамак":        {246, "sterlitamak", "sterlitamak.etagi.com", "31447638", "", "11111", "tier_3"},
	"тамбов":             {248, "tambov", "tambov.etagi.com", "56857531", "porg-4fm7mfgg", "13", "tier_3"},
	"саранск":            {243, "saransk", "saransk.etagi.com", "69271498", "porg-vncz2b2b", "42", "tier_3"},
	"нальчик":            {2592, "nalchik", "nalchik.etagi.com", "84576808", "porg-pbjdkab2", "11070", "tier_3"},
	"якутск":             {258, "yakutsk", "yakutsk.etagi.com", "40955514", "porg-5bbyxu75", "74", "tier_3"},
	"ялта":               {775, "yalta", "yalta.etagi.com", "49089640", "porg-uz5qwfj2", "11470", "tier_3"},
	"дмитров":            {2040, "dmitrov", "dmitrov.etagi.com", "72200521", "porg-5sd57v7i", "10716", "tier_3"},
	"обнинск":            {990, "obninsk", "obninsk.etagi.com", "84396994", "porg-t5zoyujy", "10857", "tier_3"},
}

// TierForLogin looks up the tier for a given client_login (linear scan; called rarely).
// Returns "tier_3" as conservative default if login is unknown.
func TierForLogin(login string) string {
	if login == "" {
		return "tier_3"
	}
	for _, c := range cities {
		if c.ClientLogin == login {
			if c.Tier != "" {
				return c.Tier
			}
			return "tier_3"
		}
	}
	return "tier_3"
}

// CityNameForLogin returns the Russian city name for a given client_login, or "".
func CityNameForLogin(login string) string {
	if login == "" {
		return ""
	}
	for name, c := range cities {
		if c.ClientLogin == login {
			return name
		}
	}
	return ""
}

func registerGetCityConfig(s *mcpserver.MCPServer) {
	tool := mcp.NewTool("get_city_config",
		mcp.WithDescription(
			"Конфигурация города Этажи: counter_id, client_login, city_id, domain, geo_region_id. "+
				"Заменяет ручной поиск в counters.md и utm_reference.md. Без параметра city — список всех городов."),
		mcp.WithString("city", mcp.Description("Название города (русский, нижний регистр). Без параметра — весь список.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cityName := strings.ToLower(strings.TrimSpace(common.GetString(req, "city")))

		if cityName == "" {
			// Return all cities summary
			type cityRow struct {
				City        string `json:"city"`
				CityID      int    `json:"city_id"`
				CounterID   string `json:"counter_id,omitempty"`
				ClientLogin string `json:"client_login,omitempty"`
				Domain      string `json:"domain"`
			}
			var rows []cityRow
			for name, c := range cities {
				rows = append(rows, cityRow{name, c.CityID, c.CounterID, c.ClientLogin, c.Domain})
			}
			out, _ := json.MarshalIndent(rows, "", "  ")
			return common.TextResult(string(out)), nil
		}

		// Exact match
		if c, ok := cities[cityName]; ok {
			out, _ := json.MarshalIndent(map[string]any{
				"city":          cityName,
				"city_id":       c.CityID,
				"subdomain":     c.Subdomain,
				"domain":        c.Domain,
				"counter_id":    c.CounterID,
				"client_login":  c.ClientLogin,
				"geo_region_id": c.GeoRegionID,
			}, "", "  ")
			return common.TextResult(string(out)), nil
		}

		// Fuzzy search — contains
		var matches []string
		for name := range cities {
			if strings.Contains(name, cityName) {
				matches = append(matches, name)
			}
		}
		if len(matches) > 0 {
			return common.TextResult(fmt.Sprintf("Город '%s' не найден. Похожие: %s", cityName, strings.Join(matches, ", "))), nil
		}
		return common.TextResult(fmt.Sprintf("Город '%s' не найден. Используй get_city_config() без параметров для списка всех городов.", cityName)), nil
	})
}

// ===== UTM CONFIG =====

type themeConfig struct {
	TypeID    string `json:"type_id"`
	Direction string `json:"direction"`
}

var themes = map[string]themeConfig{
	"вторичка":              {"3", "vtorichka"},
	"вторичка_покупатель":   {"3", "vtorichka"},
	"вторичка_продавец":     {"2", "vtorichka_seller"},
	"загородка":             {"11", "zagorodka"},
	"загородка_покупатель":  {"11", "zagorodka"},
	"загородка_продавец":    {"10", "zagorodka_seller"},
	"новостройки":           {"68", "novostroyki"},
	"ипотека":               {"70", "ipoteka"},
	"аренда":                {"5", "arenda"},
	"аренда_покупатель":     {"5", "arenda"},
	"аренда_продавец":       {"4", "arenda_seller"},
	"коммерческая":          {"14", "commerce"},
	"коммерческая_покупатель": {"14", "commerce"},
	"коммерческая_продавец": {"15", "commerce_seller"},
	"агентство":             {"59", "agency_obshie"},
	"бренд":                 {"63", "agency_obshie"},
	"hr":                    {"25", "hr"},
}

var placementTypes = map[string]string{
	"поиск":    "poisk",
	"рся":      "rsya",
	"мастер":   "master",
	"епк":      "epk",
	"динамика": "dynamic",
	"медийная": "mediynaya",
}

func registerGetUTMConfig(s *mcpserver.MCPServer) {
	tool := mcp.NewTool("get_utm_config",
		mcp.WithDescription(
			"Сгенерировать готовые UTM-параметры для tracking_params группы объявлений. "+
				"Возвращает полный utm_content и tracking_params строку."),
		mcp.WithString("city", mcp.Description("Название города (русский)"), mcp.Required()),
		mcp.WithString("theme", mcp.Description("Тематика: вторичка, загородка, новостройки, ипотека, аренда, коммерческая, агентство, бренд, hr"), mcp.Required()),
		mcp.WithString("placement", mcp.Description("Тип размещения: поиск, рся, мастер, епк, динамика, медийная (по умолчанию: поиск)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cityName := strings.ToLower(strings.TrimSpace(common.GetString(req, "city")))
		themeName := strings.ToLower(strings.TrimSpace(common.GetString(req, "theme")))
		placement := strings.ToLower(strings.TrimSpace(common.GetString(req, "placement")))
		if placement == "" {
			placement = "поиск"
		}

		city, ok := cities[cityName]
		if !ok {
			return common.ErrorResult(fmt.Sprintf("Город '%s' не найден. Используй get_city_config() для списка.", cityName)), nil
		}

		theme, ok := themes[themeName]
		if !ok {
			keys := make([]string, 0, len(themes))
			for k := range themes {
				keys = append(keys, k)
			}
			return common.ErrorResult(fmt.Sprintf("Тематика '%s' не найдена. Доступные: %s", themeName, strings.Join(keys, ", "))), nil
		}

		plType, ok := placementTypes[placement]
		if !ok {
			return common.ErrorResult(fmt.Sprintf("Тип '%s' не найден. Доступные: поиск, рся, мастер, епк, динамика, медийная", placement)), nil
		}

		// Build utm_content static part
		utmContent := fmt.Sprintf("campn:{campaign_name}|gid:{gbid}|adid:{ad_id}|pid:{phrase_id}|pos:{position_type}_{position}|device:{device_type}|city:%s|city_id:%d|type:%s|type_id:%s|direction:%s",
			titleCase(cityName), city.CityID, plType, theme.TypeID, theme.Direction)

		// Full tracking_params
		trackingParams := fmt.Sprintf("utm_source=yandex&utm_medium=cpc&utm_campaign={campaign_id}&utm_content=%s&utm_term={keyword}&utm_pos={position_type}", utmContent)

		// City display name with proper case
		displayCity := titleCase(cityName)

		out, _ := json.MarshalIndent(map[string]any{
			"city":            displayCity,
			"city_id":         city.CityID,
			"type":            plType,
			"type_id":         theme.TypeID,
			"direction":       theme.Direction,
			"utm_content":     utmContent,
			"tracking_params": trackingParams,
			"domain":          city.Domain,
		}, "", "  ")
		return common.TextResult(string(out)), nil
	})
}

// ===== BLOCKED PLACEMENTS =====

func registerGetBlockedPlacements(s *mcpserver.MCPServer) {
	tool := mcp.NewTool("get_blocked_placements",
		mcp.WithDescription(
			"Стандартный чёрный список площадок РСЯ для кампаний Этажи. 400+ площадок (игры, развлечения, мусорные приложения). "+
				"Применяй ко ВСЕМ РСЯ-кампаниям."),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		out, _ := json.MarshalIndent(map[string]any{
			"total":      len(blockedPlacements),
			"placements": blockedPlacements,
			"blocked_networks": []string{
				"Smaato", "MobFox", "MoPub", "InMobi", "Fyber", "Chartboost",
			},
			"note": "Применяй через интерфейс Директа (excluded_sites). MCP пока не поддерживает автоматическое применение.",
		}, "", "  ")
		return common.TextResult(string(out)), nil
	})
}

// ===== SITELINKS =====

type sitelink struct {
	Text        string `json:"text"`
	Description string `json:"description"`
	URLSuffix   string `json:"url_suffix"`
}

type sitelinkSet struct {
	Theme     string     `json:"theme"`
	Placement string     `json:"placement"`
	Source    string     `json:"source,omitempty"`
	Links     []sitelink `json:"links"`
}

func registerGetSitelinkTemplates(s *mcpserver.MCPServer) {
	tool := mcp.NewTool("get_sitelink_templates",
		mcp.WithDescription(
			"Библиотека шаблонов быстрых ссылок (sitelinks) для объявлений Этажи. "+
				"Возвращает наборы по тематике. URL адаптируется под город автоматически. "+
				"Для работы с существующими сайтлинками в Директе используй get_sitelinks."),
		mcp.WithString("theme", mcp.Description("Тематика: вторичка, новостройки, загородка, аренда, ипотека, коммерческая, агентство"), mcp.Required()),
		mcp.WithString("city", mcp.Description("Город — для подстановки домена в URL")),
		mcp.WithString("placement", mcp.Description("Тип: поиск или рся (по умолчанию: поиск)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		themeName := strings.ToLower(strings.TrimSpace(common.GetString(req, "theme")))
		cityName := strings.ToLower(strings.TrimSpace(common.GetString(req, "city")))
		placement := strings.ToLower(strings.TrimSpace(common.GetString(req, "placement")))
		if placement == "" {
			placement = "поиск"
		}

		domain := "etagi.com"
		if cityName != "" {
			if c, ok := cities[cityName]; ok {
				domain = c.Domain
			}
		}

		sets, ok := sitelinkLibrary[themeName]
		if !ok {
			keys := make([]string, 0, len(sitelinkLibrary))
			for k := range sitelinkLibrary {
				keys = append(keys, k)
			}
			return common.ErrorResult(fmt.Sprintf("Тематика '%s' не найдена. Доступные: %s", themeName, strings.Join(keys, ", "))), nil
		}

		// Filter by placement if applicable
		var filtered []sitelinkSet
		for _, set := range sets {
			if set.Placement == "" || strings.Contains(set.Placement, placement) {
				filtered = append(filtered, set)
			}
		}

		// Substitute domain in URLs
		for i, set := range filtered {
			for j, link := range set.Links {
				filtered[i].Links[j].URLSuffix = "https://" + domain + link.URLSuffix
			}
		}

		out, _ := json.MarshalIndent(map[string]any{
			"theme":     themeName,
			"domain":    domain,
			"placement": placement,
			"sets":      filtered,
			"rules":     "Мин. 4 ссылки, оптимально 8. Описания обязательны. URL уникальные. Формат для add_sitelinks: title|description|url;title2|description2|url2",
		}, "", "  ")
		return common.TextResult(string(out)), nil
	})
}

// ===== SEMANTIC CLUSTERS =====

type semanticCluster struct {
	Theme      string   `json:"theme"`
	Modifiers  []string `json:"modifiers"`
	Objects    []string `json:"objects"`
	ExtraKeys  []string `json:"extra_keys,omitempty"`
	NegativeKw []string `json:"negative_keywords"`
	Notes      string   `json:"notes,omitempty"`
}

func registerGetSemanticCluster(s *mcpserver.MCPServer) {
	tool := mcp.NewTool("get_semantic_cluster",
		mcp.WithDescription(
			"Семантический кластер для создания ключевых фраз кампании. "+
				"Возвращает модификаторы, типы объектов, минус-слова по тематике. "+
				"Используй как стартовую точку, расширяй через check_search_volume."),
		mcp.WithString("theme", mcp.Description("Тематика: вторичка, вторичка_продавец, загородка, новостройки, ипотека, аренда, коммерческая, бренд, hr"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		themeName := strings.ToLower(strings.TrimSpace(common.GetString(req, "theme")))

		cluster, ok := semanticClusters[themeName]
		if !ok {
			keys := make([]string, 0, len(semanticClusters))
			for k := range semanticClusters {
				keys = append(keys, k)
			}
			return common.ErrorResult(fmt.Sprintf("Тематика '%s' не найдена. Доступные: %s", themeName, strings.Join(keys, ", "))), nil
		}

		out, _ := json.MarshalIndent(map[string]any{
			"cluster": cluster,
			"usage":   "Комбинируй: {модификатор} + {объект} + {город}. Расширяй через check_search_volume. Минус-слова — добавляй сразу.",
		}, "", "  ")
		return common.TextResult(string(out)), nil
	})
}
