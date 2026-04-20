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
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Default monthly seasonality multipliers (1.0 = baseline year-round).
// Built from observed Russian real-estate demand curves — overridden automatically
// once historical year-over-year data is in.
var defaultSeasonality = map[time.Month]float64{
	time.January:   0.85,
	time.February:  0.90,
	time.March:     1.05,
	time.April:     1.10,
	time.May:       1.15,
	time.June:      1.00,
	time.July:      0.80,
	time.August:    0.75,
	time.September: 1.10,
	time.October:   1.20,
	time.November:  1.05,
	time.December:  0.95,
}

// Horizon plans supported by the forecast tool.
var defaultHorizons = []int{3, 7, 15, 30, 90}

// RegisterForecastTools exposes forecast_campaign as a single MCP tool.
// It depends on the Direct Reports API to read historical daily stats, then
// projects clicks / cost / conversions forward with a 95% confidence interval.
func RegisterForecastTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerForecastCampaign(s, client, resolver)
}

func registerForecastCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("forecast_campaign",
		mcp.WithDescription(
			"Прогноз расхода / кликов / конверсий по кампании на заданные горизонты "+
				"(3/7/15/30/90 дней по умолчанию). Берёт дневную статистику за lookback_days "+
				"(default 28), считает baseline + stddev, применяет сезонный множитель. "+
				"Возвращает JSON с point-estimate и 95% доверительным интервалом."),
		mcp.WithString("account",
			mcp.Description("Аккаунт-резолвер (для агентства)")),
		mcp.WithString("client_login",
			mcp.Description("Client-Login города (обязательно для агентского аккаунта)")),
		mcp.WithNumber("campaign_id",
			mcp.Description("ID кампании в Директе"),
			mcp.Required()),
		mcp.WithNumber("lookback_days",
			mcp.Description("Сколько дней назад брать для baseline (default 28, min 7, max 180)")),
		mcp.WithString("horizons",
			mcp.Description("Горизонты прогноза через запятую (default 3,7,15,30,90)")),
		mcp.WithString("goal_ids",
			mcp.Description("Цели для расчёта конверсий (через запятую)")),
		mcp.WithString("attribution",
			mcp.Description("Атрибуция отчёта (default LYDC)")),
		mcp.WithNumber("seasonality_multiplier",
			mcp.Description("Ручной сезонный множитель для горизонта (override дефолта)")),
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

		lookback := common.GetInt(req, "lookback_days")
		if lookback <= 0 {
			lookback = 28
		}
		if lookback < 7 {
			lookback = 7
		}
		if lookback > 180 {
			lookback = 180
		}

		horizons := parseHorizons(common.GetString(req, "horizons"))
		goalIDs := common.GetStringSlice(req, "goal_ids")
		attribution := common.GetString(req, "attribution")
		if attribution == "" {
			attribution = "LYDC"
		}
		manualSeason := 0.0
		if v := common.GetString(req, "seasonality_multiplier"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				manualSeason = f
			}
		}

		// Pull daily rows for the lookback window.
		dateTo := time.Now().UTC().AddDate(0, 0, -1) // yesterday
		dateFrom := dateTo.AddDate(0, 0, -(lookback - 1))
		rows, err := fetchDailyRows(ctx, client, token, clientLogin, campaignID, dateFrom, dateTo, goalIDs, attribution)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("fetch daily stats: %v", err)), nil
		}
		if len(rows) == 0 {
			return common.ErrorResult(fmt.Sprintf(
				"нет daily-статистики за период %s..%s для кампании %d",
				dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"), campaignID)), nil
		}

		baseline := computeBaseline(rows)
		forecasts := make([]HorizonForecast, 0, len(horizons))
		for _, h := range horizons {
			forecasts = append(forecasts, buildForecast(baseline, h, dateTo, manualSeason))
		}

		out := ForecastResponse{
			CampaignID:      campaignID,
			ClientLogin:     clientLogin,
			LookbackWindow:  fmt.Sprintf("%s..%s", dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02")),
			LookbackDays:    lookback,
			ObservedDays:    baseline.Days,
			Attribution:     attribution,
			Baseline:        baseline,
			Horizons:        forecasts,
			ManualSeasonMul: manualSeason,
			Notes:           baselineNotes(baseline),
		}
		payload, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("encode: %v", err)), nil
		}
		return common.TextResult(string(payload)), nil
	})
}

// --- data types ---

