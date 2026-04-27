package imagegen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leadgen-mcp/server/platform/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers generate_image and generate_banner_set on the MCP server.
// previewDir is the directory where generated images are saved for human review
// (e.g. docs/campaign_previews/<slug>/). If empty — a default under the current
// working directory is used.
func RegisterTools(s *mcpserver.MCPServer, client *Client, previewDir string) {
	if previewDir == "" {
		previewDir = "docs/campaign_previews"
	}
	registerGenerateImage(s, client, previewDir)
	registerGenerateBannerSet(s, client, previewDir)
}

func registerGenerateImage(s *mcpserver.MCPServer, client *Client, previewDir string) {
	tool := mcp.NewTool("generate_image",
		mcp.WithDescription("Сгенерировать одно изображение через OpenRouter (Flux, Gemini, и др.). Возвращает путь к сохранённому PNG + base64. Используется на шаге R6.5 ветки РСЯ."),
		mcp.WithString("prompt", mcp.Description("Полный промпт (стиль + сцена + контекст + негатив). Собирай через промпт-билдер из references/image_prompts.md."), mcp.Required()),
		mcp.WithString("model", mcp.Description("ID модели OpenRouter. Умолч: black-forest-labs/flux.2-pro. Альтернативы: google/gemini-3-pro-image-preview, google/gemini-3.1-flash-image-preview, black-forest-labs/flux.2-flex.")),
		mcp.WithString("aspect_ratio", mcp.Description("Соотношение сторон: 1:1, 16:9, 4:3, 3:2 (умолч. 1:1)")),
		mcp.WithString("campaign_slug", mcp.Description("Slug кампании для папки preview: docs/campaign_previews/<slug>/")),
		mcp.WithString("save_name", mcp.Description("Имя файла без расширения (умолч. timestamp)")),
		mcp.WithBoolean("return_base64", mcp.Description("Вернуть полный base64 в ответе (умолч. false — только путь)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt := common.GetString(req, "prompt")
		if prompt == "" {
			return common.ErrorResult("prompt обязателен"), nil
		}
		model := common.GetString(req, "model")
		if model == "" {
			model = "black-forest-labs/flux.2-pro"
		}
		aspect := common.GetString(req, "aspect_ratio")
		imageSize := aspectToSize(aspect)

		slug := common.GetString(req, "campaign_slug")
		saveName := common.GetString(req, "save_name")
		returnB64 := common.GetBool(req, "return_base64")

		res, err := client.Generate(ctx, Request{
			Model:     model,
			Prompt:    prompt,
			ImageSize: imageSize,
		})
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("generate failed: %v", err)), nil
		}

		// If URL came back instead of base64, we still report it but cannot save locally.
		if res.Base64 == "" && res.URL != "" {
			out := map[string]any{
				"model":  res.Model,
				"url":    res.URL,
				"status": "returned_url",
				"note":   "model returned direct URL instead of base64 — download separately or pass url to add_ad_image",
			}
			return common.JSONResult(out), nil
		}

		// Save to preview dir.
		ext := ".png"
		if strings.Contains(res.MimeType, "jpeg") || strings.Contains(res.MimeType, "jpg") {
			ext = ".jpg"
		}
		if saveName == "" {
			saveName = fmt.Sprintf("%s_%s", model, time.Now().Format("20060102_150405"))
			// Sanitize model id for filename.
			saveName = sanitizeName(saveName)
		}
		saveDir := previewDir
		if slug != "" {
			saveDir = filepath.Join(previewDir, sanitizeName(slug))
		}
		if err := os.MkdirAll(saveDir, 0o755); err != nil {
			return common.ErrorResult(fmt.Sprintf("mkdir %s: %v", saveDir, err)), nil
		}
		savePath := filepath.Join(saveDir, saveName+ext)

		bytesData, err := base64.StdEncoding.DecodeString(res.Base64)
		if err != nil {
			return common.ErrorResult(fmt.Sprintf("decode base64: %v", err)), nil
		}
		if err := os.WriteFile(savePath, bytesData, 0o644); err != nil {
			return common.ErrorResult(fmt.Sprintf("write %s: %v", savePath, err)), nil
		}

		out := map[string]any{
			"model":      res.Model,
			"mime_type":  res.MimeType,
			"file_path":  savePath,
			"size_bytes": len(bytesData),
			"status":     "saved",
		}
		if res.FallbackUsed {
			out["fallback_used"] = true
			out["primary_error"] = res.PrimaryError
			out["requested_model"] = model
		}
		// Decode actual pixel dimensions and warn if they don't match the
		// requested aspect ratio — image models often ignore the size hint.
		if w, h, derr := decodeDims(bytesData); derr == nil {
			out["width"] = w
			out["height"] = h
			if warn := checkAspectMatch(aspect, w, h); warn != "" {
				out["aspect_warning"] = warn
			}
		}
		if returnB64 {
			out["image_base64"] = res.Base64
		}
		// JSONResult formats with indent.
		data, _ := json.MarshalIndent(out, "", "  ")
		return common.TextResult(string(data)), nil
	})
}

