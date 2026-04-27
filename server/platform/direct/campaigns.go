package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterCampaignTools registers all campaign-related MCP tools.
func RegisterCampaignTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetCampaigns(s, client, resolver)
	registerAddCampaign(s, client, resolver)
	registerUpdateCampaign(s, client, resolver)
	registerManageCampaigns(s, client, resolver)
	registerSuspendCampaign(s, client, resolver)
	registerResumeCampaign(s, client, resolver)
	registerArchiveCampaign(s, client, resolver)
}

func registerGetCampaigns(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_campaigns",
		mcp.WithDescription("Получить список кампаний Яндекс Директа. Всегда фильтруй по states. Без фильтра вернёт ВСЕ включая архивные."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("states", mcp.Description("Статусы: ON, SUSPENDED, OFF, ENDED, ARCHIVED")),
		mcp.WithString("field_names", mcp.Description("Базовые поля: Id, Name, State, Status, DailyBudget и др. Для вложенных TextCampaign/UnifiedCampaign/DynamicTextCampaign используй соответствующие *_field_names.")),
		mcp.WithString("text_campaign_field_names", mcp.Description("Поля внутри TextCampaign (возвращается только если кампания типа TEXT_CAMPAIGN): BiddingStrategy, TrackingParams, Settings, CounterIds и др. Пример: BiddingStrategy,TrackingParams")),
		mcp.WithString("dynamic_text_campaign_field_names", mcp.Description("Поля внутри DynamicTextCampaign (только для DYNAMIC_TEXT_CAMPAIGN). Пример: BiddingStrategy,TrackingParams")),
		mcp.WithString("unified_campaign_field_names", mcp.Description("Поля внутри UnifiedCampaign/ЕПК (только для UNIFIED_CAMPAIGN). Пример: BiddingStrategy,TrackingParams,PlacementTypes")),
		mcp.WithNumber("limit", mcp.Description("Макс. кампаний (умолч 100)")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		// Build SelectionCriteria
		criteria := map[string]any{}

		if states := common.GetStringSlice(req, "states"); len(states) > 0 {
			criteria["States"] = states
		}

		if ids := common.GetStringSlice(req, "campaign_ids"); len(ids) > 0 {
			intIDs := make([]int64, 0, len(ids))
			for _, id := range ids {
				n, err := strconv.ParseInt(id, 10, 64)
				if err == nil {
					intIDs = append(intIDs, n)
				}
			}
			criteria["Ids"] = intIDs
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
		}

		if fields := common.GetStringSlice(req, "field_names"); len(fields) > 0 {
			params["FieldNames"] = fields
		} else {
			params["FieldNames"] = []string{"Id", "Name", "State", "Status"}
		}

		// Тип-специфичные FieldNames (для чтения вложенных объектов TextCampaign/UnifiedCampaign/DynamicTextCampaign).
		// Нужны, например, чтобы получить BiddingStrategy.Search.PlacementTypes.DynamicPlaces.
		if tf := common.GetStringSlice(req, "text_campaign_field_names"); len(tf) > 0 {
			params["TextCampaignFieldNames"] = tf
		}
		if df := common.GetStringSlice(req, "dynamic_text_campaign_field_names"); len(df) > 0 {
			params["DynamicTextCampaignFieldNames"] = df
		}
		if uf := common.GetStringSlice(req, "unified_campaign_field_names"); len(uf) > 0 {
			params["UnifiedCampaignFieldNames"] = uf
		}

		limit := common.GetInt(req, "limit")
		if limit <= 0 {
			limit = 50
		}
		params["Page"] = map[string]int{"Limit": limit}

		raw, err := client.Call(ctx, token, "campaigns", "get", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		return common.TextResult(string(result)), nil
	})
}

func registerAddCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_campaign",
		mcp.WithDescription("Создать кампанию в Яндекс Директе. Бюджет в РУБЛЯХ (недельный). Стратегия: начинай с WB_MAXIMUM_CONVERSION_RATE."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("name", mcp.Description("Название кампании"), mcp.Required()),
		mcp.WithNumber("daily_budget_amount", mcp.Description("Недельный бюджет в рублях"), mcp.Required()),
		mcp.WithString("daily_budget_mode", mcp.Description("Режим: STANDARD или DISTRIBUTED (умолч)")),
		mcp.WithString("search_strategy", mcp.Description("Стратегия поиска: WB_MAXIMUM_CONVERSION_RATE, AVERAGE_CPA, WB_MAXIMUM_CLICKS и др."), mcp.Required()),
		mcp.WithString("network_strategy", mcp.Description("Стратегия сетей: SERVING_OFF (по умолчанию), NETWORK_DEFAULT, WB_MAXIMUM_CLICKS, WB_MAXIMUM_CONVERSION_RATE, AVERAGE_CPA. Для РСЯ-кампаний используй WB_MAXIMUM_CONVERSION_RATE с search_strategy=SERVING_OFF.")),
		mcp.WithNumber("network_weekly_budget", mcp.Description("Недельный бюджет для Network-стратегии, рубли. Если не задан — берётся daily_budget_amount.")),
		mcp.WithNumber("network_average_cpa", mcp.Description("Целевая CPA для Network AVERAGE_CPA, рубли.")),
		mcp.WithNumber("network_bid_ceiling", mcp.Description("Верхний потолок ставки клика для Network-автостратегии, рубли. Применим к WB_MAXIMUM_CONVERSION_RATE / AVERAGE_CPA / WB_MAXIMUM_CLICKS. Формула для РСЯ: tCPA × 1.5. Мин. 0.3 ₽.")),
		mcp.WithNumber("search_bid_ceiling", mcp.Description("Верхний потолок ставки клика для Search-автостратегии, рубли. Применим к WB_MAXIMUM_CONVERSION_RATE / AVERAGE_CPA / WB_MAXIMUM_CLICKS. Мин. 0.3 ₽.")),
		mcp.WithNumber("goal_id", mcp.Description("ID цели Метрики (одна). Для нескольких — priority_goals.")),
		mcp.WithString("priority_goals", mcp.Description("Приоритетные цели JSON. При передаче GoalId ставится 13.")),
		mcp.WithString("counter_ids", mcp.Description("ID счётчиков Метрики")),
		mcp.WithNumber("average_cpa", mcp.Description("Целевая CPA (для AVERAGE_CPA)")),
		mcp.WithString("start_date", mcp.Description("Начало (YYYY-MM-DD)")),
		mcp.WithString("negative_keywords", mcp.Description("Минус-фразы через запятую")),
		mcp.WithString("settings", mcp.Description("Настройки JSON: [{\"option\":\"ENABLE_AREA_OF_INTEREST_TARGETING\",\"value\":false}]")),
		mcp.WithString("tracking_params", mcp.Description("UTM-метки на уровне кампании. Наследуется всеми группами. Рекомендуется вместо per-group. Вид: utm_source=yandex&utm_medium=cpc&utm_campaign={campaign_id}&...")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		name := common.GetString(req, "name")
		budgetAmount := common.GetInt(req, "daily_budget_amount")
		budgetMode := common.GetString(req, "daily_budget_mode")
		if budgetMode == "" {
			budgetMode = "DISTRIBUTED"
		}

		searchStrategy := common.GetString(req, "search_strategy")
		networkStrategy := common.GetString(req, "network_strategy")
		if networkStrategy == "" {
			networkStrategy = "SERVING_OFF"
		}

		// Build campaign object. Default StartDate to today when the caller omits it —
		// the API rejects empty StartDate (error 5003).
		startDate := common.GetString(req, "start_date")
		if startDate == "" {
			startDate = time.Now().Format("2006-01-02")
		}
		campaign := map[string]any{
			"Name":      name,
			"StartDate": startDate,
		}

		// DailyBudget only for manual strategies (not WB_* / AVERAGE_*).
		// Pure RSYA has Search=SERVING_OFF and Network=WB_*, so the Network block
		// must also suppress top-level DailyBudget.
		isAutoStrategy := strings.HasPrefix(searchStrategy, "WB_") || strings.HasPrefix(searchStrategy, "AVERAGE_") ||
			strings.HasPrefix(networkStrategy, "WB_") || strings.HasPrefix(networkStrategy, "AVERAGE_")
		if !isAutoStrategy && budgetAmount > 0 {
			campaign["DailyBudget"] = map[string]any{
				"Amount": budgetAmount * 1000000,
				"Mode":   budgetMode,
			}
		}

		// Strategy — build strategy-specific settings
		searchSettings := map[string]any{
			"BiddingStrategyType": searchStrategy,
		}

		weeklyBudgetMicros := budgetAmount * 1000000
		goalID := common.GetInt(req, "goal_id")
		avgCPA := common.GetInt(req, "average_cpa")

		// Parse priority goals (multiple goals with conversion values)
		var priorityGoals []map[string]any
		if pgStr := common.GetString(req, "priority_goals"); pgStr != "" {
			var pgItems []struct {
				GoalID int64 `json:"goal_id"`
				Value  int64 `json:"value"`
			}
			if err := json.Unmarshal([]byte(pgStr), &pgItems); err == nil && len(pgItems) > 0 {
				for _, pg := range pgItems {
					valueMicros := pg.Value * 1000000
					// API minimum is 300000 micros (0.3 RUB). Default to 1 RUB if not specified.
					if valueMicros < 300000 {
						valueMicros = 1000000 // 1 RUB
					}
					item := map[string]any{
						"GoalId":                 pg.GoalID,
						"Value":                  valueMicros,
						"IsMetrikaSourceOfValue": "NO",
					}
					priorityGoals = append(priorityGoals, item)
				}
				// When priority_goals is set, strategy GoalId should be 13 (= priority goals)
				goalID = 13
			}
		}

		// BidCeiling for auto strategies — caps the per-click bid the auto-bidder
		// can reach. Rubles → micros, Yandex API minimum is 0.3 RUB (300000 micros).
		searchBidCeilingMicros := bidCeilingMicros(common.GetInt(req, "search_bid_ceiling"))

		switch searchStrategy {
		case "WB_MAXIMUM_CLICKS":
			wbClicks := map[string]any{
				"WeeklySpendLimit": weeklyBudgetMicros,
			}
			if searchBidCeilingMicros > 0 {
				wbClicks["BidCeiling"] = searchBidCeilingMicros
			}
			searchSettings["WbMaximumClicks"] = wbClicks
		case "WB_MAXIMUM_CONVERSION_RATE":
			wbSettings := map[string]any{
				"WeeklySpendLimit": weeklyBudgetMicros,
			}
			if goalID > 0 {
				wbSettings["GoalId"] = goalID
			}
			if searchBidCeilingMicros > 0 {
				wbSettings["BidCeiling"] = searchBidCeilingMicros
			}
			searchSettings["WbMaximumConversionRate"] = wbSettings
		case "AVERAGE_CPA":
			avgSettings := map[string]any{
				"WeeklySpendLimit": weeklyBudgetMicros,
			}
			if avgCPA > 0 {
				avgSettings["AverageCpa"] = avgCPA * 1000000
			}
			if goalID > 0 {
				avgSettings["GoalId"] = goalID
			}
			if searchBidCeilingMicros > 0 {
				avgSettings["BidCeiling"] = searchBidCeilingMicros
			}
			searchSettings["AverageCpa"] = avgSettings
		}

		// Динамические места показа (договорённость с SEO: ВСЕГДА off).
		// BiddingStrategy.Search.PlacementTypes.DynamicPlaces = "NO".
		// Применимо к TextCampaign / DynamicTextCampaign.
		searchSettings["PlacementTypes"] = map[string]any{
			"DynamicPlaces": "NO",
		}

		// Network-strategy: for auto types (WB_*, AVERAGE_*) Yandex requires nested
		// settings identical in shape to Search. Needed for RSYA campaigns where
		// the network_strategy carries the autobid logic (WB_MAX_CONV_RATE etc).
		networkSettings := map[string]any{
			"BiddingStrategyType": networkStrategy,
		}
		networkWeeklyRub := common.GetInt(req, "network_weekly_budget")
		if networkWeeklyRub <= 0 {
			networkWeeklyRub = budgetAmount
		}
		networkWeeklyMicros := networkWeeklyRub * 1000000
		networkAvgCPA := common.GetInt(req, "network_average_cpa")
		networkBidCeilingMicros := bidCeilingMicros(common.GetInt(req, "network_bid_ceiling"))

		switch networkStrategy {
		case "WB_MAXIMUM_CLICKS":
			wbClicksNet := map[string]any{
				"WeeklySpendLimit": networkWeeklyMicros,
			}
			if networkBidCeilingMicros > 0 {
				wbClicksNet["BidCeiling"] = networkBidCeilingMicros
			}
			networkSettings["WbMaximumClicks"] = wbClicksNet
		case "WB_MAXIMUM_CONVERSION_RATE":
			wbNet := map[string]any{
				"WeeklySpendLimit": networkWeeklyMicros,
			}
			if goalID > 0 {
				wbNet["GoalId"] = goalID
			}
			if networkBidCeilingMicros > 0 {
				wbNet["BidCeiling"] = networkBidCeilingMicros
			}
			networkSettings["WbMaximumConversionRate"] = wbNet
		case "AVERAGE_CPA":
			avgNet := map[string]any{
				"WeeklySpendLimit": networkWeeklyMicros,
			}
			if networkAvgCPA > 0 {
				avgNet["AverageCpa"] = networkAvgCPA * 1000000
			} else if avgCPA > 0 {
				avgNet["AverageCpa"] = avgCPA * 1000000
			}
			if goalID > 0 {
				avgNet["GoalId"] = goalID
			}
			if networkBidCeilingMicros > 0 {
				avgNet["BidCeiling"] = networkBidCeilingMicros
			}
			networkSettings["AverageCpa"] = avgNet
		}

		biddingStrategy := map[string]any{
			"Search":  searchSettings,
			"Network": networkSettings,
		}

		textCampaign := map[string]any{
			"BiddingStrategy": biddingStrategy,
		}

		// Add PriorityGoals if specified
		if len(priorityGoals) > 0 {
			textCampaign["PriorityGoals"] = map[string]any{
				"Items": priorityGoals,
			}
		}

		// Campaign-level UTM (TrackingParams) — наследуется всеми группами.
		if tp := common.GetString(req, "tracking_params"); tp != "" {
			textCampaign["TrackingParams"] = tp
		}

		campaign["TextCampaign"] = textCampaign

		// Counter IDs
		if counterIDs := common.GetStringSlice(req, "counter_ids"); len(counterIDs) > 0 {
			ids := make([]int64, 0, len(counterIDs))
			for _, id := range counterIDs {
				n, _ := strconv.ParseInt(id, 10, 64)
				if n > 0 {
					ids = append(ids, n)
				}
			}
			campaign["TextCampaign"].(map[string]any)["CounterIds"] = map[string]any{"Items": ids}
		}

		// Negative keywords
		if negKw := common.GetString(req, "negative_keywords"); negKw != "" {
			campaign["NegativeKeywords"] = map[string]any{
				"Items": strings.Split(negKw, ","),
			}
		}

		// Campaign settings (e.g., ENABLE_AREA_OF_INTEREST_TARGETING)
		if settingsStr := common.GetString(req, "settings"); settingsStr != "" {
			var settingsItems []struct {
				Option string `json:"option"`
				Value  any    `json:"value"`
			}
			if err := json.Unmarshal([]byte(settingsStr), &settingsItems); err == nil && len(settingsItems) > 0 {
				var settings []map[string]any
				for _, s := range settingsItems {
					valStr := "YES"
					switch v := s.Value.(type) {
					case bool:
						if !v {
							valStr = "NO"
						}
					case string:
						valStr = v
					}
					settings = append(settings, map[string]any{
						"Option": s.Option,
						"Value":  valStr,
					})
				}
				textCampaign["Settings"] = settings
			}
		}

		params := map[string]any{
			"Campaigns": []any{campaign},
		}

		raw, err := client.Call(ctx, token, "campaigns", "add", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		return common.TextResult(string(result)), nil
	})
}

func registerUpdateCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_campaign",
		mcp.WithDescription("Обновить кампанию Яндекс Директа. Частичное обновление — указывай только изменяемые поля. Бюджет в РУБЛЯХ."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название")),
		mcp.WithNumber("daily_budget_amount", mcp.Description("Новый недельный бюджет в рублях")),
		mcp.WithString("search_strategy", mcp.Description("Новая стратегия поиска")),
		mcp.WithNumber("goal_id", mcp.Description("Новая цель Метрики (одна). Для нескольких — используй priority_goals + goal_id=13.")),
		mcp.WithString("priority_goals", mcp.Description("Приоритетные цели JSON: [{\"goal_id\":123,\"value\":0},{\"goal_id\":456,\"value\":0}]. value — ценность в рублях (0 = минимум 1₽). При передаче GoalId автоматически ставится 13.")),
		mcp.WithNumber("average_cpa", mcp.Description("Целевая CPA (для стратегии AVERAGE_CPA)")),
		mcp.WithString("tracking_params", mcp.Description("UTM-метки на уровне кампании. Наследуется всеми группами. Для миграции UTM с уровня группы на кампанию — указывай здесь.")),
		mcp.WithBoolean("disable_dynamic_places", mcp.Description("Отключить динамические места показа (BiddingStrategy.Search.PlacementTypes.DynamicPlaces=NO). Договорённость с SEO. Требует search_strategy для применения.")),
		mcp.WithString("network_strategy", mcp.Description("Стратегия сетей: SERVING_OFF, NETWORK_DEFAULT, WB_MAXIMUM_CLICKS, WB_MAXIMUM_CONVERSION_RATE, AVERAGE_CPA. Передача обновляет Network-блок BiddingStrategy целиком (включая nested settings).")),
		mcp.WithNumber("network_weekly_budget", mcp.Description("Новый недельный бюджет Network-стратегии, рубли. Требует network_strategy.")),
		mcp.WithNumber("network_average_cpa", mcp.Description("Новая целевая CPA для Network AVERAGE_CPA, рубли.")),
		mcp.WithNumber("network_bid_ceiling", mcp.Description("Новый верхний потолок ставки клика Network-автостратегии, рубли. Требует network_strategy для пересылки блока Network целиком.")),
		mcp.WithNumber("search_bid_ceiling", mcp.Description("Новый верхний потолок ставки клика Search-автостратегии, рубли. Требует search_strategy для пересылки блока Search целиком.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignID := common.GetInt(req, "campaign_id")
		campaign := map[string]any{
			"Id": campaignID,
		}

		if name := common.GetString(req, "name"); name != "" {
			campaign["Name"] = name
		}

		if budget := common.GetInt(req, "daily_budget_amount"); budget > 0 {
			campaign["DailyBudget"] = map[string]any{
				"Amount": budget * 1000000,
				"Mode":   "DISTRIBUTED",
			}
		}

		// Parse priority goals (multiple goals with conversion values)
		var priorityGoals []map[string]any
		if pgStr := common.GetString(req, "priority_goals"); pgStr != "" {
			var pgItems []struct {
				GoalID int64 `json:"goal_id"`
				Value  int64 `json:"value"`
			}
			if err := json.Unmarshal([]byte(pgStr), &pgItems); err == nil && len(pgItems) > 0 {
				for _, pg := range pgItems {
					valueMicros := pg.Value * 1000000
					if valueMicros < 300000 {
						valueMicros = 1000000 // 1 RUB default
					}
					item := map[string]any{
						"GoalId":                 pg.GoalID,
						"Value":                  valueMicros,
						"IsMetrikaSourceOfValue": "NO",
						"Operation":              "SET",
					}
					priorityGoals = append(priorityGoals, item)
				}
			}
		}

		// Strategy updates
		searchStrategy := common.GetString(req, "search_strategy")
		goalID := common.GetInt(req, "goal_id")
		avgCPA := common.GetInt(req, "average_cpa")

		// When priority_goals is set, GoalId should be 13 (= optimize for priority goals)
		if len(priorityGoals) > 0 && goalID == 0 {
			goalID = 13
		}

		trackingParams := common.GetString(req, "tracking_params")
		disableDynamicPlaces := common.GetBool(req, "disable_dynamic_places")

		networkStrategy := common.GetString(req, "network_strategy")
		networkWeeklyRub := common.GetInt(req, "network_weekly_budget")
		networkAvgCPA := common.GetInt(req, "network_average_cpa")
		networkBidCeilingMicros := bidCeilingMicros(common.GetInt(req, "network_bid_ceiling"))
		searchBidCeilingMicros := bidCeilingMicros(common.GetInt(req, "search_bid_ceiling"))

		needTextCampaign := searchStrategy != "" || goalID > 0 || avgCPA > 0 || len(priorityGoals) > 0 ||
			trackingParams != "" || disableDynamicPlaces ||
			networkStrategy != "" || networkWeeklyRub > 0 || networkAvgCPA > 0 ||
			networkBidCeilingMicros > 0 || searchBidCeilingMicros > 0

		if needTextCampaign {
			textCampaign := map[string]any{}

			biddingStrategyUpdate := map[string]any{}

			if searchStrategy != "" || goalID > 0 || avgCPA > 0 || disableDynamicPlaces || searchBidCeilingMicros > 0 {
				search := map[string]any{}
				if searchStrategy != "" {
					search["BiddingStrategyType"] = searchStrategy
				}

				// GoalId, AverageCpa, BidCeiling go inside the strategy-specific key.
				switch searchStrategy {
				case "WB_MAXIMUM_CONVERSION_RATE":
					stratSettings := map[string]any{}
					if goalID > 0 {
						stratSettings["GoalId"] = goalID
					}
					if searchBidCeilingMicros > 0 {
						stratSettings["BidCeiling"] = searchBidCeilingMicros
					}
					search["WbMaximumConversionRate"] = stratSettings
				case "AVERAGE_CPA":
					stratSettings := map[string]any{}
					if goalID > 0 {
						stratSettings["GoalId"] = goalID
					}
					if avgCPA > 0 {
						stratSettings["AverageCpa"] = avgCPA * 1000000
					}
					if searchBidCeilingMicros > 0 {
						stratSettings["BidCeiling"] = searchBidCeilingMicros
					}
					search["AverageCpa"] = stratSettings
				case "WB_MAXIMUM_CLICKS":
					stratSettings := map[string]any{}
					if searchBidCeilingMicros > 0 {
						stratSettings["BidCeiling"] = searchBidCeilingMicros
					}
					search["WbMaximumClicks"] = stratSettings
				default:
					// If no strategy specified but goalID changed, we still need the strategy type
					// to know where to put GoalId — caller must provide search_strategy
					if goalID > 0 {
						search["GoalId"] = goalID
					}
				}

				// Динамические места показа — off по договорённости с SEO.
				if disableDynamicPlaces {
					search["PlacementTypes"] = map[string]any{
						"DynamicPlaces": "NO",
					}
				}

				biddingStrategyUpdate["Search"] = search
			}

			// Network block: если передано хотя бы одно network-поле — отправляем Network целиком.
			// Yandex API требует полного BiddingStrategyType + nested settings. Для чистого
			// изменения BidCeiling — обязательно продублировать network_strategy.
			if networkStrategy != "" || networkWeeklyRub > 0 || networkAvgCPA > 0 || networkBidCeilingMicros > 0 {
				netSettings := map[string]any{}
				if networkStrategy != "" {
					netSettings["BiddingStrategyType"] = networkStrategy
				}
				netWeeklyMicros := networkWeeklyRub * 1000000

				switch networkStrategy {
				case "WB_MAXIMUM_CLICKS":
					wbClicks := map[string]any{}
					if netWeeklyMicros > 0 {
						wbClicks["WeeklySpendLimit"] = netWeeklyMicros
					}
					if networkBidCeilingMicros > 0 {
						wbClicks["BidCeiling"] = networkBidCeilingMicros
					}
					netSettings["WbMaximumClicks"] = wbClicks
				case "WB_MAXIMUM_CONVERSION_RATE":
					wbConv := map[string]any{}
					if netWeeklyMicros > 0 {
						wbConv["WeeklySpendLimit"] = netWeeklyMicros
					}
					if goalID > 0 {
						wbConv["GoalId"] = goalID
					}
					if networkBidCeilingMicros > 0 {
						wbConv["BidCeiling"] = networkBidCeilingMicros
					}
					netSettings["WbMaximumConversionRate"] = wbConv
				case "AVERAGE_CPA":
					avgSet := map[string]any{}
					if netWeeklyMicros > 0 {
						avgSet["WeeklySpendLimit"] = netWeeklyMicros
					}
					if networkAvgCPA > 0 {
						avgSet["AverageCpa"] = networkAvgCPA * 1000000
					} else if avgCPA > 0 {
						avgSet["AverageCpa"] = avgCPA * 1000000
					}
					if goalID > 0 {
						avgSet["GoalId"] = goalID
					}
					if networkBidCeilingMicros > 0 {
						avgSet["BidCeiling"] = networkBidCeilingMicros
					}
					netSettings["AverageCpa"] = avgSet
				}
				biddingStrategyUpdate["Network"] = netSettings
			}

			if len(biddingStrategyUpdate) > 0 {
				textCampaign["BiddingStrategy"] = biddingStrategyUpdate
			}

			if len(priorityGoals) > 0 {
				textCampaign["PriorityGoals"] = map[string]any{
					"Items": priorityGoals,
				}
			}

			// Campaign-level UTM (TrackingParams) — наследуется всеми группами.
			// Для миграции с уровня группы: ставим на кампанию, потом чистим у групп через update_adgroup(tracking_params="").
			if trackingParams != "" {
				textCampaign["TrackingParams"] = trackingParams
			}

			campaign["TextCampaign"] = textCampaign
		}

		params := map[string]any{
			"Campaigns": []any{campaign},
		}

		raw, err := client.Call(ctx, token, "campaigns", "update", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		return common.TextResult(string(result)), nil
	})
}

func registerManageCampaigns(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_campaigns",
		mcp.WithDescription("Массовое управление кампаниями: остановка, возобновление, архивация."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: suspend, resume, archive, unarchive, delete"), mcp.Required()),
	)

	s.AddTool(tool, campaignActionHandler(client, resolver, ""))
}

func registerSuspendCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("suspend_campaign",
		mcp.WithDescription("Остановить кампанию. Только для кампаний со статусом ON."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
	)

	s.AddTool(tool, campaignActionHandler(client, resolver, "suspend"))
}

func registerResumeCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("resume_campaign",
		mcp.WithDescription("Возобновить показы кампании. Только для кампаний со статусом SUSPENDED."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
	)

	s.AddTool(tool, campaignActionHandler(client, resolver, "resume"))
}

func registerArchiveCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("archive_campaign",
		mcp.WithDescription("Архивировать кампанию. Сначала нужно остановить (suspend), потом архивировать."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
	)

	s.AddTool(tool, campaignActionHandler(client, resolver, "archive"))
}

func campaignActionHandler(client *Client, resolver *auth.AccountResolver, fixedAction string) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		action := fixedAction
		if action == "" {
			action = common.GetString(req, "action")
		}

		var ids []int64
		if campaignID := common.GetInt(req, "campaign_id"); campaignID > 0 {
			ids = []int64{int64(campaignID)}
		} else if idStrs := common.GetStringSlice(req, "campaign_ids"); len(idStrs) > 0 {
			for _, s := range idStrs {
				n, _ := strconv.ParseInt(s, 10, 64)
				if n > 0 {
					ids = append(ids, n)
				}
			}
		}

		if len(ids) == 0 {
			return common.ErrorResult("campaign_id or campaign_ids required"), nil
		}

		params := map[string]any{
			"SelectionCriteria": map[string]any{
				"Ids": ids,
			},
		}

		raw, err := client.Call(ctx, token, "campaigns", action, params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("Действие %s выполнено, но ответ не содержит result: %v", action, err)), nil
		}

		return common.TextResult(string(result)), nil
	}
}

// bidCeilingMicros converts a BidCeiling value from rubles to Yandex API micros
// and enforces the 0.3 RUB (300000 micros) minimum documented for auto-bid ceilings.
// Returns 0 if the caller passed a non-positive value — meaning "no ceiling".
func bidCeilingMicros(rub int) int {
	if rub <= 0 {
		return 0
	}
	micros := rub * 1000000
	if micros < 300000 {
		micros = 300000
	}
	return micros
}
