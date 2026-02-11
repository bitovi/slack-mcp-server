// Package tools provides unit tests for the MCP tool handlers.
package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Bitovi/slack-mcp-server/pkg/types"
)

// createListChannelMessagesRequest creates an MCP CallToolRequest for list_channel_messages with the given arguments.
func createListChannelMessagesRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      "list_channel_messages",
			Arguments: args,
		},
	}
}

func TestListChannelMessagesHandler_Handle_Success(t *testing.T) {
	tests := []struct {
		name             string
		channelID        string
		limit            int
		oldest           string
		latest           string
		mockMessages     []types.Message
		mockHasMore      bool
		userInfoMap      map[string]*types.UserInfo
		wantMessageCount int
		wantHasMore      bool
		wantUserNames    []string // Expected user_name for each message
	}{
		{
			name:      "basic message retrieval",
			channelID: "C01234567",
			limit:     100,
			mockMessages: []types.Message{
				{
					User:       "U12345678",
					Text:       "Hello, world!",
					Timestamp:  "1355517523.000008",
					ReplyCount: 0,
				},
				{
					User:       "U87654321",
					Text:       "Hi there!",
					Timestamp:  "1355517524.000009",
					ReplyCount: 2,
				},
			},
			mockHasMore: false,
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
			},
			wantMessageCount: 2,
			wantHasMore:      false,
			wantUserNames:    []string{"alice", "bob"},
		},
		{
			name:             "empty channel returns empty array",
			channelID:        "C01234567",
			limit:            100,
			mockMessages:     []types.Message{},
			mockHasMore:      false,
			userInfoMap:      map[string]*types.UserInfo{},
			wantMessageCount: 0,
			wantHasMore:      false,
			wantUserNames:    nil,
		},
		{
			name:      "messages with user resolution",
			channelID: "C01234567",
			limit:     100,
			mockMessages: []types.Message{
				{
					User:       "U11111111",
					Text:       "First message from user 1",
					Timestamp:  "1355517523.000001",
					ReplyCount: 0,
				},
				{
					User:       "U22222222",
					Text:       "Second message from user 2",
					Timestamp:  "1355517523.000002",
					ReplyCount: 0,
				},
				{
					User:       "U11111111",
					Text:       "Third message from user 1 again",
					Timestamp:  "1355517523.000003",
					ReplyCount: 0,
				},
			},
			mockHasMore: false,
			userInfoMap: map[string]*types.UserInfo{
				"U11111111": {
					ID:          "U11111111",
					Name:        "johndoe",
					DisplayName: "John Doe",
					RealName:    "John D. Doe",
					IsBot:       false,
				},
				"U22222222": {
					ID:          "U22222222",
					Name:        "janedoe",
					DisplayName: "Jane Doe",
					RealName:    "Jane D. Doe",
					IsBot:       false,
				},
			},
			wantMessageCount: 3,
			wantHasMore:      false,
			wantUserNames:    []string{"johndoe", "janedoe", "johndoe"},
		},
		{
			name:      "with has_more flag",
			channelID: "C01234567",
			limit:     50,
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Message 1",
					Timestamp: "1355517523.000008",
				},
			},
			mockHasMore: true,
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
			},
			wantMessageCount: 1,
			wantHasMore:      true,
			wantUserNames:    []string{"alice"},
		},
		{
			name:      "with oldest and latest filters",
			channelID: "C01234567",
			oldest:    "1355500000.000000",
			latest:    "1355600000.000000",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Filtered message",
					Timestamp: "1355517523.000008",
				},
			},
			mockHasMore:      false,
			userInfoMap:      map[string]*types.UserInfo{},
			wantMessageCount: 1,
			wantHasMore:      false,
			wantUserNames:    []string{""},
		},
		{
			name:      "user resolution graceful failure",
			channelID: "C01234567",
			limit:     100,
			mockMessages: []types.Message{
				{
					User:       "UUNKNOWN1",
					Text:       "Message from unknown user",
					Timestamp:  "1355517523.000008",
					ReplyCount: 0,
				},
			},
			mockHasMore:      false,
			userInfoMap:      map[string]*types.UserInfo{}, // No user info available
			wantMessageCount: 1,
			wantHasMore:      false,
			wantUserNames:    []string{""}, // Should be empty, not fail
		},
		{
			name:      "bot user resolution",
			channelID: "C01234567",
			limit:     100,
			mockMessages: []types.Message{
				{
					User:       "UBOTUSER1",
					Text:       "Bot message",
					Timestamp:  "1355517523.000008",
					ReplyCount: 0,
				},
			},
			mockHasMore: false,
			userInfoMap: map[string]*types.UserInfo{
				"UBOTUSER1": {
					ID:          "UBOTUSER1",
					Name:        "mybot",
					DisplayName: "My Bot",
					RealName:    "My Bot App",
					IsBot:       true,
				},
			},
			wantMessageCount: 1,
			wantHasMore:      false,
			wantUserNames:    []string{"mybot"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
					if channelID != tt.channelID {
						t.Errorf("GetChannelHistory channelID = %q, want %q", channelID, tt.channelID)
					}
					if tt.oldest != "" && oldest != tt.oldest {
						t.Errorf("GetChannelHistory oldest = %q, want %q", oldest, tt.oldest)
					}
					if tt.latest != "" && latest != tt.latest {
						t.Errorf("GetChannelHistory latest = %q, want %q", latest, tt.latest)
					}
					return tt.mockMessages, tt.mockHasMore, nil
				},
				getUserInfo: func(ctx context.Context, userID string) (*types.UserInfo, error) {
					if info, ok := tt.userInfoMap[userID]; ok {
						return info, nil
					}
					return nil, nil
				},
			}

			handler := NewListChannelMessagesHandler(mock)
			args := map[string]interface{}{
				"channel_id": tt.channelID,
			}
			if tt.limit > 0 {
				args["limit"] = float64(tt.limit)
			}
			if tt.oldest != "" {
				args["oldest"] = tt.oldest
			}
			if tt.latest != "" {
				args["latest"] = tt.latest
			}
			request := createListChannelMessagesRequest(args)

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

			var listResult types.ListChannelMessagesResult
			if err := json.Unmarshal([]byte(textContent.Text), &listResult); err != nil {
				t.Fatalf("failed to parse result JSON: %v", err)
			}

			if listResult.ChannelID != tt.channelID {
				t.Errorf("result ChannelID = %q, want %q", listResult.ChannelID, tt.channelID)
			}

			if len(listResult.Messages) != tt.wantMessageCount {
				t.Errorf("result Messages length = %d, want %d", len(listResult.Messages), tt.wantMessageCount)
			}

			if listResult.HasMore != tt.wantHasMore {
				t.Errorf("result HasMore = %v, want %v", listResult.HasMore, tt.wantHasMore)
			}

			// Verify user resolution populated name fields on messages
			if tt.wantUserNames != nil {
				if len(listResult.Messages) != len(tt.wantUserNames) {
					t.Fatalf("Messages length = %d, want %d for user name verification", len(listResult.Messages), len(tt.wantUserNames))
				}
				for i, wantName := range tt.wantUserNames {
					if listResult.Messages[i].UserName != wantName {
						t.Errorf("Messages[%d].UserName = %q, want %q", i, listResult.Messages[i].UserName, wantName)
					}
				}
			}

			// Verify display and real names for messages with user resolution
			for i, msg := range listResult.Messages {
				if userInfo, ok := tt.userInfoMap[msg.User]; ok {
					if msg.DisplayName != userInfo.DisplayName {
						t.Errorf("Messages[%d].DisplayName = %q, want %q", i, msg.DisplayName, userInfo.DisplayName)
					}
					if msg.RealName != userInfo.RealName {
						t.Errorf("Messages[%d].RealName = %q, want %q", i, msg.RealName, userInfo.RealName)
					}
				}
			}
		})
	}
}

