package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// ===== Types =====

// campaignMeta holds minimal campaign info needed for benchmarking + learning/moderation checks.
type campaignMeta struct {
	ID                  int64
	Name                string
	StartDate           string
	StatusClarification string
}

// conversionBenchmark holds computed CPA data for a single campaign.
type conversionBenchmark struct {
	CampaignID   int64   `json:"campaign_id"`
	CampaignName string  `json:"campaign_name"`
	Theme        string  `json:"theme,omitempty"`
	Cost         float64 `json:"cost"`
	Clicks       int     `json:"clicks"`
	FormConv     int     `json:"form_conversions"`
	CallConv     int     `json:"call_conversions"`
	FormCPA      float64 `json:"form_cpa,omitempty"`
	CallCPA      float64 `json:"call_cpa,omitempty"`
}

// goalStats captures robust statistics for a single goal over included campaigns.
type goalStats struct {
	WeightedMean     float64 `json:"weighted_mean"`
	RobustMean       float64 `json:"robust_mean"`
	P25              float64 `json:"p25"`
	P50              float64 `json:"p50"`
	P75              float64 `json:"p75"`
	IQR              float64 `json:"iqr"`
	OutliersRemoved  int     `json:"outliers_removed"`
	OutlierNote      string  `json:"outlier_note,omitempty"`
	TotalConversions int     `json:"total_conversions"`
	TotalCost        float64 `json:"total_cost"`
	IncludedCount    int     `json:"included_count"`
	Confidence       string  `json:"confidence"`
}

// excludedCampaign describes a campaign removed from the benchmark with a reason.
type excludedCampaign struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// filterApplied records the thresholds actually used (after any relaxation).
type filterApplied struct {
	MinConversions  int     `json:"min_conversions"`
	MinClicks       int     `json:"min_clicks"`
	MinCost         float64 `json:"min_cost"`
	ExcludeLearning bool    `json:"exclude_learning"`
	LearningDays    int     `json:"learning_days"`
}

// recommendedValue is the payload for downstream priority_goals JSON.
type recommendedValue struct {
	FormValue int64  `json:"form_value_rubles"`
	CallValue int64  `json:"call_value_rubles"`
	Source    string `json:"source"`
}

// themeBreakdown is per-theme robust stats when input theme is not specified.
type themeBreakdown struct {
	Theme           string  `json:"theme"`
	NCampaigns      int     `json:"n_campaigns"`
	FormCPA         float64 `json:"form_cpa,omitempty"`
	CallCPA         float64 `json:"call_cpa,omitempty"`
	FormConversions int     `json:"form_conversions"`
	CallConversions int     `json:"call_conversions"`
	TotalCost       float64 `json:"total_cost"`
}

// trendData captures the 7d-vs-period CPA trend signal.
type trendData struct {
	FormCPA7d        float64 `json:"form_cpa_7d,omitempty"`
	CallCPA7d        float64 `json:"call_cpa_7d,omitempty"`
	FormConv7d       int     `json:"form_conv_7d"`
	CallConv7d       int     `json:"call_conv_7d"`
	FormVsPeriod     string  `json:"form_cpa_vs_period,omitempty"`
	CallVsPeriod     string  `json:"call_cpa_vs_period,omitempty"`
	Signal           string  `json:"signal"`
	InsufficientData bool    `json:"insufficient_data,omitempty"`
}

// conversionValuesResult is the tool's output.
type conversionValuesResult struct {
	// Identity / period
	City        string `json:"city,omitempty"`
	ClientLogin string `json:"client_login"`
	CounterID   string `json:"counter_id"`
	Theme       string `json:"theme,omitempty"`
	Period      string `json:"period"`
	Source      string `json:"source"` // "campaigns" | "network_average" | "mixed"

	// Phase 2: window transparency
	ActualDaysUsed int    `json:"actual_days_used"`
	WindowReason   string `json:"window_reason,omitempty"`

	// Goals
	FormGoalID   int64  `json:"form_goal_id"`
	FormGoalName string `json:"form_goal_name"`
	CallGoalID   int64  `json:"call_goal_id,omitempty"`
	CallGoalName string `json:"call_goal_name,omitempty"`
	CallGoalType string `json:"call_goal_type,omitempty"`

	// Backward-compat CPA fields (weighted mean over all fetched campaigns with conv>0)
	AvgFormCPA float64 `json:"avg_form_cpa"`
	AvgCallCPA float64 `json:"avg_call_cpa,omitempty"`
	Ratio      float64 `json:"ratio,omitempty"`

	// Phase 1: benchmark vs target
	BenchmarkFormCPA float64 `json:"benchmark_form_cpa,omitempty"`
	BenchmarkCallCPA float64 `json:"benchmark_call_cpa,omitempty"`
	TargetFormCPA    float64 `json:"target_form_cpa,omitempty"`
	TargetCallCPA    float64 `json:"target_call_cpa,omitempty"`

	// Phase 3: cost attribution model used per-goal CPA computation.
	CostAttribution string `json:"cost_attribution,omitempty"` // "yandex_per_goal" | "full_campaign_cost"

	// Phase 1: robust statistics
	Statistics struct {
		Form *goalStats `json:"form,omitempty"`
		Call *goalStats `json:"call,omitempty"`
	} `json:"statistics,omitempty"`

	// Phase 1: confidence
	Confidence       string `json:"confidence,omitempty"`
	ConfidenceReason string `json:"confidence_reason,omitempty"`

	// Phase 1: campaign-level transparency
	CampaignsIncluded []conversionBenchmark `json:"campaigns_included,omitempty"`
	CampaignsExcluded []excludedCampaign    `json:"campaigns_excluded,omitempty"`

	// Phase 1: filter transparency
	FilterApplied filterApplied `json:"filter_applied"`
	FilterRelaxed bool          `json:"filter_relaxed,omitempty"`
	RelaxedReason string        `json:"relaxed_reason,omitempty"`

	// Phase 2: per-theme breakdown when input theme is not specified.
	BreakdownByTheme map[string]*themeBreakdown `json:"breakdown_by_theme,omitempty"`

	// Phase 3: 7-day trend vs full period.
	Trend7d *trendData `json:"trend_7d,omitempty"`

	// Strategy-ready output
	RecommendedValue  recommendedValue `json:"recommended_value"`
	PriorityGoalsJSON string           `json:"priority_goals_json"`

	// Legacy: all fetched benchmarks (unfiltered)
	Campaigns []conversionBenchmark `json:"campaigns,omitempty"`
	Note      string                `json:"note,omitempty"`
}

