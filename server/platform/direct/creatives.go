package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterCreativeTools registers Creatives, AdVideos, TurboPages, Leads, Businesses tools.
func RegisterCreativeTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetCreatives(s, client, resolver)
	registerGetAdVideos(s, client, resolver)
	registerGetTurboPages(s, client, resolver)
	registerGetLeads(s, client, resolver)
	registerGetBusinesses(s, client, resolver)
}

func registerGetCreatives(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("creatives",
		mcp.WithDescription("Получить креативы (видеодополнения, смарт-баннеры и др.)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("creative_ids", mcp.Description("ID креативов через запятую (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "creative_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "Name", "Type", "PreviewUrl", "ThumbnailUrl"},
		}
		raw, err := client.Call(ctx, token, "creatives", "get", params, clientLogin)
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

func registerGetAdVideos(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("ad_videos",
		mcp.WithDescription("Получить видеодополнения для объявлений."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("video_ids", mcp.Description("ID видео через запятую (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "video_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "Name", "PreviewUrl", "ThumbnailUrl", "Type"},
		}
		raw, err := client.Call(ctx, token, "creatives", "get", params, clientLogin)
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

func registerGetTurboPages(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_turbo_pages",
		mcp.WithDescription("Получить турбо-страницы для объявлений."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("turbo_page_ids", mcp.Description("ID турбо-страниц через запятую (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "turbo_page_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "Name", "Href", "TurboSiteHref", "PreviewHref"},
		}
		raw, err := client.Call(ctx, token, "turbopages", "get", params, clientLogin)
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

func registerGetLeads(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_leads",
		mcp.WithDescription("Получить лиды (заявки) из турбо-страниц Яндекс Директа. Требуется turbo_page_ids."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("turbo_page_ids", mcp.Description("ID турбо-страниц через запятую"), mcp.Required()),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую (опционально)")),
		mcp.WithString("date_from", mcp.Description("Дата начала YYYY-MM-DD")),
		mcp.WithString("date_to", mcp.Description("Дата конца YYYY-MM-DD")),
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
		if df := common.GetString(req, "date_from"); df != "" {
			criteria["DateFrom"] = df
		}
		if dt := common.GetString(req, "date_to"); dt != "" {
			criteria["DateTo"] = dt
		}
		if tpIds := parseIntSlice(common.GetStringSlice(req, "turbo_page_ids")); len(tpIds) > 0 {
			criteria["TurboPageIds"] = tpIds
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "TurboPageId", "SubmitDate", "Data"},
		}
		raw, err := client.Call(ctx, token, "leads", "get", params, clientLogin)
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

func registerGetBusinesses(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_businesses",
		mcp.WithDescription("Получить организации из Яндекс Бизнеса, привязанные к аккаунту. Нужен хотя бы один фильтр: business_ids, name или url."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("business_ids", mcp.Description("ID организаций через запятую")),
		mcp.WithString("name", mcp.Description("Фильтр по названию (поиск подстроки)")),
		mcp.WithString("url", mcp.Description("Фильтр по URL сайта")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		criteria := map[string]any{}
		if ids := parseIntSlice(common.GetStringSlice(req, "business_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}
		if name := common.GetString(req, "name"); name != "" {
			criteria["Name"] = name
		}
		if u := common.GetString(req, "url"); u != "" {
			criteria["Url"] = u
		}
		if len(criteria) == 0 {
			return common.ErrorResult("Укажи хотя бы один фильтр: business_ids, name или url"), nil
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
			"FieldNames":        []string{"Id", "Name", "Type", "Address", "Phone", "ProfileUrl", "IsPublished"},
		}
		raw, err := client.Call(ctx, token, "businesses", "get", params, clientLogin)
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
