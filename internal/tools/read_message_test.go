// Package tools provides unit tests for the MCP tool handlers.
package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	slackclient "github.com/slack-mcp-server/slack-mcp-server/internal/slack"
	"github.com/slack-mcp-server/slack-mcp-server/pkg/types"
)

// mockSlackClient is a test double for the Slack client interface.
type mockSlackClient struct {
	getMessage      func(ctx context.Context, channelID, timestamp string) (*types.Message, error)
	getThread       func(ctx context.Context, channelID, threadTS string) ([]types.Message, error)
	hasThread       func(message *types.Message) bool
	getUserInfo     func(ctx context.Context, userID string) (*types.UserInfo, error)
	getCurrentUser  func(ctx context.Context) (*types.UserInfo, error)
	extractMentions func(text string) []string
}

// GetMessage implements slackclient.ClientInterface.
func (m *mockSlackClient) GetMessage(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
	if m.getMessage != nil {
		return m.getMessage(ctx, channelID, timestamp)
	}
	return nil, types.NewSlackError(types.ErrCodeMessageNotFound, "mock: GetMessage not configured")
}

// GetThread implements slackclient.ClientInterface.
func (m *mockSlackClient) GetThread(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
	if m.getThread != nil {
		return m.getThread(ctx, channelID, threadTS)
	}
	return nil, types.NewSlackError(types.ErrCodeMessageNotFound, "mock: GetThread not configured")
}

// HasThread implements slackclient.ClientInterface.
func (m *mockSlackClient) HasThread(message *types.Message) bool {
	if m.hasThread != nil {
		return m.hasThread(message)
	}
	// Default behavior: check ReplyCount > 0
	return message != nil && message.ReplyCount > 0
}

// GetUserInfo implements slackclient.ClientInterface.
func (m *mockSlackClient) GetUserInfo(ctx context.Context, userID string) (*types.UserInfo, error) {
	if m.getUserInfo != nil {
		return m.getUserInfo(ctx, userID)
	}
	// Default: return nil to simulate user not found
	return nil, nil
}

// GetCurrentUser implements slackclient.ClientInterface.
func (m *mockSlackClient) GetCurrentUser(ctx context.Context) (*types.UserInfo, error) {
	if m.getCurrentUser != nil {
		return m.getCurrentUser(ctx)
	}
	// Default: return a mock current user
	return &types.UserInfo{
		ID:          "UBOT12345",
		Name:        "test_bot",
		DisplayName: "Test Bot",
		RealName:    "Test Bot",
		IsBot:       true,
	}, nil
}

// ExtractMentions implements slackclient.ClientInterface.
func (m *mockSlackClient) ExtractMentions(text string) []string {
	if m.extractMentions != nil {
		return m.extractMentions(text)
	}
	// Default: return empty slice (no mentions)
	return []string{}
}

// Ensure mockSlackClient implements the interface.
var _ slackclient.ClientInterface = (*mockSlackClient)(nil)

// createToolRequest creates an MCP CallToolRequest with the given arguments.
func createToolRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      "read_message",
			Arguments: args,
		},
	}
}

