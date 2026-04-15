package vk

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterContentTools registers URL and image upload tools.
func RegisterContentTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerVKCreateURL(s, client, resolver)
	registerVKUploadImage(s, client, resolver)
}

func registerVKCreateURL(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_url",
		mcp.WithDescription("Зарегистрировать URL для объявлений VK Ads. Добавь UTM: utm_source=vk&utm_medium=cpc&utm_campaign=slug."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("url", mcp.Description("URL с UTM-метками"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"url": common.GetString(req, "url"),
		}

		result, err := client.Post(ctx, token, "/urls.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKUploadImage(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_upload_image",
		mcp.WithDescription("Загрузить изображение VK Ads. Мин: icon 256x256, image 600x600. Рек: 1080x1350."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("image_url", mcp.Description("URL внешнего изображения"), mcp.Required()),
		mcp.WithNumber("width", mcp.Description("Ширина в пикселях"), mcp.Required()),
		mcp.WithNumber("height", mcp.Description("Высота в пикселях"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"url":    common.GetString(req, "image_url"),
			"width":  common.GetInt(req, "width"),
			"height": common.GetInt(req, "height"),
		}

		result, err := client.Post(ctx, token, "/images/upload.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}
