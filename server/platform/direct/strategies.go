package direct

import (
	"context"
	"encoding/json"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterStrategyTools registers Strategies service MCP tools.
func RegisterStrategyTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetStrategies(s, client, resolver)
	registerAddStrategy(s, client, resolver)
	registerUpdateStrategy(s, client, resolver)
	registerManageStrategies(s, client, resolver)
}

func registerGetStrategies(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_strategies",
		mcp.WithDescription("Получить портфельные стратегии (пакетные стратегии для группы кампаний)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("strategy_ids", mcp.Description("ID стратегий через запятую (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "strategy_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "Name", "Type", "AttributionModel", "CounterIds", "StatusArchived"},
		}
		raw, err := client.Call(ctx, token, "strategies", "get", params, clientLogin)
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

func registerAddStrategy(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_strategy",
		mcp.WithDescription("Создать портфельную стратегию."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("strategy_json", mcp.Description("JSON стратегии: {\"Name\":\"...\",\"Type\":\"...\",\"CounterId\":123,...}"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var strategy any
		if err := json.Unmarshal([]byte(common.GetString(req, "strategy_json")), &strategy); err != nil {
			return common.ErrorResult("invalid strategy_json: " + err.Error()), nil
		}

		params := map[string]any{"Strategies": []any{strategy}}
		raw, err := client.Call(ctx, token, "strategies", "add", params, clientLogin)
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

func registerUpdateStrategy(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_strategy",
		mcp.WithDescription("Обновить портфельную стратегию."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("strategy_json", mcp.Description("JSON стратегии с Id: {\"Id\":123,\"Name\":\"...\"}"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var strategy any
		if err := json.Unmarshal([]byte(common.GetString(req, "strategy_json")), &strategy); err != nil {
			return common.ErrorResult("invalid strategy_json: " + err.Error()), nil
		}

		params := map[string]any{"Strategies": []any{strategy}}
		raw, err := client.Call(ctx, token, "strategies", "update", params, clientLogin)
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

func registerManageStrategies(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_strategies",
		mcp.WithDescription("Управление портфельными стратегиями: delete."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("strategy_ids", mcp.Description("ID стратегий через запятую"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: delete"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "strategy_ids"))
		action := common.GetString(req, "action")
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "strategies", action, params, clientLogin)
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
