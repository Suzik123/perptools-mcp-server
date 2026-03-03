package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"mcp-server/app/internal/service"
	"mcp-server/app/internal/tools"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	svc := service.NewService(service.Config{
		OrderlyBaseURL:   "https://api.orderly.org",
		PerptoolsBaseURL: "https://app.perptools.ai/api",
		BrokerID:         "dextools",
	})

	s := server.NewMCPServer(
		"perptools-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	for _, t := range tools.RegisterAuthTools(svc) {
		s.AddTool(t.Tool, t.Handler)
	}
	for _, t := range tools.RegisterPerptoolsTools(svc) {
		s.AddTool(t.Tool, t.Handler)
	}

	transport := strings.ToLower(envOrDefault("TRANSPORT", "sse"))

	switch transport {
	case "sse":
		addr := envOrDefault("ADDR", ":8080")
		basePath := envOrDefault("BASE_PATH", "/mcp")
		baseURL := envOrDefault("BASE_URL", "")

		opts := []server.SSEOption{
			server.WithStaticBasePath(basePath),
			server.WithKeepAliveInterval(30 * time.Second),
		}
		if baseURL != "" {
			opts = append(opts, server.WithBaseURL(baseURL))
		}

		sseServer := server.NewSSEServer(s, opts...)
		log.Printf("Starting SSE server on %s (base path: %s)", addr, basePath)
		if err := sseServer.Start(addr); err != nil {
			log.Fatalf("SSE server error: %v", err)
		}

	case "stdio":
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
			os.Exit(1)
		}

	default:
		log.Fatalf("Unknown transport: %s (supported: sse, stdio)", transport)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
