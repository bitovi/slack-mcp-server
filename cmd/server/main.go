// Package main provides the entry point for the Slack MCP server.
// The server reads Slack messages and threads via MCP protocol.
package main

import (
	"fmt"
	"os"

	"github.com/slack-mcp-server/slack-mcp-server/internal/server"
)

const (
	// envSlackBotToken is the environment variable name for the Slack bot token.
	envSlackBotToken = "SLACK_BOT_TOKEN"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run executes the main server logic.
// It validates configuration, creates the server, and starts it.
// Separated from main() to allow proper error handling.
func run() error {
	// Get the Slack bot token from environment
	token := os.Getenv(envSlackBotToken)
	if token == "" {
		return fmt.Errorf(
			"%s environment variable is required\n\n"+
				"To obtain a Slack bot token:\n"+
				"1. Go to https://api.slack.com/apps and create a new app\n"+
				"2. Under 'OAuth & Permissions', add the following scopes:\n"+
				"   - channels:history (read public channel messages)\n"+
				"   - groups:history (read private channel messages)\n"+
				"   - im:history (read direct messages)\n"+
				"   - mpim:history (read group direct messages)\n"+
				"3. Install the app to your workspace\n"+
				"4. Copy the 'Bot User OAuth Token' (starts with xoxb-)\n"+
				"5. Export it: export %s=xoxb-your-token-here",
			envSlackBotToken, envSlackBotToken)
	}

	// Create server configuration
	cfg := server.Config{
		SlackToken: token,
	}

	// Create the MCP server
	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Run the server using Stdio transport
	// This blocks until the server is terminated
	if err := srv.Run(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
