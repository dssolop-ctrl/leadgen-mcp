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
}

func registerGetClient(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_client",
		mcp.WithDescription("Получить информацию о клиенте: логин, ФИО, email, баланс, настройки уведомлений."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
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