func TestReadMessageHandler_Handle_Success(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockMessage    *types.Message
		mockThread     []types.Message
		hasThread      bool
		wantChannelID  string
		wantTimestamp  string
		wantThreadLen  int
	}{
		{
			name: "simple message without thread",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hello, world!",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread:     false,
			wantChannelID: "C01234567",
			wantTimestamp: "1355517523.000008",
			wantThreadLen: 0,
		},
		{
			name: "message with thread auto-detected via ReplyCount",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Thread parent",
				Timestamp:  "1355517523.000008",
				ReplyCount: 2,
			},
			mockThread: []types.Message{
				{
					User:      "U12345678",
					Text:      "Thread parent",
					Timestamp: "1355517523.000008",
				},
				{
					User:      "U87654321",
					Text:      "First reply",
					Timestamp: "1355517524.000001",
					ThreadTS:  "1355517523.000008",
				},
				{
					User:      "U12345678",
					Text:      "Second reply",
					Timestamp: "1355517525.000002",
					ThreadTS:  "1355517523.000008",
				},
			},
			hasThread:     true,
			wantChannelID: "C01234567",
			wantTimestamp: "1355517523.000008",
			wantThreadLen: 3,
		},
		{
			name: "thread URL with thread_ts parameter",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517524000001?thread_ts=1355517523.000008&cid=C01234567",
			mockMessage: &types.Message{
				User:      "U87654321",
				Text:      "Reply message",
				Timestamp: "1355517524.000001",
				ThreadTS:  "1355517523.000008",
			},
			mockThread: []types.Message{
				{
					User:      "U12345678",
					Text:      "Parent",
					Timestamp: "1355517523.000008",
				},
				{
					User:      "U87654321",
					Text:      "Reply message",
					Timestamp: "1355517524.000001",
					ThreadTS:  "1355517523.000008",
				},
			},
			hasThread:     false, // Not auto-detected, but URL indicates thread
			wantChannelID: "C01234567",
			wantTimestamp: "1355517524.000001",
			wantThreadLen: 2,
		},
		{
			name: "message in DM channel",
			url:  "https://workspace.slack.com/archives/D01234567/p1234567890123456",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "DM message",
				Timestamp:  "1234567890.123456",
				ReplyCount: 0,
			},
			hasThread:     false,
			wantChannelID: "D01234567",
			wantTimestamp: "1234567890.123456",
			wantThreadLen: 0,
		},
		{
			name: "message in private channel",
			url:  "https://workspace.slack.com/archives/G01234567/p1234567890123456",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Private channel message",
				Timestamp:  "1234567890.123456",
				ReplyCount: 0,
			},
			hasThread:     false,
			wantChannelID: "G01234567",
			wantTimestamp: "1234567890.123456",
			wantThreadLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
					if channelID != tt.wantChannelID {
						t.Errorf("GetMessage channelID = %q, want %q", channelID, tt.wantChannelID)
					}
					if timestamp != tt.wantTimestamp {
						t.Errorf("GetMessage timestamp = %q, want %q", timestamp, tt.wantTimestamp)
					}
					return tt.mockMessage, nil
				},
				getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
					return tt.mockThread, nil
				},
				hasThread: func(message *types.Message) bool {
					return tt.hasThread
				},
			}

			handler := NewReadMessageHandler(mock)
			request := createToolRequest(map[string]interface{}{
				"url": tt.url,
			})

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError {
				t.Fatalf("expected success, got error: %+v", result.Content)
			}

			// Parse the result JSON
			if len(result.Content) == 0 {
				t.Fatal("expected content in result")
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("expected TextContent, got %T", result.Content[0])
			}

			var readResult types.ReadMessageResult
			if err := json.Unmarshal([]byte(textContent.Text), &readResult); err != nil {
				t.Fatalf("failed to parse result JSON: %v", err)
			}

			if readResult.ChannelID != tt.wantChannelID {
				t.Errorf("result ChannelID = %q, want %q", readResult.ChannelID, tt.wantChannelID)
			}

			if readResult.Message.Text != tt.mockMessage.Text {
				t.Errorf("result Message.Text = %q, want %q", readResult.Message.Text, tt.mockMessage.Text)
			}

			if len(readResult.Thread) != tt.wantThreadLen {
				t.Errorf("result Thread length = %d, want %d", len(readResult.Thread), tt.wantThreadLen)
			}
		})
	}
}

func TestReadMessageHandler_Handle_MissingURL(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewReadMessageHandler(mock)

	// Test with no arguments
	request := createToolRequest(map[string]interface{}{})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing URL")
	}

	// Check error message
	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "url") {
		t.Errorf("error message should mention 'url', got: %s", textContent.Text)
	}
}

