package direct

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterImageTools registers add_ad_image and delete_ad_images MCP tools.
// These are РСЯ-specific creative management tools for the R6.5 / R7 steps
// of the leadgen skill — uploading generated banners to Yandex Direct.
func RegisterImageTools(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	registerAddAdImage(s, client, resolver)
	registerDeleteAdImages(s, client, resolver)
}

func registerAddAdImage(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("add_ad_image",
		mcp.WithDescription("Загрузить изображение в Яндекс Директ (для РСЯ-объявлений). Источник: url, file_path или base64. Возвращает AdImageHash для использования в add_ad."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Имя изображения (до 100 символов, для ориентировки в кабинете)")),
		mcp.WithString("url", mcp.Description("HTTPS URL изображения — скачается и будет загружено в Директ")),
		mcp.WithString("file_path", mcp.Description("Локальный путь к JPG/PNG файлу (альтернатива url)")),
		mcp.WithString("image_base64", mcp.Description("Base64 содержимое изображения (альтернатива url/file_path). БЕЗ data:image/... префикса.")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		if clientLogin == "" {
			return common.ErrorResult("client_login обязателен для add_ad_image"), nil
		}

		name := common.GetString(req, "name")
		url := strings.TrimSpace(common.GetString(req, "url"))
		filePath := strings.TrimSpace(common.GetString(req, "file_path"))
		b64 := strings.TrimSpace(common.GetString(req, "image_base64"))

		var imageBase64 string
		switch {
		case b64 != "":
			// Strip data URL prefix if user accidentally included it.
			if idx := strings.Index(b64, "base64,"); idx >= 0 {
				b64 = b64[idx+len("base64,"):]
			}
			imageBase64 = b64
		case filePath != "":
			data, err := os.ReadFile(filePath)
			if err != nil {
				return common.ErrorResult(fmt.Sprintf("read file_path=%s: %v", filePath, err)), nil
			}
			imageBase64 = base64.StdEncoding.EncodeToString(data)
		case url != "":
			data, err := downloadImage(ctx, url)
			if err != nil {
				return common.ErrorResult(fmt.Sprintf("download url=%s: %v", url, err)), nil
			}
			imageBase64 = base64.StdEncoding.EncodeToString(data)
		default:
			return common.ErrorResult("нужен один из: url, file_path, image_base64"), nil
		}

		if len(imageBase64) == 0 {
			return common.ErrorResult("пустое изображение после загрузки"), nil
		}

		imageAsset := map[string]any{
			"ImageData": imageBase64,
		}
		if name != "" {
			// Yandex limits name to 100 chars.
			if len(name) > 100 {
				name = name[:100]
			}
			imageAsset["Name"] = name
		}

		params := map[string]any{
			"AdImages": []any{imageAsset},
		}
		raw, err := client.Call(ctx, token, "adimages", "add", params, clientLogin)
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

func registerDeleteAdImages(s *mcpserver.MCPServer, client *Client, resolver *auth.AccountResolver) {
	tool := mcp.NewTool("delete_ad_images",
		mcp.WithDescription("Удалить изображения из Яндекс Директа по хешам. Не работает для Associated=YES (привязанных к объявлениям)."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города"), mcp.Required()),
		mcp.WithString("ad_image_hashes", mcp.Description("AdImageHash через запятую"), mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token, err := resolver.ResolveYandex(common.GetString(req, "account"))
		if err != nil {
			return common.ErrorResult(err.Error()), nil
		}
		clientLogin := common.GetString(req, "client_login")
		hashes := common.GetStringSlice(req, "ad_image_hashes")
		if len(hashes) == 0 {
			return common.ErrorResult("ad_image_hashes обязательны"), nil
		}

		params := map[string]any{
			"SelectionCriteria": map[string]any{
				"AdImageHashes": hashes,
			},
		}
		raw, err := client.Call(ctx, token, "adimages", "delete", params, clientLogin)
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

// downloadImage fetches an image URL and returns its bytes.
// Used by add_ad_image when user passes a URL instead of file/base64.
func downloadImage(ctx context.Context, url string) ([]byte, error) {
	httpClient := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	// Limit download size to 12 MB (Yandex allows up to 10 MB — leave a small buffer).
	const maxBytes = 12 * 1024 * 1024
	limited := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("image too large (>%d bytes)", maxBytes)
	}
	return data, nil
}
