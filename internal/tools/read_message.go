// Package tools provides MCP tool handler implementations for the Slack MCP server.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	slackclient "github.com/slack-mcp-server/slack-mcp-server/internal/slack"
	"github.com/slack-mcp-server/slack-mcp-server/internal/urlparser"
	"github.com/slack-mcp-server/slack-mcp-server/pkg/types"
)

// ReadMessageHandler handles the read_message MCP tool requests.
// It parses Slack URLs, retrieves messages, and optionally fetches thread replies.
type ReadMessageHandler struct {
	// slackClient is the Slack API client for retrieving messages and threads.
	slackClient slackclient.ClientInterface
}

// NewReadMessageHandler creates a new ReadMessageHandler with the given Slack client.
func NewReadMessageHandler(client slackclient.ClientInterface) *ReadMessageHandler {
	return &ReadMessageHandler{
		slackClient: client,
	}
}

// Handle processes a read_message tool call.
// It parses the Slack URL from the request, retrieves the message,
// and fetches the thread if applicable.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - request: The MCP tool call request containing the URL argument
//
// Returns an MCP tool result containing the message and optional thread,
// or an error result if the operation fails.
func (h *ReadMessageHandler) Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract the URL argument from the request
	url := mcp.ExtractString(request.Params.Arguments, "url")
	if url == "" {
		return mcp.NewToolResultError("missing required argument 'url'"), nil
	}

	// Parse the Slack URL to extract channel ID and timestamps
	parsedURL, err := urlparser.Parse(url)
	if err != nil {
		return h.handleError(err), nil
	}

	// Fetch the primary message
	message, err := h.slackClient.GetMessage(ctx, parsedURL.ChannelID, parsedURL.Timestamp)
	if err != nil {
		return h.handleError(err), nil
	}

	// Resolve user info for the primary message (populates UserName, DisplayName, RealName)
	h.resolveUserForMessage(ctx, message)

	// Build the result
	result := &types.ReadMessageResult{
		Message:   *message,
		ChannelID: parsedURL.ChannelID,
	}

	// Determine if we need to fetch thread replies
	// We fetch the thread if:
	// 1. The URL explicitly points to a thread (has thread_ts parameter), OR
	// 2. The message has replies (ReplyCount > 0)
	shouldFetchThread := parsedURL.IsThread || h.slackClient.HasThread(message)

	if shouldFetchThread {
		// Determine which timestamp to use for fetching the thread
		// If it's a thread URL, use the thread_ts from the URL
		// Otherwise, use the message's timestamp (it's the parent of the thread)
		threadTS := parsedURL.ThreadTS
		if threadTS == "" {
			// Message is a parent with replies, use its timestamp
			threadTS = message.Timestamp
		}

		// Fetch all thread replies
		thread, err := h.slackClient.GetThread(ctx, parsedURL.ChannelID, threadTS)
		if err != nil {
			// If thread fetch fails, still return the message but note the error
			// This provides partial results rather than complete failure
			return h.handlePartialResult(result, err), nil
		}

		// Resolve user info for each message in the thread
		for i := range thread {
			h.resolveUserForMessage(ctx, &thread[i])
		}

		result.Thread = thread
	}

	// Extract mentioned users from all messages and build user mapping
	result.UserMapping = h.buildUserMapping(ctx, result)

	// Return the successful result as JSON content
	return h.successResult(result)
}

