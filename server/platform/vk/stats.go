package vk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterStatsTools registers VK statistics tools.
func RegisterStatsTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerVKGetStatistics(s, client, resolver)
	registerVKGetGoalStatistics(s, client, resolver)
	registerVKGetProjection(s, client, resolver)
}

func registerVKGetStatistics(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_statistics",
		mcp.WithDescription("Статистика VK Ads: показы, клики, расход, конверсии."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("object_type", mcp.Description("Тип: ad_plan, ad_group, banner"), mcp.Required()),
		mcp.WithString("object_ids", mcp.Description("ID объектов через запятую"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("metrics", mcp.Description("Метрики: shows, clicks, spent, goals и т.д.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := url.Values{}
		params.Set("object_type", common.GetString(req, "object_type"))
		for _, id := range common.GetStringSlice(req, "object_ids") {
			params.Add("id", id)
		}
		params.Set("date_from", common.GetString(req, "date_from"))
		params.Set("date_to", common.GetString(req, "date_to"))
		if m := common.GetString(req, "metrics"); m != "" {
			params.Set("metrics", m)
		}

		// VK API v2: /statistics/{object_type}s/day.json
		objectType := common.GetString(req, "object_type")
		statsPath := fmt.Sprintf("/statistics/%ss/day.json", objectType)
		result, err := client.Get(ctx, token, statsPath, params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}

func registerVKGetGoalStatistics(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_goal_statistics",
		mcp.WithDescription("Статистика по конверсиям VK Ads."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("object_type", mcp.Description("Тип: ad_plan, ad_group, banner"), mcp.Required()),
		mcp.WithString("object_ids", mcp.Description("ID объектов через запятую"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("date_to", mcp.Description("Конец (YYYY-MM-DD)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := url.Values{}
		params.Set("object_type", common.GetString(req, "object_type"))
		for _, id := range common.GetStringSlice(req, "object_ids") {
			params.Add("id", id)
		}
		params.Set("date_from", common.GetString(req, "date_from"))
		params.Set("date_to", common.GetString(req, "date_to"))

		// VK API v2: /statistics/goals/{object_type}s/day.json
		objectType := common.GetString(req, "object_type")
		goalsPath := fmt.Sprintf("/statistics/goals/%ss/day.json", objectType)
		result, err := client.Get(ctx, token, goalsPath, params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}

func registerVKGetProjection(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_projection",
		mcp.WithDescription("Прогноз охвата VK Ads при разных ставках."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("ad_group_id", mcp.Description("ID группы для прогноза")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании")),
		mcp.WithString("budget", mcp.Description("Бюджет для прогноза")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := url.Values{}
		if agID := common.GetInt(req, "ad_group_id"); agID > 0 {
			params.Set("ad_group_id", fmt.Sprintf("%d", agID))
		}
		if cID := common.GetInt(req, "campaign_id"); cID > 0 {
			params.Set("ad_plan_id", fmt.Sprintf("%d", cID))
		}
		if b := common.GetString(req, "budget"); b != "" {
			params.Set("budget", b)
		}

		result, err := client.Get(ctx, token, "/projection.json", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}
