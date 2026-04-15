package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	metrika "github.com/leadgen-mcp/server/platform/metrika"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterBenchmarkTools registers conversion value benchmarking tools.
func RegisterBenchmarkTools(s *mcpserver.MCPServer, client *Client, metrClient *metrika.Client, resolver *auth.AccountResolver) {
	registerGetConversionValues(s, client, metrClient, resolver)
}

// conversionBenchmark holds computed CPA data for a single campaign.
type conversionBenchmark struct {
	CampaignID   int64   `json:"campaign_id"`
	CampaignName string  `json:"campaign_name"`
	Cost         float64 `json:"cost"`
	Clicks       int     `json:"clicks"`
	FormConv     int     `json:"form_conversions"`
	CallConv     int     `json:"call_conversions"`
	FormCPA      float64 `json:"form_cpa,omitempty"`
	CallCPA      float64 `json:"call_cpa,omitempty"`
}

// conversionValuesResult is the tool's output.
type conversionValuesResult struct {
	City             string  `json:"city"`
	ClientLogin      string  `json:"client_login"`
	CounterID        string  `json:"counter_id"`
	Period           string  `json:"period"`
	Source           string  `json:"source"` // "campaigns" or "network_average"
	FormGoalID       int64   `json:"form_goal_id"`
	FormGoalName     string  `json:"form_goal_name"`
	CallGoalID       int64   `json:"call_goal_id,omitempty"`
	CallGoalName     string  `json:"call_goal_name,omitempty"`
	CallGoalType     string  `json:"call_goal_type,omitempty"` // received_real_calls or all_calls
	AvgFormCPA       float64 `json:"avg_form_cpa"`
	AvgCallCPA       float64 `json:"avg_call_cpa,omitempty"`
	Ratio            float64 `json:"ratio,omitempty"` // form_cpa / call_cpa (how many times form lead is more expensive)
	RecommendedValue struct {
		FormValue int64 `json:"form_value_rubles"`
		CallValue int64 `json:"call_value_rubles"`
	} `json:"recommended_value"`
	PriorityGoalsJSON string                `json:"priority_goals_json"`
	Campaigns         []conversionBenchmark `json:"campaigns,omitempty"`
	Note              string                `json:"note,omitempty"`
}

