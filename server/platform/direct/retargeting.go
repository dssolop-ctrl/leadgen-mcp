package direct

import (
	"context"
	"encoding/json"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterRetargetingTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetRetargetingLists(s, client, resolver)
	registerAddRetargetingList(s, client, resolver)
	registerUpdateRetargetingList(s, client, resolver)
	registerDeleteRetargetingLists(s, client, resolver)
}

func registerGetRetargetingLists(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_retargeting_lists",
		mcp.WithDescription("Получить условия ретаргетинга и подбора аудитории."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("retargeting_list_ids", mcp.Description("ID списков через запятую (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "retargeting_list_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}
		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "Name", "Description", "Type"},
		}
		raw, err := client.Call(ctx, token, "retargetinglists", "get", params, clientLogin)
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

func registerAddRetargetingList(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_retargeting_list",
		mcp.WithDescription("Создать условие ретаргетинга. rules_json — массив правил [{\"GoalId\":123,\"GoalType\":\"GOAL\"}]."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("name", mcp.Description("Название списка"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Описание")),
		mcp.WithString("rules_json", mcp.Description("JSON массив правил: [{\"Items\":[{\"GoalId\":123,\"GoalType\":\"GOAL\"}],\"Operator\":\"ALL\"}]"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		var rules any
		if err := json.Unmarshal([]byte(common.GetString(req, "rules_json")), &rules); err != nil {
			return common.ErrorResult("invalid rules_json: " + err.Error()), nil
		}

		list := map[string]any{
			"Name":  common.GetString(req, "name"),
			"Rules": rules,
		}
		if desc := common.GetString(req, "description"); desc != "" {
			list["Description"] = desc
		}

		params := map[string]any{"RetargetingLists": []any{list}}
		raw, err := client.Call(ctx, token, "retargetinglists", "add", params, clientLogin)
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

func registerUpdateRetargetingList(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_retargeting_list",
		mcp.WithDescription("Обновить условие ретаргетинга."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("retargeting_list_id", mcp.Description("ID списка"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название")),
		mcp.WithString("rules_json", mcp.Description("Новые правила (JSON)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		list := map[string]any{"Id": common.GetInt(req, "retargeting_list_id")}
		if name := common.GetString(req, "name"); name != "" {
			list["Name"] = name
		}
		if rulesStr := common.GetString(req, "rules_json"); rulesStr != "" {
			var rules any
			if err := json.Unmarshal([]byte(rulesStr), &rules); err != nil {
				return common.ErrorResult("invalid rules_json: " + err.Error()), nil
			}
			list["Rules"] = rules
		}

		params := map[string]any{"RetargetingLists": []any{list}}
		raw, err := client.Call(ctx, token, "retargetinglists", "update", params, clientLogin)
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

func registerDeleteRetargetingLists(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("delete_retargeting_lists",
		mcp.WithDescription("Удалить условия ретаргетинга."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("retargeting_list_ids", mcp.Description("ID списков через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "retargeting_list_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "retargetinglists", "delete", params, clientLogin)
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
