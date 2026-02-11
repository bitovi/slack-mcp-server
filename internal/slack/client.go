// Package slack provides a wrapper around the Slack API client
// for fetching messages and threads.
package slack

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/slack-go/slack"

	"github.com/slack-mcp-server/slack-mcp-server/pkg/types"
)

// mentionPattern matches Slack user mentions in the format <@UXXXXXXXX>
var mentionPattern = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)

// Client wraps the Slack API client to provide message and thread retrieval.
type Client struct {
	api       *slack.Client
	userCache sync.Map // Maps user ID (string) to user display name (string)
}

// NewClient creates a new Slack client with the provided bot token.
func NewClient(token string) *Client {
	return &Client{
		api: slack.New(token),
	}
}

// GetMessage retrieves a single message from a Slack channel by its timestamp.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - channelID: The Slack channel ID (e.g., "C01234567")
//   - timestamp: The message timestamp in API format (e.g., "1234567890.123456")
//
// Returns the message if found, or an error if the message cannot be retrieved.
func (c *Client) GetMessage(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Oldest:    timestamp,
		Latest:    timestamp,
		Inclusive: true,
		Limit:     1,
	}

	history, err := c.api.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return nil, wrapSlackError(err)
	}

	if !history.Ok {
		return nil, types.NewSlackError(types.ErrCodeMessageNotFound,
			fmt.Sprintf("Slack API error: %s", history.Error))
	}

	if len(history.Messages) == 0 {
		return nil, types.NewSlackError(types.ErrCodeMessageNotFound,
			fmt.Sprintf("message not found in channel %s with timestamp %s", channelID, timestamp))
	}

	msg := history.Messages[0]
	return convertMessage(&msg), nil
}

// GetThread retrieves all messages in a thread, including the parent message.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - channelID: The Slack channel ID (e.g., "C01234567")
//   - threadTS: The parent message timestamp (thread_ts) in API format
//
// Returns all messages in the thread in chronological order, or an error
// if the thread cannot be retrieved.
func (c *Client) GetThread(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
	params := &slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
	}

	var allMessages []types.Message
	cursor := ""

	for {
		params.Cursor = cursor

		messages, hasMore, nextCursor, err := c.api.GetConversationRepliesContext(ctx, params)
		if err != nil {
			return nil, wrapSlackError(err)
		}

		for i := range messages {
			allMessages = append(allMessages, *convertMessage(&messages[i]))
		}

		if !hasMore {
			break
		}
		cursor = nextCursor
	}

	if len(allMessages) == 0 {
		return nil, types.NewSlackError(types.ErrCodeMessageNotFound,
			fmt.Sprintf("thread not found in channel %s with timestamp %s", channelID, threadTS))
	}

	return allMessages, nil
}

// HasThread checks if a message has thread replies.
// This is determined by checking the ReplyCount field of the message.
func (c *Client) HasThread(message *types.Message) bool {
	return message != nil && message.ReplyCount > 0
}

// GetCurrentUser retrieves information about the currently authenticated bot user.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// This method uses the auth.test API to identify the current user, then fetches
// their full profile information. Results are cached via GetUserInfo.
//
// Returns the current user info, or an error if the authentication test fails.
func (c *Client) GetCurrentUser(ctx context.Context) (*types.UserInfo, error) {
	// Call auth.test to get the current user ID
	authResp, err := c.api.AuthTestContext(ctx)
	if err != nil {
		return nil, wrapSlackError(err)
	}

	// Use GetUserInfo to fetch full user details (benefits from caching)
	return c.GetUserInfo(ctx, authResp.UserID)
}

// GetUserInfo retrieves user information from Slack, using a cache to minimize API calls.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - userID: The Slack user ID (e.g., "U06025G6B28")
//
// Returns the user info if found, or a placeholder for deleted users.
// Returns an error only for non-recoverable failures (e.g., invalid token).
func (c *Client) GetUserInfo(ctx context.Context, userID string) (*types.UserInfo, error) {
	// Check if user ID is empty
	if userID == "" {
		return nil, nil
	}

	// Check cache first
	if cached, ok := c.userCache.Load(userID); ok {
		return cached.(*types.UserInfo), nil
	}

	// Fetch from Slack API
	user, err := c.api.GetUserInfoContext(ctx, userID)
	if err != nil {
		// Check if user was not found (deleted user)
		errStr := err.Error()
		if strings.Contains(errStr, "user_not_found") || strings.Contains(errStr, "users_not_found") {
			// Return placeholder for deleted user
			deletedUser := &types.UserInfo{
				ID:          userID,
				Name:        "deleted_user",
				DisplayName: "Deleted User",
				RealName:    "Deleted User",
				IsBot:       false,
				IsDeleted:   true,
			}
			// Cache the placeholder to avoid repeated lookups
			c.userCache.Store(userID, deletedUser)
			return deletedUser, nil
		}
		return nil, wrapSlackError(err)
	}

	// Convert to our UserInfo type
	userInfo := convertUser(user)

	// Cache the result
	c.userCache.Store(userID, userInfo)

	return userInfo, nil
}

// convertUser converts a Slack API user to our UserInfo type.
func convertUser(user *slack.User) *types.UserInfo {
	displayName := user.Profile.DisplayName
	// Fall back to real name if display name is empty
	if displayName == "" {
		displayName = user.Profile.RealName
	}
	// Fall back to username if both are empty
	if displayName == "" {
		displayName = user.Name
	}

	return &types.UserInfo{
		ID:          user.ID,
		Name:        user.Name,
		DisplayName: displayName,
		RealName:    user.Profile.RealName,
		IsBot:       user.IsBot,
		IsDeleted:   user.Deleted,
	}
}

// convertMessage converts a Slack API message to our Message type.
func convertMessage(msg *slack.Message) *types.Message {
	return &types.Message{
		User:       msg.User,
		Text:       msg.Text,
		Timestamp:  msg.Timestamp,
		ThreadTS:   msg.ThreadTimestamp,
		ReplyCount: msg.ReplyCount,
	}
}

// ExtractMentions extracts unique user IDs from Slack mentions in the given text.
//
// Slack mentions follow the format <@UXXXXXXXX> where U followed by alphanumeric
// characters represents a user ID.
//
// Parameters:
//   - text: The message text that may contain user mentions
//
// Returns a slice of unique user IDs found in the text. Returns an empty slice
// if no mentions are found.
func (c *Client) ExtractMentions(text string) []string {
	matches := mentionPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return []string{}
	}

	// Use a map to deduplicate user IDs
	seen := make(map[string]bool)
	var userIDs []string

	for _, match := range matches {
		if len(match) >= 2 {
			userID := match[1]
			if !seen[userID] {
				seen[userID] = true
				userIDs = append(userIDs, userID)
			}
		}
	}

	return userIDs
}

// ClientInterface defines the interface for Slack client operations.
// This interface is useful for mocking in tests.
type ClientInterface interface {
	GetMessage(ctx context.Context, channelID, timestamp string) (*types.Message, error)
	GetThread(ctx context.Context, channelID, threadTS string) ([]types.Message, error)
	HasThread(message *types.Message) bool
}

// Ensure Client implements ClientInterface.
var _ ClientInterface = (*Client)(nil)
