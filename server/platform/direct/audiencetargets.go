package direct

import (
	"context"
	"encoding/json"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterAudienceTargetTools registers AudienceTargets service MCP tools.
func RegisterAudienceTargetTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetAudienceTargets(s, client, resolver)
	registerAddAudienceTargets(s, client, resolver)
	registerManageAudienceTargets(s, client, resolver)
	registerSetAudienceTargetBids(s, client, resolver)
}

func registerGetAudienceTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_audience_targets",
		mcp.WithDescription("Получить условия подбора аудитории для групп объявлений."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("adgroup_ids", mcp.Description("ID групп через запятую"), mcp.Required()),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "adgroup_ids")); len(ids) > 0 {
			criteria["AdGroupIds"] = ids
		}
		if ids := parseIntSlice(common.GetStringSlice(req, "campaign_ids")); len(ids) > 0 {
			criteria["CampaignIds"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "AdGroupId", "CampaignId", "RetargetingListId", "InterestId", "ContextBid", "StrategyPriority", "State"},
		}
		raw, err := client.Call(ctx, token, "audiencetargets", "get", params, clientLogin)
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

func registerAddAudienceTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_audience_targets",
		mcp.WithDescription("Добавить условия подбора аудитории в группу объявлений."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("targets_json", mcp.Description("JSON массив: [{\"AdGroupId\":123,\"RetargetingListId\":456,\"ContextBid\":5000000}]"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var targets any
		if err := json.Unmarshal([]byte(common.GetString(req, "targets_json")), &targets); err != nil {
			return common.ErrorResult("invalid targets_json: " + err.Error()), nil
		}

		params := map[string]any{"AudienceTargets": targets}
		raw, err := client.Call(ctx, token, "audiencetargets", "add", params, clientLogin)
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

func registerManageAudienceTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_audience_targets",
		mcp.WithDescription("Управление условиями подбора аудитории: suspend, resume, delete."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("target_ids", mcp.Description("ID условий через запятую"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: suspend, resume, delete"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "target_ids"))
		action := common.GetString(req, "action")
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "audiencetargets", action, params, clientLogin)
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

func registerSetAudienceTargetBids(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_audience_target_bids",
		mcp.WithDescription("Установить ставки для условий подбора аудитории."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("bids_json", mcp.Description("JSON массив: [{\"Id\":123,\"ContextBid\":5000000}]"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var bids any
		if err := json.Unmarshal([]byte(common.GetString(req, "bids_json")), &bids); err != nil {
			return common.ErrorResult("invalid bids_json: " + err.Error()), nil
		}

		params := map[string]any{"Bids": bids}
		raw, err := client.Call(ctx, token, "audiencetargets", "setBids", params, clientLogin)
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
