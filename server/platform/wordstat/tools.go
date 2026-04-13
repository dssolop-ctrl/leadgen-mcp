package wordstat

import (
	"log/slog"

	"github.com/leadgen-mcp/server/auth"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all Wordstat MCP tools.
func RegisterTools(s *mcpserver.MCPServer, resolver *auth.AccountResolver, logger *slog.Logger) {
	client := NewClient(logger)

	RegisterHandlers(s, client, resolver) // 5 tools
}
