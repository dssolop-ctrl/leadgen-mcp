package direct

import (
	"context"
	"encoding/json"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterKeywordTools registers keyword and autotargeting MCP tools.
func RegisterKeywordTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetKeywords(s, client, resolver)
	registerAddKeywords(s, client, resolver)
	registerUpdateKeywords(s, client, resolver)
	registerManageKeywords(s, client, resolver)
	registerDeduplicateKeywords(s, client, resolver)
	registerGetAutotargeting(s, client, resolver)
	registerUpdateAutotargeting(s, client, resolver)
	registerManageAutotargeting(s, client, resolver)
}

func registerGetKeywords(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_keywords",
		mcp.WithDescription("Получить ключевые фразы. Фильтр по campaign_ids или adgroup_ids. По умолчанию первые 200."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую")),
		mcp.WithString("adgroup_ids", mcp.Description("ID групп через запятую")),
		mcp.WithString("field_names", mcp.Description("Поля: Id, Keyword, AdGroupId, CampaignId, Bid, State, Status")),
		mcp.WithNumber("limit", mcp.Description("Макс. ключевых слов в ответе (по умолчанию 200)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "campaign_ids")); len(ids) > 0 {
			criteria["CampaignIds"] = ids
		}
		if ids := parseIntSlice(common.GetStringSlice(req, "adgroup_ids")); len(ids) > 0 {
			criteria["AdGroupIds"] = ids
		}

		limit := common.GetInt(req, "limit")
		if limit <= 0 {
			limit = 200
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"Page":              map[string]any{"Limit": limit},
		}
		if fields := common.GetStringSlice(req, "field_names"); len(fields) > 0 {
			params["FieldNames"] = fields
		} else {
			params["FieldNames"] = []string{"Id", "Keyword", "AdGroupId", "State", "Status"}
		}

		raw, err := client.Call(ctx, token, "keywords", "get", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerAddKeywords(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_keywords",
		mcp.WithDescription("Добавить ключевые фразы в группу объявлений. До 1000 фраз за раз."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("adgroup_id", mcp.Description("ID группы объявлений"), mcp.Required()),
		mcp.WithString("keywords", mcp.Description("Ключевые фразы через запятую. Операторы: \"точная фраза\", +обязательное, -минус"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		adGroupID := common.GetInt(req, "adgroup_id")
		keywords := common.GetStringSlice(req, "keywords")

		var kwObjects []any
		for _, kw := range keywords {
			kwObjects = append(kwObjects, map[string]any{
				"Keyword":   kw,
				"AdGroupId": adGroupID,
			})
		}

		params := map[string]any{
			"Keywords": kwObjects,
		}

		raw, err := client.Call(ctx, token, "keywords", "add", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerUpdateKeywords(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_keywords",
		mcp.WithDescription("Обновить ключевые фразы: изменить текст фразы или назначить другую группу."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("keywords_json", mcp.Description("JSON массив: [{\"Id\":123,\"Keyword\":\"новая фраза\"}]"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var keywords any
		if err := json.Unmarshal([]byte(common.GetString(req, "keywords_json")), &keywords); err != nil {
			return common.ErrorResult("invalid keywords_json: " + err.Error()), nil
		}

		params := map[string]any{"Keywords": keywords}
		raw, err := client.Call(ctx, token, "keywords", "update", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerManageKeywords(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_keywords",
		mcp.WithDescription("Управление ключевыми фразами: остановка, возобновление, удаление."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("keyword_ids", mcp.Description("ID ключевых фраз через запятую"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: suspend, resume, delete"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "keyword_ids"))
		action := common.GetString(req, "action")

		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}

		raw, err := client.Call(ctx, token, "keywords", action, params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerDeduplicateKeywords(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("deduplicate_keywords",
		mcp.WithDescription("Проверить ключевые фразы на дублирование и каннибализацию между группами/кампаниями."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "campaign_ids"))

		// Get all keywords for given campaigns, then check for duplicates
		params := map[string]any{
			"SelectionCriteria": map[string]any{"CampaignIds": ids},
			"FieldNames":       []string{"Id", "Keyword", "AdGroupId", "CampaignId", "State"},
		}

		raw, err := client.Call(ctx, token, "keywords", "get", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetAutotargeting(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_autotargeting",
		mcp.WithDescription("Получить настройки автотаргетинга для групп объявлений. Категории: EXACT, ALTERNATIVE, BROADER, ACCESSORY, COMPETITOR."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("adgroup_ids", mcp.Description("ID групп через запятую"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "adgroup_ids"))

		// Autotargeting is the special keyword "---autotargeting" in Keywords service
		// We fetch all keywords for the given ad groups and let the client filter by "---autotargeting"
		params := map[string]any{
			"SelectionCriteria": map[string]any{
				"AdGroupIds": ids,
			},
			"FieldNames": []string{"Id", "Keyword", "AdGroupId", "State", "Status"},
		}

		raw, err := client.Call(ctx, token, "keywords", "get", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerUpdateAutotargeting(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_autotargeting",
		mcp.WithDescription("Обновить категории автотаргетинга. Рекомендация: EXACT=ON, ALTERNATIVE=ON, остальные=OFF."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("adgroup_id", mcp.Description("ID группы"), mcp.Required()),
		mcp.WithString("exact", mcp.Description("Целевые запросы: ON или OFF")),
		mcp.WithString("alternative", mcp.Description("Узкие запросы: ON или OFF")),
		mcp.WithString("broader", mcp.Description("Широкие запросы: ON или OFF")),
		mcp.WithString("accessory", mcp.Description("Сопутствующие: ON или OFF")),
		mcp.WithString("competitor", mcp.Description("Конкурентные: ON или OFF")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		adGroupID := common.GetInt(req, "adgroup_id")

		// Autotargeting categories are configured via AdGroups.update
		categories := map[string]any{}
		if v := common.GetString(req, "exact"); v != "" {
			categories["Exact"] = v
		}
		if v := common.GetString(req, "alternative"); v != "" {
			categories["Alternative"] = v
		}
		if v := common.GetString(req, "broader"); v != "" {
			categories["Broader"] = v
		}
		if v := common.GetString(req, "accessory"); v != "" {
			categories["Accessory"] = v
		}
		if v := common.GetString(req, "competitor"); v != "" {
			categories["Competitor"] = v
		}

		params := map[string]any{
			"AdGroups": []any{
				map[string]any{
					"Id":                       adGroupID,
					"AutotargetingCategories":  categories,
				},
			},
		}

		raw, err := client.Call(ctx, token, "adgroups", "update", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerManageAutotargeting(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_autotargeting",
		mcp.WithDescription("Управление автотаргетингом: suspend или resume ключевой фразы ---autotargeting."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("keyword_ids", mcp.Description("ID ключевых фраз автотаргетинга через запятую (получи через get_autotargeting)"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: suspend или resume"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "keyword_ids"))
		action := common.GetString(req, "action")
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "keywords", action, params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}