// ===== Network benchmarks (FR-8) =====

// networkBenchmark holds form/call CPA for a single (theme, tier) cell.
type networkBenchmark struct {
	Form float64 `json:"form"`
	Call float64 `json:"call"`
}

// networkBenchmarksTable is the on-disk JSON used as fallback for cities without enough data.
type networkBenchmarksTable struct {
	UpdatedAt  string                                 `json:"updated_at"`
	Source     string                                 `json:"source,omitempty"`
	Note       string                                 `json:"note,omitempty"`
	Benchmarks map[string]map[string]networkBenchmark `json:"benchmarks"` // theme → tier → bench
	Default    networkBenchmark                       `json:"default"`
}

var (
	networkTable     *networkBenchmarksTable
	networkTableOnce sync.Once
	networkTablePath = filepath.Join("data", "network_benchmarks.json") // relative to working dir / /app/
)

// loadNetworkBenchmarks reads the JSON file once on first use, with hardcoded fallback if missing.
func loadNetworkBenchmarks() *networkBenchmarksTable {
	networkTableOnce.Do(func() {
		// Try /app/data first (Docker), then ./data (local), then absolute.
		paths := []string{
			"/app/data/network_benchmarks.json",
			networkTablePath,
			filepath.Join("server", "data", "network_benchmarks.json"),
		}
		for _, p := range paths {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			var t networkBenchmarksTable
			if err := json.Unmarshal(data, &t); err == nil && len(t.Benchmarks) > 0 {
				networkTable = &t
				return
			}
		}
		// Fallback: hardcoded minimal table (matches old defaults — form > call for real estate).
		networkTable = &networkBenchmarksTable{
			UpdatedAt: "hardcoded_fallback",
			Source:    "go_default",
			Default:   networkBenchmark{Form: 3000, Call: 1500},
			Benchmarks: map[string]map[string]networkBenchmark{
				"vtorichka":   {"tier_1": {3500, 1500}, "tier_2": {2500, 1100}, "tier_3": {1800, 800}},
				"novostroyki": {"tier_1": {5000, 2200}, "tier_2": {3500, 1500}, "tier_3": {2500, 1000}},
				"zagorodka":   {"tier_1": {4500, 2000}, "tier_2": {3200, 1400}, "tier_3": {2200, 950}},
				"ipoteka":     {"tier_1": {3800, 1700}, "tier_2": {2700, 1200}, "tier_3": {2000, 900}},
				"arenda":      {"tier_1": {2200, 950}, "tier_2": {1600, 700}, "tier_3": {1100, 500}},
				"commerce":    {"tier_1": {6000, 2700}, "tier_2": {4500, 2000}, "tier_3": {3000, 1300}},
				"agency":      {"tier_1": {2200, 950}, "tier_2": {1700, 750}, "tier_3": {1300, 600}},
				"imidzh":      {"tier_1": {4500, 2000}, "tier_2": {3200, 1400}, "tier_3": {2200, 950}},
				"hr":          {"tier_1": {1800, 800}, "tier_2": {1300, 600}, "tier_3": {900, 400}},
			},
		}
	})
	return networkTable
}

// networkFallback returns (form, call, source) for a (theme, tier) lookup with graceful chain:
// 1) theme + tier  →  2) tier (avg across themes)  →  3) default.
func networkFallback(theme, tier string) (float64, float64, string) {
	t := loadNetworkBenchmarks()
	if theme != "" {
		if themeMap, ok := t.Benchmarks[theme]; ok {
			if b, ok2 := themeMap[tier]; ok2 {
				return b.Form, b.Call, fmt.Sprintf("network[%s/%s]", theme, tier)
			}
		}
	}
	// Fallback to tier average across all themes.
	var sumForm, sumCall float64
	var n int
	for _, themeMap := range t.Benchmarks {
		if b, ok := themeMap[tier]; ok {
			sumForm += b.Form
			sumCall += b.Call
			n++
		}
	}
	if n > 0 {
		return math.Round(sumForm / float64(n)), math.Round(sumCall / float64(n)), fmt.Sprintf("network[*/%s]", tier)
	}
	return t.Default.Form, t.Default.Call, "network[default]"
}

// ===== Theme parser (FR-4) =====

// themePatterns maps canonical theme keys to recognition regexps. Order matters — first match wins.
// Names follow campaign_naming.md: "Город | Тип | Тематика | Детализация | [посадка]".
var themePatterns = []struct {
	key string
	re  *regexp.Regexp
}{
	{"hr", regexp.MustCompile(`(?i)\bhr\b|вакансии|career|карьер`)},
	{"ipoteka", regexp.MustCompile(`(?i)ипотек`)},
	{"arenda", regexp.MustCompile(`(?i)аренд`)},
	{"commerce", regexp.MustCompile(`(?i)коммерческ|commerce`)},
	{"novostroyki", regexp.MustCompile(`(?i)новостро`)},
	{"vtorichka", regexp.MustCompile(`(?i)вторичк`)},
	{"zagorodka", regexp.MustCompile(`(?i)загородк|загородн`)},
	{"agency", regexp.MustCompile(`(?i)агентств`)},
	{"imidzh", regexp.MustCompile(`(?i)имидж|brand|бренд`)},
}

