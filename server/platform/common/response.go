package common

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// TextResult creates a successful MCP tool result with text content.
func TextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

// JSONResult creates a successful MCP tool result with formatted JSON.
func JSONResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to serialize result: %v", err))
	}
	return TextResult(string(data))
}

// MaxResponseSize is the maximum response size in bytes before truncation.
const MaxResponseSize = 8192

// SafeTextResult creates a text result with truncation if response exceeds MaxResponseSize.
func SafeTextResult(text string) *mcp.CallToolResult {
	if len(text) > MaxResponseSize {
		truncated := text[:MaxResponseSize]
		truncated += fmt.Sprintf("\n\n... [ОБРЕЗАНО: ответ %d байт, показано %d. Используй фильтры для уточнения запроса]", len(text), MaxResponseSize)
		return TextResult(truncated)
	}
	return TextResult(text)
}

// ErrorResult creates an error MCP tool result.
func ErrorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: msg,
			},
		},
		IsError: true,
	}
}

// GetString extracts a string parameter from the MCP request arguments.
func GetString(req mcp.CallToolRequest, key string) string {
	return req.GetString(key, "")
}

// GetInt extracts an integer parameter from the MCP request arguments.
func GetInt(req mcp.CallToolRequest, key string) int {
	return req.GetInt(key, 0)
}

// GetBool extracts a boolean parameter from the MCP request arguments.
func GetBool(req mcp.CallToolRequest, key string) bool {
	return req.GetBool(key, false)
}

// GetStringSlice extracts a comma-separated string as a slice.
func GetStringSlice(req mcp.CallToolRequest, key string) []string {
	s := GetString(req, key)
	if s == "" {
		return nil
	}
	var result []string
	for _, part := range splitAndTrim(s) {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitAndTrim(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == ',' {
			parts = append(parts, trimSpace(current))
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, trimSpace(current))
	}
	return parts
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
