package direct

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

// RegisterLabelTools registers label (tag) management tools using API v4 Live.
// Labels work at the banner (ad) level but visually apply to ad groups in the interface.
//
// Model:
//  1. UpdateCampaignsTags — manage the label catalog at campaign level (name → TagID)
//  2. GetBannersTags — get labels assigned to banners
//  3. UpdateBannersTags — assign labels to banners (effectively to ad groups)
func RegisterLabelTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetLabelsV4(s, client, resolver)
	registerAddLabels(s, client, resolver)
}

// ===== GET LABELS =====

func registerGetLabelsV4(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_labels",
		mcp.WithDescription(
			"Получить метки. Два режима: (1) campaign_ids → каталог меток кампании, "+
				"(2) banner_ids → метки конкретных объявлений. "+
				"API v4 Live: GetCampaignsTags / GetBannersTags."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("campaign_ids", mcp.Description("ID кампаний через запятую (до 10) — вернёт каталог меток")),
		mcp.WithString("banner_ids", mcp.Description("ID объявлений через запятую (до 2000) — вернёт назначенные метки")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignIDs := parseIntIDs(common.GetStringSlice(req, "campaign_ids"))
		bannerIDs := parseIntIDs(common.GetStringSlice(req, "banner_ids"))

		if len(campaignIDs) == 0 && len(bannerIDs) == 0 {
			return common.ErrorResult("укажи campaign_ids или banner_ids"), nil
		}

		if len(bannerIDs) > 0 {
			// GetBannersTags — labels assigned to specific banners
			params := map[string]any{"BannerIDS": bannerIDs}
			raw, err := client.CallV4(ctx, token, "GetBannersTags", params, clientLogin)
			if err != nil {
				return common.ErrorResult(err.Error()), nil
			}
			return common.TextResult(string(raw)), nil
		}

		// GetCampaignsTags — label catalog for campaigns
		params := map[string]any{"CampaignIDS": campaignIDs}
		raw, err := client.CallV4(ctx, token, "GetCampaignsTags", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(raw)), nil
	})
}

// ===== ADD LABELS =====