// parseTheme inspects a campaign name and returns a canonical theme key, or "unknown".
// Strategy: prefer the 3rd pipe-segment ("тематика"); else scan whole name; else "unknown".
func parseTheme(name string) string {
	parts := strings.Split(name, "|")
	if len(parts) >= 3 {
		segment := strings.TrimSpace(parts[2])
		for _, p := range themePatterns {
			if p.re.MatchString(segment) {
				return p.key
			}
		}
	}
	for _, p := range themePatterns {
		if p.re.MatchString(name) {
			return p.key
		}
	}
	return "unknown"
}

// ===== Tool registration =====

func registerGetConversionValues(s *mcpserver.MCPServer, client *Client, metrClient *metrika.Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_conversion_values",
		mcp.WithDescription(
			"Рассчитать CPA-бенчмарк и ценности конверсий (value) для priority_goals. "+
				"Phase 1+2+3: робастная статистика (медиана + IQR), confidence, разделение benchmark/target, "+
				"исключение learning, авто-окно, theme-сегментация, breakdown_by_theme, network-fallback по тиру города, "+
				"тренд 7d vs период, прозрачность атрибуции cost. "+
				"Возвращает priority_goals_json + полную картину расчёта."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (город). Получи через get_agency_clients."), mcp.Required()),
		mcp.WithString("counter_id", mcp.Description("ID счётчика Метрики города. Получи из config/counters.md."), mcp.Required()),
		mcp.WithString("theme", mcp.Description("Фильтр тематики: vtorichka, novostroyki, zagorodka, ipoteka, arenda, commerce, agency, imidzh, hr. Если задан — расчёт только по совпадающим кампаниям. Если не задан — общий бенчмарк + breakdown_by_theme.")),
		mcp.WithNumber("days", mcp.Description("Период анализа в днях (default 30). Игнорируется при auto_window=true и нехватке данных.")),
		mcp.WithBoolean("auto_window", mcp.Description("Авто-окно: 30→60→90 при <20 конв, 30→14 при >150 конв. Default true.")),
		mcp.WithNumber("min_conversions", mcp.Description("Минимум конверсий на кампанию для включения в расчёт цели (default 3). Если все кампании отсечены — авто-релакс до 1 с флагом filter_relaxed.")),
		mcp.WithNumber("min_clicks", mcp.Description("Минимум кликов на кампанию (default 100).")),
		mcp.WithNumber("min_cost", mcp.Description("Минимум расхода на кампанию в рублях (default 1000).")),
		mcp.WithBoolean("exclude_learning", mcp.Description("Исключать кампании первых 14 дней после запуска и с замечаниями модерации (default true).")),
		mcp.WithNumber("target_form_cpa_override", mcp.Description("Явный таргет формы для стратегии в рублях. Если задан — priority_goals_json использует это значение, benchmark остаётся фактическим.")),
		mcp.WithNumber("target_call_cpa_override", mcp.Description("Явный таргет звонка для стратегии в рублях.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		counterID := common.GetString(req, "counter_id")
		inputTheme := strings.ToLower(strings.TrimSpace(common.GetString(req, "theme")))

		days := common.GetInt(req, "days")
		if days <= 0 {
			days = 30
		}
		autoWindow := true
		if v, ok := req.GetArguments()["auto_window"].(bool); ok {
			autoWindow = v
		}
		minConv := common.GetInt(req, "min_conversions")
		if minConv <= 0 {
			minConv = 3
		}
		minClicks := common.GetInt(req, "min_clicks")
		if minClicks <= 0 {
			minClicks = 100
		}
		minCost := float64(common.GetInt(req, "min_cost"))
		if minCost <= 0 {
			minCost = 1000
		}
		excludeLearning := true
		if v, ok := req.GetArguments()["exclude_learning"].(bool); ok {
			excludeLearning = v
		}
		const learningDays = 14

		targetFormOverride := float64(common.GetInt(req, "target_form_cpa_override"))
		targetCallOverride := float64(common.GetInt(req, "target_call_cpa_override"))

		// City + tier resolution from login.
		city := CityNameForLogin(clientLogin)
		tier := TierForLogin(clientLogin)

		// Step 1: Resolve Metrika goals (form + call).
		formGoalID, formGoalName, callGoalID, callGoalName, callGoalType, err := resolveGoals(ctx, metrClient, token, counterID)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("resolve goals: %v", err)), nil
		}
		if formGoalID == 0 {
			return common.ErrorResult("Цель form_sum_leads не найдена на счётчике " + counterID), nil
		}

		// Step 2: Get active campaigns with StartDate + StatusClarification.
		campaigns, err := getActiveCampaigns(ctx, client, token, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("get campaigns: %v", err)), nil
		}

		result := conversionValuesResult{
			City:         city,
			ClientLogin:  clientLogin,
			CounterID:    counterID,
			Theme:        inputTheme,
			FormGoalID:   formGoalID,
			FormGoalName: formGoalName,
			CallGoalID:   callGoalID,
			CallGoalName: callGoalName,
			CallGoalType: callGoalType,
			FilterApplied: filterApplied{
				MinConversions:  minConv,
				MinClicks:       minClicks,
				MinCost:         minCost,
				ExcludeLearning: excludeLearning,
				LearningDays:    learningDays,
			},
		}

		// Step 3: Apply learning/moderation filter.
		var kept []campaignMeta
		var excluded []excludedCampaign
		now := time.Now()
		if excludeLearning {
			for _, c := range campaigns {
				if reason := learningExclusionReason(c, now, learningDays); reason != "" {
					excluded = append(excluded, excludedCampaign{c.ID, c.Name, reason})
				} else {
					kept = append(kept, c)
				}
			}
		} else {
			kept = campaigns
		}

		// Step 4: Apply theme filter if user specified one.
		if inputTheme != "" && inputTheme != "unknown" {
			var themeKept []campaignMeta
			for _, c := range kept {
				ct := parseTheme(c.Name)
				if ct == inputTheme {
					themeKept = append(themeKept, c)
				} else {
					excluded = append(excluded, excludedCampaign{c.ID, c.Name, fmt.Sprintf("theme_mismatch: parsed=%s, requested=%s", ct, inputTheme)})
				}
			}
			kept = themeKept
		}

		if len(kept) == 0 {
			applyNetworkFallback(&result, inputTheme, tier, len(campaigns), len(excluded), "no_campaigns_after_filter")
			result.CampaignsExcluded = excluded
			result.ActualDaysUsed = days
			return emitResult(&result, formGoalID, callGoalID, targetFormOverride, targetCallOverride)
		}

		// Step 5: Auto-window loop. Try requested days first, expand if too little data, narrow if too much.
		goalIDs := []string{strconv.FormatInt(formGoalID, 10)}
		if callGoalID > 0 {
			goalIDs = append(goalIDs, strconv.FormatInt(callGoalID, 10))
		}

		actualDays := days
		var benchmarks []conversionBenchmark
		var costAttr string
		var windowReason string
		windowAttempts := []int{}
		for iter := 0; iter < 4; iter++ {
			windowAttempts = append(windowAttempts, actualDays)
			dateTo := now.Format("2006-01-02")
			dateFrom := now.AddDate(0, 0, -actualDays).Format("2006-01-02")
			b, attr, fetchErr := collectBenchmarks(ctx, client, token, clientLogin, kept, dateFrom, dateTo, goalIDs, formGoalID, callGoalID)
			if fetchErr != nil {
				return common.ErrorResult(fmt.Sprintf("collect stats (window=%dd): %v", actualDays, fetchErr)), nil
			}
			benchmarks = b
			costAttr = attr

			if !autoWindow {
				windowReason = fmt.Sprintf("auto_window=false, fixed at %dd", actualDays)
				break
			}
			totalConv := 0
			for _, x := range benchmarks {
				totalConv += x.FormConv + x.CallConv
			}
			// Narrow to 14 if 30d already has 150+ conv (more recent signal).
			if iter == 0 && actualDays == 30 && totalConv > 150 {
				actualDays = 14
				continue
			}
			// Expand if too little data.
			if totalConv < 20 && actualDays < 90 {
				switch actualDays {
				case 30:
					actualDays = 60
				case 60:
					actualDays = 90
				default:
					actualDays = 90
				}
				continue
			}
			windowReason = fmt.Sprintf("settled at %dd: total_conv=%d (attempts: %v)", actualDays, totalConv, windowAttempts)
			break
		}
		if windowReason == "" {
			windowReason = fmt.Sprintf("max iterations reached, settled at %dd (attempts: %v)", actualDays, windowAttempts)
		}
		result.ActualDaysUsed = actualDays
		result.WindowReason = windowReason
		result.CostAttribution = costAttr
		result.Period = fmt.Sprintf("%s — %s", now.AddDate(0, 0, -actualDays).Format("2006-01-02"), now.Format("2006-01-02"))

		// Annotate themes on every benchmark.
		for i := range benchmarks {
			benchmarks[i].Theme = parseTheme(benchmarks[i].CampaignName)
		}
		result.Campaigns = benchmarks

		if len(benchmarks) == 0 {
			applyNetworkFallback(&result, inputTheme, tier, len(campaigns), len(excluded), "no_data_in_period")
			result.Note = "Кампании активны, но за период нет показов/кликов. Используются средние значения по сети."
			result.CampaignsExcluded = excluded
			return emitResult(&result, formGoalID, callGoalID, targetFormOverride, targetCallOverride)
		}

		// Step 6: Backward-compat weighted means over ALL fetched benchmarks.
		avgForm, avgCall := computeLegacyAverages(benchmarks)
		result.AvgFormCPA = avgForm
		result.AvgCallCPA = avgCall

		// Step 7: Apply campaign-level filter (clicks, cost).
		var campaignLevelKept []conversionBenchmark
		for _, b := range benchmarks {
			if reason := campaignLevelExcludeReason(b, minClicks, minCost); reason != "" {
				excluded = append(excluded, excludedCampaign{b.CampaignID, b.CampaignName, reason})
			} else {
				campaignLevelKept = append(campaignLevelKept, b)
			}
		}

		// Step 8: Compute per-goal robust stats with min_conv filter; relax if both empty.
		effMinConv := minConv
		formStats := computeGoalStats(campaignLevelKept, goalForm, effMinConv)
		callStats := computeGoalStats(campaignLevelKept, goalCall, effMinConv)
		if formStats == nil && callStats == nil && len(campaignLevelKept) > 0 && minConv > 1 {
			effMinConv = 1
			formStats = computeGoalStats(campaignLevelKept, goalForm, effMinConv)
			callStats = computeGoalStats(campaignLevelKept, goalCall, effMinConv)
			result.FilterRelaxed = true
			result.RelaxedReason = fmt.Sprintf("min_conversions relaxed %d → 1 (все кампании города были отсечены по объёму)", minConv)
			result.FilterApplied.MinConversions = 1
		}

		// Step 9: Partition campaign-level kept into included (has conv for ≥1 goal) vs excluded.
		for _, b := range campaignLevelKept {
			if b.FormConv >= effMinConv || b.CallConv >= effMinConv {
				result.CampaignsIncluded = append(result.CampaignsIncluded, b)
			} else {
				excluded = append(excluded, excludedCampaign{
					b.CampaignID, b.CampaignName,
					fmt.Sprintf("form_conv=%d, call_conv=%d — оба < min_conversions=%d", b.FormConv, b.CallConv, effMinConv),
				})
			}
		}
		result.CampaignsExcluded = excluded

		// Step 10: No data at all for any goal → fall back to network average.
		if formStats == nil && callStats == nil {
			applyNetworkFallback(&result, inputTheme, tier, len(campaigns), len(excluded), "no_qualifying_campaigns")
			result.Note = "Ни одна кампания не прошла фильтр. Используются средние значения по сети (theme+tier)."
			// Trend still tries — may return insufficient_data.
			result.Trend7d = compute7dTrend(ctx, client, token, clientLogin, kept, goalIDs, formGoalID, callGoalID, 0, 0)
			return emitResult(&result, formGoalID, callGoalID, targetFormOverride, targetCallOverride)
		}

		result.Source = "campaigns"
		result.Statistics.Form = formStats
		result.Statistics.Call = callStats

		// Step 11: Populate benchmark CPAs (robust_mean), with cross-goal estimation when one is missing.
		// Priority: real campaign data > 2× within-city ratio heuristic > network table fallback.
		// Rationale: form/call CPA correlate strongly within a single city's market, so the 2× heuristic
		// applied to a real call_cpa is a tighter estimate than a generic network[theme/tier] average.
		mixed := false
		if formStats != nil {
			result.BenchmarkFormCPA = formStats.RobustMean
		}
		if callStats != nil {
			result.BenchmarkCallCPA = callStats.RobustMean
		}
		if result.BenchmarkFormCPA == 0 && result.BenchmarkCallCPA > 0 {
			est := math.Round(result.BenchmarkCallCPA * 2)
			result.BenchmarkFormCPA = est
			result.Note = appendNote(result.Note, fmt.Sprintf("Нет конверсий по форме за период. CPA формы оценён как 2× CPA звонка = %.0f₽ (внутригородское соотношение).", est))
			mixed = true
		} else if result.BenchmarkCallCPA == 0 && result.BenchmarkFormCPA > 0 {
			est := math.Round(result.BenchmarkFormCPA / 2)
			result.BenchmarkCallCPA = est
			result.Note = appendNote(result.Note, fmt.Sprintf("Нет конверсий по звонкам за период. CPA звонка оценён как 0.5× CPA формы = %.0f₽ (внутригородское соотношение).", est))
			mixed = true
		}
		if mixed {
			result.Source = "mixed"
		}

		// Step 12: Target = override ?? benchmark.
		result.TargetFormCPA = result.BenchmarkFormCPA
		result.TargetCallCPA = result.BenchmarkCallCPA
		if targetFormOverride > 0 {
			result.TargetFormCPA = targetFormOverride
		}
		if targetCallOverride > 0 {
			result.TargetCallCPA = targetCallOverride
		}

		// Step 13: Ratio (on benchmark).
		if result.BenchmarkFormCPA > 0 && result.BenchmarkCallCPA > 0 {
			result.Ratio = math.Round(result.BenchmarkFormCPA/result.BenchmarkCallCPA*10) / 10
		}

		// Step 14: Overall confidence.
		result.Confidence = overallConfidence(formStats, callStats)
		result.ConfidenceReason = buildConfidenceReason(formStats, callStats, len(result.CampaignsIncluded))

		// Step 15: Per-theme breakdown when no input theme.
		if inputTheme == "" && len(result.CampaignsIncluded) > 0 {
			result.BreakdownByTheme = computeBreakdown(result.CampaignsIncluded, effMinConv)
		}

		// Step 16: 7-day trend (always, on the same kept set).
		// Compare 7d CPA against the FINAL benchmark (which may include 2× heuristic for one missing goal),
		// not raw formStats.RobustMean — that way trend reflects the value users actually act on.
		result.Trend7d = compute7dTrend(ctx, client, token, clientLogin, kept, goalIDs, formGoalID, callGoalID, result.BenchmarkFormCPA, result.BenchmarkCallCPA)

		return emitResult(&result, formGoalID, callGoalID, targetFormOverride, targetCallOverride)
	})
}

