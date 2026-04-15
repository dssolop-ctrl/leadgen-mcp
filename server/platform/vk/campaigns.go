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

// RegisterCampaignTools registers VK campaign MCP tools.
func RegisterCampaignTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerVKGetCampaigns(s, client, resolver)
	registerVKCreateCampaign(s, client, resolver)
	registerVKUpdateCampaign(s, client, resolver)
	registerVKManageCampaigns(s, client, resolver)
}

func registerVKGetCampaigns(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_campaigns",
		mcp.WithDescription("Получить список кампаний VK Ads."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("status", mcp.Description("Фильтр по статусу: active, blocked, deleted")),
		mcp.WithNumber("limit", mcp.Description("Лимит (по умолчанию 50)")),
		mcp.WithNumber("offset", mcp.Description("Смещение для пагинации")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := url.Values{}
		if st := common.GetString(req, "status"); st != "" {
			params.Set("status", st)
		}
		if limit := common.GetInt(req, "limit"); limit > 0 {
			params.Set("limit", fmt.Sprintf("%d", limit))
		}
		if offset := common.GetInt(req, "offset"); offset > 0 {
			params.Set("offset", fmt.Sprintf("%d", offset))
		}

		result, err := client.Get(ctx, token, "/ad_plans.json", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKCreateCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_campaign",
		mcp.WithDescription("Создать кампанию VK Ads. Бюджет в рублях (строка). Минимум 300₽/день. Нужна хотя бы 1 группа."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("name", mcp.Description("Название кампании"), mcp.Required()),
		mcp.WithString("objective", mcp.Description("Цель: site_conversions, traffic, reach, и т.д."), mcp.Required()),
		mcp.WithString("budget_limit_day", mcp.Description("Дневной бюджет в рублях (строка, мин. 300)")),
		mcp.WithString("budget_limit", mcp.Description("Общий бюджет в рублях (строка)")),
		mcp.WithString("start_date", mcp.Description("Начало (YYYY-MM-DD)")),
		mcp.WithString("end_date", mcp.Description("Конец (YYYY-MM-DD)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"name":      common.GetString(req, "name"),
			"objective": common.GetString(req, "objective"),
		}
		if bd := common.GetString(req, "budget_limit_day"); bd != "" {
			body["budget_limit_day"] = bd
		}
		if bl := common.GetString(req, "budget_limit"); bl != "" {
			body["budget_limit"] = bl
		}
		if sd := common.GetString(req, "start_date"); sd != "" {
			body["start_date"] = sd
		}
		if ed := common.GetString(req, "end_date"); ed != "" {
			body["end_date"] = ed
		}

		result, err := client.Post(ctx, token, "/ad_plans.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKUpdateCampaign(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_update_campaign",
		mcp.WithDescription("Обновить кампанию VK Ads."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название")),
		mcp.WithString("budget_limit_day", mcp.Description("Новый дневной бюджет")),
		mcp.WithString("budget_limit", mcp.Description("Новый общий бюджет")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		campaignID := common.GetInt(req, "campaign_id")
		body := map[string]any{}
		if n := common.GetString(req, "name"); n != "" {
			body["name"] = n
		}
		if bd := common.GetString(req, "budget_limit_day"); bd != "" {
			body["budget_limit_day"] = bd
		}
		if bl := common.GetString(req, "budget_limit"); bl != "" {
			body["budget_limit"] = bl
		}

		path := fmt.Sprintf("/ad_plans/%d.json", campaignID)
		result, err := client.Patch(ctx, token, path, body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKManageCampaigns(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_manage_campaigns",
		mcp.WithDescription("Массовое управление кампаниями VK: остановка, запуск, удаление. До 200 ID."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("status", mcp.Description("Статус: active, blocked, deleted"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		ids := common.GetStringSlice(req, "campaign_ids")
		status := common.GetString(req, "status")

		body := map[string]any{
			"ids":    ids,
			"status": status,
		}

		result, err := client.Post(ctx, token, "/ad_plans/mass_action.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}
