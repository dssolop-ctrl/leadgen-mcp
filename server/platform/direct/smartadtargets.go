package direct

import (
	"context"
	"encoding/json"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterSmartAdTargetTools registers SmartAdTargets service MCP tools.
func RegisterSmartAdTargetTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetSmartAdTargets(s, client, resolver)
	registerAddSmartAdTarget(s, client, resolver)
	registerUpdateSmartAdTarget(s, client, resolver)
	registerManageSmartAdTargets(s, client, resolver)
	registerSetSmartAdTargetBids(s, client, resolver)
}

func registerGetSmartAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_smart_ad_targets",
		mcp.WithDescription("Получить фильтры смарт-баннеров."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("adgroup_ids", mcp.Description("ID групп через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "adgroup_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"AdGroupIds": ids},
			"FieldNames":        []string{"Id", "AdGroupId", "Name", "State", "Audience"},
		}
		raw, err := client.Call(ctx, token, "smartadtargets", "get", params, clientLogin)
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

func registerAddSmartAdTarget(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_smart_ad_target",
		mcp.WithDescription("Добавить фильтр смарт-баннеров."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("adgroup_id", mcp.Description("ID группы"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Название фильтра"), mcp.Required()),
		mcp.WithString("conditions_json", mcp.Description("JSON условий фильтра: [{\"Operand\":\"CATEGORY_ID\",\"Operator\":\"EQUALS_ANY\",\"Arguments\":[\"1\",\"2\"]}]")),
		mcp.WithString("audience", mcp.Description("Аудитория: INTERESTED_IN_SIMILAR (похожие) или HAS_VISITED_SITE (посещавшие)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		target := map[string]any{
			"AdGroupId": common.GetInt(req, "adgroup_id"),
			"Name":      common.GetString(req, "name"),
		}
		if condJSON := common.GetString(req, "conditions_json"); condJSON != "" {
			var cond any
			if err := json.Unmarshal([]byte(condJSON), &cond); err != nil {
				return common.ErrorResult("invalid conditions_json: " + err.Error()), nil
			}
			target["Conditions"] = cond
		}
		if audience := common.GetString(req, "audience"); audience != "" {
			target["Audience"] = audience
		}

		params := map[string]any{"SmartAdTargets": []any{target}}
		raw, err := client.Call(ctx, token, "smartadtargets", "add", params, clientLogin)
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

func registerUpdateSmartAdTarget(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_smart_ad_target",
		mcp.WithDescription("Обновить фильтр смарт-баннеров."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("target_id", mcp.Description("ID фильтра"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название фильтра")),
		mcp.WithString("conditions_json", mcp.Description("JSON новых условий фильтра")),
		mcp.WithString("audience", mcp.Description("Аудитория: INTERESTED_IN_SIMILAR или HAS_VISITED_SITE")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		target := map[string]any{
			"Id": common.GetInt(req, "target_id"),
		}
		if name := common.GetString(req, "name"); name != "" {
			target["Name"] = name
		}
		if condJSON := common.GetString(req, "conditions_json"); condJSON != "" {
			var cond any
			if err := json.Unmarshal([]byte(condJSON), &cond); err != nil {
				return common.ErrorResult("invalid conditions_json: " + err.Error()), nil
			}
			target["Conditions"] = cond
		}
		if audience := common.GetString(req, "audience"); audience != "" {
			target["Audience"] = audience
		}

		params := map[string]any{"SmartAdTargets": []any{target}}
		raw, err := client.Call(ctx, token, "smartadtargets", "update", params, clientLogin)
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

func registerManageSmartAdTargets(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_smart_ad_targets",
		mcp.WithDescription("Управление фильтрами смарт-баннеров: suspend, resume, delete."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("target_ids", mcp.Description("ID фильтров через запятую"), mcp.Required()),
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
		raw, err := client.Call(ctx, token, "smartadtargets", action, params, clientLogin)
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

func registerSetSmartAdTargetBids(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_smart_ad_target_bids",
		mcp.WithDescription("Установить ставки для фильтров смарт-баннеров."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("bids_json", mcp.Description("JSON массив: [{\"SmartAdTargetId\":123,\"ContextBid\":5000000}]"), mcp.Required()),
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
		raw, err := client.Call(ctx, token, "smartadtargets", "setBids", params, clientLogin)
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
