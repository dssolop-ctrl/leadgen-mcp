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
}

func registerGetDynamicAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_dynamic_ad_targets",
		mcp.WithDescription("Получить условия нацеливания для динамических объявлений."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
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
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
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
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
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
