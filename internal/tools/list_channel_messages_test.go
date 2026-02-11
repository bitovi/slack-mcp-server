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