func TestReadMessageHandler_Handle_InvalidURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantErrContain string
	}{
		{
			name:           "empty URL",
			url:            "",
			wantErrContain: "url", // Handler checks for empty URL first with "missing required argument 'url'"
		},
		{
			name:           "non-Slack URL",
			url:            "https://google.com/archives/C01234567/p1355517523000008",
			wantErrContain: "Invalid Slack URL",
		},
		{
			name:           "missing timestamp",
			url:            "https://workspace.slack.com/archives/C01234567",
			wantErrContain: "Invalid Slack URL",
		},
		{
			name:           "malformed URL",
			url:            "not-a-valid-url",
			wantErrContain: "Invalid Slack URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{}
			handler := NewReadMessageHandler(mock)
			request := createToolRequest(map[string]interface{}{
				"url": tt.url,
			})

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.IsError {
				t.Error("expected error result for invalid URL")
			}

			// Check error message
			if len(result.Content) == 0 {
				t.Fatal("expected error content")
			}
			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("expected TextContent, got %T", result.Content[0])
			}
			if !strings.Contains(textContent.Text, tt.wantErrContain) {
				t.Errorf("error message should contain %q, got: %s", tt.wantErrContain, textContent.Text)
			}
		})
	}
}

func TestReadMessageHandler_Handle_SlackErrors(t *testing.T) {
	tests := []struct {
		name           string
		errorCode      string
		wantErrContain string
	}{
		{
			name:           "rate limited",
			errorCode:      types.ErrCodeRateLimited,
			wantErrContain: "Rate limit exceeded",
		},
		{
			name:           "invalid token",
			errorCode:      types.ErrCodeInvalidToken,
			wantErrContain: "Authentication failed",
		},
		{
			name:           "channel not found",
			errorCode:      types.ErrCodeChannelNotFound,
			wantErrContain: "Channel not found",
		},
		{
			name:           "not in channel",
			errorCode:      types.ErrCodeNotInChannel,
			wantErrContain: "not a member of this channel",
		},
		{
			name:           "message not found",
			errorCode:      types.ErrCodeMessageNotFound,
			wantErrContain: "Message not found",
		},
		{
			name:           "permission denied",
			errorCode:      types.ErrCodePermissionDenied,
			wantErrContain: "Permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
					return nil, types.NewSlackError(tt.errorCode, "mock error")
				},
			}
			handler := NewReadMessageHandler(mock)
			request := createToolRequest(map[string]interface{}{
				"url": "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			})

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.IsError {
				t.Error("expected error result")
			}

			// Check error message
			if len(result.Content) == 0 {
				t.Fatal("expected error content")
			}
			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("expected TextContent, got %T", result.Content[0])
			}
			if !strings.Contains(textContent.Text, tt.wantErrContain) {
				t.Errorf("error message should contain %q, got: %s", tt.wantErrContain, textContent.Text)
			}
		})
	}
}

func TestReadMessageHandler_Handle_PartialResult(t *testing.T) {
	// Test that thread fetch failure returns partial result with message
	mock := &mockSlackClient{
		getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
			return &types.Message{
				User:       "U12345678",
				Text:       "Parent message",
				Timestamp:  "1355517523.000008",
				ReplyCount: 2,
			}, nil
		},
		getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
			return nil, types.NewSlackError(types.ErrCodeRateLimited, "rate limited during thread fetch")
		},
		hasThread: func(message *types.Message) bool {
			return true // Trigger thread fetch
		},
	}

	handler := NewReadMessageHandler(mock)
	request := createToolRequest(map[string]interface{}{
		"url": "https://workspace.slack.com/archives/C01234567/p1355517523000008",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT be an error result - partial success
	if result.IsError {
		t.Error("expected partial success, not full error")
	}

	// Check that result contains both the message and a warning
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	// Should contain the message data
	if !strings.Contains(textContent.Text, "Parent message") {
		t.Error("partial result should contain the message text")
	}

	// Should contain a note about the thread failure
	if !strings.Contains(textContent.Text, "Failed to fetch thread") {
		t.Error("partial result should note the thread fetch failure")
	}
}

func TestReadMessageHandler_HandleFunc(t *testing.T) {
	// Test that HandleFunc returns a usable function
	mock := &mockSlackClient{
		getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
			return &types.Message{
				User:      "U12345678",
				Text:      "Test message",
				Timestamp: "1355517523.000008",
			}, nil
		},
	}

	handler := NewReadMessageHandler(mock)
	handlerFunc := handler.HandleFunc()

	if handlerFunc == nil {
		t.Fatal("HandleFunc returned nil")
	}

	request := createToolRequest(map[string]interface{}{
		"url": "https://workspace.slack.com/archives/C01234567/p1355517523000008",
	})

	result, err := handlerFunc(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Error("expected success result")
	}
}

