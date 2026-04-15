package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально, по умолчанию — default)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("states", mcp.Description("Фильтр по статусам через запятую: ON, SUSPENDED, OFF, ENDED, CONVERTED, ARCHIVED")),
		mcp.WithString("field_names", mcp.Description("Поля через запятую: Id, Name, State, Status, DailyBudget, Statistics, и т.д. По умолчанию — все")),
		mcp.WithNumber("limit", mcp.Description("Максимум кампаний (по умолчанию 100)")),
		mcp.WithString("campaign_ids", mcp.Description("Фильтр по ID кампаний через запятую")),
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
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("name", mcp.Description("Название кампании"), mcp.Required()),
		mcp.WithNumber("daily_budget_amount", mcp.Description("Недельный бюджет в рублях (число, НЕ микроюниты)"), mcp.Required()),
		mcp.WithString("daily_budget_mode", mcp.Description("Режим бюджета: STANDARD или DISTRIBUTED (по умолчанию DISTRIBUTED)")),
		mcp.WithString("search_strategy", mcp.Description("Стратегия поиска: WB_MAXIMUM_CONVERSION_RATE, AVERAGE_CPA, WB_MAXIMUM_CLICKS и др."), mcp.Required()),
		mcp.WithString("network_strategy", mcp.Description("Стратегия сетей: SERVING_OFF (по умолчанию), NETWORK_DEFAULT, WB_MAXIMUM_CLICKS")),
		mcp.WithNumber("goal_id", mcp.Description("ID цели Метрики для оптимизации (одна цель). Если нужно несколько — используй priority_goals, а сюда 0 или не передавай.")),
		mcp.WithString("priority_goals", mcp.Description("Приоритетные цели JSON: [{\"goal_id\":123,\"value\":0},{\"goal_id\":456,\"value\":0}]. value — ценность конверсии в рублях (0 = не задана). При передаче GoalId в стратегии автоматически ставится 13.")),
		mcp.WithString("counter_ids", mcp.Description("ID счётчиков Метрики через запятую")),
		mcp.WithNumber("average_cpa", mcp.Description("Целевая цена конверсии (для стратегии AVERAGE_CPA)")),
		mcp.WithString("start_date", mcp.Description("Дата начала (YYYY-MM-DD)")),
		mcp.WithString("negative_keywords", mcp.Description("Минус-фразы через запятую")),
		mcp.WithString("settings", mcp.Description("Настройки JSON: [{\"option\":\"ENABLE_AREA_OF_INTEREST_TARGETING\",\"value\":false}]")),
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

		// Build campaign object
		campaign := map[string]any{
			"Name":      name,
			"StartDate": common.GetString(req, "start_date"),
		}

		// DailyBudget only for manual strategies (not WB_* / AVERAGE_*)
		isAutoStrategy := strings.HasPrefix(searchStrategy, "WB_") || strings.HasPrefix(searchStrategy, "AVERAGE_")
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

		switch searchStrategy {
		case "WB_MAXIMUM_CLICKS":
			searchSettings["WbMaximumClicks"] = map[string]any{
				"WeeklySpendLimit": weeklyBudgetMicros,
			}
		case "WB_MAXIMUM_CONVERSION_RATE":
			wbSettings := map[string]any{
				"WeeklySpendLimit": weeklyBudgetMicros,
			}
			if goalID > 0 {
				wbSettings["GoalId"] = goalID
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
			searchSettings["AverageCpa"] = avgSettings
		}

		biddingStrategy := map[string]any{
			"Search": searchSettings,
			"Network": map[string]any{
				"BiddingStrategyType": networkStrategy,
			},
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
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название")),
		mcp.WithNumber("daily_budget_amount", mcp.Description("Новый недельный бюджет в рублях")),
		mcp.WithString("search_strategy", mcp.Description("Новая стратегия поиска")),
		mcp.WithNumber("goal_id", mcp.Description("Новая цель Метрики (одна). Для нескольких — используй priority_goals + goal_id=13.")),
		mcp.WithString("priority_goals", mcp.Description("Приоритетные цели JSON: [{\"goal_id\":123,\"value\":0},{\"goal_id\":456,\"value\":0}]. value — ценность в рублях (0 = минимум 1₽). При передаче GoalId автоматически ставится 13.")),
		mcp.WithNumber("average_cpa", mcp.Description("Целевая CPA (для стратегии AVERAGE_CPA)")),
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

		needTextCampaign := searchStrategy != "" || goalID > 0 || avgCPA > 0 || len(priorityGoals) > 0
		if needTextCampaign {
			textCampaign := map[string]any{}

			if searchStrategy != "" || goalID > 0 || avgCPA > 0 {
				search := map[string]any{}
				if searchStrategy != "" {
					search["BiddingStrategyType"] = searchStrategy
				}

				// GoalId and other params go inside the strategy-specific key
				switch searchStrategy {
				case "WB_MAXIMUM_CONVERSION_RATE":
					stratSettings := map[string]any{}
					if goalID > 0 {
						stratSettings["GoalId"] = goalID
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
					search["AverageCpa"] = stratSettings
				case "WB_MAXIMUM_CLICKS":
					search["WbMaximumClicks"] = map[string]any{}
				default:
					// If no strategy specified but goalID changed, we still need the strategy type
					// to know where to put GoalId — caller must provide search_strategy
					if goalID > 0 {
						search["GoalId"] = goalID
					}
				}

				textCampaign["BiddingStrategy"] = map[string]any{
					"Search": search,
				}
			}

			if len(priorityGoals) > 0 {
				textCampaign["PriorityGoals"] = map[string]any{
					"Items": priorityGoals,
				}
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
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: suspend, resume, archive, unarchive, delete"), mcp.Required()),
	)

	s.AddTool(tool, campaignActionHandler(client, resolver, ""))
}

func registerSuspendCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("suspend_campaign",
		mcp.WithDescription("Остановить кампанию. Только для кампаний со статусом ON."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
	)

	s.AddTool(tool, campaignActionHandler(client, resolver, "suspend"))
}

func registerResumeCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("resume_campaign",
		mcp.WithDescription("Возобновить показы кампании. Только для кампаний со статусом SUSPENDED."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
	)

	s.AddTool(tool, campaignActionHandler(client, resolver, "resume"))
}

func registerArchiveCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("archive_campaign",
		mcp.WithDescription("Архивировать кампанию. Сначала нужно остановить (suspend), потом архивировать."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
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
