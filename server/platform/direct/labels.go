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
//  3. UpdateBannersTags — assign labels to banners (REPLACES the set per banner)
//
// Tools:
//   get_labels        — read catalog (CampaignIDS) or banner assignments (BannerIDS)
//   add_labels        — create missing tags + add to banners (idempotent, additive)
//   remove_labels     — remove specific labels from banners (no campaign_id needed)
//   set_banner_labels — REPLACE the entire tag set on banners (remove + add atomic)
func RegisterLabelTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetLabelsV4(s, client, resolver)
	registerAddLabels(s, client, resolver)
	registerRemoveLabels(s, client, resolver)
	registerSetBannerLabels(s, client, resolver)
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
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании (число, не строка) — для каталога меток. Передавай как integer."), mcp.Required()),
		mcp.WithString("banner_ids", mcp.Description("ID объявлений через запятую (метки назначаются на них)"), mcp.Required()),
		mcp.WithString("labels", mcp.Description("Метки через запятую. Пример: Лидген,Вторичка,Покупатель"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		// campaign_id — single integer (not a comma-separated string).
		// Previously declared as WithString and parsed via GetStringSlice →
		// when the client passed a number JSON-side, it produced an empty
		// slice and the tool errored with "укажи ID кампании". 2026-05-05 fix.
		campaignIDInt := common.GetInt(req, "campaign_id")
		if campaignIDInt <= 0 {
			// Backward-compat: fall back to legacy string slice parsing in case
			// some older clients still pass it as a comma string.
			legacy := parseIntIDs(common.GetStringSlice(req, "campaign_id"))
			if len(legacy) > 0 {
				campaignIDInt = int(legacy[0])
			}
		}
		if campaignIDInt <= 0 {
			return common.ErrorResult("campaign_id: укажи ID кампании (число, > 0)"), nil
		}
		campaignID := int64(campaignIDInt)

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

// ===== REMOVE LABELS =====

// removeLabels removes specific labels (by name, case-insensitive) from given banners.
// It does NOT touch the campaign tag catalog — only the banner↔tag assignments.
// Names are resolved per banner from the tags currently assigned (no campaign_id
// is needed). API v4 Live flow: GetBannersTags → filter → UpdateBannersTags
// with reduced TagIDS per banner (UpdateBannersTags REPLACES the set per banner).
func registerRemoveLabels(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("remove_labels",
		mcp.WithDescription(
			"Убрать конкретные метки с объявлений (banner-level). "+
				"Не трогает каталог кампании — только связи banner↔tag. "+
				"Имена меток резолвятся case-insensitively из текущих меток баннеров — "+
				"campaign_id не нужен. До 2000 banner_ids за вызов."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города"), mcp.Required()),
		mcp.WithString("banner_ids", mcp.Description("ID объявлений через запятую (до 2000)"), mcp.Required()),
		mcp.WithString("labels", mcp.Description("Метки на удаление через запятую. Пример: topic:vtorichka,channel:search"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		bannerIDs := parseIntIDs(common.GetStringSlice(req, "banner_ids"))
		if len(bannerIDs) == 0 {
			return common.ErrorResult("banner_ids: укажи хотя бы один ID объявления"), nil
		}
		if len(bannerIDs) > 2000 {
			return common.ErrorResult(fmt.Sprintf("banner_ids: максимум 2000 за вызов, передано %d", len(bannerIDs))), nil
		}

		toRemove := common.GetStringSlice(req, "labels")
		if len(toRemove) == 0 {
			return common.ErrorResult("labels: укажи хотя бы одну метку"), nil
		}
		removeSet := make(map[string]struct{}, len(toRemove))
		for _, l := range toRemove {
			removeSet[strings.ToLower(l)] = struct{}{}
		}

		// Step 1: read current tags per banner
		getParams := map[string]any{"BannerIDS": bannerIDs}
		raw, err := client.CallV4(ctx, token, "GetBannersTags", getParams, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("не удалось получить метки объявлений: %v", err)), nil
		}
		var getResp v4BannerTagsResponse
		if err := json.Unmarshal(raw, &getResp); err != nil {
			return common.ErrorResult(fmt.Sprintf("ошибка парсинга: %v", err)), nil
		}

		// Step 2: compute per-banner reduced TagIDS + diff
		type bannerChange struct {
			BannerID     int64    `json:"banner_id"`
			RemovedNames []string `json:"removed,omitempty"`
		}
		changedEntries := make([]map[string]any, 0)
		changes := make([]bannerChange, 0)
		totalRemoved := 0
		for _, bt := range getResp.Data {
			newIDs, removedNames := computeRemovedTagIDs(bt.Tags, removeSet)
			if len(removedNames) == 0 {
				continue
			}
			totalRemoved += len(removedNames)
			// TagIDS must always be present; empty list = clear all tags for banner.
			if newIDs == nil {
				newIDs = []int64{}
			}
			changedEntries = append(changedEntries, map[string]any{
				"BannerID": bt.BannerID,
				"TagIDS":   newIDs,
			})
			changes = append(changes, bannerChange{BannerID: bt.BannerID, RemovedNames: removedNames})
		}

		errorsPerBanner := map[string]string{}
		var updateRaw json.RawMessage
		if len(changedEntries) > 0 {
			updateRaw, err = client.CallV4(ctx, token, "UpdateBannersTags", changedEntries, clientLogin)
			if err != nil {
				// All-or-nothing failure from the API. Surface it; the caller can retry.
				errorsPerBanner["*"] = err.Error()
				return common.ErrorResult(fmt.Sprintf("не удалось обновить метки: %v", err)), nil
			}
		}

		result := map[string]any{
			"banners_processed": len(getResp.Data),
			"banners_changed":   len(changedEntries),
			"removed_count":     totalRemoved,
			"labels_requested":  toRemove,
			"errors_per_banner": errorsPerBanner,
			"changes":           changes,
		}
		if updateRaw != nil {
			result["update_result"] = json.RawMessage(updateRaw)
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		return common.TextResult(string(out)), nil
	})
}

// ===== SET BANNER LABELS =====

// setBannerLabels REPLACES the entire tag set on each banner with the provided
// labels (full replace, not merge). Equivalent to remove-all + add-new in one
// atomic API call. Mirrors add_labels for catalog management: campaign_id is
// required so missing tags can be auto-created in the campaign catalog.
//
// Pass labels="" to clear all tags from the listed banners.
func registerSetBannerLabels(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_banner_labels",
		mcp.WithDescription(
			"Полная замена меток на объявлениях. Удаляет ВСЕ текущие, ставит указанный набор. "+
				"Требует campaign_id для каталога меток (создаёт недостающие). "+
				"Пустая строка labels = очистить все метки. До 2000 banner_ids за вызов."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города"), mcp.Required()),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании — для каталога меток. Передавай как integer."), mcp.Required()),
		mcp.WithString("banner_ids", mcp.Description("ID объявлений через запятую (до 2000)"), mcp.Required()),
		mcp.WithString("labels", mcp.Description("Полный целевой набор меток через запятую. ВСЕ текущие метки заменяются. Пустая строка = очистить."), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		campaignIDInt := common.GetInt(req, "campaign_id")
		if campaignIDInt <= 0 {
			legacy := parseIntIDs(common.GetStringSlice(req, "campaign_id"))
			if len(legacy) > 0 {
				campaignIDInt = int(legacy[0])
			}
		}
		if campaignIDInt <= 0 {
			return common.ErrorResult("campaign_id: укажи ID кампании (число, > 0)"), nil
		}
		campaignID := int64(campaignIDInt)

		bannerIDs := parseIntIDs(common.GetStringSlice(req, "banner_ids"))
		if len(bannerIDs) == 0 {
			return common.ErrorResult("banner_ids: укажи хотя бы один ID объявления"), nil
		}
		if len(bannerIDs) > 2000 {
			return common.ErrorResult(fmt.Sprintf("banner_ids: максимум 2000 за вызов, передано %d", len(bannerIDs))), nil
		}

		newLabels := common.GetStringSlice(req, "labels")
		// newLabels==nil means "clear all". Validate length only for non-empty entries.
		for _, l := range newLabels {
			if len([]rune(l)) > 25 {
				return common.ErrorResult(fmt.Sprintf("метка '%s' превышает 25 символов (%d)", l, len([]rune(l)))), nil
			}
		}

		// Step 1: read current banner tags for removed_diff reporting
		bannerTagsRaw, err := client.CallV4(ctx, token, "GetBannersTags",
			map[string]any{"BannerIDS": bannerIDs}, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("не удалось получить текущие метки: %v", err)), nil
		}
		var oldResp v4BannerTagsResponse
		_ = json.Unmarshal(bannerTagsRaw, &oldResp)
		oldByBanner := make(map[int64][]string, len(oldResp.Data))
		for _, bt := range oldResp.Data {
			for _, t := range bt.Tags {
				oldByBanner[bt.BannerID] = append(oldByBanner[bt.BannerID], t.Tag)
			}
		}

		// Step 2: resolve target labels → TagIDs via campaign catalog
		tagIDs := []int64{}
		if len(newLabels) > 0 {
			resolvedIDs, err := resolveOrCreateCampaignTagIDs(ctx, client, token, clientLogin, campaignID, newLabels)
			if err != nil {
				return common.ErrorResult(err.Error()), nil
			}
			tagIDs = resolvedIDs
		}

		// Step 3: UpdateBannersTags REPLACES tag set per banner with new TagIDS
		entries := make([]map[string]any, 0, len(bannerIDs))
		for _, bid := range bannerIDs {
			entries = append(entries, map[string]any{"BannerID": bid, "TagIDS": tagIDs})
		}
		assignRaw, err := client.CallV4(ctx, token, "UpdateBannersTags", entries, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("не удалось назначить метки: %v", err)), nil
		}

		// Build removed_diff: old labels that are no longer in newLabels per banner
		newSet := make(map[string]struct{}, len(newLabels))
		for _, l := range newLabels {
			newSet[strings.ToLower(l)] = struct{}{}
		}
		removedDiff := make(map[string][]string)
		for bid, oldTags := range oldByBanner {
			var diff []string
			for _, ot := range oldTags {
				if _, kept := newSet[strings.ToLower(ot)]; !kept {
					diff = append(diff, ot)
				}
			}
			if len(diff) > 0 {
				removedDiff[fmt.Sprintf("%d", bid)] = diff
			}
		}

		summary := map[string]any{
			"campaign_id":   campaignID,
			"banner_ids":    bannerIDs,
			"assigned":      newLabels,
			"tag_ids":       tagIDs,
			"removed_diff":  removedDiff,
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

type v4BannerTagsResponse struct {
	Data []v4BannerTags `json:"data"`
}

type v4BannerTags struct {
	BannerID int64      `json:"BannerID"`
	Tags     []v4TagObj `json:"Tags"`
}

type v4TagObj struct {
	Tag   string `json:"Tag"`
	TagID int64  `json:"TagID,omitempty"`
}

// parseIntIDs converts a string slice to int64 slice, skipping invalid entries.
func parseIntIDs(strs []string) []int64 {
	return parseIntSlice(strs)
}

// computeRemovedTagIDs filters out tags whose names (case-insensitive) appear
// in removeSet. Returns the surviving TagIDs and the original names that were
// dropped. Pure function — extracted for unit testing.
func computeRemovedTagIDs(currentTags []v4TagObj, removeSet map[string]struct{}) (newTagIDs []int64, removedNames []string) {
	for _, t := range currentTags {
		if _, hit := removeSet[strings.ToLower(t.Tag)]; hit {
			removedNames = append(removedNames, t.Tag)
			continue
		}
		newTagIDs = append(newTagIDs, t.TagID)
	}
	return
}

// resolveOrCreateCampaignTagIDs resolves label names to TagIDs in the campaign
// catalog, creating any that are missing. Mirrors the dance inside add_labels.
func resolveOrCreateCampaignTagIDs(ctx context.Context, client *Client, token, clientLogin string, campaignID int64, names []string) ([]int64, error) {
	getParams := map[string]any{"CampaignIDS": []int64{campaignID}}
	raw, err := client.CallV4(ctx, token, "GetCampaignsTags", getParams, clientLogin)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить каталог меток: %w", err)
	}
	var getResp v4CampaignTagsResponse
	if err := json.Unmarshal(raw, &getResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга каталога: %w", err)
	}
	existingMap := make(map[string]int64)
	var existingTags []v4TagObj
	for _, ct := range getResp.Data {
		if ct.CampaignID == campaignID {
			existingTags = ct.Tags
			for _, t := range ct.Tags {
				existingMap[strings.ToLower(t.Tag)] = t.TagID
			}
		}
	}

	needsUpdate := false
	for _, name := range names {
		if _, ok := existingMap[strings.ToLower(name)]; !ok {
			needsUpdate = true
			break
		}
	}

	if needsUpdate {
		mergedTags := make([]map[string]any, 0, len(existingTags)+len(names))
		for _, t := range existingTags {
			mergedTags = append(mergedTags, map[string]any{"TagID": t.TagID, "Tag": t.Tag})
		}
		for _, name := range names {
			if _, ok := existingMap[strings.ToLower(name)]; !ok {
				mergedTags = append(mergedTags, map[string]any{"TagID": 0, "Tag": name})
			}
		}
		updateParams := []map[string]any{{"CampaignID": campaignID, "Tags": mergedTags}}
		if _, err := client.CallV4(ctx, token, "UpdateCampaignsTags", updateParams, clientLogin); err != nil {
			return nil, fmt.Errorf("не удалось обновить каталог: %w", err)
		}
		// Re-read to pull fresh TagIDs for newly created tags
		raw, err = client.CallV4(ctx, token, "GetCampaignsTags", getParams, clientLogin)
		if err != nil {
			return nil, fmt.Errorf("не удалось перечитать каталог: %w", err)
		}
		var refreshResp v4CampaignTagsResponse
		if err := json.Unmarshal(raw, &refreshResp); err != nil {
			return nil, fmt.Errorf("ошибка парсинга каталога: %w", err)
		}
		existingMap = make(map[string]int64)
		for _, ct := range refreshResp.Data {
			if ct.CampaignID == campaignID {
				for _, t := range ct.Tags {
					existingMap[strings.ToLower(t.Tag)] = t.TagID
				}
			}
		}
	}

	out := make([]int64, 0, len(names))
	for _, name := range names {
		id, ok := existingMap[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("метка '%s' не найдена после создания", name)
		}
		out = append(out, id)
	}
	return out, nil
}
