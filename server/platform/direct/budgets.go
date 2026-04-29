package direct

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// budgetTier holds default weekly budgets per stage for one (channel, tier) combo.
// Values in RUB, weekly.
type budgetTier struct {
	Test     int `json:"test"`
	Start    int `json:"start"`
	ScaleMin int `json:"scale_min"`
}

// budgetMatrix: channel -> tier -> stages.
// Source: PROJECTS.md (search) + .claude/skills/leadgen/references/rsya_defaults.md (rsya).
// VK approximated from search × 0.8 (lower CPM, but minimum daily floor).
var budgetMatrix = map[string]map[string]budgetTier{
	"search": {
		"tier_1": {Test: 8000, Start: 25000, ScaleMin: 80000},
		"tier_2": {Test: 5000, Start: 15000, ScaleMin: 50000},
		"tier_3": {Test: 3000, Start: 8000, ScaleMin: 25000},
	},
	"rsya": {
		"tier_1": {Test: 5000, Start: 18000, ScaleMin: 56000},
		"tier_2": {Test: 3500, Start: 10000, ScaleMin: 35000},
		"tier_3": {Test: 2500, Start: 6000, ScaleMin: 18000},
	},
	"vk": {
		"tier_1": {Test: 6000, Start: 20000, ScaleMin: 60000},
		"tier_2": {Test: 4000, Start: 12000, ScaleMin: 40000},
		"tier_3": {Test: 2500, Start: 7000, ScaleMin: 20000},
	},
}

// themeMultiplier adjusts budget by theme breadth. 1.0 default.
// Premium themes (commercial, new buildings) have higher CPA so need more budget;
// narrow themes (ипотека) usually have lower volume so smaller floor is fine.
var themeMultiplier = map[string]float64{
	"вторичка":     1.0,
	"новостройки":  1.15,
	"загородка":    0.9,
	"аренда":       0.7,
	"коммерческая": 1.3,
	"ипотека":      0.8,
	"агентство":    1.0,
	"бренд":        1.0,
	"hr":           0.6,
}

// minWeeklyFloor — hard minimum below which Yandex auto-strategies don't learn reliably.
var minWeeklyFloor = map[string]int{
	"search": 5000,
	"rsya":   3000,
	"vk":     2000,
}

// roundToStep rounds an int to the nearest multiple of step (step > 0).
func roundToStep(v int, step int) int {
	if step <= 0 {
		return v
	}
	return int(math.Round(float64(v)/float64(step))) * step
}

func registerGetDefaultBudgets(s *mcpserver.MCPServer) {
	tool := mcp.NewTool("get_default_budgets",
		mcp.WithDescription(
			"Дефолтные недельные бюджеты по уровням (test / start / scale_min) с учётом канала и tier-а города. "+
				"Возвращает три уровня + правила старта/смены, минимальный пол под автостратегии и "+
				"опционально расчёт от целевого CPA (target_cpa × 10 × 1.2). "+
				"Заменяет таблицу бюджетов из PROJECTS.md."),
		mcp.WithString("channel",
			mcp.Description("Канал размещения: search | rsya | vk"),
			mcp.Required()),
		mcp.WithString("tier",
			mcp.Description("Tier города: tier_1 (1M+ pop) | tier_2 (300K-1M) | tier_3 (<300K). Узнать через get_city_config(city).tier."),
			mcp.Required()),
		mcp.WithString("theme",
			mcp.Description("Тематика — для коэффициента (вторичка=1.0, новостройки=1.15, коммерческая=1.3, аренда=0.7, ипотека=0.8, hr=0.6). По умолчанию 1.0.")),
		mcp.WithNumber("target_cpa",
			mcp.Description("Опционально: целевой CPA в рублях (из get_conversion_values, целое число). Если задан — добавляется computed_weekly = target_cpa × 10 × 1.2, округлённый до 500₽.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		channel := strings.ToLower(strings.TrimSpace(common.GetString(req, "channel")))
		tier := strings.ToLower(strings.TrimSpace(common.GetString(req, "tier")))
		theme := strings.ToLower(strings.TrimSpace(common.GetString(req, "theme")))
		targetCPA := common.GetInt(req, "target_cpa")

		if channel == "" {
			return common.ErrorResult("Параметр channel обязателен. Допустимые: search, rsya, vk."), nil
		}
		if tier == "" {
			return common.ErrorResult("Параметр tier обязателен. Допустимые: tier_1, tier_2, tier_3."), nil
		}

		channelMap, ok := budgetMatrix[channel]
		if !ok {
			return common.ErrorResult(fmt.Sprintf("channel '%s' не поддерживается. Допустимые: search, rsya, vk.", channel)), nil
		}
		row, ok := channelMap[tier]
		if !ok {
			return common.ErrorResult(fmt.Sprintf("tier '%s' не поддерживается. Допустимые: tier_1, tier_2, tier_3.", tier)), nil
		}

		mult := 1.0
		if theme != "" {
			if m, found := themeMultiplier[theme]; found {
				mult = m
			}
		}

		applyMult := func(v int) int {
			return roundToStep(int(float64(v)*mult), 500)
		}

		floor := minWeeklyFloor[channel]

		out := map[string]any{
			"channel":             channel,
			"tier":                tier,
			"theme":               theme,
			"theme_multiplier":    mult,
			"currency":            "RUB",
			"period":              "weekly",
			"min_floor":           floor,
			"tiers": map[string]int{
				"test":      max(applyMult(row.Test), floor),
				"start":     max(applyMult(row.Start), floor),
				"scale_min": max(applyMult(row.ScaleMin), floor),
			},
			"rules": []string{
				"Старт — по уровню `start`. Можно сдвинуть ±30% от тематики/конкурентности.",
				"Тест (`test`) — минимально допустимый, при новой гипотезе или незнакомом городе.",
				"Масштаб (`scale_min`) — нижняя планка для масштабирования; рост свыше этого без потолка.",
				"Рост бюджета >30% за один шаг — только с явного подтверждения пользователя, лучше в чт-пт.",
				"Снижение бюджета — допустимо в любой момент при CPA в зоне CRITICAL (>1.5 × target).",
				"Перевод стратегии WB_MAXIMUM_CONVERSION_RATE → AVERAGE_CPA — после стабильных 10+ конверсий/нед в течение 2–3 недель.",
				fmt.Sprintf("Минимум %d ₽/нед — ниже автостратегии Директа/VK не обучаются стабильно.", floor),
			},
			"formula":         "weekly_budget = target_cpa × 10 × 1.2  (10 конверсий/нед × запас на обучение)",
			"theme_guidance": map[string]string{
				"new_city_no_history":      "test",
				"known_market_first_run":   "start",
				"profitable_scaling":       "scale_min",
				"narrow_theme_low_volume":  "снизить test/start на 20–30% от таблицы",
				"premium_theme_high_cpa":   "поднять start на 20–30% (повышенный CPA)",
			},
		}

		if targetCPA > 0 {
			computedRaw := float64(targetCPA) * 10 * 1.2
			computed := roundToStep(int(computedRaw), 500)
			if computed < floor {
				computed = floor
			}
			out["computed_from_target"] = map[string]any{
				"target_cpa":            targetCPA,
				"weekly_raw":            int(computedRaw),
				"weekly_rounded_500":    computed,
				"vs_table_start":        out["tiers"].(map[string]int)["start"],
				"recommendation":        "Старт = max(computed, table_start). Если разрыв > 30% — обсудить с пользователем перед apply.",
			}
		}

		return common.JSONResult(out), nil
	})
}
