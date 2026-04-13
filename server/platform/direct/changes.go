package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterChangeTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerCheckChanges(s, client, resolver)
	registerCheckCampaignChanges(s, client, resolver)
}

func registerCheckChanges(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("check_changes",
		mcp.WithDescription("Проверить, какие кампании/группы/объявления изменились с указанной даты. Укажи хотя бы один из: campaign_ids, adgroup_ids, ad_ids."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("timestamp", mcp.Description("Дата-время начала: YYYY-MM-DDTHH:MM:SSZ"), mcp.Required()),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую (хотя бы один фильтр обязателен)")),
		mcp.WithString("adgroup_ids", mcp.Description("ID групп через запятую")),
		mcp.WithString("ad_ids", mcp.Description("ID объявлений через запятую")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		params := map[string]any{
			"Timestamp": common.GetString(req, "timestamp"),
			"FieldNames": []string{"CampaignIds", "AdGroupIds", "AdIds"},
		}

		if ids := parseIntSlice(common.GetStringSlice(req, "campaign_ids")); len(ids) > 0 {
			params["CampaignIds"] = ids
		}
		if ids := parseIntSlice(common.GetStringSlice(req, "adgroup_ids")); len(ids) > 0 {
			params["AdGroupIds"] = ids
		}
		if ids := parseIntSlice(common.GetStringSlice(req, "ad_ids")); len(ids) > 0 {
			params["AdIds"] = ids
		}

		raw, err := client.Call(ctx, token, "changes", "check", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.SafeTextResult(string(result)), nil
	})
}

func registerCheckCampaignChanges(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("check_campaign_changes",
		mcp.WithDescription("Проверить какие кампании изменились с указанной даты. Возвращает типы изменений: SELF (параметры), CHILDREN (группы/объявления), STAT (статистика)."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("campaign_ids", mcp.Description("Не используется (проверяются все кампании). Указывай для совместимости.")),
		mcp.WithString("timestamp", mcp.Description("Дата-время начала: YYYY-MM-DDTHH:MM:SSZ"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil { return common.ErrorResult(err.Error()), nil }
		clientLogin := common.GetString(req, "client_login")

		params := map[string]any{
			"Timestamp": common.GetString(req, "timestamp"),
		}
		raw, err := client.Call(ctx, token, "changes", "checkCampaigns", params, clientLogin)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		result, err := GetResult(raw)
		if err != nil { return common.ErrorResult(err.Error()), nil }
		return common.SafeTextResult(string(result)), nil
	})
}
