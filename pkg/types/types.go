// Package types provides shared type definitions for the Slack MCP server.
package types

// UserInfo contains resolved user information from Slack.
type UserInfo struct {
	// ID is the Slack user ID (e.g., "U06025G6B28").
	ID string `json:"id"`
	// Name is the username (handle) without the @ symbol.
	Name string `json:"name"`
	// DisplayName is the user's display name, which may differ from their username.
	DisplayName string `json:"display_name"`
	// RealName is the user's full name.
	RealName string `json:"real_name"`
	// IsBot indicates whether this user is a bot account.
	IsBot bool `json:"is_bot"`
	// IsDeleted indicates whether this user account has been deleted.
	// Only set when true.
	IsDeleted bool `json:"is_deleted,omitempty"`
}

// Message represents a Slack message.
type Message struct {
	// User is the Slack user ID of the message author.
	User string `json:"user"`
	// UserName is the username (handle) of the message author, resolved from the user ID.
	// Empty if user resolution was not performed or failed.
	UserName string `json:"user_name,omitempty"`
	// DisplayName is the display name of the message author.
	// Empty if user resolution was not performed or failed.
	DisplayName string `json:"display_name,omitempty"`
	// RealName is the full name of the message author.
	// Empty if user resolution was not performed or failed.
	RealName string `json:"real_name,omitempty"`
	// Text is the message content.
	Text string `json:"text"`
	// Timestamp is the message timestamp in Slack API format (e.g., "1234567890.123456").
	Timestamp string `json:"timestamp"`
	// ThreadTS is the parent message timestamp if this message is part of a thread.
	// Empty string if the message is not a thread reply.
	ThreadTS string `json:"thread_ts,omitempty"`
	// ReplyCount is the number of replies in the thread (only set on parent messages).
	ReplyCount int `json:"reply_count,omitempty"`
}

// ParsedURL contains the components extracted from a Slack message URL.
type ParsedURL struct {
	// ChannelID is the Slack channel identifier (e.g., "C01234567").
	ChannelID string
	// Timestamp is the message timestamp in API format (e.g., "1355517523.000008").
	Timestamp string
	// ThreadTS is the parent thread timestamp, if this URL points to a thread.
	// Empty string for non-thread URLs.
	ThreadTS string
	// IsThread indicates whether this URL points to a threaded message.
	IsThread bool
}

// ReadMessageArgs is the input schema for the read_message MCP tool.
type ReadMessageArgs struct {
	// URL is the Slack message or thread URL to read.
	URL string `json:"url" jsonschema:"required,description=Slack message or thread URL to read"`
}

// ReadMessageResult is the output schema for the read_message MCP tool.
type ReadMessageResult struct {
	// Message is the primary message referenced by the URL.
	Message Message `json:"message"`
	// Thread contains all messages in the thread, including the parent.
	// Empty if the message is not part of a thread.
	Thread []Message `json:"thread,omitempty"`
	// ChannelID is the Slack channel where the message was posted.
	ChannelID string `json:"channel_id"`
	// CurrentUser contains the authenticated bot's user information.
	// Nil if user lookup was not performed or failed.
	CurrentUser *UserInfo `json:"current_user,omitempty"`
	// UserMapping maps user IDs to user info for all users mentioned in message text.
	// Empty if no mentions were found or user resolution was not performed.
	UserMapping map[string]UserInfo `json:"user_mapping,omitempty"`
}

// ListChannelMessagesResult is the output schema for the list_channel_messages MCP tool.
type ListChannelMessagesResult struct {
	// Messages contains the retrieved messages in reverse chronological order (newest first).
	Messages []Message `json:"messages"`
	// ChannelID is the Slack channel where the messages were retrieved from.
	ChannelID string `json:"channel_id"`
	// HasMore indicates whether additional messages exist beyond the requested limit.
	HasMore bool `json:"has_more"`
	// CurrentUser contains the authenticated bot's user information.
	// Nil if user lookup was not performed or failed.
	CurrentUser *UserInfo `json:"current_user,omitempty"`
	// UserMapping maps user IDs to user info for all users mentioned in message texts.
	// Empty if no mentions were found or user resolution was not performed.
	UserMapping map[string]UserInfo `json:"user_mapping,omitempty"`
}

// SearchMessagesResult is the output schema for the search_messages MCP tool.
type SearchMessagesResult struct {
	// Query is the search query that was executed.
	Query string `json:"query"`
	// Total is the total number of matching messages found.
	Total int `json:"total"`
	// Matches contains the matching messages.
	Matches []SearchMatch `json:"matches"`
	// CurrentUser contains the authenticated user's information.
	// Nil if user lookup was not performed or failed.
	CurrentUser *UserInfo `json:"current_user,omitempty"`
}

// SearchMatch represents a single message match from search results.
type SearchMatch struct {
	// ChannelID is the ID of the channel where the message was posted.
	ChannelID string `json:"channel_id"`
	// ChannelName is the name of the channel (without # prefix).
	ChannelName string `json:"channel_name"`
	// User is the Slack user ID of the message author.
	User string `json:"user"`
	// UserName is the username of the message author.
	// Empty if user resolution was not performed or failed.
	UserName string `json:"user_name,omitempty"`
	// DisplayName is the display name of the message author.
	// Empty if user resolution was not performed or failed.
	DisplayName string `json:"display_name,omitempty"`
	// RealName is the full name of the message author.
	// Empty if user resolution was not performed or failed.
	RealName string `json:"real_name,omitempty"`
	// Text is the message content.
	Text string `json:"text"`
	// Timestamp is the message timestamp in Slack API format.
	Timestamp string `json:"timestamp"`
	// Permalink is the direct URL to the message.
	Permalink string `json:"permalink"`
}

// SlackError represents an error from the Slack API or URL parsing.
type SlackError struct {
	// Code is a machine-readable error code.
	Code string `json:"code"`
	// Message is a human-readable error description.
	Message string `json:"message"`
}

// Error implements the error interface for SlackError.
func (e *SlackError) Error() string {
	return e.Message
}

// Common error codes for Slack operations.
const (
	// ErrCodeInvalidURL indicates the provided URL is not a valid Slack message URL.
	ErrCodeInvalidURL = "invalid_url"
	// ErrCodeMessageNotFound indicates the message could not be found.
	ErrCodeMessageNotFound = "message_not_found"
	// ErrCodeChannelNotFound indicates the channel could not be found.
	ErrCodeChannelNotFound = "channel_not_found"
	// ErrCodeNotInChannel indicates the bot is not a member of the channel.
	ErrCodeNotInChannel = "not_in_channel"
	// ErrCodeRateLimited indicates the Slack API rate limit was exceeded.
	ErrCodeRateLimited = "rate_limited"
	// ErrCodeInvalidToken indicates the Slack bot token is invalid or expired.
	ErrCodeInvalidToken = "invalid_token"
	// ErrCodePermissionDenied indicates the bot lacks required permissions.
	ErrCodePermissionDenied = "permission_denied"
)

// NewSlackError creates a new SlackError with the given code and message.
func NewSlackError(code, message string) *SlackError {
	return &SlackError{
		Code:    code,
		Message: message,
	}
}
