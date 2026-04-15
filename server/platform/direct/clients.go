package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterClientTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetClient(s, client, resolver)
	registerUpdateClient(s, client, resolver)
	registerGetAccountBalance(s, client, resolver)
}

func registerGetClient(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_client",
		mcp.WithDescription("Получить информацию о клиенте: логин, ФИО, email, баланс, настройки уведомлений."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		params := map[string]any{
			"FieldNames": []string{"Login", "ClientId", "ClientInfo", "AccountQuality"},
		}
		raw, err := client.Call(ctx, token, "clients", "get", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.TextResult(string(result)), nil
	})
}

func registerUpdateClient(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_client",
		mcp.WithDescription("Обновить настройки клиента: email для уведомлений, имя, язык."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("client_info", mcp.Description("Новое имя/фамилия клиента")),
		mcp.WithString("email", mcp.Description("Новый email для уведомлений")),
		mcp.WithString("phone", mcp.Description("Новый телефон")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		cl := map[string]any{}
		if v := common.GetString(req, "client_info"); v != "" {
			cl["ClientInfo"] = v
		}
		if v := common.GetString(req, "email"); v != "" {
			cl["Notification"] = map[string]any{"Email": v}
		}
		if v := common.GetString(req, "phone"); v != "" {
			cl["Phone"] = v
		}

		params := map[string]any{"Clients": []any{cl}}
		raw, err := client.Call(ctx, token, "clients", "update", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.TextResult(string(result)), nil
	})
}

func registerGetAccountBalance(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_account_balance",
		mcp.WithDescription("Финансы аккаунта: овердрафт, бонусы, гранты, общий счёт. Точный остаток через API v5 недоступен — используй get_campaign_stats."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		params := map[string]any{
			"FieldNames": []string{"Login", "ClientId", "Currency", "OverdraftSumAvailable", "Bonuses", "Grants", "Settings", "Restrictions", "AccountQuality"},
		}
		raw, err := client.Call(ctx, token, "clients", "get", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.TextResult(string(result)), nil
	})
}
