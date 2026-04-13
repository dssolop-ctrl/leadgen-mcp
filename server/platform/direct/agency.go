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
	registerAddAgencyClient(s, client, resolver)
	registerUpdateAgencyClient(s, client, resolver)
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
			"Login", "ClientId", "ClientInfo", "Archived", "AccountQuality", "Phone", "Currency", "OverdraftSumAvailable", "Bonuses", "Grants",
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

func registerAddAgencyClient(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_agency_client",
		mcp.WithDescription("Создать нового клиента агентства."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("login", mcp.Description("Логин нового клиента"), mcp.Required()),
		mcp.WithString("first_name", mcp.Description("Имя"), mcp.Required()),
		mcp.WithString("last_name", mcp.Description("Фамилия"), mcp.Required()),
		mcp.WithString("currency", mcp.Description("Валюта: RUB, USD, EUR и т.д. (по умолчанию RUB)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		currency := common.GetString(req, "currency")
		if currency == "" {
			currency = "RUB"
		}

		params := map[string]any{
			"Login":    common.GetString(req, "login"),
			"FirstName": common.GetString(req, "first_name"),
			"LastName":  common.GetString(req, "last_name"),
			"Currency":  currency,
		}
		raw, err := client.Call(ctx, token, "agencyclients", "add", params)
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

func registerUpdateAgencyClient(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_agency_client",
		mcp.WithDescription("Обновить настройки клиента агентства."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("login", mcp.Description("Логин клиента для обновления"), mcp.Required()),
		mcp.WithString("client_info", mcp.Description("Новое имя/описание клиента")),
		mcp.WithString("grants_json", mcp.Description("JSON прав: {\"Privilege\":\"EDIT_CAMPAIGNS\",\"Value\":\"YES\"}")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		cl := map[string]any{
			"Login": common.GetString(req, "login"),
		}
		if v := common.GetString(req, "client_info"); v != "" {
			cl["ClientInfo"] = v
		}

		params := map[string]any{"Clients": []any{cl}}
		raw, err := client.Call(ctx, token, "agencyclients", "update", params)
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
