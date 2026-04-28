package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterSummarizeTools registers compact-summary tools that return aggregated
// top-N data instead of raw TSV firehose. They cut 70-90% of context tokens
// versus the raw `get_search_queries` / `get_ad_stats` equivalents and pre-bucket
// the rows into the views the optimizer skill actually needs (top, waste, totals).
//
// Use these in pre-apply / optimization flows where the LLM otherwise has to
// digest hundreds of rows. The raw `get_*` tools are still available for cases
// when full data is genuinely needed.
func RegisterSummarizeTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerSummarizeSearchQueries(s, client, resolver)
	registerSummarizeAdsPerformance(s, client, resolver)
	registerSummarizeCampaignSnapshot(s, client, resolver)
}

// rowSummary is a single aggregated row for queries/ads/keywords.
// Fields are emitted in JSON only when meaningful (omitempty) to keep the
// payload compact.
type rowSummary struct {
	Query       string  `json:"query,omitempty"`
	AdID        int64   `json:"ad_id,omitempty"`
	Criteria    string  `json:"criteria,omitempty"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	Cost        float64 `json:"cost_rub"`
	Conversions int     `json:"conversions,omitempty"`
	CTR         float64 `json:"ctr,omitempty"`
	CPC         float64 `json:"cpc_rub,omitempty"`
	CPA         float64 `json:"cpa_rub,omitempty"`
}

// totalsSummary aggregates across all rows of a report.
type totalsSummary struct {
	Rows        int     `json:"unique_rows"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	Cost        float64 `json:"cost_rub"`
	Conversions int     `json:"conversions,omitempty"`
	CTR         float64 `json:"ctr,omitempty"`
	CPC         float64 `json:"cpc_rub,omitempty"`
	CPA         float64 `json:"cpa_rub,omitempty"`
}

// ---------- summarize_search_queries ----------