func TestListChannelMessagesHandler_Handle_MissingChannelID(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewListChannelMessagesHandler(mock)

	// Test with no arguments
	request := createListChannelMessagesRequest(map[string]interface{}{})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing channel_id")
	}

	// Check error message
	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "channel_id") {
		t.Errorf("error message should mention 'channel_id', got: %s", textContent.Text)
	}
}

func TestListChannelMessagesHandler_Handle_EmptyChannelID(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewListChannelMessagesHandler(mock)

	request := createListChannelMessagesRequest(map[string]interface{}{
		"channel_id": "",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for empty channel_id")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "channel_id") {
		t.Errorf("error message should mention 'channel_id', got: %s", textContent.Text)
	}
}

func TestNewListChannelMessagesHandler(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewListChannelMessagesHandler(mock)

	if handler == nil {
		t.Fatal("NewListChannelMessagesHandler returned nil")
	}

	if handler.slackClient != mock {
		t.Error("handler did not store the provided client")
	}
}

func TestListChannelMessagesHandler_HandleFunc(t *testing.T) {
	// Test that HandleFunc returns a usable function
	mock := &mockSlackClient{
		getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
			return []types.Message{
				{
					User:      "U12345678",
					Text:      "Test message",
					Timestamp: "1355517523.000008",
				},
			}, false, nil
		},
	}

	handler := NewListChannelMessagesHandler(mock)
	handlerFunc := handler.HandleFunc()

	if handlerFunc == nil {
		t.Fatal("HandleFunc returned nil")
	}

	request := createListChannelMessagesRequest(map[string]interface{}{
		"channel_id": "C01234567",
	})

	result, err := handlerFunc(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Error("expected success result")
	}
}

