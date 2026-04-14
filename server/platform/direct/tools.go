package direct

import (
	"log/slog"

	"github.com/leadgen-mcp/server/auth"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all Yandex Direct MCP tools on the server.
func RegisterTools(s *mcpserver.MCPServer, resolver *auth.AccountResolver, logger *slog.Logger) {
	client := NewClient(logger)

	RegisterCampaignTools(s, client, resolver)       // 7 tools
	RegisterAdGroupTools(s, client, resolver)         // 3 tools
	RegisterAdTools(s, client, resolver)              // 5 tools
	RegisterKeywordTools(s, client, resolver)         // 8 tools (+2: update_keywords, manage_autotargeting)
	RegisterStatsTools(s, client, resolver)           // 8 tools (+3: account_stats, custom_report, reach_frequency)
	RegisterExtensionTools(s, client, resolver)       // 6 tools
	RegisterGeoTools(s, client, resolver)             // 2 tools (+1: get_dictionaries)
	RegisterAgencyTools(s, client, resolver)          // 3 tools (+2: add/update_agency_client)
	RegisterBidModifierTools(s, client, resolver)     // 4 tools
	RegisterRetargetingTools(s, client, resolver)     // 4 tools
	RegisterNegKeywordTools(s, client, resolver)      // 4 tools
	RegisterClientTools(s, client, resolver)          // 3 tools (+2: update_client, get_account_balance)
	RegisterChangeTools(s, client, resolver)          // 3 tools (+1: check_dictionary_changes)
	RegisterContentTools(s, client, resolver)         // 4 tools
	RegisterDynamicTools(s, client, resolver)         // 8 tools (+5: set_bids, feed_ad_targets)
	RegisterKeywordBidTools(s, client, resolver)      // 3 tools (+1: set_keyword_bids_auto)
	RegisterBidTools(s, client, resolver)             // 3 tools (NEW)
	RegisterAudienceTargetTools(s, client, resolver)  // 4 tools (NEW)
	RegisterSmartAdTargetTools(s, client, resolver)   // 5 tools (NEW)
	RegisterFeedTools(s, client, resolver)            // 4 tools (NEW)
	RegisterStrategyTools(s, client, resolver)        // 4 tools (NEW)
	RegisterCreativeTools(s, client, resolver)        // 5 tools (NEW: creatives, ad_videos, turbo_pages, leads, businesses)
	RegisterLabelTools(s, client, resolver)            // 3 tools (NEW: get_labels, set_campaign_labels, add_labels — API v4 Live)
}
