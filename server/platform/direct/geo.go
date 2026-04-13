package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

var geoCache = common.NewCache(6 * time.Hour)

// RegisterGeoTools registers geo-related and dictionary MCP tools.
func RegisterGeoTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetGeoRegions(s, client, resolver)
	registerGetDictionaries(s, client, resolver)
}

func registerGetDictionaries(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_dictionaries",
		mcp.WithDescription("Получить справочники Яндекс Директа: GeoRegions, TimeZones, Currencies, AdCategories, OperationSystemVersions, ProductivityAssertions, SupplySidePlatforms, Interests, AudienceDemographicProfiles, AudienceCriteriaTypes."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов)")),
		mcp.WithString("dictionary_names", mcp.Description("Названия справочников через запятую: GeoRegions, TimeZones, Currencies, AdCategories, Interests"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		names := common.GetStringSlice(req, "dictionary_names")
		params := map[string]any{
			"DictionaryNames": names,
		}
		raw, err := client.Call(ctx, token, "dictionaries", "get", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}

func registerGetGeoRegions(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_geo_regions",
		mcp.WithDescription("Поиск регионов Яндекс Директа по названию. ОБЯЗАТЕЛЬНО укажи query для поиска. Возвращает до 20 результатов с GeoRegionId для таргетинга."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("client_login", mcp.Description("Логин клиента (для агентских аккаунтов). Получи через get_agency_clients.")),
		mcp.WithString("query", mcp.Description("Название региона/города для поиска (например: Новосибирск, Москва, Краснодарский край)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		query := strings.ToLower(strings.TrimSpace(common.GetString(req, "query")))

		if query == "" {
			return common.ErrorResult("query is required — укажи название региона для поиска"), nil
		}

		// Try cache first
		cacheKey := "direct_geo_regions"
		regionData := geoCache.Get(cacheKey)

		if regionData == nil {
			raw, err := client.Call(ctx, token, "dictionaries", "get", map[string]any{
				"DictionaryNames": []string{"GeoRegions"},
			}, clientLogin)
			if err != nil {
				return common.ErrorResult(err.Error()), nil
			}

			result, err := GetResult(raw)
			if err != nil {
				return common.ErrorResult(err.Error()), nil
			}
			regionData = result
			geoCache.Set(cacheKey, regionData)
		}

		// Parse and filter regions by query
		var dict struct {
			GeoRegions []geoRegion `json:"GeoRegions"`
		}
		if err := json.Unmarshal(regionData, &dict); err != nil {
			return common.ErrorResult(fmt.Sprintf("parse geo regions: %v", err)), nil
		}

		var matched []geoRegion
		for _, r := range dict.GeoRegions {
			if strings.Contains(strings.ToLower(r.GeoRegionName), query) {
				matched = append(matched, r)
				if len(matched) >= 20 {
					break
				}
			}
		}

		if len(matched) == 0 {
			return common.TextResult(fmt.Sprintf("Регионы по запросу '%s' не найдены", query)), nil
		}

		data, _ := json.Marshal(matched)
		return common.TextResult(string(data)), nil
	})
}

type geoRegion struct {
	GeoRegionID   int    `json:"GeoRegionId"`
	GeoRegionName string `json:"GeoRegionName"`
	GeoRegionType string `json:"GeoRegionType"`
	ParentID      *int   `json:"ParentId,omitempty"`
}