// addLabels is the main convenience tool:
// 1. Gets campaign tag catalog
// 2. Creates missing tags via UpdateCampaignsTags
// 3. Assigns tags to banners via UpdateBannersTags
func registerAddLabels(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_labels",
		mcp.WithDescription(
			"Создать метки (если не существуют) и назначить на объявления (= группу в интерфейсе). "+
				"Автоматически: получает каталог → создаёт недостающие → назначает на баннеры. "+
				"Для отчётности обязательны: Лидген, <Тематика>, <Направление>. "+
				"Нужен campaign_id (для каталога) и banner_ids (для назначения)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города"), mcp.Required()),
		mcp.WithString("campaign_id", mcp.Description("ID кампании (для каталога меток)"), mcp.Required()),
		mcp.WithString("banner_ids", mcp.Description("ID объявлений через запятую (метки назначаются на них)"), mcp.Required()),
		mcp.WithString("labels", mcp.Description("Метки через запятую. Пример: Лидген,Вторичка,Покупатель"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignIDs := parseIntIDs(common.GetStringSlice(req, "campaign_id"))
		if len(campaignIDs) == 0 {
			return common.ErrorResult("campaign_id: укажи ID кампании"), nil
		}
		campaignID := campaignIDs[0]

		bannerIDs := parseIntIDs(common.GetStringSlice(req, "banner_ids"))
		if len(bannerIDs) == 0 {
			return common.ErrorResult("banner_ids: укажи хотя бы один ID объявления"), nil
		}

		newLabels := common.GetStringSlice(req, "labels")
		if len(newLabels) == 0 {
			return common.ErrorResult("labels: укажи хотя бы одну метку"), nil
		}

		// Validate label lengths
		for _, l := range newLabels {
			if len([]rune(l)) > 25 {
				return common.ErrorResult(fmt.Sprintf("метка '%s' превышает 25 символов (%d)", l, len([]rune(l)))), nil
			}
		}

		// Step 1: Get existing campaign tag catalog
		getParams := map[string]any{"CampaignIDS": []int64{campaignID}}
		raw, err := client.CallV4(ctx, token, "GetCampaignsTags", getParams, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("не удалось получить метки кампании: %v", err)), nil
		}

		var getResp v4CampaignTagsResponse
		if err := json.Unmarshal(raw, &getResp); err != nil {
			return common.ErrorResult(fmt.Sprintf("ошибка парсинга: %v", err)), nil
		}

		// Build existing name→TagID map
		existingMap := make(map[string]int64) // lowercase name → TagID
		var existingTags []v4TagObj
		for _, ct := range getResp.Data {
			if ct.CampaignID == campaignID {
				existingTags = ct.Tags
				for _, t := range ct.Tags {
					existingMap[strings.ToLower(t.Tag)] = t.TagID
				}
			}
		}

		// Step 2: Check if all labels exist, create missing ones
		var tagIDs []int64
		needsUpdate := false

		for _, name := range newLabels {
			if id, ok := existingMap[strings.ToLower(name)]; ok {
				tagIDs = append(tagIDs, id)
			} else {
				needsUpdate = true
			}
		}

		if needsUpdate {
			// Merge existing tags with new ones
			mergedTags := make([]map[string]any, 0)

			// Keep existing tags
			for _, t := range existingTags {
				mergedTags = append(mergedTags, map[string]any{
					"TagID": t.TagID,
					"Tag":   t.Tag,
				})
			}

			// Add new tags (TagID=0 for new)
			for _, name := range newLabels {
				if _, ok := existingMap[strings.ToLower(name)]; !ok {
					mergedTags = append(mergedTags, map[string]any{
						"TagID": 0,
						"Tag":   name,
					})
				}
			}

			updateParams := []map[string]any{
				{
					"CampaignID": campaignID,
					"Tags":       mergedTags,
				},
			}

			_, err := client.CallV4(ctx, token, "UpdateCampaignsTags", updateParams, clientLogin)
			if err != nil {
				return common.ErrorResult(fmt.Sprintf("не удалось обновить каталог меток: %v", err)), nil
			}

			// Re-fetch to get new TagIDs
			raw, err = client.CallV4(ctx, token, "GetCampaignsTags", getParams, clientLogin)
			if err != nil {
				return common.ErrorResult(fmt.Sprintf("не удалось перечитать метки: %v", err)), nil
			}

			var refreshResp v4CampaignTagsResponse
			if err := json.Unmarshal(raw, &refreshResp); err != nil {
				return common.ErrorResult(fmt.Sprintf("ошибка парсинга: %v", err)), nil
			}

			// Rebuild map with new IDs
			refreshMap := make(map[string]int64)
			for _, ct := range refreshResp.Data {
				if ct.CampaignID == campaignID {
					for _, t := range ct.Tags {
						refreshMap[strings.ToLower(t.Tag)] = t.TagID
					}
				}
			}

			// Collect all TagIDs
			tagIDs = nil
			for _, name := range newLabels {
				if id, ok := refreshMap[strings.ToLower(name)]; ok {
					tagIDs = append(tagIDs, id)
				} else {
					return common.ErrorResult(fmt.Sprintf("метка '%s' не найдена после создания", name)), nil
				}
			}
		}

		// Step 3: Assign tags to banners via UpdateBannersTags
		bannerTagEntries := make([]map[string]any, 0, len(bannerIDs))
		for _, bid := range bannerIDs {
			bannerTagEntries = append(bannerTagEntries, map[string]any{
				"BannerID": bid,
				"TagIDS":   tagIDs,
			})
		}

		assignRaw, err := client.CallV4(ctx, token, "UpdateBannersTags", bannerTagEntries, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("не удалось назначить метки на объявления: %v", err)), nil
		}

		// Build result summary
		summary := map[string]any{
			"campaign_id":   campaignID,
			"banner_ids":    bannerIDs,
			"labels":        newLabels,
			"tag_ids":       tagIDs,
			"assign_result": json.RawMessage(assignRaw),
		}
		out, _ := json.MarshalIndent(summary, "", "  ")
		return common.TextResult(string(out)), nil
	})
}

// --- Helpers ---

type v4CampaignTagsResponse struct {
	Data []v4CampaignTags `json:"data"`
}

type v4CampaignTags struct {
	CampaignID int64      `json:"CampaignID"`
	Tags       []v4TagObj `json:"Tags"`
}

type v4TagObj struct {
	Tag   string `json:"Tag"`
	TagID int64  `json:"TagID,omitempty"`
}

// parseIntIDs converts a string slice to int64 slice, skipping invalid entries.
func parseIntIDs(strs []string) []int64 {
	return parseIntSlice(strs)
}
