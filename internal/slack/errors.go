// Package slack provides error types and handling for Slack API operations.
package slack

import (
	"errors"
	"fmt"
	"strings"

	"github.com/slack-mcp-server/slack-mcp-server/pkg/types"
)

// Error sentinel values for common Slack API errors.
// These can be used with errors.Is() for error checking.
var (
	// ErrRateLimited indicates the Slack API rate limit was exceeded.
	ErrRateLimited = types.NewSlackError(types.ErrCodeRateLimited, "rate limit exceeded")

	// ErrInvalidToken indicates the Slack bot token is invalid or expired.
	ErrInvalidToken = types.NewSlackError(types.ErrCodeInvalidToken, "invalid or expired token")

	// ErrChannelNotFound indicates the channel could not be found.
	ErrChannelNotFound = types.NewSlackError(types.ErrCodeChannelNotFound, "channel not found")

	// ErrNotInChannel indicates the bot is not a member of the channel.
	ErrNotInChannel = types.NewSlackError(types.ErrCodeNotInChannel, "bot not in channel")

	// ErrMessageNotFound indicates the message could not be found.
	ErrMessageNotFound = types.NewSlackError(types.ErrCodeMessageNotFound, "message not found")

	// ErrPermissionDenied indicates the bot lacks required permissions.
	ErrPermissionDenied = types.NewSlackError(types.ErrCodePermissionDenied, "permission denied")
)

// IsRateLimited checks if the error is a rate limiting error.
func IsRateLimited(err error) bool {
	return isSlackErrorCode(err, types.ErrCodeRateLimited)
}

// IsInvalidToken checks if the error is an invalid token error.
func IsInvalidToken(err error) bool {
	return isSlackErrorCode(err, types.ErrCodeInvalidToken)
}

// IsChannelNotFound checks if the error is a channel not found error.
func IsChannelNotFound(err error) bool {
	return isSlackErrorCode(err, types.ErrCodeChannelNotFound)
}

// IsNotInChannel checks if the error is a "not in channel" error.
func IsNotInChannel(err error) bool {
	return isSlackErrorCode(err, types.ErrCodeNotInChannel)
}

// IsMessageNotFound checks if the error is a message not found error.
func IsMessageNotFound(err error) bool {
	return isSlackErrorCode(err, types.ErrCodeMessageNotFound)
}

// IsPermissionDenied checks if the error is a permission denied error.
func IsPermissionDenied(err error) bool {
	return isSlackErrorCode(err, types.ErrCodePermissionDenied)
}

// isSlackErrorCode checks if the error is a SlackError with the given code.
func isSlackErrorCode(err error, code string) bool {
	var slackErr *types.SlackError
	if errors.As(err, &slackErr) {
		return slackErr.Code == code
	}
	return false
}

// GetErrorCode extracts the error code from a SlackError.
// Returns an empty string if the error is not a SlackError.
func GetErrorCode(err error) string {
	var slackErr *types.SlackError
	if errors.As(err, &slackErr) {
		return slackErr.Code
	}
	return ""
}

// wrapSlackError converts Slack API errors to our typed errors.
// This function examines the error string to determine the specific error type
// and returns an appropriate SlackError with a helpful message.
func wrapSlackError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for rate limiting
	if strings.Contains(errStr, "rate_limit") || strings.Contains(errStr, "ratelimited") {
		return types.NewSlackError(types.ErrCodeRateLimited,
			"Slack API rate limit exceeded. Please wait and try again.")
	}

	// Check for authentication errors
	if strings.Contains(errStr, "invalid_auth") || strings.Contains(errStr, "not_authed") {
		return types.NewSlackError(types.ErrCodeInvalidToken,
			"Invalid or expired Slack bot token. Please check your SLACK_BOT_TOKEN.")
	}

	// Check for token scope errors
	if strings.Contains(errStr, "missing_scope") || strings.Contains(errStr, "token_expired") {
		return types.NewSlackError(types.ErrCodeInvalidToken,
			"Slack bot token lacks required scopes or has expired.")
	}

	// Check for channel not found
	if strings.Contains(errStr, "channel_not_found") {
		return types.NewSlackError(types.ErrCodeChannelNotFound,
			"Channel not found. The channel may have been deleted or the ID is incorrect.")
	}

	// Check for not in channel
	if strings.Contains(errStr, "not_in_channel") {
		return types.NewSlackError(types.ErrCodeNotInChannel,
			"Bot is not a member of this channel. Please invite the bot to the channel.")
	}

	// Check for permission denied
	if strings.Contains(errStr, "access_denied") || strings.Contains(errStr, "is_archived") {
		return types.NewSlackError(types.ErrCodePermissionDenied,
			"Access denied. The channel may be archived or the bot lacks permissions.")
	}

	// Check for message not found
	if strings.Contains(errStr, "message_not_found") || strings.Contains(errStr, "thread_not_found") {
		return types.NewSlackError(types.ErrCodeMessageNotFound,
			"Message or thread not found.")
	}

	// Generic error wrapping
	return types.NewSlackError("slack_error", fmt.Sprintf("Slack API error: %s", errStr))
}
