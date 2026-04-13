package direct

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterStatsTools registers statistics and search query MCP tools.
func RegisterStatsTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetCampaignStats(s, client, resolver)
	registerGetAdGroupStats(s, client, resolver)
	registerGetAdStats(s, client, resolver)
	registerGetCriteriaStats(s, client, resolver)
	registerGetSearchQueries(s, client, resolver)
}

func registerGetCampaignStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_campaign_stats",
		mcp.WithDescription("Статистика кампаний за период. Без campaign_id/campaign_ids — по ВСЕМ кампаниям аккаунта. Для конверсий укажи goal_ids и добавь Conversions в field_names."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("campaign_id", mcp.Description("ID одной кампании (опционально)")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую (опционально). Без фильтра — все кампании.")),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую. Получи из metrika_get_goals. Обязательно для Conversions/CostPerConversion.")),
		mcp.WithString("attribution", mcp.Description("Модель атрибуции: LYDC (по умолчанию). Применяется только при наличии goal_ids.")),
		mcp.WithString("field_names", mcp.Description("Поля через запятую. По умолчанию: CampaignName,Impressions,Clicks,Cost. Для конверсий: Conversions,CostPerConversion (+ укажи goal_ids)")),
		mcp.WithNumber("limit", mcp.Description("Макс. строк в ответе (по умолчанию 200)")),
	)

	s.AddTool(tool, statsHandler(client, resolver, "CampaignId"))
}

func registerGetAdGroupStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_adgroup_stats",
		mcp.WithDescription("Статистика групп объявлений за период. Для конверсий укажи goal_ids."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую. Обязательно для Conversions/CostPerConversion.")),
		mcp.WithString("attribution", mcp.Description("Модель атрибуции (применяется только при наличии goal_ids)")),
		mcp.WithString("field_names", mcp.Description("Поля через запятую (по умолчанию: Impressions,Clicks,Cost,Ctr,AvgCpc)")),
	)

	s.AddTool(tool, statsHandler(client, resolver, "AdGroupId"))
}

func registerGetAdStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_ad_stats",
		mcp.WithDescription("Статистика объявлений за период. Для A/B тестирования. Для конверсий укажи goal_ids."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую. Обязательно для Conversions/CostPerConversion.")),
		mcp.WithString("attribution", mcp.Description("Модель атрибуции (применяется только при наличии goal_ids)")),
		mcp.WithString("field_names", mcp.Description("Поля через запятую (по умолчанию: Impressions,Clicks,Cost,Ctr,AvgCpc)")),
	)

	s.AddTool(tool, statsHandler(client, resolver, "AdId"))
}

func registerGetCriteriaStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_criteria_stats",
		mcp.WithDescription("Статистика по ключевым фразам (критериям) за период. Для конверсий укажи goal_ids."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую. Обязательно для Conversions/CostPerConversion.")),
		mcp.WithString("attribution", mcp.Description("Модель атрибуции (применяется только при наличии goal_ids)")),
		mcp.WithString("field_names", mcp.Description("Поля через запятую (по умолчанию: Impressions,Clicks,Cost,Ctr,AvgCpc)")),
	)

	s.AddTool(tool, statsHandler(client, resolver, "CriteriaId"))
}

func statsHandler(client *Client, resolver *auth.AccountResolver, groupBy string) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignID := common.GetInt(req, "campaign_id")
		campaignIDs := common.GetStringSlice(req, "campaign_ids")
		dateFrom := common.GetString(req, "date_from")
		dateTo := common.GetString(req, "date_to")

		// Report type and default fields depend on groupBy
		reportType := "CAMPAIGN_PERFORMANCE_REPORT"
		defaultFields := []string{"CampaignName", "Impressions", "Clicks", "Cost"}
		switch groupBy {
		case "AdGroupId":
			reportType = "ADGROUP_PERFORMANCE_REPORT"
			defaultFields = []string{"AdGroupName", "Impressions", "Clicks", "Cost", "Ctr", "AvgCpc"}
		case "AdId":
			reportType = "AD_PERFORMANCE_REPORT"
			defaultFields = []string{"AdId", "Impressions", "Clicks", "Cost", "Ctr", "AvgCpc"}
		case "CriteriaId":
			reportType = "CRITERIA_PERFORMANCE_REPORT"
			defaultFields = []string{"Criteria", "Impressions", "Clicks", "Cost", "Ctr", "AvgCpc"}
		}

		// Build field names
		var fieldNames []string
		if userFields := common.GetStringSlice(req, "field_names"); len(userFields) > 0 {
			fieldNames = userFields
		} else {
			fieldNames = defaultFields
		}

		// Build selection criteria
		criteria := map[string]any{
			"DateFrom": dateFrom,
			"DateTo":   dateTo,
		}

		// Add campaign filter if specified
		if campaignID > 0 {
			criteria["Filter"] = []any{
				map[string]any{
					"Field":    "CampaignId",
					"Operator": "EQUALS",
					"Values":   []string{intToStr(campaignID)},
				},
			}
		} else if len(campaignIDs) > 0 {
			criteria["Filter"] = []any{
				map[string]any{
					"Field":    "CampaignId",
					"Operator": "IN",
					"Values":   campaignIDs,
				},
			}
		}
		// No filter = all campaigns

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":    fieldNames,
			"ReportName":    fmt.Sprintf("stats_%d", time.Now().UnixNano()),
			"ReportType":    reportType,
			"DateRangeType": "CUSTOM_DATE",
			"Format":        "TSV",
			"IncludeVAT":    "YES",
			"IncludeDiscount": "NO",
		}

		// Add Goals + AttributionModels when goal_ids is specified
		goalIDs := common.GetStringSlice(req, "goal_ids")
		if len(goalIDs) > 0 {
			params["Goals"] = goalIDs
			attribution := common.GetString(req, "attribution")
			if attribution == "" {
				attribution = "LYDC"
			}
			params["AttributionModels"] = []string{attribution}
		}

		tsv, err := client.CallReport(ctx, token, params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		// Limit rows for large reports
		limit := common.GetInt(req, "limit")
		if limit <= 0 {
			limit = 200
		}
		tsv = truncateTSV(tsv, limit)

		return common.TextResult(tsv), nil
	}
}

