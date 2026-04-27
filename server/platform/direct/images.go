package direct

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
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
		mcp.WithDescription("Загрузить изображение в Яндекс Директ (для РСЯ-объявлений). Источник: url, file_path или base64. Возвращает AdImageHash для использования в add_ad. Перед отправкой в Директ выполняется локальная валидация: формат (JPG/PNG/GIF), вес (10 KB — 10 MB), пиксели и соотношение сторон (1:1 ≥ 450×450 или 16:9 ≥ 1080×607, ±2%). При несоответствии — ошибка с явной причиной до похода в API."),
		mcp.WithString("account", mcp.Description("Аккаунт")),
		mcp.WithString("client_login", mcp.Description("Логин клиента-города"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Имя изображения (до 100 символов, для ориентировки в кабинете)")),
		mcp.WithString("url", mcp.Description("HTTPS URL изображения — скачается и будет загружено в Директ")),
		mcp.WithString("file_path", mcp.Description("Локальный путь к JPG/PNG файлу (альтернатива url)")),
		mcp.WithString("image_base64", mcp.Description("Base64 содержимое изображения (альтернатива url/file_path). БЕЗ data:image/... префикса.")),
		mcp.WithBoolean("skip_validation", mcp.Description("Не проверять размеры локально — отправить в Директ как есть (дефолт false). Включай только для отладки.")),
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

		var rawData []byte
		switch {
		case b64 != "":
			// Strip data URL prefix if user accidentally included it.
			if idx := strings.Index(b64, "base64,"); idx >= 0 {
				b64 = b64[idx+len("base64,"):]
			}
			decoded, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return common.ErrorResult(fmt.Sprintf("decode base64: %v", err)), nil
			}
			rawData = decoded
		case filePath != "":
			data, err := os.ReadFile(filePath)
			if err != nil {
				return common.ErrorResult(fmt.Sprintf("read file_path=%s: %v", filePath, err)), nil
			}
			rawData = data
		case url != "":
			data, err := downloadImage(ctx, url)
			if err != nil {
				return common.ErrorResult(fmt.Sprintf("download url=%s: %v", url, err)), nil
			}
			rawData = data
		default:
			return common.ErrorResult("нужен один из: url, file_path, image_base64"), nil
		}

		if len(rawData) == 0 {
			return common.ErrorResult("пустое изображение после загрузки"), nil
		}

		// Local validation against Yandex Direct's image requirements before
		// burning an API call that returns the unhelpful "Размер изображения
		// некорректен" error.
		skipValidation := common.GetBool(req, "skip_validation")
		if !skipValidation {
			if verr := validateAdImage(rawData); verr != nil {
				return common.ErrorResult(fmt.Sprintf("local image validation failed: %v — fix locally (resize/crop/recompress) before retrying, or pass skip_validation=true to bypass", verr)), nil
			}
		}

		imageBase64 := base64.StdEncoding.EncodeToString(rawData)

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

// Yandex Direct image limits for AdImages.add (graphic ad creatives).
// Source: yandex.ru/dev/direct/doc — "Требования к изображениям, загружаемым через API".
//
// We enforce locally to fail fast with a useful error message instead of
// burning an API round-trip and getting "Размер изображения некорректен".
const (
	minImageBytes = 10 * 1024        // 10 KB
	maxImageBytes = 10 * 1024 * 1024 // 10 MB

	// Aspect 1:1 — square creatives.
	minSide11 = 450
	maxSide11 = 5000

	// Aspect 16:9 — wide creatives.
	minWidth169  = 1080
	minHeight169 = 607
	maxWidth169  = 5000
	maxHeight169 = 2812

	// Tolerance on aspect ratio (±2% — same as imagegen warning threshold).
	aspectTolerance = 0.02
)

// validateAdImage checks raw image bytes against Yandex Direct's published
// requirements. Returns nil if the image will be accepted, or a descriptive
// error explaining the exact failure (format / weight / pixels / aspect).
func validateAdImage(data []byte) error {
	// File weight.
	if len(data) < minImageBytes {
		return fmt.Errorf("file too small (%d bytes, min %d) — model probably returned a corrupted/tiny image",
			len(data), minImageBytes)
	}
	if len(data) > maxImageBytes {
		return fmt.Errorf("file too large (%d bytes, max %d)", len(data), maxImageBytes)
	}

	// Format + dimensions via DecodeConfig (no full pixel decode).
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("not a valid image (decode failed: %v) — Yandex Direct accepts JPG, PNG, GIF only", err)
	}
	switch format {
	case "jpeg", "png", "gif":
		// ok
	default:
		return fmt.Errorf("unsupported format %q — Yandex Direct accepts JPG, PNG, GIF", format)
	}

	w, h := cfg.Width, cfg.Height
	if w <= 0 || h <= 0 {
		return fmt.Errorf("invalid dimensions %dx%d", w, h)
	}

	// Match against the two ratios Yandex Direct accepts for graphic ads.
	got := float64(w) / float64(h)
	want11 := 1.0
	want169 := 16.0 / 9.0

	delta11 := math.Abs(got-want11) / want11
	delta169 := math.Abs(got-want169) / want169

	switch {
	case delta11 <= aspectTolerance:
		// 1:1 — check pixel bounds.
		if w < minSide11 || h < minSide11 {
			return fmt.Errorf("aspect 1:1 detected (%dx%d) but below minimum %dx%d", w, h, minSide11, minSide11)
		}
		if w > maxSide11 || h > maxSide11 {
			return fmt.Errorf("aspect 1:1 detected (%dx%d) but above maximum %dx%d", w, h, maxSide11, maxSide11)
		}
	case delta169 <= aspectTolerance:
		// 16:9 — check pixel bounds.
		if w < minWidth169 || h < minHeight169 {
			return fmt.Errorf("aspect 16:9 detected (%dx%d) but below minimum %dx%d", w, h, minWidth169, minHeight169)
		}
		if w > maxWidth169 || h > maxHeight169 {
			return fmt.Errorf("aspect 16:9 detected (%dx%d) but above maximum %dx%d", w, h, maxWidth169, maxHeight169)
		}
	default:
		return fmt.Errorf("aspect ratio %.3f (%dx%d) is neither 1:1 (≈1.000) nor 16:9 (≈1.778) within ±%.0f%% — Yandex Direct rejects other ratios. Image generation model likely ignored the size hint; resize/crop locally before upload",
			got, w, h, aspectTolerance*100)
	}
	return nil
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
