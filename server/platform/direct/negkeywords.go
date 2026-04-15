package direct

import (
	"context"
	"strings"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterNegKeywordTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetNegKeywordSets(s, client, resolver)
	registerAddNegKeywordSet(s, client, resolver)
	registerUpdateNegKeywordSet(s, client, resolver)
	registerDeleteNegKeywordSets(s, client, resolver)
}

func registerGetNegKeywordSets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_negative_keyword_sets",
		mcp.WithDescription("Получить наборы минус-фраз (библиотека). Применяются к кампаниям."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("set_ids", mcp.Description("ID наборов через запятую (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "set_ids"))
		if len(ids) == 0 {
			return common.ErrorResult("Укажи set_ids. Чтобы узнать ID наборов, используй get_campaigns с field_names=NegativeKeywordSharedSetIds"), nil
		}
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
			"FieldNames": []string{"Id", "Name", "NegativeKeywords"},
		}
		raw, err := client.Call(ctx, token, "negativekeywordsharedsets", "get", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.SafeTextResult(string(result)), nil
	})
}

func registerAddNegKeywordSet(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_negative_keyword_set",
		mcp.WithDescription("Создать набор минус-фраз."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("name", mcp.Description("Название набора"), mcp.Required()),
		mcp.WithString("negative_keywords", mcp.Description("Минус-фразы через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		keywords := strings.Split(common.GetString(req, "negative_keywords"), ",")
		for i := range keywords {
			keywords[i] = strings.TrimSpace(keywords[i])
		}

		params := map[string]any{
			"NegativeKeywordSharedSets": []any{
				map[string]any{
					"Name": common.GetString(req, "name"),
					"NegativeKeywords": keywords,
				},
			},
		}
		raw, err := client.Call(ctx, token, "negativekeywordsharedsets", "add", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.TextResult(string(result)), nil
	})
}

func registerUpdateNegKeywordSet(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_negative_keyword_set",
		mcp.WithDescription("Обновить набор минус-фраз."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("set_id", mcp.Description("ID набора"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название")),
		mcp.WithString("negative_keywords", mcp.Description("Новые минус-фразы через запятую")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		set := map[string]any{"Id": common.GetInt(req, "set_id")}
		if name := common.GetString(req, "name"); name != "" {
			set["Name"] = name
		}
		if kw := common.GetString(req, "negative_keywords"); kw != "" {
			keywords := strings.Split(kw, ",")
			for i := range keywords {
				keywords[i] = strings.TrimSpace(keywords[i])
			}
			set["NegativeKeywords"] = keywords
		}

		params := map[string]any{"NegativeKeywordSharedSets": []any{set}}
		raw, err := client.Call(ctx, token, "negativekeywordsharedsets", "update", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.TextResult(string(result)), nil
	})
}

func registerDeleteNegKeywordSets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("delete_negative_keyword_sets",
		mcp.WithDescription("Удалить наборы минус-фраз."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("set_ids", mcp.Description("ID наборов через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "set_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "negativekeywordsharedsets", "delete", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.TextResult(string(result)), nil
	})
}
