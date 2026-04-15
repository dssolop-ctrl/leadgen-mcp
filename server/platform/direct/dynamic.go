package direct

import (
	"context"
	"encoding/json"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterDynamicTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetDynamicAdTargets(s, client, resolver)
	registerAddDynamicAdTargets(s, client, resolver)
	registerManageDynamicAdTargets(s, client, resolver)
	registerSetDynamicAdTargetBids(s, client, resolver)
	registerGetDynamicFeedAdTargets(s, client, resolver)
	registerAddDynamicFeedAdTargets(s, client, resolver)
	registerManageFeedAdTargets(s, client, resolver)
	registerSetDynamicFeedAdTargetBids(s, client, resolver)
}

func registerGetDynamicAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_dynamic_ad_targets",
		mcp.WithDescription("Получить условия нацеливания для динамических объявлений."),
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
		params := map[string]any{
			"SelectionCriteria": map[string]any{"AdGroupIds": ids},
			"FieldNames":        []string{"Id", "AdGroupId", "Name", "State"},
		}
		raw, err := client.Call(ctx, token, "dynamictextadtargets", "get", params, clientLogin)
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

func registerAddDynamicAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_dynamic_ad_targets",
		mcp.WithDescription("Добавить условие нацеливания для динамических объявлений."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("adgroup_id", mcp.Description("ID группы"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Название условия"), mcp.Required()),
		mcp.WithString("conditions_json", mcp.Description("JSON условий: [{\"Operand\":\"PAGE_CONTENT\",\"Operator\":\"CONTAINS_ANY\",\"Arguments\":[\"купить\"]}]"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var conditions any
		if err := json.Unmarshal([]byte(common.GetString(req, "conditions_json")), &conditions); err != nil {
			return common.ErrorResult("invalid conditions_json: " + err.Error()), nil
		}

		params := map[string]any{
			"Webpages": []any{
				map[string]any{
					"AdGroupId":  common.GetInt(req, "adgroup_id"),
					"Name":       common.GetString(req, "name"),
					"Conditions": conditions,
				},
			},
		}
		raw, err := client.Call(ctx, token, "dynamictextadtargets", "add", params, clientLogin)
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

func registerManageDynamicAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_dynamic_ad_targets",
		mcp.WithDescription("Управление условиями динамических объявлений: suspend, resume, delete."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("target_ids", mcp.Description("ID условий через запятую"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: suspend, resume, delete"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "target_ids"))
		action := common.GetString(req, "action")
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "dynamictextadtargets", action, params, clientLogin)
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

func registerSetDynamicAdTargetBids(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_dynamic_ad_target_bids",
		mcp.WithDescription("Установить ставки для условий нацеливания динамических объявлений."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("bids_json", mcp.Description("JSON массив: [{\"WebpageId\":123,\"SearchBid\":5000000}]"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var bids any
		if err := json.Unmarshal([]byte(common.GetString(req, "bids_json")), &bids); err != nil {
			return common.ErrorResult("invalid bids_json: " + err.Error()), nil
		}

		params := map[string]any{"Bids": bids}
		raw, err := client.Call(ctx, token, "dynamictextadtargets", "setBids", params, clientLogin)
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

func registerGetDynamicFeedAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_dynamic_feed_ad_targets",
		mcp.WithDescription("Получить фильтры для динамических объявлений по фиду (каталогу товаров)."),
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
		params := map[string]any{
			"SelectionCriteria": map[string]any{"AdGroupIds": ids},
			"FieldNames":        []string{"Id", "AdGroupId", "Name", "State"},
		}
		raw, err := client.Call(ctx, token, "dynamicfeedadtargets", "get", params, clientLogin)
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

func registerAddDynamicFeedAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_dynamic_feed_ad_targets",
		mcp.WithDescription("Добавить фильтр для динамических объявлений по фиду."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("adgroup_id", mcp.Description("ID группы"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Название фильтра"), mcp.Required()),
		mcp.WithString("conditions_json", mcp.Description("JSON условий: [{\"Operand\":\"CATEGORY_ID\",\"Operator\":\"EQUALS_ANY\",\"Arguments\":[\"1\"]}]")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		target := map[string]any{
			"AdGroupId": common.GetInt(req, "adgroup_id"),
			"Name":      common.GetString(req, "name"),
		}
		if condJSON := common.GetString(req, "conditions_json"); condJSON != "" {
			var cond any
			if err := json.Unmarshal([]byte(condJSON), &cond); err != nil {
				return common.ErrorResult("invalid conditions_json: " + err.Error()), nil
			}
			target["Conditions"] = cond
		}

		params := map[string]any{"DynamicFeedAdTargets": []any{target}}
		raw, err := client.Call(ctx, token, "dynamicfeedadtargets", "add", params, clientLogin)
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

func registerManageFeedAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_feed_ad_targets",
		mcp.WithDescription("Управление фильтрами динамических объявлений по фиду: suspend, resume, delete."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("target_ids", mcp.Description("ID фильтров через запятую"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: suspend, resume, delete"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "target_ids"))
		action := common.GetString(req, "action")
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "dynamicfeedadtargets", action, params, clientLogin)
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

func registerSetDynamicFeedAdTargetBids(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_dynamic_feed_ad_target_bids",
		mcp.WithDescription("Установить ставки для фильтров динамических объявлений по фиду."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("bids_json", mcp.Description("JSON массив: [{\"DynamicFeedAdTargetId\":123,\"SearchBid\":5000000}]"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var bids any
		if err := json.Unmarshal([]byte(common.GetString(req, "bids_json")), &bids); err != nil {
			return common.ErrorResult("invalid bids_json: " + err.Error()), nil
		}

		params := map[string]any{"Bids": bids}
		raw, err := client.Call(ctx, token, "dynamicfeedadtargets", "setBids", params, clientLogin)
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
