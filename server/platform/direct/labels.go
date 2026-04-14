package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterLabelTools registers label (tag) management tools using API v4 Live.
// Labels are not available in API v5 — only through v4 Live methods:
// GetCampaignsTags, UpdateCampaignsTags.
func RegisterLabelTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetLabels(s, client, resolver)
	registerSetCampaignLabels(s, client, resolver)
	registerAddLabels(s, client, resolver)
}

// registerGetLabels — get labels (tags) for campaigns.
func registerGetLabels(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_labels",
		mcp.WithDescription("Получить метки (теги) кампаний. API v4 Live: GetCampaignsTags. Возвращает список меток для каждой кампании."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntIDs(common.GetStringSlice(req, "campaign_ids"))
		if len(ids) == 0 {
			return common.ErrorResult("campaign_ids: укажи хотя бы один ID кампании"), nil
		}

		params := map[string]any{
			"CampaignIDS": ids,
		}

		raw, err := client.CallV4(ctx, token, "GetCampaignsTags", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		return common.TextResult(string(raw)), nil
	})
}

// registerSetCampaignLabels — set (replace) labels on campaigns.
func registerSetCampaignLabels(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_campaign_labels",
		mcp.WithDescription("Установить метки на кампании (ЗАМЕНЯЕТ все текущие метки). Чтобы добавить к существующим — используй add_labels. До 200 меток на кампанию, до 25 символов каждая."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("labels", mcp.Description("Метки через запятую. Пример: Вторичка,Поиск,Тюмень"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntIDs(common.GetStringSlice(req, "campaign_ids"))
		if len(ids) == 0 {
			return common.ErrorResult("campaign_ids: укажи хотя бы один ID кампании"), nil
		}

		labels := common.GetStringSlice(req, "labels")
		if len(labels) == 0 {
			return common.ErrorResult("labels: укажи хотя бы одну метку"), nil
		}

		// Validate label length
		for _, l := range labels {
			if len([]rune(l)) > 25 {
				return common.ErrorResult(fmt.Sprintf("метка '%s' превышает 25 символов (%d)", l, len([]rune(l)))), nil
			}
		}

		// Build update entries — same labels for all campaigns
		entries := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			entries = append(entries, map[string]any{
				"CampaignID": id,
				"Tags":       labels,
			})
		}

		params := map[string]any{
			"CampaignsTags": entries,
		}

		raw, err := client.CallV4(ctx, token, "UpdateCampaignsTags", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		return common.TextResult(string(raw)), nil
	})
}

// registerAddLabels — add labels to existing campaign labels (merge, not replace).
func registerAddLabels(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_labels",
		mcp.WithDescription("Добавить метки к кампаниям (НЕ заменяя существующие). Сначала получает текущие метки, потом объединяет с новыми. Для полной замены — используй set_campaign_labels."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую"), mcp.Required()),
		mcp.WithString("labels", mcp.Description("Метки для добавления через запятую. Пример: Вторичка,Поиск"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntIDs(common.GetStringSlice(req, "campaign_ids"))
		if len(ids) == 0 {
			return common.ErrorResult("campaign_ids: укажи хотя бы один ID кампании"), nil
		}

		newLabels := common.GetStringSlice(req, "labels")
		if len(newLabels) == 0 {
			return common.ErrorResult("labels: укажи хотя бы одну метку"), nil
		}

		// Validate label length
		for _, l := range newLabels {
			if len([]rune(l)) > 25 {
				return common.ErrorResult(fmt.Sprintf("метка '%s' превышает 25 символов (%d)", l, len([]rune(l)))), nil
			}
		}

		// Step 1: Get existing labels
		getParams := map[string]any{
			"CampaignIDS": ids,
		}
		raw, err := client.CallV4(ctx, token, "GetCampaignsTags", getParams, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("не удалось получить текущие метки: %v", err)), nil
		}

		// Parse existing tags
		var getResp v4TagsResponse
		if err := json.Unmarshal(raw, &getResp); err != nil {
			return common.ErrorResult(fmt.Sprintf("ошибка парсинга меток: %v", err)), nil
		}

		// Step 2: Merge existing + new labels per campaign
		entries := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			existing := getResp.findTags(id)
			merged := mergeTags(existing, newLabels)
			entries = append(entries, map[string]any{
				"CampaignID": id,
				"Tags":       merged,
			})
		}

		// Step 3: Update with merged labels
		updateParams := map[string]any{
			"CampaignsTags": entries,
		}
		raw, err = client.CallV4(ctx, token, "UpdateCampaignsTags", updateParams, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		return common.TextResult(string(raw)), nil
	})
}

// --- Helpers ---

// v4TagsResponse is the response from GetCampaignsTags.
type v4TagsResponse struct {
	Data []v4CampaignTags `json:"data"`
}

type v4CampaignTags struct {
	CampaignID int64    `json:"CampaignID"`
	Tags       []string `json:"Tags"`
}

func (r *v4TagsResponse) findTags(campaignID int64) []string {
	for _, d := range r.Data {
		if d.CampaignID == campaignID {
			return d.Tags
		}
	}
	return nil
}

// parseIntIDs converts a string slice to int64 slice, skipping invalid entries.
func parseIntIDs(strs []string) []int64 {
	ids := make([]int64, 0, len(strs))
	for _, s := range strs {
		n, err := strconv.ParseInt(s, 10, 64)
		if err == nil && n > 0 {
			ids = append(ids, n)
		}
	}
	return ids
}

// mergeTags merges two tag slices, deduplicating case-insensitively.
// Preserves original casing of the first occurrence.
func mergeTags(existing, newTags []string) []string {
	seen := make(map[string]bool, len(existing)+len(newTags))
	result := make([]string, 0, len(existing)+len(newTags))

	for _, t := range existing {
		lower := strings.ToLower(t)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, t)
		}
	}
	for _, t := range newTags {
		lower := strings.ToLower(t)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, t)
		}
	}
	return result
}
