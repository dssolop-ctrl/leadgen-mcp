package metrika

import (
	"context"
	"fmt"
	"net/url"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterReportTools registers all Metrika report tools.
func RegisterReportTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetReport(s, client, resolver)
	registerGetReportByTime(s, client, resolver)
	registerGetReportComparison(s, client, resolver)
	registerGetDirectReport(s, client, resolver)
	registerGetTrafficSources(s, client, resolver)
	registerGetAudience(s, client, resolver)
	registerGetPopularPages(s, client, resolver)
	registerGetGoalsReport(s, client, resolver)
}

func registerGetReport(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_report",
		mcp.WithDescription("Универсальный отчёт Метрики. Любые метрики + группировки + фильтры."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("date1", mcp.Description("Начало периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date2", mcp.Description("Конец периода (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("metrics", mcp.Description("Метрики через запятую: ym:s:visits, ym:s:bounceRate, ym:s:pageDepth, ym:s:avgVisitDurationSeconds"), mcp.Required()),
		mcp.WithString("dimensions", mcp.Description("Измерения через запятую: ym:s:date, ym:s:UTMSource, ym:s:UTMCampaign")),
		mcp.WithString("filters", mcp.Description("Фильтры в формате API Метрики")),
		mcp.WithString("sort", mcp.Description("Сортировка: -ym:s:visits, ym:s:date")),
		mcp.WithNumber("limit", mcp.Description("Лимит строк (по умолчанию 100)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := buildReportParams(req)
		result, err := client.Get(ctx, token, "/stat/v1/data", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetReportByTime(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_report_bytime",
		mcp.WithDescription("Отчёт по временному ряду. Для графиков день/неделя/месяц."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("date1", mcp.Description("Начало (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date2", mcp.Description("Конец (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("metrics", mcp.Description("Метрики через запятую"), mcp.Required()),
		mcp.WithString("group", mcp.Description("Группировка: day, week, month (по умолчанию day)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := buildReportParams(req)
		if g := common.GetString(req, "group"); g != "" {
			params.Set("group", g)
		}

		result, err := client.Get(ctx, token, "/stat/v1/data/bytime", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetReportComparison(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_report_comparison",
		mcp.WithDescription("Сравнение двух периодов в Метрике."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("date1a", mcp.Description("Начало первого периода"), mcp.Required()),
		mcp.WithString("date2a", mcp.Description("Конец первого периода"), mcp.Required()),
		mcp.WithString("date1b", mcp.Description("Начало второго периода"), mcp.Required()),
		mcp.WithString("date2b", mcp.Description("Конец второго периода"), mcp.Required()),
		mcp.WithString("metrics", mcp.Description("Метрики через запятую"), mcp.Required()),
		mcp.WithString("dimensions", mcp.Description("Измерения через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		counterID := common.GetInt(req, "counter_id")
		params := url.Values{}
		params.Set("id", itoa(counterID))
		params.Set("date1_a", common.GetString(req, "date1a"))
		params.Set("date2_a", common.GetString(req, "date2a"))
		params.Set("date1_b", common.GetString(req, "date1b"))
		params.Set("date2_b", common.GetString(req, "date2b"))
		params.Set("metrics", common.GetString(req, "metrics"))
		if d := common.GetString(req, "dimensions"); d != "" {
			params.Set("dimensions", d)
		}
		params.Set("limit", "50")

		result, err := client.Get(ctx, token, "/stat/v1/data/comparison", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetDirectReport(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_direct_report",
		mcp.WithDescription("Отчёт по Яндекс Директу: визиты, отказы, глубина, конверсии после клика."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("date1", mcp.Description("Начало (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date2", mcp.Description("Конец (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("utm_campaign", mcp.Description("UTM-метка для фильтрации")),
		mcp.WithNumber("goal_id", mcp.Description("ID цели для конверсий")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		counterID := common.GetInt(req, "counter_id")
		metrics := "ym:s:visits,ym:s:bounceRate,ym:s:pageDepth,ym:s:avgVisitDurationSeconds"
		goalID := common.GetInt(req, "goal_id")
		if goalID > 0 {
			metrics += fmt.Sprintf(",ym:s:goal%dreaches", goalID)
		}

		params := url.Values{}
		params.Set("id", itoa(counterID))
		params.Set("date1", common.GetString(req, "date1"))
		params.Set("date2", common.GetString(req, "date2"))
		params.Set("metrics", metrics)
		params.Set("dimensions", "ym:s:UTMCampaign")

		if utm := common.GetString(req, "utm_campaign"); utm != "" {
			params.Set("filters", "ym:s:UTMCampaign=='"+utm+"'")
		} else {
			params.Set("filters", "ym:s:UTMSource=='yandex' AND ym:s:UTMMedium=='cpc'")
		}
		params.Set("limit", "50")

		result, err := client.Get(ctx, token, "/stat/v1/data", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetTrafficSources(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_traffic_sources",
		mcp.WithDescription("Источники трафика: Direct, organic, social, referral."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("date1", mcp.Description("Начало (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date2", mcp.Description("Конец (YYYY-MM-DD)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := url.Values{}
		params.Set("id", itoa(common.GetInt(req, "counter_id")))
		params.Set("date1", common.GetString(req, "date1"))
		params.Set("date2", common.GetString(req, "date2"))
		params.Set("metrics", "ym:s:visits,ym:s:bounceRate,ym:s:pageDepth")
		params.Set("dimensions", "ym:s:TrafficSource")
		params.Set("limit", "20")

		result, err := client.Get(ctx, token, "/stat/v1/data", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetAudience(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_audience",
		mcp.WithDescription("Аудитория: пол, возраст, устройства, гео."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("date1", mcp.Description("Начало (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date2", mcp.Description("Конец (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("dimension", mcp.Description("ym:s:gender, ym:s:ageInterval, ym:s:deviceCategory, ym:s:regionCity"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := url.Values{}
		params.Set("id", itoa(common.GetInt(req, "counter_id")))
		params.Set("date1", common.GetString(req, "date1"))
		params.Set("date2", common.GetString(req, "date2"))
		params.Set("metrics", "ym:s:visits,ym:s:bounceRate,ym:s:pageDepth")
		params.Set("dimensions", common.GetString(req, "dimension"))
		params.Set("limit", "50")

		result, err := client.Get(ctx, token, "/stat/v1/data", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetPopularPages(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_popular_pages",
		mcp.WithDescription("Популярные страницы сайта."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("date1", mcp.Description("Начало (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date2", mcp.Description("Конец (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Лимит (по умолчанию 20)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		limit := common.GetInt(req, "limit")
		if limit <= 0 {
			limit = 20
		}

		params := url.Values{}
		params.Set("id", itoa(common.GetInt(req, "counter_id")))
		params.Set("date1", common.GetString(req, "date1"))
		params.Set("date2", common.GetString(req, "date2"))
		params.Set("metrics", "ym:s:visits,ym:s:bounceRate,ym:s:pageDepth")
		params.Set("dimensions", "ym:s:startURL")
		params.Set("sort", "-ym:s:visits")
		params.Set("limit", itoa(limit))

		result, err := client.Get(ctx, token, "/stat/v1/data", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetGoalsReport(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_goals_report",
		mcp.WithDescription("Отчёт по конверсиям (воронка)."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("date1", mcp.Description("Начало (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date2", mcp.Description("Конец (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithNumber("goal_id", mcp.Description("ID цели (все цели если не указано)")),
		mcp.WithString("dimensions", mcp.Description("Измерения: ym:s:UTMSource, ym:s:TrafficSource")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		counterID := common.GetInt(req, "counter_id")
		goalID := common.GetInt(req, "goal_id")

		metrics := "ym:s:visits"
		if goalID > 0 {
			metrics += fmt.Sprintf(",ym:s:goal%dreaches,ym:s:goal%dconversionRate", goalID, goalID)
		}

		params := url.Values{}
		params.Set("id", itoa(counterID))
		params.Set("date1", common.GetString(req, "date1"))
		params.Set("date2", common.GetString(req, "date2"))
		params.Set("metrics", metrics)
		if d := common.GetString(req, "dimensions"); d != "" {
			params.Set("dimensions", d)
		} else {
			params.Set("dimensions", "ym:s:date")
		}
		params.Set("limit", "50")

		result, err := client.Get(ctx, token, "/stat/v1/data", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func buildReportParams(req mcp.CallToolRequest) url.Values {
	params := url.Values{}
	params.Set("id", itoa(common.GetInt(req, "counter_id")))
	params.Set("date1", common.GetString(req, "date1"))
	params.Set("date2", common.GetString(req, "date2"))
	params.Set("metrics", common.GetString(req, "metrics"))

	if d := common.GetString(req, "dimensions"); d != "" {
		params.Set("dimensions", d)
	}
	if f := common.GetString(req, "filters"); f != "" {
		params.Set("filters", f)
	}
	if s := common.GetString(req, "sort"); s != "" {
		params.Set("sort", s)
	}
	limit := common.GetInt(req, "limit")
	if limit <= 0 {
		limit = 50
	}
	params.Set("limit", itoa(limit))
	return params
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
