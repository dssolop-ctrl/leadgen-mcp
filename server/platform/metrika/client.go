package metrika

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/leadgen-mcp/server/platform/common"
)

const baseURL = "https://api-metrika.yandex.net"

// Client is the Yandex Metrika API client.
type Client struct {
	api    *common.APIClient
	logger *slog.Logger
}

// NewClient creates a new Yandex Metrika API client.
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		api:    common.NewAPIClient(logger),
		logger: logger,
	}
}

// Get makes a GET request to Yandex Metrika API.
func (c *Client) Get(ctx context.Context, token, path string, queryParams url.Values) (json.RawMessage, error) {
	u := fmt.Sprintf("%s%s", baseURL, path)
	if len(queryParams) > 0 {
		u += "?" + queryParams.Encode()
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
		return nil, fmt.Errorf("metrika API %s: %w", path, err)
	}

	return result, nil
}