func registerGetConversionValues(s *mcpserver.MCPServer, client *Client, metrClient *metrika.Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_conversion_values",
		mcp.WithDescription(
			"Рассчитать рекомендуемые ценности конверсий (value) для priority_goals. "+
				"Анализирует CPA по действующим кампаниям города. "+
				"Если кампаний нет (новый город) — возвращает средние по сети. "+
				"Результат: готовый JSON для priority_goals в add_campaign/update_campaign."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (город). Получи через get_agency_clients."), mcp.Required()),
		mcp.WithString("counter_id", mcp.Description("ID счётчика Метрики города. Получи из config/counters.md."), mcp.Required()),
		mcp.WithNumber("days", mcp.Description("Период для анализа в днях (по умолчанию 30)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		counterID := common.GetString(req, "counter_id")

		days := common.GetInt(req, "days")
		if days <= 0 {
			days = 30
		}

		now := time.Now()
		dateTo := now.Format("2006-01-02")
		dateFrom := now.AddDate(0, 0, -days).Format("2006-01-02")

		// Step 1: Get goals from Metrika
		formGoalID, formGoalName, callGoalID, callGoalName, callGoalType, err := resolveGoals(ctx, metrClient, token, counterID)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("resolve goals: %v", err)), nil
		}
		if formGoalID == 0 {
			return common.ErrorResult("Цель form_sum_leads не найдена на счётчике " + counterID), nil
		}

		// Step 2: Get active campaigns for this client
		campaigns, err := getActiveCampaigns(ctx, client, token, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("get campaigns: %v", err)), nil
		}

		result := conversionValuesResult{
			ClientLogin:  clientLogin,
			CounterID:    counterID,
			Period:       fmt.Sprintf("%s — %s", dateFrom, dateTo),
			FormGoalID:   formGoalID,
			FormGoalName: formGoalName,
			CallGoalID:   callGoalID,
			CallGoalName: callGoalName,
			CallGoalType: callGoalType,
		}

		if len(campaigns) == 0 {
			// No campaigns — use network average
			result.Source = "network_average"
			result.Note = "Нет активных кампаний. Используются средние значения по сети."
			// Sensible defaults based on real estate vertical
			result.AvgFormCPA = 1500
			result.AvgCallCPA = 3000
			result.Ratio = 2.0
		} else {
			// Step 3: Get stats per campaign with goal breakdowns
			result.Source = "campaigns"
			goalIDs := []string{strconv.FormatInt(formGoalID, 10)}
			if callGoalID > 0 {
				goalIDs = append(goalIDs, strconv.FormatInt(callGoalID, 10))
			}

			benchmarks, err := collectBenchmarks(ctx, client, token, clientLogin, campaigns, dateFrom, dateTo, goalIDs, formGoalID, callGoalID)
			if err != nil {
				return common.ErrorResult(fmt.Sprintf("collect stats: %v", err)), nil
			}

			result.Campaigns = benchmarks

			// Calculate weighted average CPA per goal type
			// Use only campaigns that have both cost > 0 and conversions for that goal
			var formCostSum float64
			var callCostSum float64
			var totalFormConv, totalCallConv int
			for _, b := range benchmarks {
				if b.FormConv > 0 {
					formCostSum += b.Cost
					totalFormConv += b.FormConv
				}
				if b.CallConv > 0 {
					callCostSum += b.Cost
					totalCallConv += b.CallConv
				}
			}

			if totalFormConv > 0 {
				result.AvgFormCPA = math.Round(formCostSum / float64(totalFormConv))
			}
			if totalCallConv > 0 {
				result.AvgCallCPA = math.Round(callCostSum / float64(totalCallConv))
			}

			// Ratio = form_cpa / call_cpa (how many times more expensive a form lead is vs a call)
			if result.AvgFormCPA > 0 && result.AvgCallCPA > 0 {
				result.Ratio = math.Round(result.AvgFormCPA/result.AvgCallCPA*10) / 10
			}

			// Fallbacks for missing data
			if totalFormConv == 0 && totalCallConv == 0 {
				result.Source = "network_average"
				result.Note = "Есть кампании, но нет конверсий за период. Используются средние значения по сети."
				result.AvgFormCPA = 1500
				result.AvgCallCPA = 3000
				result.Ratio = 2.0
			} else if totalFormConv == 0 && totalCallConv > 0 {
				// Have calls but no forms — estimate form CPA as 2x call CPA (typical ratio)
				result.AvgFormCPA = result.AvgCallCPA * 2
				result.Ratio = 2.0
				result.Note = "Нет конверсий по заявкам за период. CPA заявки оценён как 2× CPA звонка."
			} else if totalCallConv == 0 && totalFormConv > 0 {
				// Have forms but no calls — estimate call CPA as 0.5x form CPA
				result.AvgCallCPA = result.AvgFormCPA / 2
				result.Ratio = 2.0
				result.Note = "Нет конверсий по звонкам за период. CPA звонка оценён как 0.5× CPA заявки."
			}
		}

		// Step 4: Generate recommended values
		// Use CPA as conversion value — the algorithm will use the ratio
		formValue := int64(result.AvgFormCPA)
		if formValue < 1 {
			formValue = 1
		}
		callValue := int64(result.AvgCallCPA)
		if callValue < 1 {
			callValue = 1
		}
		result.RecommendedValue.FormValue = formValue
		result.RecommendedValue.CallValue = callValue

		// Generate ready-to-use priority_goals JSON
		pgItems := []map[string]any{
			{"goal_id": formGoalID, "value": formValue},
		}
		if callGoalID > 0 {
			pgItems = append(pgItems, map[string]any{"goal_id": callGoalID, "value": callValue})
		}
		pgJSON, _ := json.Marshal(pgItems)
		result.PriorityGoalsJSON = string(pgJSON)

		out, _ := json.MarshalIndent(result, "", "  ")
		return common.TextResult(string(out)), nil
	})
}

// resolveGoals fetches Metrika goals and finds form_sum_leads + received_real_calls/all_calls.
func resolveGoals(ctx context.Context, metrClient *metrika.Client, token, counterID string) (
	formID int64, formName string, callID int64, callName string, callType string, err error,
) {
	cID, _ := strconv.ParseInt(counterID, 10, 64)
	path := fmt.Sprintf("/management/v1/counter/%d/goals", cID)

	raw, err := metrClient.Get(ctx, token, path, nil)
	if err != nil {
		return 0, "", 0, "", "", err
	}

	var resp struct {
		Goals []struct {
			ID         int64  `json:"id"`
			Name       string `json:"name"`
			Type       string `json:"type"`
			Conditions []struct {
				Type string `json:"type"`
				URL  string `json:"url"`
			} `json:"conditions"`
		} `json:"goals"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return 0, "", 0, "", "", fmt.Errorf("parse goals: %w", err)
	}

	for _, g := range resp.Goals {
		for _, c := range g.Conditions {
			switch c.URL {
			case "form_sum_leads":
				formID = g.ID
				formName = g.Name
			case "received_real_calls":
				// Always prefer received_real_calls (overwrites all_calls if found earlier)
				callID = g.ID
				callName = g.Name
				callType = "received_real_calls"
			case "all_calls":
				if callID == 0 { // fallback
					callID = g.ID
					callName = g.Name
					callType = "all_calls"
				}
			}
		}
	}

	return formID, formName, callID, callName, callType, nil
}

// getActiveCampaigns fetches ON campaigns for a client.
func getActiveCampaigns(ctx context.Context, client *Client, token, clientLogin string) ([]struct {
	ID   int64
	Name string
}, error) {
	params := map[string]any{
		"SelectionCriteria": map[string]any{
			"States": []string{"ON"},
		},
		"FieldNames": []string{"Id", "Name"},
	}

	raw, err := client.Call(ctx, token, "campaigns", "get", params, clientLogin)
	if err != nil {
		return nil, err
	}

	result, err := GetResult(raw)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Campaigns []struct {
			ID   int64  `json:"Id"`
			Name string `json:"Name"`
		} `json:"Campaigns"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parse campaigns: %w", err)
	}

	out := make([]struct {
		ID   int64
		Name string
	}, len(resp.Campaigns))
	for i, c := range resp.Campaigns {
		out[i] = struct {
			ID   int64
			Name string
		}{c.ID, c.Name}
	}
	return out, nil
}

