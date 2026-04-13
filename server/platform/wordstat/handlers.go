package wordstat

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

var wordstatCache = common.NewCache(6 * time.Hour)

// RegisterHandlers registers all Wordstat tool handlers.
func RegisterHandlers(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerCheckSearchVolume(s, client, resolver)
	registerWordstatDynamics(s, client, resolver)
	registerWordstatRegions(s, client, resolver)
	registerWordstatRegionsTree(s, client, resolver)
	registerWordstatUserInfo(s, client, resolver)
}

func registerCheckSearchVolume(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("check_search_volume",
		mcp.WithDescription("Проверить частотность фраз через Яндекс Вордстат. Показывает количество запросов в месяц + похожие фразы. До 128 фраз за раз."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("phrases", mcp.Description("Фразы через запятую. Операторы: \"точная фраза\", [точный порядок]"), mcp.Required()),
		mcp.WithString("region_ids", mcp.Description("ID регионов через запятую (опционально)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		phrases := common.GetStringSlice(req, "phrases")

		param := map[string]any{
			"Phrases": phrases,
		}
		if regions := common.GetStringSlice(req, "region_ids"); len(regions) > 0 {
			param["GeoID"] = regions
		}

		// CreateNewWordstatReport
		resp, err := client.Call(ctx, token, "CreateNewWordstatReport", param)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		// API v4 returns {"data": <reportID>} — extract report ID
		var envelope struct {
			Data int `json:"data"`
		}
		if err := json.Unmarshal(resp, &envelope); err != nil || envelope.Data == 0 {
			return common.TextResult(string(resp)), nil
		}
		reportID := envelope.Data

		// Poll for report — GetWordstatReport (with delay between attempts)
		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			result, err := client.Call(ctx, token, "GetWordstatReport", reportID)
			if err != nil {
				// HTTP error — wait and retry
				continue
			}

			// Check if API returned an error in JSON body (v4 format)
			var apiErr struct {
				ErrorCode int `json:"error_code"`
			}
			if err := json.Unmarshal(result, &apiErr); err == nil && apiErr.ErrorCode != 0 {
				// Report still generating (error_code 92) — continue polling
				continue
			}

			// Delete report after getting results
			_, _ = client.Call(ctx, token, "DeleteWordstatReport", reportID)

			// Extract data field from v4 response
			var dataEnvelope struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(result, &dataEnvelope); err == nil && len(dataEnvelope.Data) > 0 {
				return common.SafeTextResult(string(dataEnvelope.Data)), nil
			}
			return common.SafeTextResult(string(result)), nil
		}

		return common.ErrorResult("Wordstat report timeout — попробуйте позже"), nil
	})
}

func registerWordstatDynamics(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("wordstat_dynamics",
		mcp.WithDescription("Динамика запросов в Вордстате: тренды по дням/неделям/месяцам. Историческое с 2018 года."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("phrase", mcp.Description("Фраза для анализа"), mcp.Required()),
		mcp.WithString("date_from", mcp.Description("Начало периода (YYYY-MM-DD)")),
		mcp.WithString("date_to", mcp.Description("Конец периода (YYYY-MM-DD)")),
		mcp.WithString("region_ids", mcp.Description("ID регионов через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		param := map[string]any{
			"Phrases": []string{common.GetString(req, "phrase")},
		}
		if df := common.GetString(req, "date_from"); df != "" {
			param["DateFrom"] = df
		}
		if dt := common.GetString(req, "date_to"); dt != "" {
			param["DateTo"] = dt
		}
		if regions := common.GetStringSlice(req, "region_ids"); len(regions) > 0 {
			param["GeoID"] = regions
		}

		resp, err := client.Call(ctx, token, "CreateNewWordstatReport", param)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		// Extract report ID and poll for results
		var envelope struct {
			Data int `json:"data"`
		}
		if err := json.Unmarshal(resp, &envelope); err != nil || envelope.Data == 0 {
			return common.TextResult(string(resp)), nil
		}
		reportID := envelope.Data

		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			result, err := client.Call(ctx, token, "GetWordstatReport", reportID)
			if err != nil {
				continue
			}
			var apiErr struct {
				ErrorCode int `json:"error_code"`
			}
			if err := json.Unmarshal(result, &apiErr); err == nil && apiErr.ErrorCode != 0 {
				continue
			}
			_, _ = client.Call(ctx, token, "DeleteWordstatReport", reportID)
			var dataEnvelope struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(result, &dataEnvelope); err == nil && len(dataEnvelope.Data) > 0 {
				return common.SafeTextResult(string(dataEnvelope.Data)), nil
			}
			return common.SafeTextResult(string(result)), nil
		}

		return common.ErrorResult("Wordstat report timeout — попробуйте позже"), nil
	})
}

func registerWordstatRegions(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("wordstat_regions",
		mcp.WithDescription("Региональный спрос в Вордстате: индекс интереса по регионам."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("phrase", mcp.Description("Фраза для анализа"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		param := map[string]any{
			"Phrases": []string{common.GetString(req, "phrase")},
		}

		resp, err := client.Call(ctx, token, "CreateNewWordstatReport", param)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		// Extract report ID and poll for results
		var envelope struct {
			Data int `json:"data"`
		}
		if err := json.Unmarshal(resp, &envelope); err != nil || envelope.Data == 0 {
			return common.TextResult(string(resp)), nil
		}
		reportID := envelope.Data

		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			result, err := client.Call(ctx, token, "GetWordstatReport", reportID)
			if err != nil {
				continue
			}
			var apiErr struct {
				ErrorCode int `json:"error_code"`
			}
			if err := json.Unmarshal(result, &apiErr); err == nil && apiErr.ErrorCode != 0 {
				continue
			}
			_, _ = client.Call(ctx, token, "DeleteWordstatReport", reportID)
			var dataEnvelope struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(result, &dataEnvelope); err == nil && len(dataEnvelope.Data) > 0 {
				return common.SafeTextResult(string(dataEnvelope.Data)), nil
			}
			return common.SafeTextResult(string(result)), nil
		}

		return common.ErrorResult("Wordstat report timeout — попробуйте позже"), nil
	})
}

func registerWordstatRegionsTree(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("wordstat_regions_tree",
		mcp.WithDescription("Поиск регионов Вордстата по названию. ОБЯЗАТЕЛЬНО укажи query. Возвращает ID для фильтрации по регионам."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
		mcp.WithString("query", mcp.Description("Название региона для поиска (например: Москва, Новосибирск)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		query := strings.ToLower(strings.TrimSpace(common.GetString(req, "query")))
		if query == "" {
			return common.ErrorResult("query обязателен — укажи название региона"), nil
		}

		// Try cache first
		cacheKey := "wordstat_regions"
		resp := wordstatCache.Get(cacheKey)

		if resp == nil {
			var err error
			resp, err = client.Call(ctx, token, "GetRegions", nil)
			if err != nil {
				return common.ErrorResult(err.Error()), nil
			}
			wordstatCache.Set(cacheKey, resp)
		}

		// Parse and filter regions by query
		var regions []wordstatRegion
		if err := json.Unmarshal(resp, &regions); err != nil {
			return common.SafeTextResult(string(resp)), nil
		}

		var matched []wordstatRegion
		for _, r := range regions {
			if strings.Contains(strings.ToLower(r.RegionName), query) {
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

type wordstatRegion struct {
	RegionID   int    `json:"RegionID"`
	RegionName string `json:"RegionName"`
	ParentID   int    `json:"ParentID,omitempty"`
}

func registerWordstatUserInfo(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("wordstat_user_info",
		mcp.WithDescription("Информация о квоте Вордстат API: дневные лимиты и оставшиеся запросы."),
		mcp.WithString("account", mcp.Description("Имя аккаунта (опционально)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		resp, err := client.Call(ctx, token, "GetWordstatReportList", nil)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		return common.TextResult(string(resp)), nil
	})
}
