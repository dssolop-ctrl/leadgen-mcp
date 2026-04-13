package wordstat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/leadgen-mcp/server/platform/common"
)

const baseURL = "https://api.direct.yandex.com/v4/json"

// Client is the Yandex Wordstat API client.
// Wordstat uses the older Yandex Direct API v4.
type Client struct {
	api    *common.APIClient
	logger *slog.Logger
}

// NewClient creates a new Wordstat client.
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		api:    common.NewAPIClient(logger),
		logger: logger,
	}
}

// wordstatRequest represents a v4 API request.
type wordstatRequest struct {
	Method string `json:"method"`
	Param  any    `json:"param,omitempty"`
	Token  string `json:"token"`
}

// Call makes a request to Yandex Wordstat via Direct API v4.
func (c *Client) Call(ctx context.Context, token, method string, param any) (json.RawMessage, error) {
	body := wordstatRequest{
		Method: method,
		Param:  param,
		Token:  token,
	}

	var resp json.RawMessage
	err := c.api.DoJSON(ctx, common.RequestOpts{
		Method: "POST",
		URL:    baseURL,
		Headers: map[string]string{
			"Accept-Language": "ru",
		},
		Body: body,
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("wordstat %s: %w", method, err)
	}

	return resp, nil
}

// GetStat makes a GET to Wordstat stat API.
func (c *Client) GetStat(ctx context.Context, token, path string, params url.Values) (json.RawMessage, error) {
	u := fmt.Sprintf("https://wordstat.yandex.ru/api%s", path)
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	var result json.RawMessage
	err := c.api.DoJSON(ctx, common.RequestOpts{
		Method: "GET",
		URL:    u,
		Headers: map[string]string{
			"Authorization": "OAuth " + token,
		},
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("wordstat stat %s: %w", path, err)
	}

	return result, nil
}
