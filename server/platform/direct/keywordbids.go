package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterKeywordBidTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetKeywordBids(s, client, resolver)
	registerSetKeywordBids(s, client, resolver)
}

func registerGetKeywordBids(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_keyword_bids",
		mcp.WithDescription("Получить ставки по ключевым фразам."),
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
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"KeywordId", "AdGroupId", "CampaignId", "ServingStatus", "StrategyPriority"},
		}
		raw, err := client.Call(ctx, token, "keywordbids", "get", params, clientLogin)
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

func registerSetKeywordBids(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_keyword_bids",
		mcp.WithDescription("Установить ставки для ключевых фраз. Ставки в рублях (будут конвертированы в микроюниты)."),
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
				"SearchBid": bid * 1000000,
			}
			if contextBid > 0 {
				b["NetworkBid"] = contextBid * 1000000
			}
			bids = append(bids, b)
		}

		params := map[string]any{"KeywordBids": bids}
		raw, err := client.Call(ctx, token, "keywordbids", "set", params, clientLogin)
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
