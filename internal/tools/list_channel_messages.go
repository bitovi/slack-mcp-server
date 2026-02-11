// Package tools provides MCP tool handler implementations for the Slack MCP server.
package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	slackclient "github.com/Bitovi/slack-mcp-server/internal/slack"
)

// ListChannelMessagesHandler handles the list_channel_messages MCP tool requests.
// It retrieves messages from a Slack channel and resolves user information.
type ListChannelMessagesHandler struct {
	// slackClient is the Slack API client for retrieving channel history.
	slackClient slackclient.ClientInterface
}

// NewListChannelMessagesHandler creates a new ListChannelMessagesHandler with the given Slack client.
func NewListChannelMessagesHandler(client slackclient.ClientInterface) *ListChannelMessagesHandler {
	return &ListChannelMessagesHandler{
		slackClient: client,
	}
}

// Handle processes a list_channel_messages tool call.
// It retrieves messages from the specified channel, resolves user information,
// and builds a user mapping for mentioned users.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - request: The MCP tool call request containing channel_id and optional parameters
//
// Returns an MCP tool result containing the messages and metadata,
// or an error result if the operation fails.
func (h *ListChannelMessagesHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Implement in subsequent subtask
	return mcp.NewToolResultError("not implemented"), nil
}

// HandleFunc returns a function that can be used directly as an MCP tool handler.
// This is a convenience method for registering the handler with the MCP server.
func (h *ListChannelMessagesHandler) HandleFunc() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return h.Handle
}
