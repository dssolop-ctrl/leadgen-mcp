package metrika

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// goal represents a Metrika goal for filtering.
type goal struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Status     string          `json:"status,omitempty"`
	Conditions json.RawMessage `json:"conditions,omitempty"`
}

// goalsResponse is the API response envelope.
type goalsResponse struct {
	Goals []json.RawMessage `json:"goals"`
}

// goalCondition is a single condition entry.
type goalCondition struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// RegisterGoalTools registers goal-related tools.
func RegisterGoalTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("metrika_get_goals",
		mcp.WithDescription("Получить цели счётчика Метрики. Без фильтра — все цели (может быть 100+). Используй conditions для поиска конкретных целей по условию."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("conditions", mcp.Description("Фильтр по conditions.url через запятую. Пример: form_sum_leads,received_real_calls,all_calls")),
		mcp.WithString("goal_type", mcp.Description("Фильтр по типу цели: action, url, step, call, conditional_call")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		counterID := common.GetInt(req, "counter_id")
		path := fmt.Sprintf("/management/v1/counter/%d/goals", counterID)

		raw, err := client.Get(ctx, token, path, nil)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		// If no filters — return as-is (backward compatible)
		condFilter := common.GetStringSlice(req, "conditions")
		typeFilter := common.GetString(req, "goal_type")

		if len(condFilter) == 0 && typeFilter == "" {
			return common.TextResult(string(raw)), nil
		}

		// Parse and filter
		var resp goalsResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return common.ErrorResult(fmt.Sprintf("parse goals: %v", err)), nil
		}

		// Build condition lookup set
		condSet := make(map[string]bool, len(condFilter))
		for _, c := range condFilter {
			condSet[strings.TrimSpace(c)] = true
		}

		var filtered []json.RawMessage
		for _, rawGoal := range resp.Goals {
			var g goal
			if err := json.Unmarshal(rawGoal, &g); err != nil {
				continue
			}

			// Filter by type
			if typeFilter != "" && g.Type != typeFilter {
				continue
			}

			// Filter by conditions
			if len(condSet) > 0 {
				if !goalMatchesConditions(rawGoal, condSet) {
					continue
				}
			}

			filtered = append(filtered, rawGoal)
		}

		result := map[string]any{
			"goals": filtered,
			"total": len(filtered),
		}
		out, _ := json.Marshal(result)
		return common.TextResult(string(out)), nil
	})
}

// goalMatchesConditions checks if any of the goal's conditions match the filter set.
func goalMatchesConditions(rawGoal json.RawMessage, condSet map[string]bool) bool {
	var g struct {
		Conditions []goalCondition `json:"conditions"`
	}
	if err := json.Unmarshal(rawGoal, &g); err != nil {
		return false
	}
	for _, c := range g.Conditions {
		if condSet[c.URL] || condSet[c.Type] {
			return true
		}
	}
	return false
}
