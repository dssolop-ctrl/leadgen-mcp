package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterExtensionTools registers sitelinks and ad extensions MCP tools.
func RegisterExtensionTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetSitelinks(s, client, resolver)
	registerAddSitelinks(s, client, resolver)
	registerDeleteSitelinks(s, client, resolver)
	registerGetAdExtensions(s, client, resolver)
	registerAddAdExtension(s, client, resolver)
	registerDeleteAdExtensions(s, client, resolver)
}

func registerGetSitelinks(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_sitelinks",
		mcp.WithDescription("Получить наборы быстрых ссылок."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("sitelink_set_ids", mcp.Description("ID наборов через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "sitelink_set_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":       []string{"Id", "Sitelinks"},
		}

		raw, err := client.Call(ctx, token, "sitelinks", "get", params, clientLogin)
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

func registerAddSitelinks(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_sitelinks",
		mcp.WithDescription("Создать набор быстрых ссылок. До 8 ссылок в наборе. Формат: title|description|href через точку с запятой."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("sitelinks", mcp.Description("Ссылки: title1|description1|href1;title2|description2|href2;... (title до 30, description до 60 символов)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		sitelinksStr := common.GetString(req, "sitelinks")
		var sitelinks []any

		for _, part := range splitBySemicolon(sitelinksStr) {
			fields := splitByPipe(part)
			if len(fields) >= 3 {
				sitelinks = append(sitelinks, map[string]any{
					"Title":       fields[0],
					"Description": fields[1],
					"Href":        fields[2],
				})
			}
		}

		params := map[string]any{
			"SitelinksSets": []any{
				map[string]any{
					"Sitelinks": sitelinks,
				},
			},
		}

		raw, err := client.Call(ctx, token, "sitelinks", "add", params, clientLogin)
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

func registerDeleteSitelinks(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("delete_sitelinks",
		mcp.WithDescription("Удалить наборы быстрых ссылок."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("sitelink_set_ids", mcp.Description("ID наборов через запятую"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "sitelink_set_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}

		raw, err := client.Call(ctx, token, "sitelinks", "delete", params, clientLogin)
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

func registerGetAdExtensions(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_ad_extensions",
		mcp.WithDescription("Получить уточнения (callouts) для объявлений."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("ad_extension_ids", mcp.Description("ID уточнений через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "ad_extension_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria":          criteria,
			"FieldNames":                 []string{"Id", "Type", "State", "Status", "StatusClarification", "Associated"},
			"CalloutFieldNames": []string{"CalloutText"},
		}

		raw, err := client.Call(ctx, token, "adextensions", "get", params, clientLogin)
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

func registerAddAdExtension(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_ad_extension",
		mcp.WithDescription("Создать уточнение (callout) для объявлений. До 25 символов."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("callout_text", mcp.Description("Текст уточнения (до 25 символов)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		params := map[string]any{
			"AdExtensions": []any{
				map[string]any{
					"Callout": map[string]any{
						"CalloutText": common.GetString(req, "callout_text"),
					},
				},
			},
		}

		raw, err := client.Call(ctx, token, "adextensions", "add", params, clientLogin)
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

func registerDeleteAdExtensions(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("delete_ad_extensions",
		mcp.WithDescription("Удалить уточнения."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("ad_extension_ids", mcp.Description("ID уточнений через запятую"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "ad_extension_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}

		raw, err := client.Call(ctx, token, "adextensions", "delete", params, clientLogin)
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

func splitBySemicolon(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == ';' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func splitByPipe(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == '|' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
