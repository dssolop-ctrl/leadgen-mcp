package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/leadgen-mcp/server/platform/common"
)

const (
	baseURL = "https://api.direct.yandex.com/json/v5"
)

// Client is the Yandex Direct API v5 client.
type Client struct {
	api    *common.APIClient
	logger *slog.Logger
}

// NewClient creates a new Yandex Direct API client.
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		api:    common.NewAPIClient(logger),
		logger: logger,
	}
}

// directRequest represents a Yandex Direct API v5 request body.
type directRequest struct {
	Method string `json:"method"`
	Params any    `json:"params"`
}

// Call makes a request to Yandex Direct API v5.
// service: "Campaigns", "AdGroups", "Ads", "Keywords", etc.
// method: "get", "add", "update", "delete", etc.
// clientLogin: optional client login for agency accounts (pass "" to skip).
func (c *Client) Call(ctx context.Context, token, service, method string, params any, clientLogin ...string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/%s", baseURL, service)

	body := directRequest{
		Method: method,
		Params: params,
	}

	headers := map[string]string{
		"Authorization":   "Bearer " + token,
		"Accept-Language": "ru",
	}

	if len(clientLogin) > 0 && clientLogin[0] != "" {
		headers["Client-Login"] = clientLogin[0]
	}

	var resp json.RawMessage
	err := c.api.DoJSON(ctx, common.RequestOpts{
		Method:  "POST",
		URL:     url,
		Headers: headers,
		Body:    body,
	}, &resp)

	if err != nil {
		return nil, fmt.Errorf("direct API %s.%s: %w", service, method, err)
	}

	return resp, nil
}

// CallReport makes a request to Yandex Direct Reports API.
// Reports API expects params directly in body (no method/params wrapper).
// Returns TSV text, not JSON.
func (c *Client) CallReport(ctx context.Context, token string, params any, clientLogin ...string) (string, error) {
	url := fmt.Sprintf("%s/%s", baseURL, "reports")

	// Reports API requires "params" wrapper at top level
	body := map[string]any{
		"params": params,
	}

	headers := map[string]string{
		"Authorization":       "Bearer " + token,
		"Accept-Language":     "ru",
		"processingMode":      "auto",
		"returnMoneyInMicros": "false",
		"skipReportHeader":    "true",
		"skipReportSummary":   "true",
	}

	if len(clientLogin) > 0 && clientLogin[0] != "" {
		headers["Client-Login"] = clientLogin[0]
	}

	result, err := c.api.DoText(ctx, common.RequestOpts{
		Method:  "POST",
		URL:     url,
		Headers: headers,
		Body:    body,
	})
	if err != nil {
		return "", fmt.Errorf("direct reports API: %w", err)
	}

	return result, nil
}

// GetResult extracts the "result" field from a Direct API response.
func GetResult(raw json.RawMessage) (json.RawMessage, error) {
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			ErrorCode   json.RawMessage `json:"error_code"`
			ErrorString string          `json:"error_string"`
			ErrorDetail string          `json:"error_detail"`
		} `json:"error"`
	}

	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("parse response envelope: %w", err)
	}

	if envelope.Error != nil {
		code := string(envelope.Error.ErrorCode)
		return nil, fmt.Errorf("API error %s: %s (%s)",
			code,
			envelope.Error.ErrorString,
			envelope.Error.ErrorDetail,
		)
	}

	return envelope.Result, nil
}
