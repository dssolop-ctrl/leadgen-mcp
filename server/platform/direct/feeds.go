package direct

import (
	"context"
	"encoding/json"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterFeedTools registers Feeds service MCP tools.
func RegisterFeedTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetFeeds(s, client, resolver)
	registerAddFeed(s, client, resolver)
	registerUpdateFeed(s, client, resolver)
	registerDeleteFeeds(s, client, resolver)
}

func registerGetFeeds(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_feeds",
		mcp.WithDescription("Получить фиды (товарные каталоги) для динамических и смарт-объявлений."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("feed_ids", mcp.Description("ID фидов через запятую (опционально, без фильтра — все)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "feed_ids"))
		if len(ids) == 0 {
			return common.ErrorResult("Укажи feed_ids. Чтобы узнать ID фидов, посмотри FeedId в группах через get_adgroups."), nil
		}
		criteria := map[string]any{"Ids": ids}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "Name", "BusinessType", "SourceType", "Status", "NumberOfItems"},
		}
		raw, err := client.Call(ctx, token, "feeds", "get", params, clientLogin)
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

func registerAddFeed(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_feed",
		mcp.WithDescription("Добавить фид (товарный каталог)."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("name", mcp.Description("Название фида"), mcp.Required()),
		mcp.WithString("business_type", mcp.Description("Тип бизнеса: RETAIL, HOTELS, REALTY, AUTOS, FLIGHTS"), mcp.Required()),
		mcp.WithString("source_type", mcp.Description("Тип источника: URL или FILE"), mcp.Required()),
		mcp.WithString("url", mcp.Description("URL фида (для source_type=URL)")),
		mcp.WithString("login", mcp.Description("Логин для доступа к URL (опционально)")),
		mcp.WithString("password", mcp.Description("Пароль для доступа к URL (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		feed := map[string]any{
			"Name":         common.GetString(req, "name"),
			"BusinessType": common.GetString(req, "business_type"),
			"SourceType":   common.GetString(req, "source_type"),
		}
		if u := common.GetString(req, "url"); u != "" {
			urlFeed := map[string]any{"Url": u}
			if login := common.GetString(req, "login"); login != "" {
				urlFeed["Login"] = login
			}
			if pwd := common.GetString(req, "password"); pwd != "" {
				urlFeed["Password"] = pwd
			}
			feed["UrlFeed"] = urlFeed
		}

		params := map[string]any{"Feeds": []any{feed}}
		raw, err := client.Call(ctx, token, "feeds", "add", params, clientLogin)
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

func registerUpdateFeed(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_feed",
		mcp.WithDescription("Обновить параметры фида."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithNumber("feed_id", mcp.Description("ID фида"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название фида")),
		mcp.WithString("feed_json", mcp.Description("JSON с полями для обновления (Name, UrlFeed и т.д.)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		feed := map[string]any{
			"Id": common.GetInt(req, "feed_id"),
		}
		if name := common.GetString(req, "name"); name != "" {
			feed["Name"] = name
		}
		if feedJSON := common.GetString(req, "feed_json"); feedJSON != "" {
			var extra map[string]any
			if err := json.Unmarshal([]byte(feedJSON), &extra); err != nil {
				return common.ErrorResult("invalid feed_json: " + err.Error()), nil
			}
			for k, v := range extra {
				feed[k] = v
			}
		}

		params := map[string]any{"Feeds": []any{feed}}
		raw, err := client.Call(ctx, token, "feeds", "update", params, clientLogin)
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

func registerDeleteFeeds(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("delete_feeds",
		mcp.WithDescription("Удалить фиды."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("feed_ids", mcp.Description("ID фидов через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "feed_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "feeds", "delete", params, clientLogin)
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
