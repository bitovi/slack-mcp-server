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
	// Extract the channel_id argument (required)
	channelIDArg, ok := request.Params.Arguments["channel_id"]
	if !ok {
		return mcp.NewToolResultError("missing required argument 'channel_id'"), nil
	}

	channelID, ok := channelIDArg.(string)
	if !ok {
		return mcp.NewToolResultError("argument 'channel_id' must be a string"), nil
	}

	if channelID == "" {
		return mcp.NewToolResultError("argument 'channel_id' cannot be empty"), nil
	}

	// Extract limit (default 100, max 200)
	limit := 100
	if limitArg, exists := request.Params.Arguments["limit"]; exists {
		switch v := limitArg.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		default:
			return mcp.NewToolResultError("argument 'limit' must be a number"), nil
		}
	}

	// Validate limit range
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}

	// Extract oldest parameter (optional Unix timestamp)
	oldest := ""
	if oldestArg, exists := request.Params.Arguments["oldest"]; exists {
		if v, ok := oldestArg.(string); ok {
			oldest = v
		} else {
			return mcp.NewToolResultError("argument 'oldest' must be a string (Unix timestamp)"), nil
		}
	}

	// Extract latest parameter (optional Unix timestamp)
	latest := ""
	if latestArg, exists := request.Params.Arguments["latest"]; exists {
		if v, ok := latestArg.(string); ok {
			latest = v
		} else {
			return mcp.NewToolResultError("argument 'latest' must be a string (Unix timestamp)"), nil
		}
	}

	// Call GetChannelHistory to retrieve messages
	messages, hasMore, err := h.slackClient.GetChannelHistory(ctx, channelID, limit, oldest, latest)
	if err != nil {
		return h.handleError(err), nil
	}

	// Resolve user info for each message
	for i := range messages {
		h.resolveUserForMessage(ctx, &messages[i])
	}

	// Build the result
	result := &types.ListChannelMessagesResult{
		Messages:  messages,
		ChannelID: channelID,
		HasMore:   hasMore,
	}

	// Extract mentioned users from all messages and build user mapping
	result.UserMapping = h.buildUserMapping(ctx, messages)

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
func (h *ListChannelMessagesHandler) handleError(err error) *mcp.CallToolResult {
	// Check for known error types and provide appropriate messages
	if slackclient.IsRateLimited(err) {
		return mcp.NewToolResultError(
			"Rate limit exceeded. Slack limits API requests to approximately 1 per minute " +
				"for non-marketplace apps. Please wait and try again.")
	}

	if slackclient.IsInvalidToken(err) {
		return mcp.NewToolResultError(
			"Authentication failed. Please check that SLACK_BOT_TOKEN is valid and not expired.")
	}

	if slackclient.IsChannelNotFound(err) {
		return mcp.NewToolResultError(
			"Channel not found. The channel may have been deleted, or the channel_id is incorrect.")
	}

	if slackclient.IsNotInChannel(err) {
		return mcp.NewToolResultError(
			"The bot is not a member of this channel. Please invite the bot to the channel first.")
	}

	if slackclient.IsPermissionDenied(err) {
		return mcp.NewToolResultError(
			"Permission denied. The bot may lack required scopes or the channel is archived.")
	}

	// Generic error handling
	return mcp.NewToolResultError(fmt.Sprintf("Failed to list channel messages: %s", err.Error()))
}

// successResult creates a successful MCP tool result with the given data.
func (h *ListChannelMessagesHandler) successResult(result *types.ListChannelMessagesResult) (*mcp.CallToolResult, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to encode result: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// resolveUserForMessage populates user name fields on a message by fetching user info.
//
// This method fetches user information for the message author and populates
// the UserName, DisplayName, and RealName fields on the message. If the user
// lookup fails, the message is left unchanged (graceful degradation).
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - msg: Pointer to the message to populate with user info
//
// This method does not return an error. If user resolution fails, the message
// will simply not have user name fields populated.
func (h *ListChannelMessagesHandler) resolveUserForMessage(ctx context.Context, msg *types.Message) {
	// Skip if message has no user ID (e.g., system messages)
	if msg.User == "" {
		return
	}

	// Fetch user info from Slack (or cache)
	userInfo, err := h.slackClient.GetUserInfo(ctx, msg.User)
	if err != nil {
		// Graceful degradation: log the error but don't fail
		// The message will be returned without user name fields
		return
	}

	// Handle case where GetUserInfo returns nil without error
	if userInfo == nil {
		return
	}

	// Populate the user name fields on the message
	msg.UserName = userInfo.Name
	msg.DisplayName = userInfo.DisplayName
	msg.RealName = userInfo.RealName
}

// buildUserMapping extracts mentioned user IDs from all messages and resolves them to UserInfo.
//
// This method scans all messages for Slack mentions (e.g., <@U06025G6B28>) and builds
// a mapping of user IDs to their UserInfo. If a user lookup fails, that user is
// simply omitted from the mapping.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - messages: The messages to scan for mentions
//
// Returns a map of user IDs to UserInfo for all mentioned users, or nil if no mentions found.
func (h *ListChannelMessagesHandler) buildUserMapping(ctx context.Context, messages []types.Message) map[string]types.UserInfo {
	// Collect all unique mentioned user IDs
	mentionedUserIDs := make(map[string]bool)

	// Extract mentions from all messages
	for _, msg := range messages {
		for _, userID := range h.slackClient.ExtractMentions(msg.Text) {
			mentionedUserIDs[userID] = true
		}
	}

	// If no mentions found, return nil
	if len(mentionedUserIDs) == 0 {
		return nil
	}

	// Build the user mapping by resolving each mentioned user
	userMapping := make(map[string]types.UserInfo)
	for userID := range mentionedUserIDs {
		userInfo, err := h.slackClient.GetUserInfo(ctx, userID)
		if err != nil {
			// Graceful degradation: skip users we can't resolve
			continue
		}
		if userInfo != nil {
			userMapping[userID] = *userInfo
		}
	}

	// Return nil if no users were resolved (to avoid empty map in JSON)
	if len(userMapping) == 0 {
		return nil
	}

	return userMapping
}

// HandleFunc returns a function that can be used directly as an MCP tool handler.
// This is a convenience method for registering the handler with the MCP server.
func (h *ListChannelMessagesHandler) HandleFunc() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return h.Handle
}