func TestReadMessage_Standalone(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		mockMessage   *types.Message
		mockThread    []types.Message
		wantErr       bool
		wantErrCode   string
		wantThreadLen int
	}{
		{
			name: "success without thread",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hello",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			wantErr:       false,
			wantThreadLen: 0,
		},
		{
			name: "success with thread",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Parent",
				Timestamp:  "1355517523.000008",
				ReplyCount: 1,
			},
			mockThread: []types.Message{
				{User: "U12345678", Text: "Parent", Timestamp: "1355517523.000008"},
				{User: "U87654321", Text: "Reply", Timestamp: "1355517524.000001"},
			},
			wantErr:       false,
			wantThreadLen: 2,
		},
		{
			name:        "invalid URL",
			url:         "not-a-valid-url",
			wantErr:     true,
			wantErrCode: types.ErrCodeInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
					if tt.mockMessage == nil {
						return nil, types.NewSlackError(types.ErrCodeMessageNotFound, "not found")
					}
					return tt.mockMessage, nil
				},
				getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
					if tt.mockThread == nil {
						return nil, types.NewSlackError(types.ErrCodeMessageNotFound, "no thread")
					}
					return tt.mockThread, nil
				},
			}

			result, err := ReadMessage(context.Background(), mock, tt.url)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.wantErrCode != "" {
					slackErr, ok := err.(*types.SlackError)
					if !ok {
						t.Errorf("expected *types.SlackError, got %T", err)
					} else if slackErr.Code != tt.wantErrCode {
						t.Errorf("error code = %q, want %q", slackErr.Code, tt.wantErrCode)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected result, got nil")
			}

			if len(result.Thread) != tt.wantThreadLen {
				t.Errorf("thread length = %d, want %d", len(result.Thread), tt.wantThreadLen)
			}
		})
	}
}

func TestReadMessage_Standalone_ThreadFetchError(t *testing.T) {
	// Standalone function should return error when thread fetch fails
	mock := &mockSlackClient{
		getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
			return &types.Message{
				User:       "U12345678",
				Text:       "Parent",
				Timestamp:  "1355517523.000008",
				ReplyCount: 2, // Has thread
			}, nil
		},
		getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
			return nil, types.NewSlackError(types.ErrCodeRateLimited, "rate limited")
		},
	}

	_, err := ReadMessage(context.Background(), mock, "https://workspace.slack.com/archives/C01234567/p1355517523000008")

	if err == nil {
		t.Error("expected error for thread fetch failure")
	}

	if !strings.Contains(err.Error(), "thread") {
		t.Errorf("error should mention 'thread', got: %s", err.Error())
	}
}

