package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterAdTools registers ad-related MCP tools.
func RegisterAdTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetAds(s, client, resolver)
	registerAddAd(s, client, resolver)
	registerUpdateAd(s, client, resolver)
	registerManageAds(s, client, resolver)
	registerModerateAds(s, client, resolver)
}

func registerGetAds(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_ads",
		mcp.WithDescription("Получить объявления. Фильтр по campaign_ids или adgroup_ids."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую")),
		mcp.WithString("adgroup_ids", mcp.Description("ID групп через запятую")),
		mcp.WithString("ad_ids", mcp.Description("ID объявлений через запятую")),
		mcp.WithString("field_names", mcp.Description("Поля: Id, AdGroupId, CampaignId, State, Status, и т.д.")),
		mcp.WithString("text_ad_field_names", mcp.Description("Поля текстового объявления: Title, Title2, Text, Href, SitelinkSetId, AdExtensionIds, AdImageHash")),
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
		if ids := parseIntSlice(common.GetStringSlice(req, "ad_ids")); len(ids) > 0 {
			criteria["Ids"] = ids
		}

		params := map[string]any{
			"SelectionCriteria": criteria,
		}
		if fields := common.GetStringSlice(req, "field_names"); len(fields) > 0 {
			params["FieldNames"] = fields
		} else {
			params["FieldNames"] = []string{"Id", "AdGroupId", "CampaignId", "State", "Status", "Type"}
		}
		if tf := common.GetStringSlice(req, "text_ad_field_names"); len(tf) > 0 {
			params["TextAdFieldNames"] = tf
		}

		raw, err := client.Call(ctx, token, "ads", "get", params, clientLogin)
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

func registerAddAd(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_ad",
		mcp.WithDescription("Создать текстовое объявление. Лимиты: Title 56, Title2 30, Text 81 символов."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("adgroup_id", mcp.Description("ID группы объявлений"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Заголовок 1 (до 56 символов)"), mcp.Required()),
		mcp.WithString("title2", mcp.Description("Заголовок 2 (до 30 символов)")),
		mcp.WithString("text", mcp.Description("Текст объявления (до 81 символа)"), mcp.Required()),
		mcp.WithString("href", mcp.Description("Ссылка на сайт"), mcp.Required()),
		mcp.WithNumber("sitelink_set_id", mcp.Description("ID набора быстрых ссылок")),
		mcp.WithString("ad_extension_ids", mcp.Description("ID уточнений через запятую")),
		mcp.WithString("ad_image_hash", mcp.Description("Хеш изображения")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		textAd := map[string]any{
			"Title": common.GetString(req, "title"),
			"Text":  common.GetString(req, "text"),
			"Href":  common.GetString(req, "href"),
		}
		if t2 := common.GetString(req, "title2"); t2 != "" {
			textAd["Title2"] = t2
		}
		if slID := common.GetInt(req, "sitelink_set_id"); slID > 0 {
			textAd["SitelinkSetId"] = slID
		}
		if extIDs := parseIntSlice(common.GetStringSlice(req, "ad_extension_ids")); len(extIDs) > 0 {
			textAd["AdExtensionIds"] = extIDs
		}
		if imgHash := common.GetString(req, "ad_image_hash"); imgHash != "" {
			textAd["AdImageHash"] = imgHash
		}

		ad := map[string]any{
			"AdGroupId": common.GetInt(req, "adgroup_id"),
			"TextAd":    textAd,
		}

		params := map[string]any{
			"Ads": []any{ad},
		}

		raw, err := client.Call(ctx, token, "ads", "add", params, clientLogin)
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

func registerUpdateAd(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("update_ad",
		mcp.WithDescription("Обновить объявление. Частичное обновление — только указанные поля. Никогда не удаляй+пересоздавай — используй update."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithNumber("ad_id", mcp.Description("ID объявления"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Новый заголовок 1")),
		mcp.WithString("title2", mcp.Description("Новый заголовок 2")),
		mcp.WithString("text", mcp.Description("Новый текст")),
		mcp.WithString("href", mcp.Description("Новая ссылка")),
		mcp.WithNumber("sitelink_set_id", mcp.Description("Новый ID набора быстрых ссылок")),
		mcp.WithString("ad_extension_ids", mcp.Description("Новые ID уточнений через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		textAd := map[string]any{}
		if t := common.GetString(req, "title"); t != "" {
			textAd["Title"] = t
		}
		if t2 := common.GetString(req, "title2"); t2 != "" {
			textAd["Title2"] = t2
		}
		if t := common.GetString(req, "text"); t != "" {
			textAd["Text"] = t
		}
		if h := common.GetString(req, "href"); h != "" {
			textAd["Href"] = h
		}
		if slID := common.GetInt(req, "sitelink_set_id"); slID > 0 {
			textAd["SitelinkSetId"] = slID
		}
		if extIDs := parseIntSlice(common.GetStringSlice(req, "ad_extension_ids")); len(extIDs) > 0 {
			textAd["AdExtensionIds"] = extIDs
		}

		ad := map[string]any{
			"Id":     common.GetInt(req, "ad_id"),
			"TextAd": textAd,
		}

		params := map[string]any{
			"Ads": []any{ad},
		}

		raw, err := client.Call(ctx, token, "ads", "update", params, clientLogin)
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

func registerManageAds(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("manage_ads",
		mcp.WithDescription("Массовое управление объявлениями: остановка, возобновление, архивация."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("ad_ids", mcp.Description("ID объявлений через запятую"), mcp.Required()),
		mcp.WithString("action", mcp.Description("Действие: suspend, resume, archive, unarchive"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "ad_ids"))
		action := common.GetString(req, "action")

		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}

		raw, err := client.Call(ctx, token, "ads", action, params, clientLogin)
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

func registerModerateAds(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("moderate_ads",
		mcp.WithDescription("Отправить объявления на модерацию."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("ad_ids", mcp.Description("ID объявлений через запятую"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "ad_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}

		raw, err := client.Call(ctx, token, "ads", "moderate", params, clientLogin)
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
