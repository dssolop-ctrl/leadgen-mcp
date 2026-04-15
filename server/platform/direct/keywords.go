package direct

import (
	"context"
	"encoding/json"
	"fmt"

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
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
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
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
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
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
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
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
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
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
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
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
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
		mcp.WithDescription("Обновить категории автотаргетинга. Находит ---autotargeting в группе, удаляет и пересоздаёт с нужными категориями. Рекомендация: EXACT+ALTERNATIVE=YES, остальные=NO."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("adgroup_id", mcp.Description("ID группы"), mcp.Required()),
		mcp.WithString("exact", mcp.Description("Целевые: YES или NO (умолч YES)")),
		mcp.WithString("alternative", mcp.Description("Узкие: YES или NO (умолч YES)")),
		mcp.WithString("broader", mcp.Description("Широкие: YES или NO (умолч NO)")),
		mcp.WithString("accessory", mcp.Description("Сопутствующие: YES или NO (умолч NO)")),
		mcp.WithString("competitor", mcp.Description("Конкурентные: YES или NO (умолч NO)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		adGroupID := common.GetInt(req, "adgroup_id")

		// Build categories array for Keywords.add
		type catItem struct {
			Category string `json:"Category"`
			Value    string `json:"Value"`
		}
		cats := []catItem{}
		addCat := func(name, param, def string) {
			v := common.GetString(req, param)
			if v == "" {
				v = def
			}
			cats = append(cats, catItem{Category: name, Value: v})
		}
		addCat("EXACT", "exact", "YES")
		addCat("ALTERNATIVE", "alternative", "YES")
		addCat("BROADER", "broader", "NO")
		addCat("ACCESSORY", "accessory", "NO")
		addCat("COMPETITOR", "competitor", "NO")

		// Step 1: Find existing ---autotargeting keyword in the group
		getParams := map[string]any{
			"SelectionCriteria": map[string]any{
				"AdGroupIds": []int64{int64(adGroupID)},
			},
			"FieldNames": []string{"Id", "Keyword", "AdGroupId"},
		}
		getRaw, err := client.Call(ctx, token, "keywords", "get", getParams, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("get keywords: %v", err)), nil
		}

		// Parse to find ---autotargeting keyword ID
		var getResp struct {
			Result struct {
				Keywords []struct {
					Id      int64  `json:"Id"`
					Keyword string `json:"Keyword"`
				} `json:"Keywords"`
			} `json:"result"`
		}
		if err := json.Unmarshal(getRaw, &getResp); err != nil {
			return common.ErrorResult(fmt.Sprintf("parse keywords: %v", err)), nil
		}

		// Step 2: Delete existing autotargeting keyword if found
		for _, kw := range getResp.Result.Keywords {
			if kw.Keyword == "---autotargeting" {
				delParams := map[string]any{
					"SelectionCriteria": map[string]any{
						"Ids": []int64{kw.Id},
					},
				}
				_, _ = client.Call(ctx, token, "keywords", "delete", delParams, clientLogin)
				break
			}
		}

		// Step 3: Add autotargeting with desired categories
		addParams := map[string]any{
			"Keywords": []any{
				map[string]any{
					"Keyword":                  "---autotargeting",
					"AdGroupId":                adGroupID,
					"AutotargetingCategories":  cats,
				},
			},
		}

		raw, err := client.Call(ctx, token, "keywords", "add", addParams, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("add autotargeting: %v", err)), nil
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
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
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