func registerSummarizeSearchQueries(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("summarize_search_queries",
		mcp.WithDescription("Компактная сводка по поисковым запросам кампании за период: топ по кликам, топ по конверсиям, waste-список (клики без конверсий), агрегаты. Экономит 70-90% контекста vs `get_search_queries` (сырой TSV из 100+ строк). Используй в шагах оптимизации (O3.x) и аудита (A4)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую — для блока top_by_conversions и waste-фильтра")),
		mcp.WithString("attribution", mcp.Description("Атрибуция (умолч LYDC, применяется при наличии goal_ids)")),
		mcp.WithNumber("top_n", mcp.Description("Сколько строк в каждом топе (умолч 15)")),
		mcp.WithNumber("waste_min_clicks", mcp.Description("Минимум кликов для попадания в waste (умолч 5)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignIDs := common.GetStringSlice(req, "campaign_ids")
		if len(campaignIDs) == 0 {
			return common.ErrorResult("campaign_ids обязателен"), nil
		}
		dateFrom := common.GetString(req, "date_from")
		dateTo := common.GetString(req, "date_to")
		topN := common.GetInt(req, "top_n")
		if topN <= 0 {
			topN = 15
		}
		wasteMinClicks := common.GetInt(req, "waste_min_clicks")
		if wasteMinClicks <= 0 {
			wasteMinClicks = 5
		}

		goalIDs := common.GetStringSlice(req, "goal_ids")
		hasGoals := len(goalIDs) > 0

		fieldNames := []string{"Query", "Impressions", "Clicks", "Cost"}
		if hasGoals {
			fieldNames = append(fieldNames, "Conversions")
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
			"FieldNames":      fieldNames,
			"ReportName":      fmt.Sprintf("sumq_%d", time.Now().UnixNano()),
			"ReportType":      "SEARCH_QUERY_PERFORMANCE_REPORT",
			"DateRangeType":   "CUSTOM_DATE",
			"Format":          "TSV",
			"IncludeVAT":      "YES",
			"IncludeDiscount": "NO",
		}

		attribution := common.GetString(req, "attribution")
		if attribution == "" {
			attribution = "LYDC"
		}
		if hasGoals {
			params["Goals"] = goalIDs
			params["AttributionModels"] = []string{attribution}
		}

		tsv, err := client.CallReport(ctx, token, params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		rows := parseReportTSV(tsv, fieldNames)

		totals := aggregateTotals(rows)

		// Top by clicks (always).
		byClicks := append([]rowSummary(nil), rows...)
		sort.Slice(byClicks, func(i, j int) bool { return byClicks[i].Clicks > byClicks[j].Clicks })
		topClicks := head(byClicks, topN)

		// Top by cost — useful even without goals to spot expensive queries.
		byCost := append([]rowSummary(nil), rows...)
		sort.Slice(byCost, func(i, j int) bool { return byCost[i].Cost > byCost[j].Cost })
		topCost := head(byCost, topN)

		out := map[string]any{
			"period":       fmt.Sprintf("%s — %s", dateFrom, dateTo),
			"campaign_ids": campaignIDs,
			"totals":       totals,
			"top_by_clicks": topClicks,
			"top_by_cost":   topCost,
			"note":          fmt.Sprintf("Top-%d по бакету. Полный TSV — get_search_queries.", topN),
		}

		if hasGoals {
			out["attribution"] = attribution

			// Top by conversions (drop tail rows with 0 conversions).
			byConv := append([]rowSummary(nil), rows...)
			sort.Slice(byConv, func(i, j int) bool { return byConv[i].Conversions > byConv[j].Conversions })
			topConv := head(byConv, topN)
			cut := 0
			for cut < len(topConv) && topConv[cut].Conversions > 0 {
				cut++
			}
			out["top_by_conversions"] = topConv[:cut]

			// Waste: clicks ≥ wasteMinClicks AND zero conversions, sorted by cost.
			waste := make([]rowSummary, 0)
			for _, r := range rows {
				if r.Conversions == 0 && r.Clicks >= wasteMinClicks {
					waste = append(waste, r)
				}
			}
			sort.Slice(waste, func(i, j int) bool { return waste[i].Cost > waste[j].Cost })
			out["waste"] = head(waste, topN)
		}

		return common.JSONResult(out), nil
	})
}

// ---------- summarize_ads_performance ----------

func registerSummarizeAdsPerformance(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("summarize_ads_performance",
		mcp.WithDescription("Компактная сводка по объявлениям кампании/группы: топ по CTR, топ по конверсиям, кандидаты на A/B (низкий CTR при достаточных показах), агрегаты. Экономит 70-85% контекста vs `get_ad_stats`. Используй в R/C-шагах ревизии креативов и в OR.2 (A/B картинок РСЯ)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("goal_ids", mcp.Description("ID целей через запятую — для top_by_conversions")),
		mcp.WithString("attribution", mcp.Description("Атрибуция (умолч LYDC)")),
		mcp.WithNumber("top_n", mcp.Description("Сколько строк в каждом топе (умолч 10)")),
		mcp.WithNumber("low_ctr_min_impressions", mcp.Description("Мин. показов чтобы попасть в low_ctr-кандидаты A/B (умолч 200)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignID := common.GetInt(req, "campaign_id")
		if campaignID <= 0 {
			return common.ErrorResult("campaign_id обязателен"), nil
		}
		dateFrom := common.GetString(req, "date_from")
		dateTo := common.GetString(req, "date_to")
		topN := common.GetInt(req, "top_n")
		if topN <= 0 {
			topN = 10
		}
		minImps := common.GetInt(req, "low_ctr_min_impressions")
		if minImps <= 0 {
			minImps = 200
		}

		goalIDs := common.GetStringSlice(req, "goal_ids")
		hasGoals := len(goalIDs) > 0

		fieldNames := []string{"AdId", "Impressions", "Clicks", "Cost"}
		if hasGoals {
			fieldNames = append(fieldNames, "Conversions")
		}

		params := map[string]any{
			"SelectionCriteria": map[string]any{
				"Filter": []any{
					map[string]any{
						"Field":    "CampaignId",
						"Operator": "EQUALS",
						"Values":   []string{strconv.Itoa(campaignID)},
					},
				},
				"DateFrom": dateFrom,
				"DateTo":   dateTo,
			},
			"FieldNames":      fieldNames,
			"ReportName":      fmt.Sprintf("sumads_%d", time.Now().UnixNano()),
			"ReportType":      "AD_PERFORMANCE_REPORT",
			"DateRangeType":   "CUSTOM_DATE",
			"Format":          "TSV",
			"IncludeVAT":      "YES",
			"IncludeDiscount": "NO",
		}

		attribution := common.GetString(req, "attribution")
		if attribution == "" {
			attribution = "LYDC"
		}
		if hasGoals {
			params["Goals"] = goalIDs
			params["AttributionModels"] = []string{attribution}
		}

		tsv, err := client.CallReport(ctx, token, params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		rows := parseReportTSV(tsv, fieldNames)
		totals := aggregateTotals(rows)

		// Top by CTR — only ads with ≥ minImps to avoid noise.
		filtered := make([]rowSummary, 0, len(rows))
		for _, r := range rows {
			if r.Impressions >= minImps {
				filtered = append(filtered, r)
			}
		}
		byCTR := append([]rowSummary(nil), filtered...)
		sort.Slice(byCTR, func(i, j int) bool { return byCTR[i].CTR > byCTR[j].CTR })
		topCTR := head(byCTR, topN)

		// Bottom by CTR (low CTR + enough impressions = A/B candidate).
		bottomCTR := append([]rowSummary(nil), filtered...)
		sort.Slice(bottomCTR, func(i, j int) bool { return bottomCTR[i].CTR < bottomCTR[j].CTR })
		lowCTR := head(bottomCTR, topN)

		out := map[string]any{
			"period":      fmt.Sprintf("%s — %s", dateFrom, dateTo),
			"campaign_id": campaignID,
			"totals":      totals,
			"top_by_ctr":  topCTR,
			"low_ctr_candidates_for_ab": lowCTR,
			"low_ctr_min_impressions":   minImps,
			"note": fmt.Sprintf("Top-%d с фильтром Impressions ≥ %d. Полный отчёт — get_ad_stats.", topN, minImps),
		}

		if hasGoals {
			out["attribution"] = attribution

			byConv := append([]rowSummary(nil), rows...)
			sort.Slice(byConv, func(i, j int) bool { return byConv[i].Conversions > byConv[j].Conversions })
			topConv := head(byConv, topN)
			cut := 0
			for cut < len(topConv) && topConv[cut].Conversions > 0 {
				cut++
			}
			out["top_by_conversions"] = topConv[:cut]
		}

		return common.JSONResult(out), nil
	})
}

// ---------- summarize_campaign_snapshot ----------

func registerSummarizeCampaignSnapshot(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("summarize_campaign_snapshot",
		mcp.WithDescription("Один компактный JSON по кампании: базовые поля (Type/State/Status/стратегия/бюджет), счётчики групп/объявлений/ключевых, метрики за last_n_days. Один вызов вместо 4–5 (get_campaigns + get_adgroups + get_ads + get_keywords + get_campaign_stats). Используй в начале любой ветки оптимизации/анализа."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города"), mcp.Required()),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithNumber("last_n_days", mcp.Description("За сколько дней метрики (умолч 7)")),
		mcp.WithString("goal_ids", mcp.Description("ID целей для расчёта Conversions/CPA в snapshot")),
		mcp.WithString("attribution", mcp.Description("Атрибуция для метрик (умолч LYDC)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		if clientLogin == "" {
			return common.ErrorResult("client_login обязателен для summarize_campaign_snapshot"), nil
		}
		campaignID := common.GetInt(req, "campaign_id")
		if campaignID <= 0 {
			return common.ErrorResult("campaign_id обязателен"), nil
		}
		days := common.GetInt(req, "last_n_days")
		if days <= 0 {
			days = 7
		}

		// 1) Campaign basics — single get_campaigns call with the most useful fields.
		campParams := map[string]any{
			"SelectionCriteria": map[string]any{
				"Ids":    []int{campaignID},
				"States": []string{"ON", "OFF", "SUSPENDED", "ENDED", "DRAFT"},
			},
			"FieldNames": []string{
				"Id", "Name", "Type", "State", "Status",
				"DailyBudget", "StartDate", "EndDate",
			},
			"TextCampaignFieldNames":    []string{"BiddingStrategy", "TrackingParams"},
			"UnifiedCampaignFieldNames": []string{"BiddingStrategy", "TrackingParams"},
		}
		campRaw, err := client.Call(ctx, token, "campaigns", "get", campParams, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("get campaign: %v", err)), nil
		}
		var campResp struct {
			Result struct {
				Campaigns []map[string]any `json:"Campaigns"`
			} `json:"result"`
		}
		_ = json.Unmarshal(campRaw, &campResp)
		if len(campResp.Result.Campaigns) == 0 {
			return common.ErrorResult(fmt.Sprintf("campaign %d не найдена", campaignID)), nil
		}
		camp := campResp.Result.Campaigns[0]

		// 2) AdGroup count — light call.
		agParams := map[string]any{
			"SelectionCriteria": map[string]any{
				"CampaignIds": []int{campaignID},
			},
			"FieldNames": []string{"Id"},
		}
		agRaw, _ := client.Call(ctx, token, "adgroups", "get", agParams, clientLogin)
		var agResp struct {
			Result struct {
				AdGroups []map[string]any `json:"AdGroups"`
			} `json:"result"`
		}
		_ = json.Unmarshal(agRaw, &agResp)
		adGroupIDs := make([]int64, 0, len(agResp.Result.AdGroups))
		for _, g := range agResp.Result.AdGroups {
			if id, ok := g["Id"].(float64); ok {
				adGroupIDs = append(adGroupIDs, int64(id))
			}
		}

		// 3) Ad count.
		var adsCount int
		if len(adGroupIDs) > 0 {
			adsParams := map[string]any{
				"SelectionCriteria": map[string]any{
					"AdGroupIds": adGroupIDs,
				},
				"FieldNames": []string{"Id"},
			}
			adsRaw, _ := client.Call(ctx, token, "ads", "get", adsParams, clientLogin)
			var adsResp struct {
				Result struct {
					Ads []map[string]any `json:"Ads"`
				} `json:"result"`
			}
			_ = json.Unmarshal(adsRaw, &adsResp)
			adsCount = len(adsResp.Result.Ads)
		}

		// 4) Keyword count.
		var kwCount int
		if len(adGroupIDs) > 0 {
			kwParams := map[string]any{
				"SelectionCriteria": map[string]any{
					"AdGroupIds": adGroupIDs,
				},
				"FieldNames": []string{"Id"},
			}
			kwRaw, _ := client.Call(ctx, token, "keywords", "get", kwParams, clientLogin)
			var kwResp struct {
				Result struct {
					Keywords []map[string]any `json:"Keywords"`
				} `json:"result"`
			}
			_ = json.Unmarshal(kwRaw, &kwResp)
			kwCount = len(kwResp.Result.Keywords)
		}

		// 5) Last-N-day metrics — campaign-level report.
		dateTo := time.Now().Format("2006-01-02")
		dateFrom := time.Now().AddDate(0, 0, -days+1).Format("2006-01-02")

		goalIDs := common.GetStringSlice(req, "goal_ids")
		hasGoals := len(goalIDs) > 0

		statsFields := []string{"CampaignName", "Impressions", "Clicks", "Cost"}
		if hasGoals {
			statsFields = append(statsFields, "Conversions")
		}

		statsParams := map[string]any{
			"SelectionCriteria": map[string]any{
				"Filter": []any{
					map[string]any{
						"Field":    "CampaignId",
						"Operator": "EQUALS",
						"Values":   []string{strconv.Itoa(campaignID)},
					},
				},
				"DateFrom": dateFrom,
				"DateTo":   dateTo,
			},
			"FieldNames":      statsFields,
			"ReportName":      fmt.Sprintf("snap_%d", time.Now().UnixNano()),
			"ReportType":      "CAMPAIGN_PERFORMANCE_REPORT",
			"DateRangeType":   "CUSTOM_DATE",
			"Format":          "TSV",
			"IncludeVAT":      "YES",
			"IncludeDiscount": "NO",
		}

		attribution := common.GetString(req, "attribution")
		if attribution == "" {
			attribution = "LYDC"
		}
		if hasGoals {
			statsParams["Goals"] = goalIDs
			statsParams["AttributionModels"] = []string{attribution}
		}

		var metrics map[string]any
		if tsv, err := client.CallReport(ctx, token, statsParams, clientLogin); err == nil {
			rows := parseReportTSV(tsv, statsFields)
			t := aggregateTotals(rows)
			metrics = map[string]any{
				"period":      fmt.Sprintf("%s — %s (%d дн)", dateFrom, dateTo, days),
				"impressions": t.Impressions,
				"clicks":      t.Clicks,
				"cost_rub":    t.Cost,
				"ctr":         t.CTR,
				"cpc_rub":     t.CPC,
			}
			if hasGoals {
				metrics["conversions"] = t.Conversions
				metrics["cpa_rub"] = t.CPA
				metrics["attribution"] = attribution
			}
		}

		out := map[string]any{
			"id":                campaignID,
			"name":              camp["Name"],
			"type":              camp["Type"],
			"state":             camp["State"],
			"status":            camp["Status"],
			"start_date":        camp["StartDate"],
			"groups_count":      len(adGroupIDs),
			"ads_count":         adsCount,
			"keywords_count":    kwCount,
			"daily_budget":      camp["DailyBudget"],
		}

		// Strategy is nested under TextCampaign / UnifiedCampaign — pull out the
		// most useful piece.
		if tc, ok := camp["TextCampaign"].(map[string]any); ok {
			out["bidding_strategy"] = tc["BiddingStrategy"]
			out["tracking_params"] = tc["TrackingParams"]
		} else if uc, ok := camp["UnifiedCampaign"].(map[string]any); ok {
			out["bidding_strategy"] = uc["BiddingStrategy"]
			out["tracking_params"] = uc["TrackingParams"]
		}

		if metrics != nil {
			out["metrics"] = metrics
		}

		return common.JSONResult(out), nil
	})
}

// ---------- shared parsing/aggregation helpers ----------

// parseReportTSV decodes a Yandex Direct Reports API TSV into rowSummary slice.
// Handles three field layouts (Query / AdId / CriteriaId as the first column).
// Skips header, footer (lines starting with "Total" or "..."), and empty rows.
//
// Cost field comes back from the API in basic units (rubles, no Money=MICROS),
// so we parse it as float64 directly. Conversions come as float in TSV when
// goals are requested — we coerce to int (Yandex returns whole-number conversions
// for Reports API).
func parseReportTSV(tsv string, fieldNames []string) []rowSummary {
	lines := strings.Split(strings.TrimRight(tsv, "\n"), "\n")
	if len(lines) < 2 {
		return nil
	}

	// Header is line 0. Build column index by name.
	header := strings.Split(lines[0], "\t")
	idx := map[string]int{}
	for i, c := range header {
		idx[strings.TrimSpace(c)] = i
	}

	rows := make([]rowSummary, 0, len(lines)-1)
	for i := 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if line == "" {
			continue
		}
		// Footer or truncate-marker.
		if strings.HasPrefix(line, "Total") || strings.HasPrefix(line, "...") {
			continue
		}
		cols := strings.Split(line, "\t")

		var r rowSummary

		// First-column identifier varies by report type.
		if pos, ok := idx["Query"]; ok && pos < len(cols) {
			r.Query = cols[pos]
		}
		if pos, ok := idx["AdId"]; ok && pos < len(cols) {
			if v, err := strconv.ParseInt(strings.TrimSpace(cols[pos]), 10, 64); err == nil {
				r.AdID = v
			}
		}
		if pos, ok := idx["Criteria"]; ok && pos < len(cols) {
			r.Criteria = cols[pos]
		}

		if pos, ok := idx["Impressions"]; ok && pos < len(cols) {
			r.Impressions, _ = strconv.Atoi(strings.TrimSpace(cols[pos]))
		}
		if pos, ok := idx["Clicks"]; ok && pos < len(cols) {
			r.Clicks, _ = strconv.Atoi(strings.TrimSpace(cols[pos]))
		}
		if pos, ok := idx["Cost"]; ok && pos < len(cols) {
			r.Cost, _ = strconv.ParseFloat(strings.TrimSpace(cols[pos]), 64)
		}
		if pos, ok := idx["Conversions"]; ok && pos < len(cols) {
			// Yandex returns whole numbers for Conversions in Reports API even
			// though field is float-shaped.
			f, _ := strconv.ParseFloat(strings.TrimSpace(cols[pos]), 64)
			r.Conversions = int(f + 0.5)
		}

		// Skip rows that look like nothing happened.
		if r.Impressions == 0 && r.Clicks == 0 && r.Cost == 0 && r.Conversions == 0 {
			continue
		}

		// Derived metrics.
		if r.Impressions > 0 {
			r.CTR = roundTo(float64(r.Clicks)/float64(r.Impressions)*100, 2)
		}
		if r.Clicks > 0 {
			r.CPC = roundTo(r.Cost/float64(r.Clicks), 2)
		}
		if r.Conversions > 0 {
			r.CPA = roundTo(r.Cost/float64(r.Conversions), 2)
		}

		rows = append(rows, r)
	}
	return rows
}

// aggregateTotals sums a rowSummary slice and computes derived metrics.
func aggregateTotals(rows []rowSummary) totalsSummary {
	var t totalsSummary
	for _, r := range rows {
		t.Rows++
		t.Impressions += r.Impressions
		t.Clicks += r.Clicks
		t.Cost += r.Cost
		t.Conversions += r.Conversions
	}
	if t.Impressions > 0 {
		t.CTR = roundTo(float64(t.Clicks)/float64(t.Impressions)*100, 2)
	}
	if t.Clicks > 0 {
		t.CPC = roundTo(t.Cost/float64(t.Clicks), 2)
	}
	if t.Conversions > 0 {
		t.CPA = roundTo(t.Cost/float64(t.Conversions), 2)
	}
	t.Cost = roundTo(t.Cost, 2)
	return t
}

// head returns the first n elements of s (or all of s if shorter).
func head[T any](s []T, n int) []T {
	if n >= len(s) {
		return s
	}
	return s[:n]
}

// roundTo rounds f to `digits` decimal places.
func roundTo(f float64, digits int) float64 {
	pow := 1.0
	for i := 0; i < digits; i++ {
		pow *= 10
	}
	return float64(int(f*pow+0.5)) / pow
}
