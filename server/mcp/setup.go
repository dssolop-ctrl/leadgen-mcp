package mcpsetup

import (
	"log/slog"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/direct"
	"github.com/leadgen-mcp/server/platform/metrika"
	"github.com/leadgen-mcp/server/platform/vk"
	"github.com/leadgen-mcp/server/platform/wordstat"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates and configures the MCP server with all tools registered.
func NewServer(resolver *auth.AccountResolver, logger *slog.Logger) *server.MCPServer {
	s := server.NewMCPServer(
		"leadgen-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	// Yandex Direct (33 tools)
	direct.RegisterTools(s, resolver, logger)

	// Yandex Metrika (11 tools)
	metrika.RegisterTools(s, resolver, logger)

	// Yandex Wordstat (5 tools)
	wordstat.RegisterTools(s, resolver, logger)

	// VK Ads (30 tools)
	vk.RegisterTools(s, resolver, logger)

	return s
}