func registerGetSearchQueries(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_search_queries",
		mcp.WithDescription("Получить реальные поисковые запросы пользователей. По умолчанию топ-100 по кликам. Для конверсий укажи goal_ids."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую. Обязательно для Conversions/CostPerConversion.")),
		mcp.WithString("attribution", mcp.Description("Модель атрибуции: LYDC (по умолчанию). Применяется только при наличии goal_ids.")),
		mcp.WithString("field_names", mcp.Description("Поля через запятую. По умолчанию: Query,Impressions,Clicks,Cost")),
		mcp.WithNumber("limit", mcp.Description("Макс. строк в ответе (по умолчанию 100)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignIDs := common.GetStringSlice(req, "campaign_ids")
		dateFrom := common.GetString(req, "date_from")
		dateTo := common.GetString(req, "date_to")

		// Build field names
		var fieldNames []string
		if userFields := common.GetStringSlice(req, "field_names"); len(userFields) > 0 {
			fieldNames = userFields
		} else {
			fieldNames = []string{"Query", "Impressions", "Clicks", "Cost"}
		}

		params := map[string]any{
			"SelectionCriteria": map[string]any{
				"Filter": []any{
					map[string]any{
						"Field":    "CampaignId",
						"Operator": "IN",
						"Values":   campaignIDs,
					},
				},
				"DateFrom": dateFrom,
				"DateTo":   dateTo,
			},
			"FieldNames":    fieldNames,
			"ReportName":    fmt.Sprintf("sq_%d", time.Now().UnixNano()),
			"ReportType":    "SEARCH_QUERY_PERFORMANCE_REPORT",
			"DateRangeType": "CUSTOM_DATE",
			"Format":        "TSV",
			"IncludeVAT":    "YES",
			"IncludeDiscount": "NO",
		}

		// Add Goals + AttributionModels when goal_ids is specified
		goalIDs := common.GetStringSlice(req, "goal_ids")
		if len(goalIDs) > 0 {
			params["Goals"] = goalIDs
			attribution := common.GetString(req, "attribution")
			if attribution == "" {
				attribution = "LYDC"
			}
			params["AttributionModels"] = []string{attribution}
		}

		tsv, err := client.CallReport(ctx, token, params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		// Limit rows
		limit := common.GetInt(req, "limit")
		if limit <= 0 {
			limit = 100
		}
		tsv = truncateTSV(tsv, limit)

		return common.TextResult(tsv), nil
	})
}

// truncateTSV limits TSV output to maxRows data rows (header always kept).
// Appends a summary line if truncated.
func truncateTSV(tsv string, maxRows int) string {
	if maxRows <= 0 {
		return tsv
	}
	lines := strings.Split(strings.TrimRight(tsv, "\n"), "\n")
	if len(lines) <= maxRows+1 { // header + data
		return tsv
	}
	result := make([]string, 0, maxRows+2)
	result = append(result, lines[0]) // header
	result = append(result, lines[1:maxRows+1]...)
	result = append(result, fmt.Sprintf("... (%d строк всего, показано %d)", len(lines)-1, maxRows))
	return strings.Join(result, "\n")
}

func intToStr(n int) string {
	return fmt.Sprintf("%d", n)
}

func hasField(fields []string, target string) bool {
	target = strings.ToLower(target)
	for _, f := range fields {
		if strings.ToLower(strings.TrimSpace(f)) == target {
			return true
		}
	}
	return false
}
