package vk

import (
	"log/slog"

	"github.com/leadgen-mcp/server/auth"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all VK Ads MCP tools.
func RegisterTools(s *mcpserver.MCPServer, resolver *auth.AccountResolver, logger *slog.Logger) {
	client := NewClient(logger)

	RegisterCampaignTools(s, client, resolver)  // 4 tools
	RegisterAdGroupTools(s, client, resolver)   // 4 tools
	RegisterBannerTools(s, client, resolver)    // 5 tools
	RegisterContentTools(s, client, resolver)   // 2 tools
	RegisterAudienceTools(s, client, resolver)  // 9 tools
	RegisterTargetingTools(s, client, resolver) // 2 tools
	RegisterStatsTools(s, client, resolver)     // 3 tools
	RegisterPackageTools(s, client, resolver)   // 1 tool
}