// Baseline is the per-day aggregate used as the forecast's center of gravity.
type Baseline struct {
	Days               int     `json:"days"`
	ClicksMean         float64 `json:"clicks_mean"`
	ClicksStd          float64 `json:"clicks_std"`
	CostMean           float64 `json:"cost_mean"`
	CostStd            float64 `json:"cost_std"`
	ConversionsMean    float64 `json:"conversions_mean"`
	ConversionsStd     float64 `json:"conversions_std"`
	HasConversionsData bool    `json:"has_conversions_data"`
}

// HorizonForecast is the projection for a single horizon (in days).
type HorizonForecast struct {
	HorizonDays          int     `json:"horizon_days"`
	SeasonalityMultiplier float64 `json:"seasonality_multiplier"`
	ClicksPoint          float64 `json:"clicks_point"`
	ClicksLow            float64 `json:"clicks_ci95_low"`
	ClicksHigh           float64 `json:"clicks_ci95_high"`
	CostPoint            float64 `json:"cost_point"`
	CostLow              float64 `json:"cost_ci95_low"`
	CostHigh             float64 `json:"cost_ci95_high"`
	ConversionsPoint     float64 `json:"conversions_point,omitempty"`
	ConversionsLow       float64 `json:"conversions_ci95_low,omitempty"`
	ConversionsHigh      float64 `json:"conversions_ci95_high,omitempty"`
	CPA                  float64 `json:"cpa,omitempty"` // point cost / point conversions
}

// ForecastResponse is the full payload returned to the agent.
type ForecastResponse struct {
	CampaignID      int               `json:"campaign_id"`
	ClientLogin     string            `json:"client_login,omitempty"`
	LookbackWindow  string            `json:"lookback_window"`
	LookbackDays    int               `json:"lookback_days"`
	ObservedDays    int               `json:"observed_days"`
	Attribution     string            `json:"attribution"`
	Baseline        Baseline          `json:"baseline"`
	Horizons        []HorizonForecast `json:"horizons"`
	ManualSeasonMul float64           `json:"manual_season_multiplier,omitempty"`
	Notes           []string          `json:"notes,omitempty"`
}

// --- helpers ---

type dailyRow struct {
	Date        string
	Clicks      float64
	Cost        float64
	Conversions float64
	HasConv     bool
}

func fetchDailyRows(ctx context.Context, client *Client, token, clientLogin string,
	campaignID int, dateFrom, dateTo time.Time, goalIDs []string, attribution string,
) ([]dailyRow, error) {
	fields := []string{"Date", "Impressions", "Clicks", "Cost"}
	if len(goalIDs) > 0 {
		fields = append(fields, "Conversions")
	}

	params := map[string]any{
		"SelectionCriteria": map[string]any{
			"DateFrom": dateFrom.Format("2006-01-02"),
			"DateTo":   dateTo.Format("2006-01-02"),
			"Filter": []any{
				map[string]any{
					"Field":    "CampaignId",
					"Operator": "EQUALS",
					"Values":   []string{strconv.Itoa(campaignID)},
				},
			},
		},
		"FieldNames":      fields,
		"ReportName":      fmt.Sprintf("forecast_%d_%d", campaignID, time.Now().UnixNano()),
		"ReportType":      "CAMPAIGN_PERFORMANCE_REPORT",
		"DateRangeType":   "CUSTOM_DATE",
		"Format":          "TSV",
		"IncludeVAT":      "YES",
		"IncludeDiscount": "NO",
	}
	if len(goalIDs) > 0 {
		params["Goals"] = goalIDs
		params["AttributionModels"] = []string{attribution}
	}

	tsv, err := client.CallReport(ctx, token, params, clientLogin)
	if err != nil {
		return nil, err
	}
	return parseDailyTSV(tsv, len(goalIDs) > 0), nil
}

// parseDailyTSV parses a Reports API TSV with Date/Clicks/Cost/(Conversions) columns.
// Skips header, "Total rows:" trailer and malformed lines silently.
func parseDailyTSV(tsv string, expectConv bool) []dailyRow {
	lines := strings.Split(strings.TrimRight(tsv, "\n"), "\n")
	if len(lines) < 2 {
		return nil
	}
	header := strings.Split(lines[0], "\t")
	idx := make(map[string]int)
	for i, h := range header {
		idx[strings.TrimSpace(h)] = i
	}
	dateI, clI, costI := idx["Date"], idx["Clicks"], idx["Cost"]
	convI, hasConv := idx["Conversions"]

	rows := make([]dailyRow, 0, len(lines)-1)
	for _, ln := range lines[1:] {
		if ln == "" || strings.HasPrefix(ln, "Total rows:") {
			continue
		}
		parts := strings.Split(ln, "\t")
		if len(parts) <= clI || len(parts) <= costI {
			continue
		}
		r := dailyRow{
			Date:   strings.TrimSpace(parts[dateI]),
			Clicks: atof(parts[clI]),
			Cost:   atof(parts[costI]),
		}
		if expectConv && hasConv && len(parts) > convI {
			r.Conversions = atof(parts[convI])
			r.HasConv = true
		}
		rows = append(rows, r)
	}
	return rows
}

