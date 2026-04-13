package metrika

import (
	"context"
	"fmt"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterGoalTools registers goal-related tools.
func RegisterGoalTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_goals",
		mcp.WithDescription("Получить цели счётчика Метрики. Возвращает goal_id для конверсий."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		counterID := common.GetInt(req, "counter_id")
		path := fmt.Sprintf("/management/v1/counter/%d/goals", counterID)

		result, err := client.Get(ctx, token, path, nil)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}
