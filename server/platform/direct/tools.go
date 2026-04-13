package direct

import (
	"log/slog"

	"github.com/leadgen-mcp/server/auth"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all Yandex Direct MCP tools on the server.
func RegisterTools(s *mcpserver.MCPServer, resolver *auth.AccountResolver, logger *slog.Logger) {
	client := NewClient(logger)

	RegisterCampaignTools(s, client, resolver)   // 7 tools
	RegisterAdGroupTools(s, client, resolver)     // 3 tools
	RegisterAdTools(s, client, resolver)          // 5 tools
	RegisterKeywordTools(s, client, resolver)     // 6 tools
	RegisterStatsTools(s, client, resolver)       // 5 tools
	RegisterExtensionTools(s, client, resolver)   // 6 tools
	RegisterGeoTools(s, client, resolver)         // 1 tool
	RegisterAgencyTools(s, client, resolver)      // 1 tool
	RegisterBidModifierTools(s, client, resolver) // 4 tools
	RegisterRetargetingTools(s, client, resolver) // 4 tools
	RegisterNegKeywordTools(s, client, resolver)  // 4 tools
	RegisterClientTools(s, client, resolver)      // 1 tool
	RegisterChangeTools(s, client, resolver)      // 2 tools
	RegisterContentTools(s, client, resolver)     // 4 tools
	RegisterDynamicTools(s, client, resolver)     // 3 tools
	RegisterKeywordBidTools(s, client, resolver)  // 2 tools
}