func atof(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "--" {
		return 0
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return 0
	}
	return v
}

func computeBaseline(rows []dailyRow) Baseline {
	n := len(rows)
	if n == 0 {
		return Baseline{}
	}
	var clicks, cost, conv []float64
	anyConv := false
	for _, r := range rows {
		clicks = append(clicks, r.Clicks)
		cost = append(cost, r.Cost)
		if r.HasConv {
			anyConv = true
			conv = append(conv, r.Conversions)
		}
	}
	clMean, clStd := meanStd(clicks)
	coMean, coStd := meanStd(cost)
	b := Baseline{
		Days:       n,
		ClicksMean: clMean, ClicksStd: clStd,
		CostMean: coMean, CostStd: coStd,
	}
	if anyConv {
		cvMean, cvStd := meanStd(conv)
		b.ConversionsMean = cvMean
		b.ConversionsStd = cvStd
		b.HasConversionsData = true
	}
	return b
}

func meanStd(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	mean := sum / float64(len(xs))
	if len(xs) == 1 {
		return mean, 0
	}
	var sq float64
	for _, v := range xs {
		d := v - mean
		sq += d * d
	}
	return mean, math.Sqrt(sq / float64(len(xs)-1))
}

func buildForecast(b Baseline, horizon int, refDay time.Time, manualSeason float64) HorizonForecast {
	// Use the midpoint of the forecast window to pick a month multiplier.
	mid := refDay.AddDate(0, 0, horizon/2)
	season := defaultSeasonality[mid.Month()]
	if season == 0 {
		season = 1.0
	}
	if manualSeason > 0 {
		season = manualSeason
	}

	h := float64(horizon)
	// 95% CI for the sum over `h` iid days: mean*h ± 1.96 * std * sqrt(h).
	z := 1.96
	sqrtH := math.Sqrt(h)
	f := HorizonForecast{
		HorizonDays:           horizon,
		SeasonalityMultiplier: season,
		ClicksPoint:           b.ClicksMean * h * season,
		CostPoint:             b.CostMean * h * season,
	}
	f.ClicksLow = max0(b.ClicksMean*h*season - z*b.ClicksStd*sqrtH*season)
	f.ClicksHigh = b.ClicksMean*h*season + z*b.ClicksStd*sqrtH*season
	f.CostLow = max0(b.CostMean*h*season - z*b.CostStd*sqrtH*season)
	f.CostHigh = b.CostMean*h*season + z*b.CostStd*sqrtH*season

	if b.HasConversionsData {
		f.ConversionsPoint = b.ConversionsMean * h
		f.ConversionsLow = max0(b.ConversionsMean*h - z*b.ConversionsStd*sqrtH)
		f.ConversionsHigh = b.ConversionsMean*h + z*b.ConversionsStd*sqrtH
		if f.ConversionsPoint > 0 {
			f.CPA = f.CostPoint / f.ConversionsPoint
		}
	}
	return f
}

func max0(x float64) float64 {
	if x < 0 {
		return 0
	}
	return x
}

func parseHorizons(s string) []int {
	s = strings.TrimSpace(s)
	if s == "" {
		return defaultHorizons
	}
	var out []int
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if v, err := strconv.Atoi(p); err == nil && v > 0 && v <= 365 {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return defaultHorizons
	}
	return out
}

func baselineNotes(b Baseline) []string {
	var notes []string
	if b.Days < 14 {
		notes = append(notes, fmt.Sprintf(
			"Baseline построен на %d днях — меньше 14. Прогноз менее надёжен, расширь lookback_days.", b.Days))
	}
	if b.ClicksMean < 1 {
		notes = append(notes, "Средний дневной клик <1: статистика разрежена, CI 95% может перекрывать ноль.")
	}
	if !b.HasConversionsData {
		notes = append(notes, "Нет goal_ids — прогноз конверсий не рассчитан. Передай goal_ids для полного прогноза.")
	}
	return notes
}
