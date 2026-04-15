package direct

import (
	"context"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterContentTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerGetAdImages(s, client, resolver)
	registerGetVCards(s, client, resolver)
	registerAddVCard(s, client, resolver)
	registerDeleteVCards(s, client, resolver)
}

func registerGetAdImages(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_ad_images",
		mcp.WithDescription("Получить изображения для объявлений (РСЯ). Возвращает хеши для использования в ad_image_hash."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		params := map[string]any{
			"SelectionCriteria": map[string]any{},
			"FieldNames":        []string{"AdImageHash", "Name", "Type", "Associated"},
			"Page":              map[string]int{"Limit": 50},
		}
		raw, err := client.Call(ctx, token, "adimages", "get", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerGetVCards(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("get_vcards",
		mcp.WithDescription("Получить визитки (адрес, телефон, компания) для объявлений."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("vcard_ids", mcp.Description("ID визиток через запятую (опционально)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "vcard_ids"))
		if len(ids) == 0 {
			return common.ErrorResult("Укажи vcard_ids. Чтобы узнать ID визиток, посмотри VCardId в объявлениях через get_ads."), nil
		}
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
			"FieldNames":        []string{"Id", "CompanyName", "Phone", "Street", "City", "WorkTime"},
		}
		raw, err := client.Call(ctx, token, "vcards", "get", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerAddVCard(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_vcard",
		mcp.WithDescription("Создать визитку для объявлений."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("company_name", mcp.Description("Название компании"), mcp.Required()),
		mcp.WithString("phone_country", mcp.Description("Код страны (по умолчанию +7)")),
		mcp.WithString("phone_city", mcp.Description("Код города"), mcp.Required()),
		mcp.WithString("phone_number", mcp.Description("Номер телефона"), mcp.Required()),
		mcp.WithString("city", mcp.Description("Город")),
		mcp.WithString("street", mcp.Description("Улица и дом")),
		mcp.WithString("work_time", mcp.Description("Время работы: 0;1;2;3;4;5;6;9;00;00;18;00")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		phoneCountry := common.GetString(req, "phone_country")
		if phoneCountry == "" {
			phoneCountry = "+7"
		}

		vcard := map[string]any{
			"CompanyName": common.GetString(req, "company_name"),
			"Phone": map[string]any{
				"CountryCode": phoneCountry,
				"CityCode":    common.GetString(req, "phone_city"),
				"PhoneNumber": common.GetString(req, "phone_number"),
			},
		}
		if city := common.GetString(req, "city"); city != "" {
			vcard["City"] = city
		}
		if street := common.GetString(req, "street"); street != "" {
			vcard["Street"] = street
		}
		if wt := common.GetString(req, "work_time"); wt != "" {
			vcard["WorkTime"] = wt
		}

		params := map[string]any{"VCards": []any{vcard}}
		raw, err := client.Call(ctx, token, "vcards", "add", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}

func registerDeleteVCards(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("delete_vcards",
		mcp.WithDescription("Удалить визитки."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города")),
		mcp.WithString("vcard_ids", mcp.Description("ID визиток через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")

		ids := parseIntSlice(common.GetStringSlice(req, "vcard_ids"))
		params := map[string]any{
			"SelectionCriteria": map[string]any{"Ids": ids},
		}
		raw, err := client.Call(ctx, token, "vcards", "delete", params, clientLogin)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		result, err := GetResult(raw)
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		return common.TextResult(string(result)), nil
	})
}
