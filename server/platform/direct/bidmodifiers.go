package direct

import (
	"context"
	"encoding/json"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterBidModifierTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetBidModifiers(s, client, resolver)
	registerAddBidModifiers(s, client, resolver)
	registerSetBidModifiers(s, client, resolver)
	registerDeleteBidModifiers(s, client, resolver)
}

func registerGetBidModifiers(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_bid_modifiers",
		mcp.WithDescription("Получить корректировки ставок (пол, возраст, устройство, регион, погода и др.) для кампаний."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "campaign_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{
				"CampaignIds": ids,
				"Levels":      []string{"CAMPAIGN", "AD_GROUP"},
			},
			"FieldNames":       []string{"Id", "CampaignId", "AdGroupId", "Type", "Level"},
			"DemographicsAdjustmentFieldNames": []string{"Gender", "Age", "BidModifier"},
			"RegionalAdjustmentFieldNames":     []string{"RegionId", "BidModifier"},
			"MobileAdjustmentFieldNames":       []string{"BidModifier"},
			"DesktopAdjustmentFieldNames":      []string{"BidModifier"},
		}

		raw, err := client.Call(ctx, token, "bidmodifiers", "get", params, clientLogin)
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

func registerAddBidModifiers(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_bid_modifiers",
		mcp.WithDescription("Добавить корректировку ставок. Типы: demographics (пол+возраст), regional (регион), mobile (мобильные), desktop (десктоп)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Тип: DEMOGRAPHICS, REGIONAL, MOBILE, DESKTOP"), mcp.Required()),
		mcp.WithString("adjustment_json", mcp.Description("JSON корректировки. Пример demographics: {\"Gender\":\"MALE\",\"Age\":\"AGE_25_34\",\"BidModifier\":120}. Пример mobile: {\"BidModifier\":50}"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignID := common.GetInt(req, "campaign_id")
		modType := common.GetString(req, "type")
		adjJSON := common.GetString(req, "adjustment_json")

		var adj any
		if err := json.Unmarshal([]byte(adjJSON), &adj); err != nil {
			return common.ErrorResult("invalid adjustment_json: " + err.Error()), nil
		}

		modifier := map[string]any{"CampaignId": campaignID}
		switch modType {
		case "DEMOGRAPHICS":
			modifier["DemographicsAdjustment"] = adj
		case "REGIONAL":
			modifier["RegionalAdjustment"] = adj
		case "MOBILE":
			modifier["MobileAdjustment"] = adj
		case "DESKTOP":
			modifier["DesktopAdjustment"] = adj
		default:
			return common.ErrorResult("type must be DEMOGRAPHICS, REGIONAL, MOBILE or DESKTOP"), nil
		}

		params := map[string]any{"BidModifiers": []any{modifier}}
		raw, err := client.Call(ctx, token, "bidmodifiers", "add", params, clientLogin)
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

func registerSetBidModifiers(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_bid_modifiers",
		mcp.WithDescription("Изменить значение корректировки ставок."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("bid_modifier_id", mcp.Description("ID корректировки"), mcp.Required()),
		mcp.WithNumber("bid_modifier", mcp.Description("Новое значение корректировки (процент, 0-1300)"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		params := map[string]any{
			"BidModifiers": []any{
				map[string]any{
					"Id":          common.GetInt(req, "bid_modifier_id"),
					"BidModifier": common.GetInt(req, "bid_modifier"),
				},
			},
		}
		raw, err := client.Call(ctx, token, "bidmodifiers", "set", params, clientLogin)
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

func registerDeleteBidModifiers(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("delete_bid_modifiers",
		mcp.WithDescription("Удалить корректировки ставок."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("bid_modifier_ids", mcp.Description("ID корректировок через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "bid_modifier_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "bidmodifiers", "delete", params, clientLogin)
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