// collectBenchmarks gathers stats for each campaign and parses TSV into benchmarks.
func collectBenchmarks(ctx context.Context, client *Client, token, clientLogin string,
	campaigns []struct {
		ID   int64
		Name string
	}, dateFrom, dateTo string, goalIDs []string, formGoalID, callGoalID int64,
) ([]conversionBenchmark, error) {

	// Build campaign IDs list
	campIDs := make([]string, len(campaigns))
	campNames := make(map[int64]string, len(campaigns))
	for i, c := range campaigns {
		campIDs[i] = strconv.FormatInt(c.ID, 10)
		campNames[c.ID] = c.Name
	}

	// Single report request for all campaigns
	fieldNames := []string{"CampaignId", "Cost", "Clicks", "Conversions"}

	params := map[string]any{
		"SelectionCriteria": map[string]any{
			"Filter": []any{
				map[string]any{
					"Field":    "CampaignId",
					"Operator": "IN",
					"Values":   campIDs,
				},
			},
			"DateFrom": dateFrom,
			"DateTo":   dateTo,
		},
		"FieldNames":        fieldNames,
		"Goals":             goalIDs,
		"AttributionModels": []string{"LYDC"},
		"ReportName":        fmt.Sprintf("bench_%d", time.Now().UnixNano()),
		"ReportType":        "CAMPAIGN_PERFORMANCE_REPORT",
		"DateRangeType":     "CUSTOM_DATE",
		"Format":            "TSV",
		"IncludeVAT":        "NO",
		"IncludeDiscount":   "NO",
	}

	tsv, err := client.CallReport(ctx, token, params, clientLogin)
	if err != nil {
		return nil, err
	}

	return parseBenchmarkTSV(tsv, campNames, formGoalID, callGoalID)
}

// parseBenchmarkTSV parses TSV report output into conversionBenchmark slice.
func parseBenchmarkTSV(tsv string, campNames map[int64]string, formGoalID, callGoalID int64) ([]conversionBenchmark, error) {
	lines := strings.Split(strings.TrimSpace(tsv), "\n")
	if len(lines) < 2 {
		return nil, nil // Empty report
	}

	// Parse header to find column indices
	headers := strings.Split(lines[0], "\t")
	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[h] = i
	}

	// Find conversions columns — they follow pattern "Conversions" with goal suffix
	formConvCol := -1
	callConvCol := -1
	formGoalStr := strconv.FormatInt(formGoalID, 10)
	callGoalStr := strconv.FormatInt(callGoalID, 10)

	for i, h := range headers {
		if strings.Contains(h, "Conversions") {
			if strings.Contains(h, formGoalStr) {
				formConvCol = i
			} else if callGoalID > 0 && strings.Contains(h, callGoalStr) {
				callConvCol = i
			}
		}
	}

	var benchmarks []conversionBenchmark
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "--") || strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")

		campID := parseIntField(cols, colIdx, "CampaignId")
		cost := parseFloatField(cols, colIdx, "Cost")
		clicks := int(parseIntField(cols, colIdx, "Clicks"))

		var formConv, callConv int
		if formConvCol >= 0 && formConvCol < len(cols) {
			formConv, _ = strconv.Atoi(strings.TrimSpace(cols[formConvCol]))
		}
		if callConvCol >= 0 && callConvCol < len(cols) {
			callConv, _ = strconv.Atoi(strings.TrimSpace(cols[callConvCol]))
		}

		b := conversionBenchmark{
			CampaignID:   campID,
			CampaignName: campNames[campID],
			Cost:         cost,
			Clicks:       clicks,
			FormConv:     formConv,
			CallConv:     callConv,
		}
		if formConv > 0 {
			b.FormCPA = math.Round(cost / float64(formConv))
		}
		if callConv > 0 {
			b.CallCPA = math.Round(cost / float64(callConv))
		}

		benchmarks = append(benchmarks, b)
	}

	return benchmarks, nil
}

func parseIntField(cols []string, colIdx map[string]int, name string) int64 {
	idx, ok := colIdx[name]
	if !ok || idx >= len(cols) {
		return 0
	}
	v, _ := strconv.ParseInt(strings.TrimSpace(cols[idx]), 10, 64)
	return v
}

func parseFloatField(cols []string, colIdx map[string]int, name string) float64 {
	idx, ok := colIdx[name]
	if !ok || idx >= len(cols) {
		return 0
	}
	v, _ := strconv.ParseFloat(strings.TrimSpace(cols[idx]), 64)
	return v
}
