package vk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

var vkRefCache = common.NewCache(6 * time.Hour)

// RegisterTargetingTools registers targeting reference tools.
func RegisterTargetingTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerVKGetTargetingsTree(s, client, resolver)
	registerVKGetRegions(s, client, resolver)
}

func registerVKGetTargetingsTree(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_targetings_tree",
		mcp.WithDescription("Поиск интересов и хобби для таргетинга VK Ads по названию. ОБЯЗАТЕЛЬНО укажи query. Возвращает до 30 совпадений с ID для таргетинга."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("query", mcp.Description("Название интереса для поиска (например: авто, недвижимость, спорт)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		query := strings.ToLower(strings.TrimSpace(common.GetString(req, "query")))
		if query == "" {
			return common.ErrorResult("query обязателен — укажи название интереса для поиска"), nil
		}

		// Try cache first
		cacheKey := "vk_targetings_tree"
		treeData := vkRefCache.Get(cacheKey)

		if treeData == nil {
			result, err := client.Get(ctx, token, "/targetings/tree.json", nil)
			if err != nil {
				return common.ErrorResult(err.Error()), nil
			}
			treeData = result
			vkRefCache.Set(cacheKey, treeData)
		}

		// Search the tree
		matched := searchTree(json.RawMessage(treeData), query, 30)
		if len(matched) == 0 {
			return common.TextResult(fmt.Sprintf("Интересы по запросу '%s' не найдены", query)), nil
		}

		data, _ := json.Marshal(matched)
		return common.TextResult(string(data)), nil
	})
}

type targetingNode struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID int    `json:"parent_id,omitempty"`
}

func searchTree(raw json.RawMessage, query string, limit int) []targetingNode {
	var results []targetingNode
	searchJSON(raw, query, &results, limit, 0)
	return results
}

func searchJSON(raw json.RawMessage, query string, results *[]targetingNode, limit int, parentID int) {
	if len(*results) >= limit {
		return
	}

	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		for _, item := range arr {
			searchJSON(item, query, results, limit, parentID)
			if len(*results) >= limit {
				return
			}
		}
		return
	}

	var node struct {
		ID       int             `json:"id"`
		Name     string          `json:"name"`
		Children json.RawMessage `json:"children"`
		Items    json.RawMessage `json:"items"`
	}
	if json.Unmarshal(raw, &node) == nil && node.Name != "" {
		if strings.Contains(strings.ToLower(node.Name), query) {
			*results = append(*results, targetingNode{
				ID:       node.ID,
				Name:     node.Name,
				ParentID: parentID,
			})
		}
		if len(node.Children) > 0 {
			searchJSON(node.Children, query, results, limit, node.ID)
		}
		if len(node.Items) > 0 {
			searchJSON(node.Items, query, results, limit, node.ID)
		}
	}
}

func registerVKGetRegions(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_regions",
		mcp.WithDescription("Поиск регионов VK Ads по названию для гео-таргетинга. ОБЯЗАТЕЛЬНО укажи search."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("search", mcp.Description("Название региона/города (например: Москва, Краснодар)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		search := common.GetString(req, "search")
		if search == "" {
			return common.ErrorResult("search обязателен — укажи название региона"), nil
		}

		params := url.Values{}
		params.Set("q", search)

		result, err := client.Get(ctx, token, "/regions.json", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}