func registerGenerateBannerSet(s *mcpserver.MCPServer, client *Client, previewDir string) {
	tool := mcp.NewTool("generate_banner_set",
		mcp.WithDescription("Пакетная генерация набора баннеров РСЯ: один визуал в нескольких форматах. Соблюдай лимит 20 генераций на кампанию (учитывай уже созданные). Используется на шаге R6.5."),
		mcp.WithString("prompt", mcp.Description("Базовый промпт (подходит для всех форматов)"), mcp.Required()),
		mcp.WithString("aspect_ratios", mcp.Description("Соотношения через запятую: 1:1,16:9 (умолч.). Допустимы: 1:1,16:9,4:3,3:2")),
		mcp.WithNumber("n_variants", mcp.Description("Кол-во вариантов на формат (умолч. 1)")),
		mcp.WithString("model", mcp.Description("Модель OpenRouter (умолч. black-forest-labs/flux.2-pro)")),
		mcp.WithString("campaign_slug", mcp.Description("Slug кампании для папки preview")),
		mcp.WithString("base_name", mcp.Description("База имени файла (умолч. visual)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt := common.GetString(req, "prompt")
		if prompt == "" {
			return common.ErrorResult("prompt обязателен"), nil
		}
		ratios := common.GetStringSlice(req, "aspect_ratios")
		if len(ratios) == 0 {
			ratios = []string{"1:1", "16:9"}
		}
		n := common.GetInt(req, "n_variants")
		if n <= 0 {
			n = 1
		}
		if n > 4 {
			n = 4
		}
		model := common.GetString(req, "model")
		if model == "" {
			model = "black-forest-labs/flux.2-pro"
		}
		slug := common.GetString(req, "campaign_slug")
		baseName := common.GetString(req, "base_name")
		if baseName == "" {
			baseName = "visual"
		}

		results := make([]map[string]any, 0, len(ratios)*n)
		for _, ar := range ratios {
			imageSize := aspectToSize(ar)
			for i := 0; i < n; i++ {
				res, err := client.Generate(ctx, Request{
					Model:     model,
					Prompt:    prompt,
					ImageSize: imageSize,
				})
				if err != nil {
					results = append(results, map[string]any{
						"aspect_ratio": ar,
						"variant":      i + 1,
						"status":       "error",
						"error":        err.Error(),
					})
					continue
				}
				if res.Base64 == "" {
					results = append(results, map[string]any{
						"aspect_ratio": ar,
						"variant":      i + 1,
						"status":       "returned_url",
						"url":          res.URL,
						"model":        res.Model,
					})
					continue
				}
				ext := ".png"
				if strings.Contains(res.MimeType, "jpeg") {
					ext = ".jpg"
				}
				saveDir := previewDir
				if slug != "" {
					saveDir = filepath.Join(previewDir, sanitizeName(slug))
				}
				if err := os.MkdirAll(saveDir, 0o755); err != nil {
					results = append(results, map[string]any{
						"aspect_ratio": ar,
						"variant":      i + 1,
						"status":       "error",
						"error":        fmt.Sprintf("mkdir: %v", err),
					})
					continue
				}
				fname := fmt.Sprintf("%s_%s_v%d_%s%s",
					sanitizeName(baseName),
					strings.ReplaceAll(ar, ":", "x"),
					i+1,
					time.Now().Format("150405"),
					ext,
				)
				savePath := filepath.Join(saveDir, fname)
				data, err := base64.StdEncoding.DecodeString(res.Base64)
				if err != nil {
					results = append(results, map[string]any{
						"aspect_ratio": ar,
						"variant":      i + 1,
						"status":       "error",
						"error":        fmt.Sprintf("decode: %v", err),
					})
					continue
				}
				if err := os.WriteFile(savePath, data, 0o644); err != nil {
					results = append(results, map[string]any{
						"aspect_ratio": ar,
						"variant":      i + 1,
						"status":       "error",
						"error":        fmt.Sprintf("write: %v", err),
					})
					continue
				}
				entry := map[string]any{
					"aspect_ratio": ar,
					"variant":      i + 1,
					"status":       "saved",
					"file_path":    savePath,
					"size_bytes":   len(data),
					"mime_type":    res.MimeType,
					"model":        res.Model,
				}
				if res.FallbackUsed {
					entry["fallback_used"] = true
					entry["primary_error"] = res.PrimaryError
					entry["requested_model"] = model
				}
				if w, h, derr := decodeDims(data); derr == nil {
					entry["width"] = w
					entry["height"] = h
					if warn := checkAspectMatch(ar, w, h); warn != "" {
						entry["aspect_warning"] = warn
					}
				}
				results = append(results, entry)
			}
		}

		out := map[string]any{
			"total_requested": len(ratios) * n,
			"results":         results,
			"prompt":          prompt,
			"model":           model,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		return common.TextResult(string(data)), nil
	})
}

// aspectToSize maps an aspect ratio like "16:9" to a concrete size hint
// understood by most OpenRouter image models. The numbers stay below Yandex
// banner caps and above the minimums for 1:1 (450×450) and 16:9 (1080×607).
func aspectToSize(aspect string) string {
	switch strings.TrimSpace(aspect) {
	case "", "1:1":
		return "1024x1024"
	case "16:9":
		return "1920x1080"
	case "4:3":
		return "1440x1080"
	case "3:2":
		return "1620x1080"
	case "9:16":
		return "1080x1920"
	default:
		return "1024x1024"
	}
}

// decodeDims returns width and height of the image bytes without decoding
// the full pixel data. Supports JPG, PNG, GIF.
func decodeDims(data []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// checkAspectMatch returns a non-empty warning string if the actual w×h does
// not match the requested aspect ratio within ±2% tolerance. Empty string
// means the dimensions match. Used to flag cases where an image model
// (especially Gemini) ignored our size hint and returned its own dimensions.
func checkAspectMatch(requested string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	var want float64
	switch strings.TrimSpace(requested) {
	case "", "1:1":
		want = 1.0
	case "16:9":
		want = 16.0 / 9.0
	case "4:3":
		want = 4.0 / 3.0
	case "3:2":
		want = 3.0 / 2.0
	case "9:16":
		want = 9.0 / 16.0
	default:
		return ""
	}
	got := float64(w) / float64(h)
	delta := math.Abs(got-want) / want
	if delta > 0.02 {
		return fmt.Sprintf("model returned %dx%d (ratio %.3f) but %s was requested (ratio %.3f) — likely Yandex Direct will reject for IncorrectImageSize",
			w, h, got, requested, want)
	}
	return ""
}

func sanitizeName(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		} else if r == '/' || r == '.' {
			sb.WriteRune('_')
		}
	}
	out := sb.String()
	if out == "" {
		return "unnamed"
	}
	return out
}