// ===== emitResult / fallback / helpers =====

// emitResult builds RecommendedValue + priority_goals_json and marshals the final result.
func emitResult(result *conversionValuesResult, formGoalID, callGoalID int64, formOverride, callOverride float64) (*mcp.CallToolResult, error) {
	source := "robust_mean"
	if result.Source == "network_average" {
		source = "network_average"
	} else if result.Source == "mixed" {
		source = "robust_with_2x_heuristic"
	}
	if formOverride > 0 || callOverride > 0 {
		source = "target_override"
	}

	if result.TargetFormCPA == 0 {
		result.TargetFormCPA = result.BenchmarkFormCPA
	}
	if result.TargetCallCPA == 0 {
		result.TargetCallCPA = result.BenchmarkCallCPA
	}

	formValue := int64(result.TargetFormCPA)
	if formValue < 1 {
		formValue = 1
	}
	callValue := int64(result.TargetCallCPA)
	if callValue < 1 {
		callValue = 1
	}
	result.RecommendedValue = recommendedValue{
		FormValue: formValue,
		CallValue: callValue,
		Source:    source,
	}

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
}

// applyNetworkFallback fills CPAs from network table (theme+tier) when we have no usable campaign data.
func applyNetworkFallback(result *conversionValuesResult, theme, tier string, totalCampaigns, excludedCount int, reason string) {
	form, call, src := networkFallback(theme, tier)
	result.Source = "network_average"
	result.AvgFormCPA = form
	result.AvgCallCPA = call
	result.BenchmarkFormCPA = form
	result.BenchmarkCallCPA = call
	if call > 0 {
		result.Ratio = math.Round(form/call*10) / 10
	}
	result.Confidence = "low"
	result.ConfidenceReason = fmt.Sprintf("%s fallback (%s)", src, reason)
	if result.Note == "" {
		if totalCampaigns == 0 {
			result.Note = fmt.Sprintf("Нет активных кампаний. Использован %s.", src)
		} else {
			result.Note = fmt.Sprintf("Все %d кампаний отфильтрованы (%d в excluded). Использован %s.", totalCampaigns, excludedCount, src)
		}
	}
}

