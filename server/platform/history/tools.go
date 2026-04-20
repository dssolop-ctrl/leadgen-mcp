package history

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers MCP tools for the centralized change history.
func RegisterTools(s *server.MCPServer, store *Store) {
	registerLogChangeEvent(s, store)
	registerGetChangeHistory(s, store)
	registerUpdateDailySummary(s, store)
	registerGetDailySummary(s, store)
}

// --- log_change_event ---

func registerLogChangeEvent(s *server.MCPServer, store *Store) {
	tool := mcp.NewTool("log_change_event",
		mcp.WithDescription(
			"Записать в централизованную историю изменений кабинета критичное действие "+
				"(смена стратегии, цели, бюджета >30%, пауза/запуск, новое объявление). "+
				"Используется после успешного read-back. Для суточных итогов используй update_daily_summary."),
		mcp.WithString("agency_account",
			mcp.Description("Агентский аккаунт (например: 'etagi click')")),
		mcp.WithString("city_login",
			mcp.Description("Клиентский логин Директа города (обязательно)"),
			mcp.Required()),
		mcp.WithString("city",
			mcp.Description("Человекочитаемый слаг города: omsk, spb, ekb")),
		mcp.WithString("campaign_id",
			mcp.Description("ID кампании в Директе")),
		mcp.WithString("campaign_name",
			mcp.Description("Читаемое имя кампании")),
		mcp.WithString("entity_type",
			mcp.Description("Тип сущности: campaign|adgroup|ad|keyword|negative|bid|strategy|budget|target"),
			mcp.Required()),
		mcp.WithString("entity_id",
			mcp.Description("ID сущности, если применимо")),
		mcp.WithString("action_type",
			mcp.Description("Действие: create|update|pause|resume|archive|delete|moderate"),
			mcp.Required()),
		mcp.WithString("tool_name",
			mcp.Description("Имя MCP-инструмента, через который сделано (например update_campaign)")),
		mcp.WithString("before_value",
			mcp.Description("Состояние ДО (короткий JSON или текст)")),
		mcp.WithString("after_value",
			mcp.Description("Состояние ПОСЛЕ (короткий JSON или текст)")),
		mcp.WithString("reason",
			mcp.Description("Почему это изменение сделано")),
		mcp.WithString("operator_note",
			mcp.Description("Свободная заметка специалиста")),
		mcp.WithString("correlation_key",
			mcp.Description("Ключ сессии/задачи: связывает события одной работы (например UUID чата)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		e := ChangeEvent{
			AgencyAccount:  common.GetString(req, "agency_account"),
			CityLogin:      common.GetString(req, "city_login"),
			City:           common.GetString(req, "city"),
			CampaignID:     common.GetString(req, "campaign_id"),
			CampaignName:   common.GetString(req, "campaign_name"),
			EntityType:     common.GetString(req, "entity_type"),
			EntityID:       common.GetString(req, "entity_id"),
			ActionType:     common.GetString(req, "action_type"),
			ToolName:       common.GetString(req, "tool_name"),
			BeforeValue:    common.GetString(req, "before_value"),
			AfterValue:     common.GetString(req, "after_value"),
			Reason:         common.GetString(req, "reason"),
			OperatorNote:   common.GetString(req, "operator_note"),
			CorrelationKey: common.GetString(req, "correlation_key"),
		}
		id, err := store.LogEvent(e)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("log error: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf(
			"Change event logged (id=%d): %s/%s/%s → %s %s",
			id, e.CityLogin, e.CampaignID, e.EntityType, e.ActionType, e.EntityID)), nil
	})
}

// --- get_change_history ---

func registerGetChangeHistory(s *server.MCPServer, store *Store) {
	tool := mcp.NewTool("get_change_history",
		mcp.WithDescription(
			"Получить детальную историю изменений. Фильтры: по campaign_id / city_login / "+
				"agency_account / диапазону дат / correlation_key. Результат — последние события "+
				"в обратном порядке. Возвращается как JSON-массив."),
		mcp.WithString("campaign_id",
			mcp.Description("Фильтр по ID кампании")),
		mcp.WithString("city_login",
			mcp.Description("Фильтр по клиентскому логину города")),
		mcp.WithString("agency_account",
			mcp.Description("Фильтр по агентскому аккаунту")),
		mcp.WithString("date_from",
			mcp.Description("YYYY-MM-DD или RFC3339 — начало периода")),
		mcp.WithString("date_to",
			mcp.Description("YYYY-MM-DD или RFC3339 — конец периода")),
		mcp.WithString("correlation_key",
			mcp.Description("Фильтр по ключу задачи/сессии")),
		mcp.WithNumber("limit",
			mcp.Description("Максимум записей (default 100, max 500)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		events, err := store.QueryEvents(
			common.GetString(req, "campaign_id"),
			common.GetString(req, "city_login"),
			common.GetString(req, "agency_account"),
			common.GetString(req, "date_from"),
			common.GetString(req, "date_to"),
			common.GetString(req, "correlation_key"),
			common.GetInt(req, "limit"),
		)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
		}
		if len(events) == 0 {
			return mcp.NewToolResultText("История пуста: под эти фильтры записей нет."), nil
		}
		payload, err := json.MarshalIndent(map[string]any{
			"count":  len(events),
			"events": events,
		}, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode: %v", err)), nil
		}
		return mcp.NewToolResultText(string(payload)), nil
	})
}