func TestReadMessageHandler_Handle_ThreadTSFromURL(t *testing.T) {
	// Test that thread_ts from URL is used correctly
	var capturedThreadTS string

	mock := &mockSlackClient{
		getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
			// URL points to a reply message
			return &types.Message{
				User:       "U87654321",
				Text:       "Reply message",
				Timestamp:  "1355517524.000001",
				ThreadTS:   "1355517523.000008",
				ReplyCount: 0, // Reply doesn't have ReplyCount
			}, nil
		},
		getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
			capturedThreadTS = threadTS
			return []types.Message{
				{User: "U12345678", Text: "Parent", Timestamp: "1355517523.000008"},
				{User: "U87654321", Text: "Reply", Timestamp: "1355517524.000001"},
			}, nil
		},
		hasThread: func(message *types.Message) bool {
			return false // Don't auto-detect, rely on URL
		},
	}

	handler := NewReadMessageHandler(mock)
	// URL includes thread_ts parameter
	request := createToolRequest(map[string]interface{}{
		"url": "https://workspace.slack.com/archives/C01234567/p1355517524000001?thread_ts=1355517523.000008&cid=C01234567",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// Verify thread_ts from URL was used
	if capturedThreadTS != "1355517523.000008" {
		t.Errorf("expected thread_ts from URL (1355517523.000008), got: %s", capturedThreadTS)
	}
}

func TestReadMessageHandler_Handle_ThreadTSFromMessage(t *testing.T) {
	// Test that when URL doesn't have thread_ts but message has replies,
	// the message's timestamp is used as thread_ts
	var capturedThreadTS string

	mock := &mockSlackClient{
		getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
			return &types.Message{
				User:       "U12345678",
				Text:       "Parent with replies",
				Timestamp:  "1355517523.000008",
				ReplyCount: 3,
			}, nil
		},
		getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
			capturedThreadTS = threadTS
			return []types.Message{
				{User: "U12345678", Text: "Parent", Timestamp: "1355517523.000008"},
			}, nil
		},
		hasThread: func(message *types.Message) bool {
			return true // Auto-detect thread
		},
	}

	handler := NewReadMessageHandler(mock)
	// URL without thread_ts parameter
	request := createToolRequest(map[string]interface{}{
		"url": "https://workspace.slack.com/archives/C01234567/p1355517523000008",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// Verify message's timestamp was used as thread_ts
	if capturedThreadTS != "1355517523.000008" {
		t.Errorf("expected message timestamp (1355517523.000008), got: %s", capturedThreadTS)
	}
}

func TestNewReadMessageHandler(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewReadMessageHandler(mock)

	if handler == nil {
		t.Fatal("NewReadMessageHandler returned nil")
	}

	if handler.slackClient != mock {
		t.Error("handler did not store the provided client")
	}
}

// TestReadMessage_UserResolution tests that user resolution populates name fields on messages.
func TestReadMessage_UserResolution(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		mockMessage     *types.Message
		mockThread      []types.Message
		hasThread       bool
		userInfoMap     map[string]*types.UserInfo
		wantUserName    string
		wantDisplayName string
		wantRealName    string
		wantThreadNames []string // Expected user_name for each thread message
	}{
		{
			name: "single message user resolution",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hello, world!",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread: false,
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {
					ID:          "U12345678",
					Name:        "johndoe",
					DisplayName: "John Doe",
					RealName:    "John D. Doe",
					IsBot:       false,
				},
			},
			wantUserName:    "johndoe",
			wantDisplayName: "John Doe",
			wantRealName:    "John D. Doe",
		},
		{
			name: "thread messages user resolution",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Thread parent",
				Timestamp:  "1355517523.000008",
				ReplyCount: 2,
			},
			mockThread: []types.Message{
				{
					User:      "U12345678",
					Text:      "Thread parent",
					Timestamp: "1355517523.000008",
				},
				{
					User:      "U87654321",
					Text:      "First reply",
					Timestamp: "1355517524.000001",
					ThreadTS:  "1355517523.000008",
				},
				{
					User:      "UAAAAAAAA",
					Text:      "Second reply",
					Timestamp: "1355517525.000002",
					ThreadTS:  "1355517523.000008",
				},
			},
			hasThread: true,
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {
					ID:          "U12345678",
					Name:        "alice",
					DisplayName: "Alice",
					RealName:    "Alice Smith",
					IsBot:       false,
				},
				"U87654321": {
					ID:          "U87654321",
					Name:        "bob",
					DisplayName: "Bob",
					RealName:    "Bob Jones",
					IsBot:       false,
				},
				"UAAAAAAAA": {
					ID:          "UAAAAAAAA",
					Name:        "charlie",
					DisplayName: "Charlie",
					RealName:    "Charlie Brown",
					IsBot:       false,
				},
			},
			wantUserName:    "alice",
			wantDisplayName: "Alice",
			wantRealName:    "Alice Smith",
			wantThreadNames: []string{"alice", "bob", "charlie"},
		},
		{
			name: "bot user resolution",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "UBOTUSER1",
				Text:       "Bot message",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread: false,
			userInfoMap: map[string]*types.UserInfo{
				"UBOTUSER1": {
					ID:          "UBOTUSER1",
					Name:        "mybot",
					DisplayName: "My Bot",
					RealName:    "My Bot App",
					IsBot:       true,
				},
			},
			wantUserName:    "mybot",
			wantDisplayName: "My Bot",
			wantRealName:    "My Bot App",
		},
		{
			name: "user resolution graceful failure",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Message from unknown user",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread:       false,
			userInfoMap:     map[string]*types.UserInfo{}, // No user info available
			wantUserName:    "",                           // Should be empty, not fail
			wantDisplayName: "",
			wantRealName:    "",
		},
		{
			name: "system message with empty user field",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "", // System message has no user
				Text:       "Channel joined",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread:       false,
			userInfoMap:     map[string]*types.UserInfo{},
			wantUserName:    "", // Should handle gracefully
			wantDisplayName: "",
			wantRealName:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
					return tt.mockMessage, nil
				},
				getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
					return tt.mockThread, nil
				},
				hasThread: func(message *types.Message) bool {
					return tt.hasThread
				},
				getUserInfo: func(ctx context.Context, userID string) (*types.UserInfo, error) {
					if info, ok := tt.userInfoMap[userID]; ok {
						return info, nil
					}
					return nil, nil // User not found, graceful degradation
				},
			}

			handler := NewReadMessageHandler(mock)
			request := createToolRequest(map[string]interface{}{
				"url": tt.url,
			})

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError {
				t.Fatalf("expected success, got error: %+v", result.Content)
			}

			// Parse the result JSON
			if len(result.Content) == 0 {
				t.Fatal("expected content in result")
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("expected TextContent, got %T", result.Content[0])
			}

			var readResult types.ReadMessageResult
			if err := json.Unmarshal([]byte(textContent.Text), &readResult); err != nil {
				t.Fatalf("failed to parse result JSON: %v", err)
			}

			// Verify primary message user resolution
			if readResult.Message.UserName != tt.wantUserName {
				t.Errorf("Message.UserName = %q, want %q", readResult.Message.UserName, tt.wantUserName)
			}
			if readResult.Message.DisplayName != tt.wantDisplayName {
				t.Errorf("Message.DisplayName = %q, want %q", readResult.Message.DisplayName, tt.wantDisplayName)
			}
			if readResult.Message.RealName != tt.wantRealName {
				t.Errorf("Message.RealName = %q, want %q", readResult.Message.RealName, tt.wantRealName)
			}

			// Verify thread message user resolution if applicable
			if tt.wantThreadNames != nil {
				if len(readResult.Thread) != len(tt.wantThreadNames) {
					t.Fatalf("Thread length = %d, want %d", len(readResult.Thread), len(tt.wantThreadNames))
				}
				for i, wantName := range tt.wantThreadNames {
					if readResult.Thread[i].UserName != wantName {
						t.Errorf("Thread[%d].UserName = %q, want %q", i, readResult.Thread[i].UserName, wantName)
					}
				}
			}
		})
	}
}

