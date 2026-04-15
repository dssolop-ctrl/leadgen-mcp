package metrika

import (
	"log/slog"

	"github.com/leadgen-mcp/server/auth"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all Yandex Metrika MCP tools (creates its own client).
func RegisterTools(s *mcpserver.MCPServer, resolver *auth.AccountResolver, logger *slog.Logger) {
	client := NewClient(logger)
	RegisterToolsWithClient(s, client, resolver)
}

// RegisterToolsWithClient registers all Yandex Metrika MCP tools using a shared client.
func RegisterToolsWithClient(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	RegisterCounterTools(s, client, resolver)  // 2 tools
	RegisterGoalTools(s, client, resolver)     // 1 tool
	RegisterReportTools(s, client, resolver)   // 8 tools
}
