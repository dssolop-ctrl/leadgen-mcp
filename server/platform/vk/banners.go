package vk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterBannerTools registers VK banner (ad) tools.
func RegisterBannerTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerVKGetBanners(s, client, resolver)
	registerVKCreateBanner(s, client, resolver)
	registerVKUpdateBanner(s, client, resolver)
	registerVKManageBanners(s, client, resolver)
	registerVKRemoderateBanners(s, client, resolver)
}

func registerVKGetBanners(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_banners",
		mcp.WithDescription("Получить объявления (баннеры) VK Ads."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("ad_group_id", mcp.Description("ID группы объявлений")),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании")),
		mcp.WithNumber("limit", mcp.Description("Лимит")),
		mcp.WithNumber("offset", mcp.Description("Смещение")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := url.Values{}
		if agID := common.GetInt(req, "ad_group_id"); agID > 0 {
			params.Set("ad_group_id", fmt.Sprintf("%d", agID))
		}
		if cID := common.GetInt(req, "campaign_id"); cID > 0 {
			params.Set("ad_plan_id", fmt.Sprintf("%d", cID))
		}
		if l := common.GetInt(req, "limit"); l > 0 {
			params.Set("limit", fmt.Sprintf("%d", l))
		}
		if o := common.GetInt(req, "offset"); o > 0 {
			params.Set("offset", fmt.Sprintf("%d", o))
		}

		result, err := client.Get(ctx, token, "/banners.json", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKCreateBanner(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_banner",
		mcp.WithDescription("Создать объявление VK Ads. ЗАПРЕЩЁН символ → — используй —, запятую, точку. Лимиты: title_40=40, text_90=90, text_long=220, title_30=30, about_company=115."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("ad_group_id", mcp.Description("ID группы"), mcp.Required()),
		mcp.WithNumber("url_id", mcp.Description("ID URL (из vk_create_url)"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Заголовок (до 40 символов)"), mcp.Required()),
		mcp.WithString("text", mcp.Description("Короткий текст (до 90 символов)"), mcp.Required()),
		mcp.WithString("text_long", mcp.Description("Длинный текст (до 220 символов)")),
		mcp.WithString("title_additional", mcp.Description("Дополнительный заголовок (до 30 символов)")),
		mcp.WithString("about_company", mcp.Description("О компании (до 115 символов)")),
		mcp.WithString("icon_id", mcp.Description("content_id иконки 256x256")),
		mcp.WithString("image_id", mcp.Description("content_id изображения 600x600")),
		mcp.WithString("image_vertical_id", mcp.Description("content_id вертикального 1080x1350 (рекомендуется)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		content := map[string]any{
			"title_40_vkads": common.GetString(req, "title"),
			"text_90":        common.GetString(req, "text"),
		}
		if tl := common.GetString(req, "text_long"); tl != "" {
			content["text_long"] = tl
		}
		if ta := common.GetString(req, "title_additional"); ta != "" {
			content["title_30_additional"] = ta
		}
		if ac := common.GetString(req, "about_company"); ac != "" {
			content["about_company_115"] = ac
		}
		if iconID := common.GetString(req, "icon_id"); iconID != "" {
			content["icon_256x256"] = map[string]any{"id": iconID}
		}
		if imgID := common.GetString(req, "image_id"); imgID != "" {
			content["image_600x600"] = map[string]any{"id": imgID}
		}
		if imgV := common.GetString(req, "image_vertical_id"); imgV != "" {
			content["image_1080x1350"] = map[string]any{"id": imgV}
		}

		body := map[string]any{
			"ad_group_id": common.GetInt(req, "ad_group_id"),
			"url_id":      common.GetInt(req, "url_id"),
			"content":     content,
		}

		result, err := client.Post(ctx, token, "/banners.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKUpdateBanner(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_update_banner",
		mcp.WithDescription("Обновить объявление VK Ads."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("banner_id", mcp.Description("ID объявления"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Новый заголовок")),
		mcp.WithString("text", mcp.Description("Новый текст")),
		mcp.WithString("text_long", mcp.Description("Новый длинный текст")),
		mcp.WithNumber("url_id", mcp.Description("Новый URL ID")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		bannerID := common.GetInt(req, "banner_id")
		body := map[string]any{}

		content := map[string]any{}
		if t := common.GetString(req, "title"); t != "" {
			content["title_40_vkads"] = t
		}
		if t := common.GetString(req, "text"); t != "" {
			content["text_90"] = t
		}
		if tl := common.GetString(req, "text_long"); tl != "" {
			content["text_long"] = tl
		}
		if len(content) > 0 {
			body["content"] = content
		}
		if urlID := common.GetInt(req, "url_id"); urlID > 0 {
			body["url_id"] = urlID
		}

		path := fmt.Sprintf("/banners/%d.json", bannerID)
		result, err := client.Patch(ctx, token, path, body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKManageBanners(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_manage_banners",
		mcp.WithDescription("Массовое управление объявлениями VK: активация, остановка. До 200 ID."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("banner_ids", mcp.Description("ID объявлений через запятую"), mcp.Required()),
		mcp.WithString("status", mcp.Description("Статус: active, blocked"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"ids":    common.GetStringSlice(req, "banner_ids"),
			"status": common.GetString(req, "status"),
		}

		result, err := client.Post(ctx, token, "/banners/mass_action.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKRemoderateBanners(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_remoderate_banners",
		mcp.WithDescription("Повторно отправить объявления на модерацию."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("banner_ids", mcp.Description("ID объявлений через запятую"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"ids": common.GetStringSlice(req, "banner_ids"),
		}

		result, err := client.Post(ctx, token, "/banners/remoderate.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}
