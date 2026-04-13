package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/leadgen-mcp/server/auth"
	"github.com/leadgen-mcp/server/config"
	mcpsetup "github.com/leadgen-mcp/server/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Create account resolver
	resolver := auth.NewAccountResolver(cfg.Accounts)

	// Create MCP server
	mcpServer := mcpsetup.NewServer(resolver, logger)

	// Create SSE handler
	sseHandler := server.NewSSEServer(mcpServer)

	// Wrap with auth middleware
	var handler http.Handler = sseHandler
	if cfg.Server.BearerToken != "" {
		handler = auth.BearerMiddleware(cfg.Server.BearerToken, handler)
	}

	// Health check
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})
	mux.Handle("/", handler)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logger.Info("starting leadgen-mcp server", "addr", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
