package vk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/leadgen-mcp/server/platform/common"
)

const baseURL = "https://ads.vk.com/api/v2"

// Client is the VK Ads API v2 client.
type Client struct {
	api    *common.APIClient
	logger *slog.Logger
}

// NewClient creates a new VK Ads API client.
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		api:    common.NewAPIClient(logger),
		logger: logger,
	}
}

func (c *Client) headers(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

// Get makes a GET request to VK Ads API.
func (c *Client) Get(ctx context.Context, token, path string, params url.Values) (json.RawMessage, error) {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	var result json.RawMessage
	err := c.api.DoJSON(ctx, common.RequestOpts{
		Method:  "GET",
		URL:     u,
		Headers: c.headers(token),
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("vk ads GET %s: %w", path, err)
	}
	return result, nil
}

// Post makes a POST request to VK Ads API.
func (c *Client) Post(ctx context.Context, token, path string, body any) (json.RawMessage, error) {
	var result json.RawMessage
	err := c.api.DoJSON(ctx, common.RequestOpts{
		Method:  "POST",
		URL:     baseURL + path,
		Headers: c.headers(token),
		Body:    body,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("vk ads POST %s: %w", path, err)
	}
	return result, nil
}

// Patch makes a PATCH request to VK Ads API.
func (c *Client) Patch(ctx context.Context, token, path string, body any) (json.RawMessage, error) {
	var result json.RawMessage
	err := c.api.DoJSON(ctx, common.RequestOpts{
		Method:  "PATCH",
		URL:     baseURL + path,
		Headers: c.headers(token),
		Body:    body,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("vk ads PATCH %s: %w", path, err)
	}
	return result, nil
}

// Delete makes a DELETE request to VK Ads API.
func (c *Client) Delete(ctx context.Context, token, path string) (json.RawMessage, error) {
	var result json.RawMessage
	err := c.api.DoJSON(ctx, common.RequestOpts{
		Method:  "DELETE",
		URL:     baseURL + path,
		Headers: c.headers(token),
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("vk ads DELETE %s: %w", path, err)
	}
	return result, nil
}