// TestReadMessage_MentionMapping tests that mentioned users are mapped correctly.
func TestReadMessage_MentionMapping(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		mockMessage      *types.Message
		mockThread       []types.Message
		hasThread        bool
		extractedIDs     map[string][]string // text -> extracted user IDs
		userInfoMap      map[string]*types.UserInfo
		wantMappingCount int
		wantMappedUsers  []string // Expected user IDs in mapping
	}{
		{
			name: "single mention in message",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hey <@U87654321>, can you help?",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread: false,
			extractedIDs: map[string][]string{
				"Hey <@U87654321>, can you help?": {"U87654321"},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
				"U87654321": {ID: "U87654321", Name: "bob", DisplayName: "Bob", RealName: "Bob Jones"},
			},
			wantMappingCount: 1,
			wantMappedUsers:  []string{"U87654321"},
		},
		{
			name: "multiple mentions in message",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hey <@U87654321> and <@UAAAAAAAA>, please review",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread: false,
			extractedIDs: map[string][]string{
				"Hey <@U87654321> and <@UAAAAAAAA>, please review": {"U87654321", "UAAAAAAAA"},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
				"U87654321": {ID: "U87654321", Name: "bob"},
				"UAAAAAAAA": {ID: "UAAAAAAAA", Name: "charlie"},
			},
			wantMappingCount: 2,
			wantMappedUsers:  []string{"U87654321", "UAAAAAAAA"},
		},
		{
			name: "mentions in thread messages",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Thread parent",
				Timestamp:  "1355517523.000008",
				ReplyCount: 1,
			},
			mockThread: []types.Message{
				{
					User:      "U12345678",
					Text:      "Thread parent",
					Timestamp: "1355517523.000008",
				},
				{
					User:      "U87654321",
					Text:      "Hey <@UAAAAAAAA>, what do you think?",
					Timestamp: "1355517524.000001",
					ThreadTS:  "1355517523.000008",
				},
			},
			hasThread: true,
			extractedIDs: map[string][]string{
				"Thread parent":                           {},
				"Hey <@UAAAAAAAA>, what do you think?": {"UAAAAAAAA"},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
				"U87654321": {ID: "U87654321", Name: "bob"},
				"UAAAAAAAA": {ID: "UAAAAAAAA", Name: "charlie"},
			},
			wantMappingCount: 1,
			wantMappedUsers:  []string{"UAAAAAAAA"},
		},
		{
			name: "no mentions in message",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hello, world!",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread: false,
			extractedIDs: map[string][]string{
				"Hello, world!": {},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
			},
			wantMappingCount: 0,
			wantMappedUsers:  nil,
		},
		{
			name: "mentioned user not found gracefully",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hey <@UDELETED1>, are you there?",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			hasThread: false,
			extractedIDs: map[string][]string{
				"Hey <@UDELETED1>, are you there?": {"UDELETED1"},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
				// UDELETED1 is not in the map - simulates deleted user
			},
			wantMappingCount: 0, // Should gracefully skip unresolvable users
			wantMappedUsers:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
					return tt.mockMessage, nil
				},
				getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
					return tt.mockThread, nil
				},
				hasThread: func(message *types.Message) bool {
					return tt.hasThread
				},
				getUserInfo: func(ctx context.Context, userID string) (*types.UserInfo, error) {
					if info, ok := tt.userInfoMap[userID]; ok {
						return info, nil
					}
					return nil, nil // User not found
				},
				extractMentions: func(text string) []string {
					if ids, ok := tt.extractedIDs[text]; ok {
						return ids
					}
					return []string{}
				},
			}

			handler := NewReadMessageHandler(mock)
			request := createToolRequest(map[string]interface{}{
				"url": tt.url,
			})

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError {
				t.Fatalf("expected success, got error: %+v", result.Content)
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("expected TextContent, got %T", result.Content[0])
			}

			var readResult types.ReadMessageResult
			if err := json.Unmarshal([]byte(textContent.Text), &readResult); err != nil {
				t.Fatalf("failed to parse result JSON: %v", err)
			}

			// Verify user mapping count
			if len(readResult.UserMapping) != tt.wantMappingCount {
				t.Errorf("UserMapping length = %d, want %d", len(readResult.UserMapping), tt.wantMappingCount)
			}

			// Verify expected users are in the mapping
			for _, wantUserID := range tt.wantMappedUsers {
				if _, ok := readResult.UserMapping[wantUserID]; !ok {
					t.Errorf("UserMapping missing expected user %q", wantUserID)
				}
			}
		})
	}
}

