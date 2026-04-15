package metrika

import (
	"context"
	"fmt"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterCounterTools registers counter-related tools.
func RegisterCounterTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetCounters(s, client, resolver)
	registerGetCounter(s, client, resolver)
}

func registerGetCounters(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_counters",
		mcp.WithDescription("Получить список счётчиков Яндекс Метрики."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		result, err := client.Get(ctx, token, "/management/v1/counters", nil)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}

func registerGetCounter(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_counter",
		mcp.WithDescription("Получить информацию о конкретном счётчике Метрики."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		counterID := common.GetInt(req, "counter_id")
		path := fmt.Sprintf("/management/v1/counter/%d", counterID)

		result, err := client.Get(ctx, token, path, nil)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}
