package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterBidTools registers Bids service MCP tools.
func RegisterBidTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetBids(s, client, resolver)
	registerSetBids(s, client, resolver)
	registerSetBidsAuto(s, client, resolver)
}

func registerGetBids(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_bids",
		mcp.WithDescription("Получить ставки для ключевых фраз (устаревший сервис Bids). Для новых кампаний используй get_keyword_bids."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую")),
		mcp.WithString("adgroup_ids", mcp.Description("ID групп через запятую")),
		mcp.WithString("keyword_ids", mcp.Description("ID ключевых фраз через запятую")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "campaign_ids")); len(ids) > 0 {
			criteria["CampaignIds"] = ids
		}
		if ids := parseIntSlice(common.GetStringSlice(req, "adgroup_ids")); len(ids) > 0 {
			criteria["AdGroupIds"] = ids
		}
		if ids := parseIntSlice(common.GetStringSlice(req, "keyword_ids")); len(ids) > 0 {
			criteria["KeywordIds"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"KeywordId", "AdGroupId", "CampaignId", "Bid", "ContextBid"},
		}
		raw, err := client.Call(ctx, token, "bids", "get", params, clientLogin)
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

func registerSetBids(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_bids",
		mcp.WithDescription("Установить ставки для ключевых фраз (сервис Bids). Ставки в рублях."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("keyword_ids", mcp.Description("ID ключевых фраз через запятую"), mcp.Required()),
		mcp.WithNumber("bid", mcp.Description("Ставка для поиска в рублях"), mcp.Required()),
		mcp.WithNumber("context_bid", mcp.Description("Ставка для сетей в рублях (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "keyword_ids"))
		bid := common.GetInt(req, "bid")
		contextBid := common.GetInt(req, "context_bid")

		var bids []any
		for _, id := range ids {
			b := map[string]any{
				"KeywordId": id,
				"Bid":       bid * 1000000,
			}
			if contextBid > 0 {
				b["ContextBid"] = contextBid * 1000000
			}
			bids = append(bids, b)
		}

		params := map[string]any{"Bids": bids}
		raw, err := client.Call(ctx, token, "bids", "set", params, clientLogin)
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

func registerSetBidsAuto(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_bids_auto",
		mcp.WithDescription("Установить автоматические ставки для ключевых фраз (сервис Bids). Задаёт целевую позицию и ограничения."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("keyword_ids", mcp.Description("ID ключевых фраз через запятую"), mcp.Required()),
		mcp.WithNumber("max_bid", mcp.Description("Максимальная ставка в рублях"), mcp.Required()),
		mcp.WithString("position", mcp.Description("Целевая позиция: PREMIUMBLOCK (спецразмещение) или FOOTERBLOCK (гарантия)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "keyword_ids"))
		maxBid := common.GetInt(req, "max_bid")
		position := common.GetString(req, "position")
		if position == "" {
			position = "PREMIUMBLOCK"
		}

		var bids []any
		for _, id := range ids {
			bids = append(bids, map[string]any{
				"KeywordId": id,
				"Bid":       maxBid * 1000000,
			})
		}

		params := map[string]any{"Bids": bids}
		raw, err := client.Call(ctx, token, "bids", "setAuto", params, clientLogin)
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
