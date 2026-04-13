package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterAgencyTools registers agency-related MCP tools.
func RegisterAgencyTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetAgencyClients(s, client, resolver)
}

func registerGetAgencyClients(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_agency_clients",
		mcp.WithDescription("Получить список клиентских аккаунтов агентства Яндекс Директ. Возвращает логины клиентов — используй их в параметре client_login других инструментов. ОБЯЗАТЕЛЬНО вызови первым при работе с агентским аккаунтом."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально, по умолчанию — default)")),
		mcp.WithString("logins", mcp.Description("Фильтр по логинам клиентов через запятую (опционально)")),
		mcp.WithBoolean("archived", mcp.Description("Включить архивных клиентов (по умолчанию false)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		criteria := map[string]any{}

		if logins := common.GetStringSlice(req, "logins"); len(logins) > 0 {
			criteria["Logins"] = logins
		}

		archived := false
		if v, ok := req.GetArguments()["archived"]; ok {
			if b, ok := v.(bool); ok {
				archived = b
			}
		}
		if archived {
			criteria["Archived"] = "YES"
		}

		fieldNames := []string{
			"Login", "ClientId", "ClientInfo", "Archived",
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        fieldNames,
		}

		raw, err := client.Call(ctx, token, "agencyclients", "get", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		return common.JSONResult(result), nil
	})
}
