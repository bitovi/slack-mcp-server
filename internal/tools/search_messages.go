// Package tools provides MCP tool handler implementations for the Slack MCP server.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	slackclient "github.com/Bitovi/slack-mcp-server/internal/slack"
	"github.com/Bitovi/slack-mcp-server/pkg/types"
)

// SearchMessagesHandler handles the search_messages MCP tool requests.
// It searches for messages across the Slack workspace and resolves user information.
type SearchMessagesHandler struct {
	// slackClient is the Slack API client for searching messages.
	slackClient slackclient.ClientInterface
}

// NewSearchMessagesHandler creates a new SearchMessagesHandler with the given Slack client.
func NewSearchMessagesHandler(client slackclient.ClientInterface) *SearchMessagesHandler {
	return &SearchMessagesHandler{
		slackClient: client,
	}
}

// Handle processes a search_messages tool call.
// It searches for messages matching the query, resolves user information,
// and returns the matching results.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - request: The MCP tool call request containing query and optional parameters
//
// Returns an MCP tool result containing the search matches and metadata,
// or an error result if the operation fails.
func (h *SearchMessagesHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract the query argument (required)
	queryArg, ok := request.Params.Arguments["query"]
	if !ok {
		return mcp.NewToolResultError("missing required argument 'query'"), nil
	}

	query, ok := queryArg.(string)
	if !ok {
		return mcp.NewToolResultError("argument 'query' must be a string"), nil
	}

	if query == "" {
		return mcp.NewToolResultError("argument 'query' cannot be empty"), nil
	}

	// Extract count (default 20, max 100)
	count := 20
	if countArg, exists := request.Params.Arguments["count"]; exists {
		switch v := countArg.(type) {
		case float64:
			count = int(v)
		case int:
			count = v
		default:
			return mcp.NewToolResultError("argument 'count' must be a number"), nil
		}
	}

	// Validate count range
	if count < 1 {
		count = 1
	}
	if count > 100 {
		count = 100
	}

	// Extract sort parameter (optional, default "score")
	sort := "score"
	if sortArg, exists := request.Params.Arguments["sort"]; exists {
		if v, ok := sortArg.(string); ok {
			// Only accept valid sort values, otherwise keep default
			if v == "score" || v == "timestamp" {
				sort = v
			}
		}
		// Invalid sort values are silently ignored, defaulting to "score"
	}

	// Call SearchMessages to search for messages
	matches, total, err := h.slackClient.SearchMessages(ctx, query, count, sort)
	if err != nil {
		return h.handleError(err), nil
	}

	// Resolve user info for each match
	for i := range matches {
		h.resolveUserForMatch(ctx, &matches[i])
	}

	// Build the result
	result := &types.SearchMessagesResult{
		Query:   query,
		Total:   total,
		Matches: matches,
	}

	// Fetch the authenticated user's identity (graceful degradation on failure)
	currentUser, err := h.slackClient.GetCurrentUser(ctx)
	if err == nil && currentUser != nil {
		result.CurrentUser = currentUser
	}
	// Note: If GetCurrentUser fails, we continue without current_user rather than failing

	// Return the successful result as JSON content
	return h.successResult(result)
}

// handleError converts an error into an MCP tool error result.
// It examines the error type to provide helpful, user-friendly messages.
func (h *SearchMessagesHandler) handleError(err error) *mcp.CallToolResult {
	// Check for user token not configured error (most common for search_messages)
	if slackclient.IsUserTokenNotConfigured(err) {
		return mcp.NewToolResultError(
			"SLACK_USER_TOKEN not configured. The search_messages tool requires a user token (xoxp-) " +
				"with the search:read scope. Please set the SLACK_USER_TOKEN environment variable.")
	}

	// Check for rate limiting
	if slackclient.IsRateLimited(err) {
		return mcp.NewToolResultError(
			"Rate limit exceeded. Slack limits API requests. Please wait and try again.")
	}

	// Check for authentication errors
	if slackclient.IsInvalidToken(err) {
		return mcp.NewToolResultError(
			"Authentication failed. Please check that SLACK_USER_TOKEN is valid and not expired.")
	}

	// Check for permission denied
	if slackclient.IsPermissionDenied(err) {
		return mcp.NewToolResultError(
			"Permission denied. The user token may lack the search:read scope.")
	}

	// Generic error handling
	return mcp.NewToolResultError(fmt.Sprintf("Failed to search messages: %s", err.Error()))
}

// successResult creates a successful MCP tool result with the given data.
func (h *SearchMessagesHandler) successResult(result *types.SearchMessagesResult) (*mcp.CallToolResult, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to encode result: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// resolveUserForMatch populates user name fields on a search match by fetching user info.
//
// This method fetches user information for the message author and populates
// the UserName, DisplayName, and RealName fields on the match. If the user
// lookup fails, the match is left unchanged (graceful degradation).
//
// Note: The Slack search API already provides UserName in some cases, but we
// resolve the full user info for consistency with other tools and to get
// DisplayName and RealName.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - match: Pointer to the search match to populate with user info
//
// This method does not return an error. If user resolution fails, the match
// will simply not have additional user name fields populated.
func (h *SearchMessagesHandler) resolveUserForMatch(ctx context.Context, match *types.SearchMatch) {
	// Skip if match has no user ID (e.g., system messages)
	if match.User == "" {
		return
	}

	// Fetch user info from Slack (or cache)
	userInfo, err := h.slackClient.GetUserInfo(ctx, match.User)
	if err != nil {
		// Graceful degradation: log the error but don't fail
		// The match will be returned without additional user name fields
		return
	}

	// Handle case where GetUserInfo returns nil without error
	if userInfo == nil {
		return
	}

	// Populate the user name fields on the match
	match.UserName = userInfo.Name
	match.DisplayName = userInfo.DisplayName
	match.RealName = userInfo.RealName
}

// HandleFunc returns a function that can be used directly as an MCP tool handler.
// This is a convenience method for registering the handler with the MCP server.
func (h *SearchMessagesHandler) HandleFunc() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return h.Handle
}
