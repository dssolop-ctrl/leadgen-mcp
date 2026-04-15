package direct

import (
	"context"
	"encoding/json"
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
	registerGetAccountStats(s, client, resolver)
	registerGetCustomReport(s, client, resolver)
	registerGetReachFrequencyStats(s, client, resolver)
}

func registerGetCampaignStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_campaign_stats",
		mcp.WithDescription("Статистика кампаний за период. Без campaign_id/campaign_ids — по ВСЕМ кампаниям аккаунта. Для конверсий укажи goal_ids и добавь Conversions в field_names."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID одной кампании (опционально)")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую (опционально). Без фильтра — все кампании.")),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую. Получи из metrika_get_goals. Обязательно для Conversions/CostPerConversion.")),
		mcp.WithString("attribution", mcp.Description("Атрибуция: LYDC (умолч, при наличии goal_ids)")),
		mcp.WithString("field_names", mcp.Description("Поля через запятую (умолч: CampaignName,Impressions,Clicks,Cost)")),
		mcp.WithNumber("limit", mcp.Description("Макс. строк (умолч 200)")),
	)

	s.AddTool(tool, statsHandler(client, resolver, "CampaignId"))
}

func registerGetAdGroupStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_adgroup_stats",
		mcp.WithDescription("Статистика групп объявлений за период. Для конверсий укажи goal_ids."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую (нужен для Conversions)")),
		mcp.WithString("attribution", mcp.Description("Атрибуция (при наличии goal_ids)")),
		mcp.WithString("field_names", mcp.Description("Поля (умолч: Impressions,Clicks,Cost,Ctr,AvgCpc)")),
	)

	s.AddTool(tool, statsHandler(client, resolver, "AdGroupId"))
}

func registerGetAdStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_ad_stats",
		mcp.WithDescription("Статистика объявлений за период. Для A/B тестирования. Для конверсий укажи goal_ids."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую (нужен для Conversions)")),
		mcp.WithString("attribution", mcp.Description("Атрибуция (при наличии goal_ids)")),
		mcp.WithString("field_names", mcp.Description("Поля (умолч: Impressions,Clicks,Cost,Ctr,AvgCpc)")),
	)

	s.AddTool(tool, statsHandler(client, resolver, "AdId"))
}

func registerGetCriteriaStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_criteria_stats",
		mcp.WithDescription("Статистика по ключевым фразам (критериям) за период. Для конверсий укажи goal_ids."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую (нужен для Conversions)")),
		mcp.WithString("attribution", mcp.Description("Атрибуция (при наличии goal_ids)")),
		mcp.WithString("field_names", mcp.Description("Поля (умолч: Impressions,Clicks,Cost,Ctr,AvgCpc)")),
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
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую (нужен для Conversions)")),
		mcp.WithString("attribution", mcp.Description("Атрибуция: LYDC (умолч, при наличии goal_ids)")),
		mcp.WithString("field_names", mcp.Description("Поля (умолч: Query,Impressions,Clicks,Cost)")),
		mcp.WithNumber("limit", mcp.Description("Макс. строк (умолч 100)")),
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

func registerGetAccountStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_account_stats",
		mcp.WithDescription("Статистика по всему аккаунту за период (без разбивки по кампаниям). Суммарные показы, клики, расход."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("date_from", mcp.Description("Начало периода YYYY-MM-DD"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода YYYY-MM-DD"), mcp.Required()),
		mcp.WithString("field_names", mcp.Description("Поля (умолч: Impressions,Clicks,Cost,Ctr,AvgCpc)")),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую (опционально)")),
		mcp.WithString("attribution", mcp.Description("Модель атрибуции (по умолчанию LYDC)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var fieldNames []string
		if userFields := common.GetStringSlice(req, "field_names"); len(userFields) > 0 {
			fieldNames = userFields
		} else {
			fieldNames = []string{"Impressions", "Clicks", "Cost", "Ctr", "AvgCpc"}
		}

		params := map[string]any{
			"SelectionCriteria": map[string]any{
				"DateFrom": common.GetString(req, "date_from"),
				"DateTo":   common.GetString(req, "date_to"),
			},
			"FieldNames":    fieldNames,
			"ReportName":    fmt.Sprintf("account_%d", time.Now().UnixNano()),
			"ReportType":    "ACCOUNT_PERFORMANCE_REPORT",
			"DateRangeType": "CUSTOM_DATE",
			"Format":        "TSV",
			"IncludeVAT":    "YES",
			"IncludeDiscount": "NO",
		}

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
		return common.TextResult(tsv), nil
	})
}

func registerGetCustomReport(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_custom_report",
		mcp.WithDescription("Произвольный отчёт Reports API. Позволяет запросить любой тип отчёта с любыми полями."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("report_type", mcp.Description("Тип: CAMPAIGN_PERFORMANCE_REPORT, ADGROUP_PERFORMANCE_REPORT, AD_PERFORMANCE_REPORT, CRITERIA_PERFORMANCE_REPORT, SEARCH_QUERY_PERFORMANCE_REPORT, REACH_AND_FREQUENCY_PERFORMANCE_REPORT и др."), mcp.Required()),
		mcp.WithString("field_names", mcp.Description("Поля через запятую"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода YYYY-MM-DD"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода YYYY-MM-DD"), mcp.Required()),
		mcp.WithString("filter_json", mcp.Description("JSON фильтров: [{\"Field\":\"CampaignId\",\"Operator\":\"IN\",\"Values\":[\"123\"]}]")),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую")),
		mcp.WithString("attribution", mcp.Description("Модель атрибуции")),
		mcp.WithNumber("limit", mcp.Description("Макс. строк (по умолчанию 200)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{
			"DateFrom": common.GetString(req, "date_from"),
			"DateTo":   common.GetString(req, "date_to"),
		}
		if filterJSON := common.GetString(req, "filter_json"); filterJSON != "" {
			var filters any
			if err := json.Unmarshal([]byte(filterJSON), &filters); err != nil {
				return common.ErrorResult("invalid filter_json: " + err.Error()), nil
			}
			criteria["Filter"] = filters
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":    common.GetStringSlice(req, "field_names"),
			"ReportName":    fmt.Sprintf("custom_%d", time.Now().UnixNano()),
			"ReportType":    common.GetString(req, "report_type"),
			"DateRangeType": "CUSTOM_DATE",
			"Format":        "TSV",
			"IncludeVAT":    "YES",
			"IncludeDiscount": "NO",
		}

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
		limit := common.GetInt(req, "limit")
		if limit <= 0 {
			limit = 200
		}
		tsv = truncateTSV(tsv, limit)
		return common.TextResult(tsv), nil
	})
}

func registerGetReachFrequencyStats(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_reach_frequency_stats",
		mcp.WithDescription("Отчёт по охвату и частоте показов (REACH_AND_FREQUENCY_PERFORMANCE_REPORT)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода YYYY-MM-DD"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода YYYY-MM-DD"), mcp.Required()),
		mcp.WithString("field_names", mcp.Description("Поля через запятую (по умолчанию: Impressions,ImpressionReach,AvgImpressionFrequency)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var fieldNames []string
		if userFields := common.GetStringSlice(req, "field_names"); len(userFields) > 0 {
			fieldNames = userFields
		} else {
			fieldNames = []string{"CampaignId", "CampaignName", "Impressions", "ImpressionReach", "AvgImpressionFrequency"}
		}

		campaignIDs := common.GetStringSlice(req, "campaign_ids")

		params := map[string]any{
			"SelectionCriteria": map[string]any{
				"DateFrom": common.GetString(req, "date_from"),
				"DateTo":   common.GetString(req, "date_to"),
				"Filter": []any{
					map[string]any{
						"Field":    "CampaignId",
						"Operator": "IN",
						"Values":   campaignIDs,
					},
				},
			},
			"FieldNames":    fieldNames,
			"ReportName":    fmt.Sprintf("reach_%d", time.Now().UnixNano()),
			"ReportType":    "REACH_AND_FREQUENCY_PERFORMANCE_REPORT",
			"DateRangeType": "CUSTOM_DATE",
			"Format":        "TSV",
			"IncludeVAT":    "YES",
			"IncludeDiscount": "NO",
		}

		tsv, err := client.CallReport(ctx, token, params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(tsv), nil
	})
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
