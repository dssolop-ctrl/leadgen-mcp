package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"
)

// APIClient is a shared HTTP client with retry, backoff, and rate limiting.
type APIClient struct {
	client  *http.Client
	logger  *slog.Logger
	maxRetries int
}

// NewAPIClient creates a new API client with sensible defaults.
func NewAPIClient(logger *slog.Logger) *APIClient {
	return &APIClient{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger:     logger,
		maxRetries: 3,
	}
}

// RequestOpts configures a single API request.
type RequestOpts struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    any // will be JSON-encoded if not nil
}

// DoJSON sends an HTTP request and decodes the JSON response into result.
func (c *APIClient) DoJSON(ctx context.Context, opts RequestOpts, result any) error {
	var bodyReader io.Reader
	if opts.Body != nil {
		data, err := json.Marshal(opts.Body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	if opts.Method == "" {
		opts.Method = http.MethodPost
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			c.logger.Debug("retrying request", "attempt", attempt, "delay", delay, "url", opts.URL)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			// Reset body reader for retry
			if opts.Body != nil {
				data, _ := json.Marshal(opts.Body)
				bodyReader = bytes.NewReader(data)
			}
		}

		req, err := http.NewRequestWithContext(ctx, opts.Method, opts.URL, bodyReader)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		// Retry on 429, 502, 503
		if resp.StatusCode == 429 || resp.StatusCode == 502 || resp.StatusCode == 503 {
			lastErr = &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
			continue
		}

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return &AuthError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("decode response: %w (body: %s)", err, truncate(string(respBody), 500))
			}
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// DoRaw sends an HTTP request and returns raw response body.
func (c *APIClient) DoRaw(ctx context.Context, opts RequestOpts) ([]byte, error) {
	var raw json.RawMessage
	// Use DoJSON but capture raw response
	opts2 := opts
	err := c.DoJSON(ctx, opts2, &raw)
	if err != nil {
		return nil, err
	}
	return []byte(raw), nil
}

// DoText sends an HTTP request and returns raw response body as string.
// Unlike DoJSON, it does NOT try to parse the response as JSON.
func (c *APIClient) DoText(ctx context.Context, opts RequestOpts) (string, error) {
	var bodyReader io.Reader
	if opts.Body != nil {
		data, err := json.Marshal(opts.Body)
		if err != nil {
			return "", fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	if opts.Method == "" {
		opts.Method = http.MethodPost
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
			if opts.Body != nil {
				data, _ := json.Marshal(opts.Body)
				bodyReader = bytes.NewReader(data)
			}
		}

		req, err := http.NewRequestWithContext(ctx, opts.Method, opts.URL, bodyReader)
		if err != nil {
			return "", fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		// Use raw header assignment to preserve exact casing
		// (Yandex Direct Reports API requires exact header names like returnMoneyInMicros)
		for k, v := range opts.Headers {
			req.Header[k] = []string{v}
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode == 429 || resp.StatusCode == 502 || resp.StatusCode == 503 {
			lastErr = &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
			continue
		}

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return "", &AuthError{StatusCode: resp.StatusCode, Message: string(respBody)}
		}

		// 201 = report created (offline), 202 = report still processing
		if resp.StatusCode == 201 || resp.StatusCode == 202 {
			retryIn := resp.Header.Get("retryIn")
			if retryIn != "" {
				lastErr = &APIError{StatusCode: resp.StatusCode, Message: "report not ready, retryIn=" + retryIn}
				continue
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
		}

		return string(respBody), nil
	}

	return "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