// TestReadMessage_CurrentUser tests that the authenticated user is included in the response.
func TestReadMessage_CurrentUser(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		mockMessage     *types.Message
		currentUser     *types.UserInfo
		currentUserErr  error
		wantCurrentUser bool
		wantUserID      string
		wantUserName    string
		wantIsBot       bool
	}{
		{
			name: "current user included in response",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hello, world!",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			currentUser: &types.UserInfo{
				ID:          "UBOTUSER1",
				Name:        "my_slack_bot",
				DisplayName: "My Slack Bot",
				RealName:    "My Slack Bot",
				IsBot:       true,
			},
			wantCurrentUser: true,
			wantUserID:      "UBOTUSER1",
			wantUserName:    "my_slack_bot",
			wantIsBot:       true,
		},
		{
			name: "current user fetch failure graceful degradation",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hello, world!",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			currentUser:     nil,
			currentUserErr:  types.NewSlackError(types.ErrCodeRateLimited, "rate limited"),
			wantCurrentUser: false, // Should be nil, not fail
		},
		{
			name: "current user nil without error",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hello, world!",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			currentUser:     nil,
			currentUserErr:  nil,
			wantCurrentUser: false,
		},
		{
			name: "non-bot current user",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			mockMessage: &types.Message{
				User:       "U12345678",
				Text:       "Hello, world!",
				Timestamp:  "1355517523.000008",
				ReplyCount: 0,
			},
			currentUser: &types.UserInfo{
				ID:          "UREGUSER1",
				Name:        "regular_user",
				DisplayName: "Regular User",
				RealName:    "Regular User",
				IsBot:       false,
			},
			wantCurrentUser: true,
			wantUserID:      "UREGUSER1",
			wantUserName:    "regular_user",
			wantIsBot:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getMessage: func(ctx context.Context, channelID, timestamp string) (*types.Message, error) {
					return tt.mockMessage, nil
				},
				getThread: func(ctx context.Context, channelID, threadTS string) ([]types.Message, error) {
					return nil, nil
				},
				hasThread: func(message *types.Message) bool {
					return false
				},
				getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
					return tt.currentUser, tt.currentUserErr
				},
			}

			handler := NewReadMessageHandler(mock)
			request := createToolRequest(map[string]interface{}{
				"url": tt.url,
			})

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError {
				t.Fatalf("expected success, got error: %+v", result.Content)
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("expected TextContent, got %T", result.Content[0])
			}

			var readResult types.ReadMessageResult
			if err := json.Unmarshal([]byte(textContent.Text), &readResult); err != nil {
				t.Fatalf("failed to parse result JSON: %v", err)
			}

			// Verify current user presence
			if tt.wantCurrentUser {
				if readResult.CurrentUser == nil {
					t.Fatal("expected CurrentUser, got nil")
				}
				if readResult.CurrentUser.ID != tt.wantUserID {
					t.Errorf("CurrentUser.ID = %q, want %q", readResult.CurrentUser.ID, tt.wantUserID)
				}
				if readResult.CurrentUser.Name != tt.wantUserName {
					t.Errorf("CurrentUser.Name = %q, want %q", readResult.CurrentUser.Name, tt.wantUserName)
				}
				if readResult.CurrentUser.IsBot != tt.wantIsBot {
					t.Errorf("CurrentUser.IsBot = %v, want %v", readResult.CurrentUser.IsBot, tt.wantIsBot)
				}
			} else {
				if readResult.CurrentUser != nil {
					t.Errorf("expected CurrentUser to be nil, got %+v", readResult.CurrentUser)
				}
			}
		})
	}
}