// handleError converts an error into an MCP tool error result.
// It examines the error type to provide helpful, user-friendly messages.
func (h *ReadMessageHandler) handleError(err error) *mcp.CallToolResult {
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
			"Channel not found. The channel may have been deleted, or the ID in the URL is incorrect.")
	}

	if slackclient.IsNotInChannel(err) {
		return mcp.NewToolResultError(
			"The bot is not a member of this channel. Please invite the bot to the channel first.")
	}

	if slackclient.IsMessageNotFound(err) {
		return mcp.NewToolResultError(
			"Message not found. The message may have been deleted, or the timestamp in the URL is incorrect.")
	}

	if slackclient.IsPermissionDenied(err) {
		return mcp.NewToolResultError(
			"Permission denied. The bot may lack required scopes or the channel is archived.")
	}

	// Check for URL parsing errors
	code := slackclient.GetErrorCode(err)
	if code == types.ErrCodeInvalidURL {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Invalid Slack URL format. Expected: https://workspace.slack.com/archives/{channel_id}/p{timestamp}\n\nDetails: %s",
			err.Error()))
	}

	// Generic error handling
	return mcp.NewToolResultError(fmt.Sprintf("Failed to read message: %s", err.Error()))
}

// handlePartialResult creates a result that includes the message but notes a thread fetch failure.
func (h *ReadMessageHandler) handlePartialResult(result *types.ReadMessageResult, threadErr error) *mcp.CallToolResult {
	// Create a modified result that indicates partial success
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to encode partial result: %s", err.Error()))
	}

	// Return both the partial result and a warning about the thread
	return mcp.NewToolResultText(fmt.Sprintf(
		"%s\n\nNote: Failed to fetch thread replies: %s",
		string(resultJSON),
		threadErr.Error()))
}

// successResult creates a successful MCP tool result with the given data.
func (h *ReadMessageHandler) successResult(result *types.ReadMessageResult) (*mcp.CallToolResult, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to encode result: %s", err.Error())), nil
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// HandleFunc returns a function that can be used directly as an MCP tool handler.
// This is a convenience method for registering the handler with the MCP server.
func (h *ReadMessageHandler) HandleFunc() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return h.Handle
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
func (h *ReadMessageHandler) resolveUserForMessage(ctx context.Context, msg *types.Message) {
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
// This method scans the primary message and all thread messages for Slack mentions
// (e.g., <@U06025G6B28>) and builds a mapping of user IDs to their UserInfo.
// If a user lookup fails, that user is simply omitted from the mapping.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - result: The ReadMessageResult containing the message and optional thread
//
// Returns a map of user IDs to UserInfo for all mentioned users, or nil if no mentions found.
func (h *ReadMessageHandler) buildUserMapping(ctx context.Context, result *types.ReadMessageResult) map[string]types.UserInfo {
	// Collect all unique mentioned user IDs
	mentionedUserIDs := make(map[string]bool)

	// Extract mentions from the primary message
	for _, userID := range h.slackClient.ExtractMentions(result.Message.Text) {
		mentionedUserIDs[userID] = true
	}

	// Extract mentions from all thread messages
	for _, msg := range result.Thread {
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

// ReadMessage is a standalone function that processes a read_message request.
// This provides a simpler interface for cases where a handler struct is not needed.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - client: The Slack client to use for API calls
//   - url: The Slack message URL to read
//
// Returns the message result or an error.
func ReadMessage(ctx context.Context, client slackclient.ClientInterface, url string) (*types.ReadMessageResult, error) {
	// Parse the Slack URL
	parsedURL, err := urlparser.Parse(url)
	if err != nil {
		return nil, err
	}

	// Fetch the primary message
	message, err := client.GetMessage(ctx, parsedURL.ChannelID, parsedURL.Timestamp)
	if err != nil {
		return nil, err
	}

	// Build the result
	result := &types.ReadMessageResult{
		Message:   *message,
		ChannelID: parsedURL.ChannelID,
	}

	// Determine if we need to fetch thread replies
	shouldFetchThread := parsedURL.IsThread || client.HasThread(message)

	if shouldFetchThread {
		threadTS := parsedURL.ThreadTS
		if threadTS == "" {
			threadTS = message.Timestamp
		}

		thread, err := client.GetThread(ctx, parsedURL.ChannelID, threadTS)
		if err != nil {
			// Return error for standalone function - caller can decide how to handle
			return nil, fmt.Errorf("failed to fetch thread: %w", err)
		}

		result.Thread = thread
	}

	return result, nil
}
