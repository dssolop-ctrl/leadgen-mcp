package direct

import (
	"context"
	"fmt"
	"strconv"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterAdGroupTools registers ad group MCP tools.
func RegisterAdGroupTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetAdGroups(s, client, resolver)
	registerAddAdGroup(s, client, resolver)
	registerUpdateAdGroup(s, client, resolver)
}

func registerGetAdGroups(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_adgroups",
		mcp.WithDescription("Получить группы объявлений. Требуется campaign_id или adgroup_ids."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("field_names", mcp.Description("Поля: Id, Name, CampaignId, Status, RegionIds, TrackingParams и т.д.")),
		mcp.WithString("adgroup_ids", mcp.Description("ID групп через запятую (фильтр)")),
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
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
		}
		if fields := common.GetStringSlice(req, "field_names"); len(fields) > 0 {
			params["FieldNames"] = fields
		} else {
			params["FieldNames"] = []string{"Id", "Name", "CampaignId", "Status"}
		}

		raw, err := client.Call(ctx, token, "adgroups", "get", params, clientLogin)
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

func registerAddAdGroup(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_adgroup",
		mcp.WithDescription("Создать группу объявлений. Автотаргетинг: целевые+узкие=ON, остальные=OFF."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Название группы"), mcp.Required()),
		mcp.WithString("region_ids", mcp.Description("ID регионов через запятую"), mcp.Required()),
		mcp.WithString("tracking_params", mcp.Description("UTM-метки: utm_source=yandex&utm_medium=cpc&...")),
		mcp.WithString("negative_keywords", mcp.Description("Минус-фразы через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		regionIDs := parseIntSlice(common.GetStringSlice(req, "region_ids"))

		adGroup := map[string]any{
			"Name":       common.GetString(req, "name"),
			"CampaignId": common.GetInt(req, "campaign_id"),
			"RegionIds":  regionIDs,
		}

		if tp := common.GetString(req, "tracking_params"); tp != "" {
			adGroup["TrackingParams"] = tp
		}

		if negKw := common.GetStringSlice(req, "negative_keywords"); len(negKw) > 0 {
			adGroup["NegativeKeywords"] = map[string]any{"Items": negKw}
		}

		params := map[string]any{
			"AdGroups": []any{adGroup},
		}

		raw, err := client.Call(ctx, token, "adgroups", "add", params, clientLogin)
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

func registerUpdateAdGroup(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_adgroup",
		mcp.WithDescription("Обновить группу объявлений. Частичное обновление. Основное применение — добавить tracking_params (UTM)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithNumber("adgroup_id", mcp.Description("ID группы"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Новое название")),
		mcp.WithString("region_ids", mcp.Description("Новые регионы через запятую")),
		mcp.WithString("tracking_params", mcp.Description("UTM-метки")),
		mcp.WithString("negative_keywords", mcp.Description("Минус-фразы через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		adGroup := map[string]any{
			"Id": common.GetInt(req, "adgroup_id"),
		}

		if name := common.GetString(req, "name"); name != "" {
			adGroup["Name"] = name
		}
		if regions := parseIntSlice(common.GetStringSlice(req, "region_ids")); len(regions) > 0 {
			adGroup["RegionIds"] = regions
		}
		if tp := common.GetString(req, "tracking_params"); tp != "" {
			adGroup["TrackingParams"] = tp
		}
		if negKw := common.GetStringSlice(req, "negative_keywords"); len(negKw) > 0 {
			adGroup["NegativeKeywords"] = map[string]any{"Items": negKw}
		}

		params := map[string]any{
			"AdGroups": []any{adGroup},
		}

		raw, err := client.Call(ctx, token, "adgroups", "update", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("Группа обновлена: %s", string(result))), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func parseIntSlice(strs []string) []int64 {
	var result []int64
	for _, s := range strs {
		n, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			result = append(result, n)
		}
	}
	return result
}
