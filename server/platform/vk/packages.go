package vk

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterPackageTools registers package reference tools.
func RegisterPackageTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_packages",
		mcp.WithDescription("Получить пакеты размещения VK Ads (форматы рекламы). 3858 = мультиформат (основной)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		result, err := client.Get(ctx, token, "/packages.json", nil)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}
