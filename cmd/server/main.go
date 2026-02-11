// Package main provides the entry point for the Slack MCP server.
// The server reads Slack messages and threads via MCP protocol.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Bitovi/slack-mcp-server/internal/server"
)

const (
	// envSlackBotToken is the environment variable name for the Slack bot token.
	envSlackBotToken = "SLACK_BOT_TOKEN"
	// envSlackUserToken is the environment variable name for the Slack user token.
	envSlackUserToken = "SLACK_USER_TOKEN"
	// botTokenPrefix is the expected prefix for Slack bot tokens.
	botTokenPrefix = "xoxb-"
	// userTokenPrefix is the expected prefix for Slack user tokens.
	userTokenPrefix = "xoxp-"
)

// Version information (set during build with ldflags if needed)
var (
	version   = "1.0.0"
	buildTime = "unknown"
)

// flags holds the command-line flags.
type flags struct {
	showHelp    bool
	showVersion bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run executes the main server logic.
// It validates configuration, creates the server, and starts it.
// Separated from main() to allow proper error handling and testing.
func run(args []string) error {
	// Parse command-line flags
	f, err := parseFlags(args)
	if err != nil {
		return err
	}

	// Handle version flag
	if f.showVersion {
		printVersion()
		return nil
	}

	// Handle help flag
	if f.showHelp {
		printUsage()
		return nil
	}

	// Validate configuration
	config, err := validateConfig()
	if err != nil {
		return err
	}

	// Create server configuration
	cfg := server.Config{
		SlackToken:     config.botToken,
		SlackUserToken: config.userToken,
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

// parseFlags parses command-line flags and returns the parsed flags.
func parseFlags(args []string) (*flags, error) {
	f := &flags{}
	fs := flag.NewFlagSet("slack-mcp-server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.BoolVar(&f.showHelp, "help", false, "Show help message")
	fs.BoolVar(&f.showHelp, "h", false, "Show help message (shorthand)")
	fs.BoolVar(&f.showVersion, "version", false, "Show version information")
	fs.BoolVar(&f.showVersion, "v", false, "Show version information (shorthand)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			// flag.ErrHelp is returned when -h is passed with default error handling
			// but since we use ContinueOnError, we handle it ourselves
			f.showHelp = true
			return f, nil
		}
		return nil, err
	}

	return f, nil
}

// configResult holds the validated configuration values.
type configResult struct {
	botToken  string
	userToken string
}

// validateConfig validates the server configuration from environment variables.
// Returns the validated config if valid, or an error with helpful guidance.
func validateConfig() (*configResult, error) {
	botToken := os.Getenv(envSlackBotToken)

	// Check if bot token is provided
	if botToken == "" {
		return nil, fmt.Errorf(
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

	// Validate bot token format
	if !strings.HasPrefix(botToken, botTokenPrefix) {
		return nil, fmt.Errorf(
			"invalid %s: token must start with '%s'\n\n"+
				"The token you provided does not appear to be a valid Slack bot token.\n"+
				"Bot tokens always start with '%s'.\n\n"+
				"Common token prefixes:\n"+
				"  - xoxb-  : Bot tokens (required for this server)\n"+
				"  - xoxp-  : User tokens (optional, for search_messages)\n"+
				"  - xoxa-  : App-level tokens (not supported)\n\n"+
				"Please use the Bot User OAuth Token from your Slack app settings.",
			envSlackBotToken, botTokenPrefix, botTokenPrefix)
	}

	// Validate bot token length (basic sanity check)
	// Slack tokens are typically at least 50 characters
	if len(botToken) < 50 {
		return nil, fmt.Errorf(
			"invalid %s: token appears too short\n\n"+
				"Slack bot tokens are typically at least 50 characters long.\n"+
				"Please verify you copied the complete token from your Slack app settings.",
			envSlackBotToken)
	}

	result := &configResult{
		botToken: botToken,
	}

	// Load optional user token
	userToken := os.Getenv(envSlackUserToken)
	if userToken != "" {
		// Validate user token format
		if !strings.HasPrefix(userToken, userTokenPrefix) {
			return nil, fmt.Errorf(
				"invalid %s: token must start with '%s'\n\n"+
					"The token you provided does not appear to be a valid Slack user token.\n"+
					"User tokens always start with '%s'.\n\n"+
					"To obtain a user token:\n"+
					"1. Go to https://api.slack.com/apps and select your app\n"+
					"2. Under 'OAuth & Permissions', add the 'search:read' scope\n"+
					"3. Install or reinstall the app to your workspace\n"+
					"4. Copy the 'User OAuth Token' (starts with xoxp-)\n"+
					"5. Export it: export %s=xoxp-your-token-here",
				envSlackUserToken, userTokenPrefix, userTokenPrefix, envSlackUserToken)
		}

		// Validate user token length (basic sanity check)
		if len(userToken) < 50 {
			return nil, fmt.Errorf(
				"invalid %s: token appears too short\n\n"+
					"Slack user tokens are typically at least 50 characters long.\n"+
					"Please verify you copied the complete token from your Slack app settings.",
				envSlackUserToken)
		}

		result.userToken = userToken
	}

	return result, nil
}

// printVersion prints version information to stdout.
func printVersion() {
	fmt.Printf("slack-mcp-server version %s (built: %s)\n", version, buildTime)
}

// printUsage prints usage information to stdout.
func printUsage() {
	usage := `Slack MCP Server

An MCP (Model Context Protocol) server that enables AI agents to read Slack
messages and threads by providing a Slack message URL.

USAGE:
    slack-mcp-server [OPTIONS]

OPTIONS:
    -h, --help      Show this help message
    -v, --version   Show version information

ENVIRONMENT VARIABLES:
    SLACK_BOT_TOKEN    Required. The Slack bot token for API authentication.
                       Must start with 'xoxb-'.

    SLACK_USER_TOKEN   Optional. The Slack user token for search functionality.
                       Must start with 'xoxp-'. Required for search_messages tool.
                       Requires 'search:read' scope.

REQUIRED SLACK SCOPES:
    The Slack bot must have the following OAuth scopes:
    - channels:history   Read public channel messages
    - groups:history     Read private channel messages
    - im:history         Read direct messages
    - mpim:history       Read group direct messages

EXAMPLE:
    export SLACK_BOT_TOKEN=xoxb-your-bot-token-here
    ./slack-mcp-server

MCP TOOLS:
    read_message    Read a Slack message and its thread by URL.
                    Accepts a Slack message URL and returns the message
                    content, author, timestamp, and any thread replies.

For more information, visit: https://github.com/slack-mcp-server/slack-mcp-server
`
	fmt.Print(usage)
}
