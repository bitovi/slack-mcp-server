// Package server provides the MCP server setup and tool registration
// for the Slack MCP server.
package server

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	slackclient "github.com/slack-mcp-server/slack-mcp-server/internal/slack"
	"github.com/slack-mcp-server/slack-mcp-server/internal/tools"
)

const (
	// ServerName is the name of the MCP server.
	ServerName = "slack-mcp"
	// ServerVersion is the version of the MCP server.
	ServerVersion = "1.0.0"
)

// Server represents the Slack MCP server.
// It wraps the MCP server and holds the Slack client for message retrieval.
type Server struct {
	// mcpServer is the underlying MCP server instance.
	mcpServer *server.MCPServer
	// slackClient is the Slack API client for retrieving messages.
	slackClient slackclient.ClientInterface
	// readMessageHandler handles the read_message tool.
	readMessageHandler *tools.ReadMessageHandler
}

// Config holds the configuration for creating a new Server.
type Config struct {
	// SlackToken is the Slack bot token for API authentication.
	// Required for creating the Slack client.
	SlackToken string
}

// New creates a new Slack MCP server with the provided configuration.
// It initializes the MCP server with tool capabilities and creates
// the Slack client for message retrieval.
//
// Parameters:
//   - cfg: Server configuration containing the Slack bot token
//
// Returns a new Server instance or an error if initialization fails.
func New(cfg Config) (*Server, error) {
	if cfg.SlackToken == "" {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN is required")
	}

	// Create the Slack client
	slackClient := slackclient.NewClient(cfg.SlackToken)

	// Create the MCP server with tool capabilities enabled
	mcpServer := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(true),
	)

	// Create the read_message handler
	readMessageHandler := tools.NewReadMessageHandler(slackClient)

	s := &Server{
		mcpServer:          mcpServer,
		slackClient:        slackClient,
		readMessageHandler: readMessageHandler,
	}

	// Register tools
	s.registerTools()

	return s, nil
}

// NewWithClient creates a new Slack MCP server with a custom Slack client.
// This is primarily useful for testing with mock clients.
//
// Parameters:
//   - client: A SlackClient interface implementation
//
// Returns a new Server instance.
func NewWithClient(client slackclient.ClientInterface) *Server {
	// Create the MCP server with tool capabilities enabled
	mcpServer := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(true),
	)

	// Create the read_message handler
	readMessageHandler := tools.NewReadMessageHandler(client)

	s := &Server{
		mcpServer:          mcpServer,
		slackClient:        client,
		readMessageHandler: readMessageHandler,
	}

	// Register tools
	s.registerTools()

	return s
}

// registerTools registers all MCP tools with the server.
// This method is called during server initialization.
func (s *Server) registerTools() {
	// Create the read_message tool
	readMessageTool := mcp.NewTool("read_message",
		mcp.WithDescription("Read a Slack message and its thread by URL. "+
			"Provide a Slack message URL to retrieve the message content, author, "+
			"timestamp, and any thread replies."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Slack message or thread URL to read. "+
				"Format: https://workspace.slack.com/archives/{channel_id}/p{timestamp}"),
		),
	)

	// Register the tool with the ReadMessageHandler
	s.mcpServer.AddTool(readMessageTool, s.readMessageHandler.HandleFunc())
}

// Run starts the MCP server using Stdio transport.
// This method blocks until the server is terminated.
//
// Returns an error if the server fails to start or encounters an error during operation.
func (s *Server) Run() error {
	return server.ServeStdio(s.mcpServer)
}

// MCPServer returns the underlying MCP server instance.
// This is useful for testing or advanced customization.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// SlackClient returns the Slack client interface.
// This is useful for testing or advanced customization.
func (s *Server) SlackClient() slackclient.ClientInterface {
	return s.slackClient
}
