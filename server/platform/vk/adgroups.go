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

// RegisterAdGroupTools registers VK ad group tools.
func RegisterAdGroupTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerVKGetAdGroups(s, client, resolver)
	registerVKCreateAdGroup(s, client, resolver)
	registerVKUpdateAdGroup(s, client, resolver)
	registerVKManageAdGroups(s, client, resolver)
}

func registerVKGetAdGroups(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_ad_groups",
		mcp.WithDescription("Получить группы объявлений VK Ads."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Лимит")),
		mcp.WithNumber("offset", mcp.Description("Смещение")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		campaignID := common.GetInt(req, "campaign_id")
		params := url.Values{}
		params.Set("ad_plan_id", fmt.Sprintf("%d", campaignID))
		if l := common.GetInt(req, "limit"); l > 0 {
			params.Set("limit", fmt.Sprintf("%d", l))
		}
		if o := common.GetInt(req, "offset"); o > 0 {
			params.Set("offset", fmt.Sprintf("%d", o))
		}

		result, err := client.Get(ctx, token, "/ad_groups.json", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKCreateAdGroup(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_ad_group",
		mcp.WithDescription("Создать группу объявлений VK Ads. Минимум бюджет 300₽/день. package_id 3858 = мультиформат."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Название группы"), mcp.Required()),
		mcp.WithNumber("package_id", mcp.Description("ID пакета размещения (3858 = мультиформат)"), mcp.Required()),
		mcp.WithString("budget_limit_day", mcp.Description("Дневной бюджет (строка, мин. 300)")),
		mcp.WithString("priced_goal_name", mcp.Description("Цель: condition:substr (напр. uss:example.com)")),
		mcp.WithNumber("priced_goal_source_id", mcp.Description("ID счётчика VK для цели")),
		mcp.WithString("region_ids", mcp.Description("ID регионов через запятую")),
		mcp.WithString("age_from", mcp.Description("Минимальный возраст")),
		mcp.WithString("age_to", mcp.Description("Максимальный возраст")),
		mcp.WithString("sex", mcp.Description("Пол: male, female или пусто (все)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"ad_plan_id": common.GetInt(req, "campaign_id"),
			"name":       common.GetString(req, "name"),
			"package_id": common.GetInt(req, "package_id"),
		}

		if bd := common.GetString(req, "budget_limit_day"); bd != "" {
			body["budget_limit_day"] = bd
		}

		// Priced goal
		if pgName := common.GetString(req, "priced_goal_name"); pgName != "" {
			pricedGoal := map[string]any{"name": pgName}
			if pgSrc := common.GetInt(req, "priced_goal_source_id"); pgSrc > 0 {
				pricedGoal["source_id"] = pgSrc
			}
			body["priced_goal"] = pricedGoal
		}

		// Targetings
		targetings := map[string]any{}
		if regions := common.GetStringSlice(req, "region_ids"); len(regions) > 0 {
			targetings["geo"] = map[string]any{"regions": regions}
		}
		if af := common.GetString(req, "age_from"); af != "" {
			targetings["age_from"] = af
		}
		if at := common.GetString(req, "age_to"); at != "" {
			targetings["age_to"] = at
		}
		if sex := common.GetString(req, "sex"); sex != "" {
			targetings["sex"] = sex
		}
		if len(targetings) > 0 {
			body["targetings"] = targetings
		}

		result, err := client.Post(ctx, token, "/ad_groups.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKUpdateAdGroup(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_update_ad_group",
		mcp.WithDescription("Обновить группу объявлений VK Ads: таргетинг, бюджет, название."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("ad_group_id", mcp.Description("ID группы"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название")),
		mcp.WithString("budget_limit_day", mcp.Description("Новый дневной бюджет")),
		mcp.WithString("region_ids", mcp.Description("Новые регионы через запятую")),
		mcp.WithString("age_from", mcp.Description("Мин. возраст")),
		mcp.WithString("age_to", mcp.Description("Макс. возраст")),
		mcp.WithString("sex", mcp.Description("Пол: male, female")),
		mcp.WithString("segment_ids", mcp.Description("ID сегментов через запятую")),
		mcp.WithString("interest_ids", mcp.Description("ID интересов через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		adGroupID := common.GetInt(req, "ad_group_id")
		body := map[string]any{}

		if n := common.GetString(req, "name"); n != "" {
			body["name"] = n
		}
		if bd := common.GetString(req, "budget_limit_day"); bd != "" {
			body["budget_limit_day"] = bd
		}

		targetings := map[string]any{}
		if regions := common.GetStringSlice(req, "region_ids"); len(regions) > 0 {
			targetings["geo"] = map[string]any{"regions": regions}
		}
		if af := common.GetString(req, "age_from"); af != "" {
			targetings["age_from"] = af
		}
		if at := common.GetString(req, "age_to"); at != "" {
			targetings["age_to"] = at
		}
		if sex := common.GetString(req, "sex"); sex != "" {
			targetings["sex"] = sex
		}
		if segs := common.GetStringSlice(req, "segment_ids"); len(segs) > 0 {
			targetings["segments"] = segs
		}
		if interests := common.GetStringSlice(req, "interest_ids"); len(interests) > 0 {
			targetings["interests"] = interests
		}
		if len(targetings) > 0 {
			body["targetings"] = targetings
		}

		path := fmt.Sprintf("/ad_groups/%d.json", adGroupID)
		result, err := client.Patch(ctx, token, path, body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKManageAdGroups(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_manage_ad_groups",
		mcp.WithDescription("Массовое управление группами VK Ads. До 200 ID."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("ad_group_ids", mcp.Description("ID групп через запятую"), mcp.Required()),
		mcp.WithString("status", mcp.Description("Статус: active, blocked"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"ids":    common.GetStringSlice(req, "ad_group_ids"),
			"status": common.GetString(req, "status"),
		}

		result, err := client.Post(ctx, token, "/ad_groups/mass_action.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}