func appendNote(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + " " + addition
}

// computeLegacyAverages returns weighted-mean CPA across ALL fetched benchmarks (pre-filter).
func computeLegacyAverages(benchmarks []conversionBenchmark) (avgForm, avgCall float64) {
	var formCost, callCost float64
	var formConv, callConv int
	for _, b := range benchmarks {
		if b.FormConv > 0 {
			formCost += b.Cost
			formConv += b.FormConv
		}
		if b.CallConv > 0 {
			callCost += b.Cost
			callConv += b.CallConv
		}
	}
	if formConv > 0 {
		avgForm = math.Round(formCost / float64(formConv))
	}
	if callConv > 0 {
		avgCall = math.Round(callCost / float64(callConv))
	}
	return avgForm, avgCall
}

// moderationRedFlags — substrings in StatusClarification that signal real serving/moderation issues.
// "Идут показы" and variants are NORMAL running state, so a naive non-empty check rejects everything.
var moderationRedFlags = []string{
	"отклон",      // отклонено модератором
	"закончил",    // закончился период / лимит
	"запрещ",      // запрещено
	"не принят",   // не принято
	"ошибк",       // ошибка
	"приостановл", // приостановлено
}

// learningExclusionReason returns "" if the campaign should be kept, or a human reason string.
func learningExclusionReason(c campaignMeta, today time.Time, learningDays int) string {
	if c.StatusClarification != "" {
		sc := strings.ToLower(c.StatusClarification)
		for _, flag := range moderationRedFlags {
			if strings.Contains(sc, flag) {
				return fmt.Sprintf("moderation: %s", truncate(c.StatusClarification, 120))
			}
		}
	}
	if c.StartDate == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", c.StartDate)
	if err != nil {
		return ""
	}
	age := int(today.Sub(t).Hours() / 24)
	if age < learningDays {
		return fmt.Sprintf("learning_period: запущена %d дн. назад (< %d)", age, learningDays)
	}
	return ""
}

