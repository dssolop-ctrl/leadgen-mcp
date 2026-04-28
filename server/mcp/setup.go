package mcpsetup

import (
	"log/slog"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/direct"
	"github.com/leadgen-mcp/server/platform/filters"
	"github.com/leadgen-mcp/server/platform/history"
	"github.com/leadgen-mcp/server/platform/imagegen"
	"github.com/leadgen-mcp/server/platform/metrika"
	"github.com/leadgen-mcp/server/platform/vk"
	"github.com/leadgen-mcp/server/platform/wordstat"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates and configures the MCP server with all tools registered.
func NewServer(
	resolver *auth.AccountResolver,
	logger *slog.Logger,
	filterStore *filters.Store,
	historyStore *history.Store,
	imgClient *imagegen.Client,
	previewDir string,
) *server.MCPServer {
	s := server.NewMCPServer(
		"leadgen-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	// Shared Metrika client (used by both Metrika tools and Direct benchmarks)
	metrClient := metrika.NewClient(logger)

	// Yandex Direct (33 tools + benchmarks + reference lookups + РСЯ image upload)
	direct.RegisterTools(s, resolver, logger)
	direct.RegisterBenchmarkTools(s, direct.NewClient(logger), metrClient, resolver)
	direct.RegisterReferenceTools(s)

	// Yandex Metrika (11 tools)
	metrika.RegisterToolsWithClient(s, metrClient, resolver)

	// Forecast (1 tool) — прогноз spend/clicks/conversions по dailyстатам кампании.
	direct.RegisterForecastTools(s, direct.NewClient(logger), resolver)

	// Summarize (3 tools) — компактные read-back-ы для оптимизации/аудита:
	// summarize_search_queries / summarize_ads_performance / summarize_campaign_snapshot.
	// Возвращают top-N + агрегаты вместо сырого TSV — экономия ~70-90% контекста.
	direct.RegisterSummarizeTools(s, direct.NewClient(logger), resolver)

	// Yandex Wordstat (5 tools)
	wordstat.RegisterTools(s, resolver, logger)

	// VK Ads (30 tools)
	vk.RegisterTools(s, resolver, logger)

	// Site filters — SQLite-backed landing URL builder (3 tools)
	if filterStore != nil {
		filters.RegisterTools(s, filterStore)
	}

	// Centralized change history — events + daily summaries (4 tools)
	if historyStore != nil {
		history.RegisterTools(s, historyStore)
	}

	// Image generation via OpenRouter for RSYA banners (2 tools).
	// Registered unconditionally: without a key the tools error out on call with a clear message.
	if imgClient != nil {
		imagegen.RegisterTools(s, imgClient, previewDir)
	}

	return s
}
