// Package slack provides a wrapper around the Slack API client
// for fetching messages and threads.
package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"

	"github.com/slack-mcp-server/slack-mcp-server/pkg/types"
)

// Client wraps the Slack API client to provide message and thread retrieval.
type Client struct {
	api *slack.Client
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

// ClientInterface defines the interface for Slack client operations.
// This interface is useful for mocking in tests.
type ClientInterface interface {
	GetMessage(ctx context.Context, channelID, timestamp string) (*types.Message, error)
	GetThread(ctx context.Context, channelID, threadTS string) ([]types.Message, error)
	HasThread(message *types.Message) bool
}

// Ensure Client implements ClientInterface.
var _ ClientInterface = (*Client)(nil)