// campaignLevelExcludeReason returns "" if the benchmark passes clicks/cost thresholds.
func campaignLevelExcludeReason(b conversionBenchmark, minClicks int, minCost float64) string {
	if b.Clicks < minClicks {
		return fmt.Sprintf("clicks=%d < min_clicks=%d", b.Clicks, minClicks)
	}
	if b.Cost < minCost {
		return fmt.Sprintf("cost=%.0f < min_cost=%.0f", b.Cost, minCost)
	}
	return ""
}

// ===== goal kind + stats =====

type goalKind int

const (
	goalForm goalKind = iota
	goalCall
)

func cpaFor(b conversionBenchmark, g goalKind) float64 {
	if g == goalForm {
		return b.FormCPA
	}
	return b.CallCPA
}

func convFor(b conversionBenchmark, g goalKind) int {
	if g == goalForm {
		return b.FormConv
	}
	return b.CallConv
}

// computeGoalStats returns robust stats for a single goal, or nil if no campaign has enough data.
func computeGoalStats(benchmarks []conversionBenchmark, g goalKind, minConv int) *goalStats {
	var included []conversionBenchmark
	for _, b := range benchmarks {
		if convFor(b, g) >= minConv {
			included = append(included, b)
		}
	}
	if len(included) == 0 {
		return nil
	}

	stats := &goalStats{IncludedCount: len(included)}
	for _, b := range included {
		stats.TotalCost += b.Cost
		stats.TotalConversions += convFor(b, g)
	}
	if stats.TotalConversions == 0 {
		return nil
	}
	stats.WeightedMean = math.Round(stats.TotalCost / float64(stats.TotalConversions))

	cpas := make([]float64, len(included))
	for i, b := range included {
		cpas[i] = cpaFor(b, g)
	}
	sort.Float64s(cpas)
	stats.P25 = math.Round(quantile(cpas, 0.25))
	stats.P50 = math.Round(quantile(cpas, 0.50))
	stats.P75 = math.Round(quantile(cpas, 0.75))
	stats.IQR = stats.P75 - stats.P25

	if len(included) < 5 {
		stats.RobustMean = stats.WeightedMean
		stats.OutlierNote = fmt.Sprintf("n=%d < 5, IQR-отсечка пропущена", len(included))
	} else {
		lower := stats.P25 - 1.5*stats.IQR
		upper := stats.P75 + 1.5*stats.IQR
		var rCost float64
		var rConv int
		rN := 0
		for _, b := range included {
			cpa := cpaFor(b, g)
			if cpa >= lower && cpa <= upper {
				rCost += b.Cost
				rConv += convFor(b, g)
				rN++
			}
		}
		stats.OutliersRemoved = len(included) - rN
		if rConv > 0 {
			stats.RobustMean = math.Round(rCost / float64(rConv))
		} else {
			stats.RobustMean = stats.WeightedMean
		}
	}

	stats.Confidence = confidenceFor(stats.TotalConversions, stats.IncludedCount)
	return stats
}

