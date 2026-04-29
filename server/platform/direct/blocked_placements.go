package direct

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterBlockedPlacementsTools registers write-tools for blocked placements (excludedsites API).
// The read-only get_blocked_placements is in references.go; this file adds the apply tool.
func RegisterBlockedPlacementsTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerApplyBlockedPlacements(s, client, resolver)
	registerSetExcludedSites(s, client, resolver)
}

// apply_blocked_placements — applies the standard Etazhi RSYA blacklist (~400 placements)
// to a given campaign. One-call convenience wrapper around set_excluded_sites.
func registerApplyBlockedPlacements(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("apply_blocked_placements",
		mcp.WithDescription(
			"Применить стандартный чёрный список РСЯ-площадок Этажи (~400 сайтов: игры, мусорные приложения, развлекательные домены) "+
				"к указанной кампании через API Яндекс Директа (excludedsites.set). "+
				"Вызывай ОДИН РАЗ при создании РСЯ-кампании сразу после add_campaign — это страховка от слива бюджета "+
				"на бесполезные площадки в первые дни обучения автостратегии. "+
				"Для кастомного списка используй set_excluded_sites."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (город). Получи через get_agency_clients."), mcp.Required()),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании, к которой применить список."), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		campaignID := common.GetInt(req, "campaign_id")
		if campaignID <= 0 {
			return common.ErrorResult("campaign_id обязателен и должен быть > 0"), nil
		}

		params := map[string]any{
			"SetItems": []any{
				map[string]any{
					"CampaignId":    campaignID,
					"ExcludedSites": blockedPlacements,
				},
			},
		}

		raw, err := client.Call(ctx, token, "excludedsites", "set", params, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("excludedsites.set failed: %v", err)), nil
		}

		// API returns {"result": {"SetResults": [{"Warnings"?, "Errors"?}]}} on success
		out, _ := json.MarshalIndent(map[string]any{
			"campaign_id":         campaignID,
			"applied_count":       len(blockedPlacements),
			"source":              "Etazhi standard RSYA blacklist (references_data.go)",
			"api_response":        json.RawMessage(raw),
		}, "", "  ")
		return common.TextResult(string(out)), nil
	})
}

// set_excluded_sites — низкоуровневый инструмент: применить произвольный список площадок к кампании.
// Используй apply_blocked_placements для дефолтного списка Этажи.
func registerSetExcludedSites(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("set_excluded_sites",
		mcp.WithDescription(
			"Установить произвольный список заблокированных площадок (excluded_sites) для кампании через excludedsites.set. "+
				"Заменяет существующий список целиком (НЕ дополняет — будь внимателен). "+
				"Лимит API: 1000 хостов на кампанию. Для дефолтного списка Этажи используй apply_blocked_placements."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (город)"), mcp.Required()),
		mcp.WithNumber("campaign_id", mcp.Description("ID кампании"), mcp.Required()),
		mcp.WithString("sites", mcp.Description("Список хостов через запятую (например: 'site1.com,site2.com,partner.ru'). Лимит 1000."), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		campaignID := common.GetInt(req, "campaign_id")
		sites := common.GetStringSlice(req, "sites")
		if campaignID <= 0 {
			return common.ErrorResult("campaign_id обязателен и должен быть > 0"), nil
		}
		if len(sites) == 0 {
			return common.ErrorResult("sites обязательно и не может быть пустым"), nil
		}
		if len(sites) > 1000 {
			return common.ErrorResult(fmt.Sprintf("sites: %d > 1000 — превышен лимит API. Сократи список.", len(sites))), nil
		}

		params := map[string]any{
			"SetItems": []any{
				map[string]any{
					"CampaignId":    campaignID,
					"ExcludedSites": sites,
				},
			},
		}

		raw, err := client.Call(ctx, token, "excludedsites", "set", params, clientLogin)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("excludedsites.set failed: %v", err)), nil
		}

		out, _ := json.MarshalIndent(map[string]any{
			"campaign_id":   campaignID,
			"applied_count": len(sites),
			"api_response":  json.RawMessage(raw),
		}, "", "  ")
		return common.TextResult(string(out)), nil
	})
}