func TestListChannelMessagesHandler_Handle_InvalidLimitType(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewListChannelMessagesHandler(mock)

	// Test with string type limit (invalid)
	request := createListChannelMessagesRequest(map[string]interface{}{
		"channel_id": "C01234567",
		"limit":      "not a number",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for invalid limit type")
	}

	// Check error message
	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "limit") {
		t.Errorf("error message should mention 'limit', got: %s", textContent.Text)
	}
}

func TestListChannelMessagesHandler_Handle_ZeroLimitUsesMinimum(t *testing.T) {
	var capturedLimit int
	mock := &mockSlackClient{
		getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
			capturedLimit = limit
			return []types.Message{}, false, nil
		},
	}

	handler := NewListChannelMessagesHandler(mock)

	// Test with zero limit - should be normalized to 1
	request := createListChannelMessagesRequest(map[string]interface{}{
		"channel_id": "C01234567",
		"limit":      float64(0),
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// Zero limit should be normalized to 1 (minimum valid value)
	if capturedLimit != 1 {
		t.Errorf("zero limit should be normalized to 1, got: %d", capturedLimit)
	}
}

func TestListChannelMessagesHandler_Handle_NegativeLimitUsesMinimum(t *testing.T) {
	var capturedLimit int
	mock := &mockSlackClient{
		getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
			capturedLimit = limit
			return []types.Message{}, false, nil
		},
	}

	handler := NewListChannelMessagesHandler(mock)

	// Test with negative limit - should be normalized to 1
	request := createListChannelMessagesRequest(map[string]interface{}{
		"channel_id": "C01234567",
		"limit":      float64(-10),
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// Negative limit should be normalized to 1 (minimum valid value)
	if capturedLimit != 1 {
		t.Errorf("negative limit should be normalized to 1, got: %d", capturedLimit)
	}
}

func TestListChannelMessagesHandler_Handle_LimitExceedsMaximum(t *testing.T) {
	var capturedLimit int
	mock := &mockSlackClient{
		getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
			capturedLimit = limit
			return []types.Message{}, false, nil
		},
	}

	handler := NewListChannelMessagesHandler(mock)

	// Test with limit exceeding max (200) - should be capped at 200
	request := createListChannelMessagesRequest(map[string]interface{}{
		"channel_id": "C01234567",
		"limit":      float64(500),
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// Limit exceeding max should be capped at 200
	if capturedLimit != 200 {
		t.Errorf("limit exceeding max should be capped at 200, got: %d", capturedLimit)
	}
}

func TestListChannelMessagesHandler_Handle_DefaultLimit(t *testing.T) {
	var capturedLimit int
	mock := &mockSlackClient{
		getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
			capturedLimit = limit
			return []types.Message{}, false, nil
		},
	}

	handler := NewListChannelMessagesHandler(mock)

	// Test with no limit specified - should use default of 100
	request := createListChannelMessagesRequest(map[string]interface{}{
		"channel_id": "C01234567",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// No limit specified should use default of 100
	if capturedLimit != 100 {
		t.Errorf("default limit should be 100, got: %d", capturedLimit)
	}
}

func TestListChannelMessagesHandler_Handle_SlackErrors(t *testing.T) {
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
			name:           "permission denied",
			errorCode:      types.ErrCodePermissionDenied,
			wantErrContain: "Permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
					return nil, false, types.NewSlackError(tt.errorCode, "mock error")
				},
			}
			handler := NewListChannelMessagesHandler(mock)
			request := createListChannelMessagesRequest(map[string]interface{}{
				"channel_id": "C01234567",
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

// TestListChannelMessagesHandler_Handle_Pagination tests pagination behavior including has_more flag and limit capping.
func TestListChannelMessagesHandler_Handle_Pagination(t *testing.T) {
	tests := []struct {
		name           string
		channelID      string
		requestLimit   float64
		mockHasMore    bool
		mockMessages   []types.Message
		wantLimit      int  // Expected limit passed to GetChannelHistory
		wantHasMore    bool // Expected has_more in result
	}{
		{
			name:         "has_more true when more messages available",
			channelID:    "C01234567",
			requestLimit: 50,
			mockHasMore:  true,
			mockMessages: []types.Message{
				{User: "U12345678", Text: "Message 1", Timestamp: "1355517523.000001"},
				{User: "U12345678", Text: "Message 2", Timestamp: "1355517523.000002"},
			},
			wantLimit:   50,
			wantHasMore: true,
		},
		{
			name:         "has_more false when no more messages",
			channelID:    "C01234567",
			requestLimit: 50,
			mockHasMore:  false,
			mockMessages: []types.Message{
				{User: "U12345678", Text: "Message 1", Timestamp: "1355517523.000001"},
			},
			wantLimit:   50,
			wantHasMore: false,
		},
		{
			name:         "has_more false with empty channel",
			channelID:    "C01234567",
			requestLimit: 100,
			mockHasMore:  false,
			mockMessages: []types.Message{},
			wantLimit:    100,
			wantHasMore:  false,
		},
		{
			name:         "limit capped at 200 when requesting 201",
			channelID:    "C01234567",
			requestLimit: 201,
			mockHasMore:  false,
			mockMessages: []types.Message{
				{User: "U12345678", Text: "Message", Timestamp: "1355517523.000001"},
			},
			wantLimit:   200,
			wantHasMore: false,
		},
		{
			name:         "limit capped at 200 when requesting 500",
			channelID:    "C01234567",
			requestLimit: 500,
			mockHasMore:  true,
			mockMessages: []types.Message{
				{User: "U12345678", Text: "Message", Timestamp: "1355517523.000001"},
			},
			wantLimit:   200,
			wantHasMore: true,
		},
		{
			name:         "limit capped at 200 when requesting 1000",
			channelID:    "C01234567",
			requestLimit: 1000,
			mockHasMore:  false,
			mockMessages: []types.Message{},
			wantLimit:    200,
			wantHasMore:  false,
		},
		{
			name:         "limit exactly 200 passed through unchanged",
			channelID:    "C01234567",
			requestLimit: 200,
			mockHasMore:  true,
			mockMessages: []types.Message{
				{User: "U12345678", Text: "Message", Timestamp: "1355517523.000001"},
			},
			wantLimit:   200,
			wantHasMore: true,
		},
		{
			name:         "limit below 200 passed through unchanged",
			channelID:    "C01234567",
			requestLimit: 100,
			mockHasMore:  true,
			mockMessages: []types.Message{
				{User: "U12345678", Text: "Message", Timestamp: "1355517523.000001"},
			},
			wantLimit:   100,
			wantHasMore: true,
		},
		{
			name:         "has_more with max limit",
			channelID:    "C01234567",
			requestLimit: 200,
			mockHasMore:  true,
			mockMessages: func() []types.Message {
				// Simulate 200 messages returned with more available
				msgs := make([]types.Message, 200)
				for i := 0; i < 200; i++ {
					msgs[i] = types.Message{
						User:      "U12345678",
						Text:      "Message",
						Timestamp: "1355517523.000001",
					}
				}
				return msgs
			}(),
			wantLimit:   200,
			wantHasMore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedLimit int
			mock := &mockSlackClient{
				getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
					capturedLimit = limit
					if channelID != tt.channelID {
						t.Errorf("GetChannelHistory channelID = %q, want %q", channelID, tt.channelID)
					}
					return tt.mockMessages, tt.mockHasMore, nil
				},
				getUserInfo: func(ctx context.Context, userID string) (*types.UserInfo, error) {
					return nil, nil // User resolution not the focus of this test
				},
			}

			handler := NewListChannelMessagesHandler(mock)
			request := createListChannelMessagesRequest(map[string]interface{}{
				"channel_id": tt.channelID,
				"limit":      tt.requestLimit,
			})

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError {
				t.Fatalf("expected success, got error: %+v", result.Content)
			}

			// Verify the limit was capped correctly
			if capturedLimit != tt.wantLimit {
				t.Errorf("limit passed to GetChannelHistory = %d, want %d", capturedLimit, tt.wantLimit)
			}

			// Parse the result JSON to verify has_more
			if len(result.Content) == 0 {
				t.Fatal("expected content in result")
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("expected TextContent, got %T", result.Content[0])
			}

			var listResult types.ListChannelMessagesResult
			if err := json.Unmarshal([]byte(textContent.Text), &listResult); err != nil {
				t.Fatalf("failed to parse result JSON: %v", err)
			}

			if listResult.HasMore != tt.wantHasMore {
				t.Errorf("result HasMore = %v, want %v", listResult.HasMore, tt.wantHasMore)
			}

			// Verify message count matches
			if len(listResult.Messages) != len(tt.mockMessages) {
				t.Errorf("result Messages length = %d, want %d", len(listResult.Messages), len(tt.mockMessages))
			}
		})
	}
}

// TestListChannelMessagesHandler_Handle_UserMapping tests that mentioned users are resolved and included in user_mapping.
func TestListChannelMessagesHandler_Handle_UserMapping(t *testing.T) {
	tests := []struct {
		name             string
		channelID        string
		mockMessages     []types.Message
		extractedIDs     map[string][]string // text -> extracted user IDs
		userInfoMap      map[string]*types.UserInfo
		wantMappingCount int
		wantMappedUsers  []string // Expected user IDs in mapping
	}{
		{
			name:      "single mention in message",
			channelID: "C01234567",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Hey <@U87654321>, can you help?",
					Timestamp: "1355517523.000008",
				},
			},
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
			name:      "multiple mentions in message",
			channelID: "C01234567",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Hey <@U87654321> and <@UAAAAAAAA>, please review",
					Timestamp: "1355517523.000008",
				},
			},
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
			name:      "mentions across multiple messages",
			channelID: "C01234567",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Hey <@U87654321>, check this out",
					Timestamp: "1355517523.000008",
				},
				{
					User:      "U87654321",
					Text:      "Thanks <@UAAAAAAAA>, I'll take a look",
					Timestamp: "1355517524.000009",
				},
			},
			extractedIDs: map[string][]string{
				"Hey <@U87654321>, check this out":       {"U87654321"},
				"Thanks <@UAAAAAAAA>, I'll take a look": {"UAAAAAAAA"},
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
			name:      "duplicate mentions deduplicated",
			channelID: "C01234567",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Hey <@U87654321>, what do you think?",
					Timestamp: "1355517523.000008",
				},
				{
					User:      "UAAAAAAAA",
					Text:      "I agree with <@U87654321>",
					Timestamp: "1355517524.000009",
				},
			},
			extractedIDs: map[string][]string{
				"Hey <@U87654321>, what do you think?": {"U87654321"},
				"I agree with <@U87654321>":            {"U87654321"},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
				"U87654321": {ID: "U87654321", Name: "bob"},
				"UAAAAAAAA": {ID: "UAAAAAAAA", Name: "charlie"},
			},
			wantMappingCount: 1, // Bob should only appear once
			wantMappedUsers:  []string{"U87654321"},
		},
		{
			name:      "no mentions in messages",
			channelID: "C01234567",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Hello, world!",
					Timestamp: "1355517523.000008",
				},
				{
					User:      "U87654321",
					Text:      "Hi there!",
					Timestamp: "1355517524.000009",
				},
			},
			extractedIDs: map[string][]string{
				"Hello, world!": {},
				"Hi there!":     {},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
				"U87654321": {ID: "U87654321", Name: "bob"},
			},
			wantMappingCount: 0,
			wantMappedUsers:  nil,
		},
		{
			name:      "mentioned user not found gracefully (deleted user)",
			channelID: "C01234567",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Hey <@UDELETED1>, are you there?",
					Timestamp: "1355517523.000008",
				},
			},
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
		{
			name:      "mixed resolvable and deleted users",
			channelID: "C01234567",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Hey <@U87654321> and <@UDELETED1>, please review",
					Timestamp: "1355517523.000008",
				},
			},
			extractedIDs: map[string][]string{
				"Hey <@U87654321> and <@UDELETED1>, please review": {"U87654321", "UDELETED1"},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
				"U87654321": {ID: "U87654321", Name: "bob", DisplayName: "Bob", RealName: "Bob Jones"},
				// UDELETED1 is not in the map - simulates deleted user
			},
			wantMappingCount: 1, // Only bob should be in the mapping
			wantMappedUsers:  []string{"U87654321"},
		},
		{
			name:      "user mapping includes full user info",
			channelID: "C01234567",
			mockMessages: []types.Message{
				{
					User:      "U12345678",
					Text:      "Hey <@U87654321>, can you help?",
					Timestamp: "1355517523.000008",
				},
			},
			extractedIDs: map[string][]string{
				"Hey <@U87654321>, can you help?": {"U87654321"},
			},
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
				"U87654321": {
					ID:          "U87654321",
					Name:        "bob",
					DisplayName: "Robert Smith",
					RealName:    "Robert J. Smith",
					IsBot:       false,
				},
			},
			wantMappingCount: 1,
			wantMappedUsers:  []string{"U87654321"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				getChannelHistory: func(ctx context.Context, channelID string, limit int, oldest, latest string) ([]types.Message, bool, error) {
					return tt.mockMessages, false, nil
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

			handler := NewListChannelMessagesHandler(mock)
			request := createListChannelMessagesRequest(map[string]interface{}{
				"channel_id": tt.channelID,
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

			var listResult types.ListChannelMessagesResult
			if err := json.Unmarshal([]byte(textContent.Text), &listResult); err != nil {
				t.Fatalf("failed to parse result JSON: %v", err)
			}

			// Verify user mapping count
			if len(listResult.UserMapping) != tt.wantMappingCount {
				t.Errorf("UserMapping length = %d, want %d", len(listResult.UserMapping), tt.wantMappingCount)
			}

			// Verify expected users are in the mapping
			for _, wantUserID := range tt.wantMappedUsers {
				userInfo, ok := listResult.UserMapping[wantUserID]
				if !ok {
					t.Errorf("UserMapping missing expected user %q", wantUserID)
					continue
				}

				// Verify the user info has the expected data
				expectedInfo := tt.userInfoMap[wantUserID]
				if userInfo.ID != expectedInfo.ID {
					t.Errorf("UserMapping[%q].ID = %q, want %q", wantUserID, userInfo.ID, expectedInfo.ID)
				}
				if userInfo.Name != expectedInfo.Name {
					t.Errorf("UserMapping[%q].Name = %q, want %q", wantUserID, userInfo.Name, expectedInfo.Name)
				}
				if userInfo.DisplayName != expectedInfo.DisplayName {
					t.Errorf("UserMapping[%q].DisplayName = %q, want %q", wantUserID, userInfo.DisplayName, expectedInfo.DisplayName)
				}
				if userInfo.RealName != expectedInfo.RealName {
					t.Errorf("UserMapping[%q].RealName = %q, want %q", wantUserID, userInfo.RealName, expectedInfo.RealName)
				}
			}
		})
	}
}