// confidenceFor grades a single goal's data volume.
func confidenceFor(totalConv, includedCount int) string {
	if totalConv >= 50 && includedCount >= 3 {
		return "high"
	}
	if totalConv >= 15 && includedCount >= 2 {
		return "medium"
	}
	return "low"
}

// overallConfidence = min of available per-goal confidences.
func overallConfidence(form, call *goalStats) string {
	rank := map[string]int{"low": 0, "medium": 1, "high": 2}
	names := []string{"low", "medium", "high"}
	minLevel := 2
	seen := false
	if form != nil {
		if l, ok := rank[form.Confidence]; ok {
			minLevel = l
			seen = true
		}
	}
	if call != nil {
		if l, ok := rank[call.Confidence]; ok {
			if !seen || l < minLevel {
				minLevel = l
			}
			seen = true
		}
	}
	if !seen {
		return "low"
	}
	return names[minLevel]
}

func buildConfidenceReason(form, call *goalStats, campaignsIncluded int) string {
	var fConv, cConv int
	if form != nil {
		fConv = form.TotalConversions
	}
	if call != nil {
		cConv = call.TotalConversions
	}
	return fmt.Sprintf("%d form + %d call conv across %d campaigns", fConv, cConv, campaignsIncluded)
}

// ===== Per-theme breakdown (FR-4) =====

// computeBreakdown groups included campaigns by theme and computes simple weighted CPA per theme.
// Uses weighted_mean (not robust_mean) since per-theme n is usually too small for IQR.
// Reports ACTUAL conversion counts (not gated by min_conv) — the breakdown is informational,
// callers can see thin themes via low conversion totals.
func computeBreakdown(included []conversionBenchmark, _ int) map[string]*themeBreakdown {
	groups := make(map[string][]conversionBenchmark)
	for _, b := range included {
		t := b.Theme
		if t == "" {
			t = "unknown"
		}
		groups[t] = append(groups[t], b)
	}
	if len(groups) <= 1 {
		// Only one theme — no breakdown value-add.
		return nil
	}
	out := make(map[string]*themeBreakdown, len(groups))
	for theme, items := range groups {
		tb := &themeBreakdown{Theme: theme, NCampaigns: len(items)}
		var formCost, callCost float64
		for _, b := range items {
			tb.TotalCost += b.Cost
			if b.FormConv > 0 {
				formCost += b.Cost
				tb.FormConversions += b.FormConv
			}
			if b.CallConv > 0 {
				callCost += b.Cost
				tb.CallConversions += b.CallConv
			}
		}
		if tb.FormConversions > 0 {
			tb.FormCPA = math.Round(formCost / float64(tb.FormConversions))
		}
		if tb.CallConversions > 0 {
			tb.CallCPA = math.Round(callCost / float64(tb.CallConversions))
		}
		out[theme] = tb
	}
	return out
}

// ===== 7-day trend (FR-9) =====

func compute7dTrend(ctx context.Context, client *Client, token, clientLogin string, kept []campaignMeta,
	goalIDs []string, formGoalID, callGoalID int64, periodFormCPA, periodCallCPA float64,
) *trendData {
	if len(kept) == 0 {
		return &trendData{Signal: "no_data", InsufficientData: true}
	}
	now := time.Now()
	dateTo := now.Format("2006-01-02")
	dateFrom := now.AddDate(0, 0, -7).Format("2006-01-02")
	bench, _, err := collectBenchmarks(ctx, client, token, clientLogin, kept, dateFrom, dateTo, goalIDs, formGoalID, callGoalID)
	if err != nil {
		return &trendData{Signal: "error", InsufficientData: true}
	}
	td := &trendData{}
	var formCost, callCost float64
	for _, b := range bench {
		if b.FormConv > 0 {
			formCost += b.Cost
			td.FormConv7d += b.FormConv
		}
		if b.CallConv > 0 {
			callCost += b.Cost
			td.CallConv7d += b.CallConv
		}
	}
	if td.FormConv7d > 0 {
		td.FormCPA7d = math.Round(formCost / float64(td.FormConv7d))
	}
	if td.CallConv7d > 0 {
		td.CallCPA7d = math.Round(callCost / float64(td.CallConv7d))
	}

	// Insufficient if both goals < 3 conv in 7d.
	if td.FormConv7d < 3 && td.CallConv7d < 3 {
		td.InsufficientData = true
		td.Signal = "insufficient_data"
		return td
	}

	formSignal := signalFromDelta(td.FormCPA7d, periodFormCPA, td.FormConv7d)
	callSignal := signalFromDelta(td.CallCPA7d, periodCallCPA, td.CallConv7d)
	td.FormVsPeriod = pctDelta(td.FormCPA7d, periodFormCPA)
	td.CallVsPeriod = pctDelta(td.CallCPA7d, periodCallCPA)

	// Combined: degrading if any degrades, improving if both improve, else stable.
	switch {
	case formSignal == "degrading" || callSignal == "degrading":
		td.Signal = "degrading"
	case formSignal == "improving" && (callSignal == "improving" || callSignal == ""):
		td.Signal = "improving"
	case callSignal == "improving" && formSignal == "":
		td.Signal = "improving"
	default:
		td.Signal = "stable"
	}
	return td
}

func signalFromDelta(cpa7d, cpaPeriod float64, conv7d int) string {
	if cpaPeriod <= 0 || cpa7d <= 0 || conv7d < 3 {
		return ""
	}
	delta := (cpa7d - cpaPeriod) / cpaPeriod
	switch {
	case delta > 0.15:
		return "degrading"
	case delta < -0.15:
		return "improving"
	default:
		return "stable"
	}
}