// --- update_daily_summary ---

func registerUpdateDailySummary(s *server.MCPServer, store *Store) {
	tool := mcp.NewTool("update_daily_summary",
		mcp.WithDescription(
			"Обновить краткое суточное саммари для города. Одна запись на день на city_login. "+
				"По умолчанию добавляет к существующей записи (append=true). На новый день создаётся "+
				"новая запись — предыдущие не перезаписываются. Вызывать в конце рабочей сессии."),
		mcp.WithString("city_login",
			mcp.Description("Клиентский логин Директа города (обязательно)"),
			mcp.Required()),
		mcp.WithString("city",
			mcp.Description("Человекочитаемый слаг города: omsk, spb, ekb")),
		mcp.WithString("agency_account",
			mcp.Description("Агентский аккаунт")),
		mcp.WithString("summary",
			mcp.Description("Текст саммари: что сделано за день/сессию. Краткий и по делу."),
			mcp.Required()),
		mcp.WithString("operator_name",
			mcp.Description("Имя специалиста (для мульти-пользовательского учёта)")),
		mcp.WithNumber("event_count",
			mcp.Description("Сколько отдельных событий это саммари объединяет (default 1)")),
		mcp.WithString("date",
			mcp.Description("Дата YYYY-MM-DD (default — сегодня UTC)")),
		mcp.WithString("mode",
			mcp.Description("append (default) — дописать к сегодняшней записи; replace — заменить")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cityLogin := common.GetString(req, "city_login")
		summary := common.GetString(req, "summary")
		if cityLogin == "" || summary == "" {
			return mcp.NewToolResultError("city_login и summary обязательны"), nil
		}
		count := common.GetInt(req, "event_count")
		if count <= 0 {
			count = 1
		}
		mode := strings.ToLower(common.GetString(req, "mode"))
		appendMode := mode != "replace"

		d := DailySummary{
			Date:          common.GetString(req, "date"),
			AgencyAccount: common.GetString(req, "agency_account"),
			CityLogin:     cityLogin,
			City:          common.GetString(req, "city"),
			Summary:       summary,
			OperatorName:  common.GetString(req, "operator_name"),
			EventCount:    count,
		}
		res, err := store.UpsertDailySummary(d, appendMode)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update error: %v", err)), nil
		}
		payload, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(payload)), nil
	})
}

// --- get_daily_summary ---

func registerGetDailySummary(s *server.MCPServer, store *Store) {
	tool := mcp.NewTool("get_daily_summary",
		mcp.WithDescription(
			"Получить суточные саммари по городу (что делали и когда). Без city_login — "+
				"возвращает недавние записи по всем городам."),
		mcp.WithString("city_login",
			mcp.Description("Клиентский логин города (опционально)")),
		mcp.WithString("date_from",
			mcp.Description("YYYY-MM-DD — начало периода (опционально)")),
		mcp.WithString("date_to",
			mcp.Description("YYYY-MM-DD — конец периода (опционально)")),
		mcp.WithNumber("limit",
			mcp.Description("Максимум записей (default 30, max 365)")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rows, err := store.GetDailySummaries(
			common.GetString(req, "city_login"),
			common.GetString(req, "date_from"),
			common.GetString(req, "date_to"),
			common.GetInt(req, "limit"),
		)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
		}
		if len(rows) == 0 {
			return mcp.NewToolResultText("Саммари за указанный период не найдены."), nil
		}
		payload, err := json.MarshalIndent(map[string]any{
			"count":   len(rows),
			"results": rows,
		}, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode: %v", err)), nil
		}
		return mcp.NewToolResultText(string(payload)), nil
	})
}
