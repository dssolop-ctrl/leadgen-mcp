package vk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterAudienceTools registers remarketing, segments, and search phrases tools.
func RegisterAudienceTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerVKCreateRemarketingCounter(s, client, resolver)
	registerVKCreateCounterGoal(s, client, resolver)
	registerVKCreateRemarketingList(s, client, resolver)
	registerVKCreateSegment(s, client, resolver)
	registerVKManageSegmentRelations(s, client, resolver)
	registerVKCreateSearchPhrases(s, client, resolver)
	registerVKAddVKGroup(s, client, resolver)
	registerVKResolveURL(s, client, resolver)
	registerVKGetVKGroups(s, client, resolver)
}

func registerVKCreateRemarketingCounter(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_remarketing_counter",
		mcp.WithDescription("Создать пиксель/счётчик ретаргетинга VK."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("name", mcp.Description("Название счётчика"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{"name": common.GetString(req, "name")}
		result, err := client.Post(ctx, token, "/remarketing/counters.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKCreateCounterGoal(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_counter_goal",
		mcp.WithDescription("Создать цель для счётчика VK: url_substring и т.д."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Название цели"), mcp.Required()),
		mcp.WithString("goal_type", mcp.Description("Тип: url_substring, url_match, и т.д."), mcp.Required()),
		mcp.WithString("value", mcp.Description("Значение (напр. подстрока URL)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		counterID := common.GetInt(req, "counter_id")
		body := map[string]any{
			"name":      common.GetString(req, "name"),
			"goal_type": common.GetString(req, "goal_type"),
			"value":     common.GetString(req, "value"),
		}

		path := fmt.Sprintf("/remarketing/counters/%d/goals.json", counterID)
		result, err := client.Post(ctx, token, path, body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKCreateRemarketingList(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_remarketing_list",
		mcp.WithDescription("Создать список ретаргетинга из счётчика (напр. посетители за 30 дней)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithNumber("counter_id", mcp.Description("ID счётчика"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Название списка"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Тип: positive (по умолчанию)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"counter_id": common.GetInt(req, "counter_id"),
			"name":       common.GetString(req, "name"),
		}
		if t := common.GetString(req, "type"); t != "" {
			body["type"] = t
		}

		result, err := client.Post(ctx, token, "/remarketing/lists.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKCreateSegment(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_segment",
		mcp.WithDescription("Создать сегмент аудитории VK Ads."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("name", mcp.Description("Название сегмента"), mcp.Required()),
		mcp.WithString("pass_condition", mcp.Description("Условие: or, and (по умолчанию or)")),
		mcp.WithString("object_type", mcp.Description("Тип: remarketing_player, remarketing_vk_group, и т.д.")),
		mcp.WithNumber("source_id", mcp.Description("ID источника (list_id, group_id)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"name": common.GetString(req, "name"),
		}
		if pc := common.GetString(req, "pass_condition"); pc != "" {
			body["pass_condition"] = pc
		}

		if ot := common.GetString(req, "object_type"); ot != "" {
			relation := map[string]any{"object_type": ot}
			if srcID := common.GetInt(req, "source_id"); srcID > 0 {
				relation["params"] = map[string]any{"source_id": srcID}
			}
			body["relations"] = []any{relation}
		}

		result, err := client.Post(ctx, token, "/remarketing/segments.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKManageSegmentRelations(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_manage_segment_relations",
		mcp.WithDescription("Обновить связи сегмента VK (добавить/удалить источники)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithNumber("segment_id", mcp.Description("ID сегмента"), mcp.Required()),
		mcp.WithString("object_type", mcp.Description("Тип: remarketing_player, remarketing_vk_group"), mcp.Required()),
		mcp.WithNumber("source_id", mcp.Description("ID источника"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		segmentID := common.GetInt(req, "segment_id")
		body := map[string]any{
			"relations": []any{
				map[string]any{
					"object_type": common.GetString(req, "object_type"),
					"params":      map[string]any{"source_id": common.GetInt(req, "source_id")},
				},
			},
		}

		path := fmt.Sprintf("/remarketing/segments/%d/relations.json", segmentID)
		result, err := client.Post(ctx, token, path, body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKCreateSearchPhrases(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_create_search_phrases",
		mcp.WithDescription("Создать контекстные фразы VK. Автоматически создаётся сегмент — используй его segment_id в targetings.segments."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("name", mcp.Description("Название списка фраз"), mcp.Required()),
		mcp.WithString("phrases", mcp.Description("Фразы через запятую"), mcp.Required()),
		mcp.WithString("stop_phrases", mcp.Description("Минус-фразы через запятую")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"name":    common.GetString(req, "name"),
			"phrases": common.GetStringSlice(req, "phrases"),
		}
		if sp := common.GetStringSlice(req, "stop_phrases"); len(sp) > 0 {
			body["stop_phrases"] = sp
		}

		result, err := client.Post(ctx, token, "/remarketing/search_phrases.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKAddVKGroup(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_add_vk_group",
		mcp.WithDescription("Зарегистрировать VK-сообщество для таргетинга. Используй object_id из vk_resolve_url."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithNumber("object_id", mcp.Description("ID VK-сообщества (из resolve_url)"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		body := map[string]any{
			"object_id": common.GetInt(req, "object_id"),
		}

		result, err := client.Post(ctx, token, "/vk_groups.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKResolveURL(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_resolve_url",
		mcp.WithDescription("Получить ID VK-сообщества по URL. Возвращает url_object_id = object_id."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("url", mcp.Description("URL VK-сообщества"), mcp.Required()),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		// VK API v2: POST /urls.json creates/resolves a URL object
		body := map[string]any{
			"url": common.GetString(req, "url"),
		}

		result, err := client.Post(ctx, token, "/urls.json", body)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerVKGetVKGroups(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("vk_get_vk_groups",
		mcp.WithDescription("Получить зарегистрированные VK-сообщества. Используй search для поиска."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("search", mcp.Description("Поиск по названию (обязательно)"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Лимит (по умолчанию 50)")),
		mcp.WithNumber("offset", mcp.Description("Смещение для пагинации")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveVK(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}

		params := url.Values{}
		if s := common.GetString(req, "search"); s != "" {
			params.Set("q", s)
		}
		limit := common.GetInt(req, "limit")
		if limit <= 0 {
			limit = 50
		}
		params.Set("limit", fmt.Sprintf("%d", limit))
		if offset := common.GetInt(req, "offset"); offset > 0 {
			params.Set("offset", fmt.Sprintf("%d", offset))
		}

		result, err := client.Get(ctx, token, "/vk_groups.json", params)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.SafeTextResult(string(result)), nil
	})
}