func pctDelta(a, b float64) string {
	if b <= 0 || a <= 0 {
		return ""
	}
	pct := (a - b) / b * 100
	if pct >= 0 {
		return fmt.Sprintf("+%.0f%%", pct)
	}
	return fmt.Sprintf("%.0f%%", pct)
}

// ===== Misc =====

func quantile(sorted []float64, q float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	pos := q * float64(n-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	frac := pos - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ===== Yandex API calls =====

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
				callID = g.ID
				callName = g.Name
				callType = "received_real_calls"
			case "all_calls":
				if callID == 0 {
					callID = g.ID
					callName = g.Name
					callType = "all_calls"
				}
			}
		}
	}

	return formID, formName, callID, callName, callType, nil
}

// getActiveCampaigns fetches ON campaigns with fields needed for learning/moderation filter.
func getActiveCampaigns(ctx context.Context, client *Client, token, clientLogin string) ([]campaignMeta, error) {
	params := map[string]any{
		"SelectionCriteria": map[string]any{
			"States": []string{"ON"},
		},
		"FieldNames": []string{"Id", "Name", "StartDate", "StatusClarification"},
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
			ID                  int64  `json:"Id"`
			Name                string `json:"Name"`
			StartDate           string `json:"StartDate"`
			StatusClarification string `json:"StatusClarification"`
		} `json:"Campaigns"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("parse campaigns: %w", err)
	}

	out := make([]campaignMeta, len(resp.Campaigns))
	for i, c := range resp.Campaigns {
		out[i] = campaignMeta{
			ID:                  c.ID,
			Name:                c.Name,
			StartDate:           c.StartDate,
			StatusClarification: c.StatusClarification,
		}
	}
	return out, nil
}

// collectBenchmarks gathers stats for each campaign and parses TSV into benchmarks.
// Returns benchmarks, cost_attribution_label, error.
// FR-1: requests CostPerConversion column; if Yandex returns per-goal values, uses them
// as authoritative per-campaign per-goal CPA. Else falls back to Cost/Conversions and tags as "full_campaign_cost".
func collectBenchmarks(ctx context.Context, client *Client, token, clientLogin string,
	campaigns []campaignMeta, dateFrom, dateTo string, goalIDs []string, formGoalID, callGoalID int64,
) ([]conversionBenchmark, string, error) {

	campIDs := make([]string, len(campaigns))
	campNames := make(map[int64]string, len(campaigns))
	for i, c := range campaigns {
		campIDs[i] = strconv.FormatInt(c.ID, 10)
		campNames[c.ID] = c.Name
	}

	// FR-1: include CostPerConversion to attempt per-goal cost attribution.
	fieldNames := []string{"CampaignId", "Cost", "Clicks", "Conversions", "CostPerConversion"}

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
		return nil, "", err
	}

	bench, attr := parseBenchmarkTSV(tsv, campNames, formGoalID, callGoalID)
	return bench, attr, nil
}

// parseBenchmarkTSV parses TSV report output into conversionBenchmark slice and attribution label.
func parseBenchmarkTSV(tsv string, campNames map[int64]string, formGoalID, callGoalID int64) ([]conversionBenchmark, string) {
	lines := strings.Split(strings.TrimSpace(tsv), "\n")
	if len(lines) < 2 {
		return nil, "no_data"
	}

	headers := strings.Split(lines[0], "\t")
	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[h] = i
	}

	formConvCol := -1
	callConvCol := -1
	formCPCCol := -1 // CostPerConversion (form goal)
	callCPCCol := -1
	formGoalStr := strconv.FormatInt(formGoalID, 10)
	callGoalStr := strconv.FormatInt(callGoalID, 10)

	for i, h := range headers {
		// Conversions and CostPerConversion columns include the goal id in parentheses.
		// Order matters: check CostPerConversion first since it contains "Conversion" substring.
		if strings.Contains(h, "CostPerConversion") {
			if strings.Contains(h, formGoalStr) {
				formCPCCol = i
			} else if callGoalID > 0 && strings.Contains(h, callGoalStr) {
				callCPCCol = i
			}
			continue
		}
		if strings.Contains(h, "Conversions") {
			if strings.Contains(h, formGoalStr) {
				formConvCol = i
			} else if callGoalID > 0 && strings.Contains(h, callGoalStr) {
				callConvCol = i
			}
		}
	}

	// Determine attribution model. If CostPerConversion columns exist and have non-zero values,
	// trust Yandex's per-goal computation; else fall back to Cost / Conversions.
	usedYandexAttribution := false

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

		// Per-goal CPA: prefer Yandex's CostPerConversion column when present and positive.
		if formConv > 0 {
			if formCPCCol >= 0 && formCPCCol < len(cols) {
				if v, err := strconv.ParseFloat(strings.TrimSpace(cols[formCPCCol]), 64); err == nil && v > 0 {
					b.FormCPA = math.Round(v)
					usedYandexAttribution = true
				} else {
					b.FormCPA = math.Round(cost / float64(formConv))
				}
			} else {
				b.FormCPA = math.Round(cost / float64(formConv))
			}
		}
		if callConv > 0 {
			if callCPCCol >= 0 && callCPCCol < len(cols) {
				if v, err := strconv.ParseFloat(strings.TrimSpace(cols[callCPCCol]), 64); err == nil && v > 0 {
					b.CallCPA = math.Round(v)
					usedYandexAttribution = true
				} else {
					b.CallCPA = math.Round(cost / float64(callConv))
				}
			} else {
				b.CallCPA = math.Round(cost / float64(callConv))
			}
		}

		benchmarks = append(benchmarks, b)
	}

	attribution := "full_campaign_cost"
	if usedYandexAttribution {
		attribution = "yandex_per_goal_cost_per_conversion"
	}
	return benchmarks, attribution
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
