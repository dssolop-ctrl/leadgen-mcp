package filters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers the 3 site-filter MCP tools.
func RegisterTools(s *server.MCPServer, store *Store) {
	registerBuildLandingURL(s, store)
	registerUpsertSiteFilters(s, store)
	registerGetSiteFilters(s, store)
}

// --- build_landing_url ---

func registerBuildLandingURL(s *server.MCPServer, store *Store) {
	tool := mcp.NewTool("build_landing_url",
		mcp.WithDescription("Собрать URL посадочной: домен города + путь тематики + фильтры из БД. Резолвит названия в URL-значения."),
		mcp.WithString("city",
			mcp.Description("Слаг города: omsk, spb, ekb"),
			mcp.Required()),
		mcp.WithString("theme",
			mcp.Description("Тематика: вторичка, загородка, новостройки, аренда, ипотека, коммерческая"),
			mcp.Required()),
		mcp.WithString("rooms",
			mcp.Description("Комнаты: 1, 2, 3, >4. Через запятую: '1,2'")),
		mcp.WithString("studio",
			mcp.Description("Студии: 'true'")),
		mcp.WithString("district",
			mcp.Description("Район (резолвится в ID из БД). Через запятую.")),
		mcp.WithString("price_min",
			mcp.Description("Мин. цена")),
		mcp.WithString("price_max",
			mcp.Description("Макс. цена")),
		mcp.WithString("square_min",
			mcp.Description("Мин. площадь м²")),
		mcp.WithString("square_max",
			mcp.Description("Макс. площадь м²")),
		mcp.WithString("property_type",
			mcp.Description("Тип объекта вторичка (из БД): квартира, апартаменты")),
		mcp.WithString("property_type_out",
			mcp.Description("Тип объекта загородка (из БД): дом, дача, участок")),
		mcp.WithString("property_type_commerce",
			mcp.Description("Тип объекта коммерческая (из БД): офис, склад")),
		mcp.WithString("old_price",
			mcp.Description("Фильтр скидки: мин. старая цена")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		city := common.GetString(req, "city")
		theme := common.GetString(req, "theme")
		if city == "" || theme == "" {
			return mcp.NewToolResultError("city and theme are required"), nil
		}

		filterParams := []string{
			"rooms", "studio", "district",
			"price_min", "price_max", "square_min", "square_max",
			"property_type", "property_type_out", "property_type_commerce",
			"old_price",
		}
		filters := make(map[string]string)
		for _, param := range filterParams {
			if v := common.GetString(req, param); v != "" {
				filters[param] = v
			}
		}

		result, err := store.BuildLandingURL(city, theme, filters)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		// Return info about which ID-based filters were resolved or skipped.
		var notes []string
		idFilters := []string{"district", "property_type", "property_type_out", "property_type_commerce"}
		for _, f := range idFilters {
			if val, ok := filters[f]; ok {
				for _, v := range strings.Split(val, ",") {
					v = strings.TrimSpace(v)
					if v == "" {
						continue
					}
					fv, err := store.ResolveFilterValue(city, f, v)
					if err != nil {
						notes = append(notes, fmt.Sprintf("WARNING: %s %q not found in DB for %s — filter skipped. Use upsert_site_filters to add it.", f, v, city))
					} else {
						notes = append(notes, fmt.Sprintf("%s: %s → %s", f, fv.Name, fv.URLValue))
					}
				}
			}
		}

		output := result
		if len(notes) > 0 {
			output += "\n\n" + strings.Join(notes, "\n")
		}

		return mcp.NewToolResultText(output), nil
	})
}

// --- upsert_site_filters ---

