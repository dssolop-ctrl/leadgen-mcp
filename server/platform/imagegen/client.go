// Package imagegen provides image generation MCP tools backed by OpenRouter.
// Used by the RSYA branch (R6.5) of the leadgen skill to generate banner
// creatives via Flux, Gemini, and other image-capable models.
package imagegen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultEndpoint is the OpenRouter chat completions URL used for image generation.
const DefaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"

// Client wraps OpenRouter calls for image generation.
type Client struct {
	apiKey   string
	endpoint string
	http     *http.Client
}

// NewClient constructs an OpenRouter image generation client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: DefaultEndpoint,
		http:     &http.Client{Timeout: 180 * time.Second},
	}
}

// Request describes a single image generation call.
type Request struct {
	Model  string
	Prompt string
	// ImageSize like "1024x1024" / "1920x1080". Sent to models that
	// accept size hints via the "image_size" extra body parameter.
	ImageSize string
	// Extra is merged into the request body (e.g. response_format, quality).
	Extra map[string]any
}

// Result is a single generated image.
type Result struct {
	Base64   string // raw base64 (without data URL prefix)
	MimeType string // image/png, image/jpeg, etc.
	URL      string // direct URL if the model returned one (rare for OpenRouter)
	Model    string
}

// Generate performs a single image generation call and returns the decoded image.
func (c *Client) Generate(ctx context.Context, req Request) (*Result, error) {
	if c.apiKey == "" {
		return nil, errors.New("openrouter api key not configured (config.yaml → openrouter.api_key or env OPENROUTER_API_KEY)")
	}
	if req.Prompt == "" {
		return nil, errors.New("prompt is required")
	}
	if req.Model == "" {
		req.Model = "google/gemini-3-pro-image-preview"
	}

	// OpenRouter unified API: chat/completions with modalities: ["image"] or ["image","text"].
	body := map[string]any{
		"model": req.Model,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": req.Prompt,
			},
		},
		"modalities": []string{"image", "text"},
	}

	if req.ImageSize != "" {
		body["image_size"] = req.ImageSize
	}
	for k, v := range req.Extra {
		body[k] = v
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "https://github.com/leadgen-mcp")
	httpReq.Header.Set("X-Title", "leadgen-mcp")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 60*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openrouter http %d: %s", resp.StatusCode, truncate(string(raw), 600))
	}

	// Parse the response. OpenRouter places image output either in:
	//   choices[0].message.images[].image_url.url   (data:<mime>;base64,<data> or direct URL)
	// or choices[0].message.content[].image_url.url (some providers)
	var parsed struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
				Images  []struct {
					Type     string `json:"type"`
					ImageURL struct {
						URL string `json:"url"`
					} `json:"image_url"`
				} `json:"images"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w; body=%s", err, truncate(string(raw), 400))
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("openrouter error (code=%d): %s", parsed.Error.Code, parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned: %s", truncate(string(raw), 400))
	}

	choice := parsed.Choices[0]

	// 1) Primary location: message.images[].image_url.url.
	if len(choice.Message.Images) > 0 {
		u := choice.Message.Images[0].ImageURL.URL
		return decodeImageURL(u, req.Model)
	}

	// 2) Fallback: message.content array with image_url entries.
	if arr, ok := choice.Message.Content.([]any); ok {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := m["type"].(string); t == "image_url" {
				if iu, ok := m["image_url"].(map[string]any); ok {
					if u, _ := iu["url"].(string); u != "" {
						return decodeImageURL(u, req.Model)
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("image not found in response: %s", truncate(string(raw), 400))
}

func decodeImageURL(u, model string) (*Result, error) {
	if strings.HasPrefix(u, "data:") {
		// data:image/png;base64,XXXX
		idx := strings.Index(u, ";base64,")
		if idx < 0 {
			return nil, fmt.Errorf("malformed data URL: %s", truncate(u, 120))
		}
		mime := strings.TrimPrefix(u[:idx], "data:")
		b64 := u[idx+len(";base64,"):]
		// Validate base64 to catch garbled responses early.
		if _, err := base64.StdEncoding.DecodeString(b64); err != nil {
			return nil, fmt.Errorf("invalid base64 in data URL: %w", err)
		}
		return &Result{Base64: b64, MimeType: mime, Model: model}, nil
	}
	// Direct URL — caller may need to fetch separately.
	return &Result{URL: u, Model: model}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