func registerUpsertSiteFilters(s *server.MCPServer, store *Store) {
	tool := mcp.NewTool("upsert_site_filters",
		mcp.WithDescription("Добавить/обновить фильтры сайта в БД. Одиночный или bulk (filters_json)."),
		mcp.WithString("city",
			mcp.Description("Слаг города (для одиночного)")),
		mcp.WithString("filter_type",
			mcp.Description("Тип фильтра: rooms, district, price_from и др. (для одиночного)")),
		mcp.WithString("name",
			mcp.Description("Название: 'Нефтяники', '1-комнатная' (для одиночного)")),
		mcp.WithString("url_value",
			mcp.Description("Значение URL-параметра: '42', '1' (для одиночного)")),
		mcp.WithString("aliases",
			mcp.Description("Алиасы через запятую: 'нефтяники,нефтяник'")),
		mcp.WithString("filters_json",
			mcp.Description("Bulk: JSON [{\"city_slug\":\"\",\"filter_type\":\"\",\"name\":\"\",\"url_value\":\"\",\"aliases\":\"\"}]")),
		mcp.WithString("filter_type_name",
			mcp.Description("Новый тип фильтра: имя. С filter_type_param.")),
		mcp.WithString("filter_type_param",
			mcp.Description("Новый тип фильтра: URL-параметр. С filter_type_name.")),
		mcp.WithString("filter_type_value_type",
			mcp.Description("Тип значения: int, string, id, bool. Умолч: string.")),
		mcp.WithString("url_template_theme",
			mcp.Description("URL-шаблон: тематика. С url_template_path.")),
		mcp.WithString("url_template_path",
			mcp.Description("URL-шаблон: путь (напр. '/realty/'). С url_template_theme.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// New filter type?
		if ftName := common.GetString(req, "filter_type_name"); ftName != "" {
			ftParam := common.GetString(req, "filter_type_param")
			ftValType := common.GetString(req, "filter_type_value_type")
			if ftParam == "" {
				return mcp.NewToolResultError("filter_type_param is required with filter_type_name"), nil
			}
			if ftValType == "" {
				ftValType = "string"
			}
			if err := store.UpsertFilterType(ftName, ftParam, ftValType); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error creating filter type: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Filter type created/updated: %s (param=%s, type=%s)", ftName, ftParam, ftValType)), nil
		}

		// New URL template?
		if theme := common.GetString(req, "url_template_theme"); theme != "" {
			path := common.GetString(req, "url_template_path")
			if path == "" {
				return mcp.NewToolResultError("url_template_path is required with url_template_theme"), nil
			}
			if err := store.UpsertTemplate(theme, path); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error creating template: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("URL template created/updated: %s → %s", theme, path)), nil
		}

		// Bulk import?
		if jsonStr := common.GetString(req, "filters_json"); jsonStr != "" {
			var values []FilterValue
			if err := json.Unmarshal([]byte(jsonStr), &values); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid filters_json: %v", err)), nil
			}
			count, err := store.BulkUpsert(values)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Bulk upsert error after %d records: %v", count, err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Bulk upsert complete: %d filter values imported", count)), nil
		}

		// Single upsert.
		city := common.GetString(req, "city")
		filterType := common.GetString(req, "filter_type")
		name := common.GetString(req, "name")
		urlValue := common.GetString(req, "url_value")
		aliases := common.GetString(req, "aliases")

		if city == "" || filterType == "" || name == "" || urlValue == "" {
			return mcp.NewToolResultError("Required: city, filter_type, name, url_value (or use filters_json for bulk)"), nil
		}

		if err := store.UpsertFilterValue(city, filterType, name, urlValue, aliases); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Filter value saved: %s/%s: %s = %s", city, filterType, name, urlValue)), nil
	})
}

// --- get_site_filters ---

func registerGetSiteFilters(s *server.MCPServer, store *Store) {
	tool := mcp.NewTool("get_site_filters",
		mcp.WithDescription(
			"View available site filters. "+
				"Without params: shows database stats and available filter types. "+
				"With city: shows all filter values for that city. "+
				"With city + filter_type: shows specific filter values."),
		mcp.WithString("city",
			mcp.Description("City slug to show filters for. Optional — without it shows global stats.")),
		mcp.WithString("filter_type",
			mcp.Description("Filter by type: rooms, district, price_from, etc. Optional.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		city := common.GetString(req, "city")
		filterType := common.GetString(req, "filter_type")

		if city == "" {
			// Show global stats + filter types.
			stats, err := store.Stats()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
			}
			types, err := store.GetFilterTypes()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
			}
			var sb strings.Builder
			sb.WriteString("=== Database Stats ===\n")
			sb.WriteString(stats)
			sb.WriteString("\n=== Filter Types ===\n")
			for _, ft := range types {
				fmt.Fprintf(&sb, "  %s → URL param: %s (type: %s)\n", ft.Name, ft.URLParam, ft.ValueType)
			}
			return mcp.NewToolResultText(sb.String()), nil
		}

		// Show filter values for a city.
		values, err := store.GetFilterValues(city, filterType)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}
		if len(values) == 0 {
			msg := fmt.Sprintf("No filter values for city=%s", city)
			if filterType != "" {
				msg += fmt.Sprintf(", type=%s", filterType)
			}
			msg += ". Use upsert_site_filters to add data."
			return mcp.NewToolResultText(msg), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "Filters for %s", city)
		if filterType != "" {
			fmt.Fprintf(&sb, " (type: %s)", filterType)
		}
		fmt.Fprintf(&sb, ": %d values\n\n", len(values))

		currentType := ""
		for _, fv := range values {
			if fv.FilterType != currentType {
				currentType = fv.FilterType
				fmt.Fprintf(&sb, "--- %s ---\n", currentType)
			}
			line := fmt.Sprintf("  %s = %s", fv.Name, fv.URLValue)
			if fv.Aliases != "" {
				line += fmt.Sprintf("  (aliases: %s)", fv.Aliases)
			}
			sb.WriteString(line + "\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	})
}
