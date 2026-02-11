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

// createSearchMessagesRequest creates an MCP CallToolRequest for search_messages with the given arguments.
func createSearchMessagesRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      "search_messages",
			Arguments: args,
		},
	}
}

func TestSearchMessagesHandler_Handle_Success(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		count           int
		sort            string
		mockMatches     []types.SearchMatch
		mockTotal       int
		userInfoMap     map[string]*types.UserInfo
		currentUser     *types.UserInfo
		wantMatchCount  int
		wantTotal       int
		wantQuery       string
		wantUserNames   []string // Expected user_name for each match
		wantCurrentUser bool
	}{
		{
			name:  "basic message search",
			query: "hello world",
			count: 20,
			mockMatches: []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "U12345678",
					Text:        "Hello, world!",
					Timestamp:   "1355517523.000008",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000008",
				},
				{
					ChannelID:   "C87654321",
					ChannelName: "random",
					User:        "U87654321",
					Text:        "Hello world from random",
					Timestamp:   "1355517524.000009",
					Permalink:   "https://slack.com/archives/C87654321/p1355517524000009",
				},
			},
			mockTotal: 2,
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
			currentUser: &types.UserInfo{
				ID:          "UCURRENT1",
				Name:        "currentuser",
				DisplayName: "Current User",
				RealName:    "Current U. User",
			},
			wantMatchCount:  2,
			wantTotal:       2,
			wantQuery:       "hello world",
			wantUserNames:   []string{"alice", "bob"},
			wantCurrentUser: true,
		},
		{
			name:            "empty search results",
			query:           "nonexistent search term",
			count:           20,
			mockMatches:     []types.SearchMatch{},
			mockTotal:       0,
			userInfoMap:     map[string]*types.UserInfo{},
			currentUser:     nil,
			wantMatchCount:  0,
			wantTotal:       0,
			wantQuery:       "nonexistent search term",
			wantUserNames:   nil,
			wantCurrentUser: false,
		},
		{
			name:  "search with user resolution",
			query: "test message",
			count: 20,
			mockMatches: []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "U11111111",
					Text:        "This is a test message from user 1",
					Timestamp:   "1355517523.000001",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000001",
				},
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "U22222222",
					Text:        "Another test message from user 2",
					Timestamp:   "1355517523.000002",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000002",
				},
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "U11111111",
					Text:        "Test message again from user 1",
					Timestamp:   "1355517523.000003",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000003",
				},
			},
			mockTotal: 3,
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
			currentUser:     nil,
			wantMatchCount:  3,
			wantTotal:       3,
			wantQuery:       "test message",
			wantUserNames:   []string{"johndoe", "janedoe", "johndoe"},
			wantCurrentUser: false,
		},
		{
			name:  "search with custom count",
			query: "important",
			count: 50,
			mockMatches: []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "U12345678",
					Text:        "Important message",
					Timestamp:   "1355517523.000008",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000008",
				},
			},
			mockTotal: 100, // Total is higher than returned count
			userInfoMap: map[string]*types.UserInfo{
				"U12345678": {ID: "U12345678", Name: "alice"},
			},
			currentUser:     nil,
			wantMatchCount:  1,
			wantTotal:       100,
			wantQuery:       "important",
			wantUserNames:   []string{"alice"},
			wantCurrentUser: false,
		},
		{
			name:  "user resolution graceful failure",
			query: "from unknown",
			count: 20,
			mockMatches: []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "UUNKNOWN1",
					Text:        "Message from unknown user",
					Timestamp:   "1355517523.000008",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000008",
				},
			},
			mockTotal:       1,
			userInfoMap:     map[string]*types.UserInfo{}, // No user info available
			currentUser:     nil,
			wantMatchCount:  1,
			wantTotal:       1,
			wantQuery:       "from unknown",
			wantUserNames:   []string{""}, // Should be empty, not fail
			wantCurrentUser: false,
		},
		{
			name:  "bot user resolution",
			query: "bot message",
			count: 20,
			mockMatches: []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "UBOTUSER1",
					Text:        "Automated bot message",
					Timestamp:   "1355517523.000008",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000008",
				},
			},
			mockTotal: 1,
			userInfoMap: map[string]*types.UserInfo{
				"UBOTUSER1": {
					ID:          "UBOTUSER1",
					Name:        "mybot",
					DisplayName: "My Bot",
					RealName:    "My Bot App",
					IsBot:       true,
				},
			},
			currentUser:     nil,
			wantMatchCount:  1,
			wantTotal:       1,
			wantQuery:       "bot message",
			wantUserNames:   []string{"mybot"},
			wantCurrentUser: false,
		},
		{
			name:  "message without user ID (system message)",
			query: "system",
			count: 20,
			mockMatches: []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "", // No user ID
					Text:        "System notification",
					Timestamp:   "1355517523.000008",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000008",
				},
			},
			mockTotal:       1,
			userInfoMap:     map[string]*types.UserInfo{},
			currentUser:     nil,
			wantMatchCount:  1,
			wantTotal:       1,
			wantQuery:       "system",
			wantUserNames:   []string{""},
			wantCurrentUser: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
					if query != tt.query {
						t.Errorf("SearchMessages query = %q, want %q", query, tt.query)
					}
					if tt.count > 0 && count != tt.count {
						t.Errorf("SearchMessages count = %d, want %d", count, tt.count)
					}
					return tt.mockMatches, tt.mockTotal, nil
				},
				getUserInfo: func(ctx context.Context, userID string) (*types.UserInfo, error) {
					if info, ok := tt.userInfoMap[userID]; ok {
						return info, nil
					}
					return nil, nil
				},
				getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
					return tt.currentUser, nil
				},
			}

			handler := NewSearchMessagesHandler(mock)
			args := map[string]interface{}{
				"query": tt.query,
			}
			if tt.count > 0 {
				args["count"] = float64(tt.count)
			}
			if tt.sort != "" {
				args["sort"] = tt.sort
			}
			request := createSearchMessagesRequest(args)

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

			var searchResult types.SearchMessagesResult
			if err := json.Unmarshal([]byte(textContent.Text), &searchResult); err != nil {
				t.Fatalf("failed to parse result JSON: %v", err)
			}

			if searchResult.Query != tt.wantQuery {
				t.Errorf("result Query = %q, want %q", searchResult.Query, tt.wantQuery)
			}

			if len(searchResult.Matches) != tt.wantMatchCount {
				t.Errorf("result Matches length = %d, want %d", len(searchResult.Matches), tt.wantMatchCount)
			}

			if searchResult.Total != tt.wantTotal {
				t.Errorf("result Total = %d, want %d", searchResult.Total, tt.wantTotal)
			}

			// Verify current user
			if tt.wantCurrentUser {
				if searchResult.CurrentUser == nil {
					t.Error("expected CurrentUser to be set")
				}
			}

			// Verify user resolution populated name fields on matches
			if tt.wantUserNames != nil {
				if len(searchResult.Matches) != len(tt.wantUserNames) {
					t.Fatalf("Matches length = %d, want %d for user name verification", len(searchResult.Matches), len(tt.wantUserNames))
				}
				for i, wantName := range tt.wantUserNames {
					if searchResult.Matches[i].UserName != wantName {
						t.Errorf("Matches[%d].UserName = %q, want %q", i, searchResult.Matches[i].UserName, wantName)
					}
				}
			}

			// Verify display and real names for matches with user resolution
			for i, match := range searchResult.Matches {
				if userInfo, ok := tt.userInfoMap[match.User]; ok {
					if match.DisplayName != userInfo.DisplayName {
						t.Errorf("Matches[%d].DisplayName = %q, want %q", i, match.DisplayName, userInfo.DisplayName)
					}
					if match.RealName != userInfo.RealName {
						t.Errorf("Matches[%d].RealName = %q, want %q", i, match.RealName, userInfo.RealName)
					}
				}
			}
		})
	}
}

func TestSearchMessagesHandler_Handle_MissingQuery(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewSearchMessagesHandler(mock)

	// Test with no arguments
	request := createSearchMessagesRequest(map[string]interface{}{})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing query")
	}

	// Check error message
	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "query") {
		t.Errorf("error message should mention 'query', got: %s", textContent.Text)
	}
}

func TestSearchMessagesHandler_Handle_EmptyQuery(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewSearchMessagesHandler(mock)

	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for empty query")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "query") {
		t.Errorf("error message should mention 'query', got: %s", textContent.Text)
	}
}

func TestSearchMessagesHandler_Handle_InvalidQueryType(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewSearchMessagesHandler(mock)

	// Test with numeric query (invalid type)
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": 12345,
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for invalid query type")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "query") {
		t.Errorf("error message should mention 'query', got: %s", textContent.Text)
	}
}

func TestNewSearchMessagesHandler(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewSearchMessagesHandler(mock)

	if handler == nil {
		t.Fatal("NewSearchMessagesHandler returned nil")
	}

	if handler.slackClient != mock {
		t.Error("handler did not store the provided client")
	}
}

func TestSearchMessagesHandler_HandleFunc(t *testing.T) {
	// Test that HandleFunc returns a usable function
	mock := &mockSlackClient{
		searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
			return []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "U12345678",
					Text:        "Test message",
					Timestamp:   "1355517523.000008",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000008",
				},
			}, 1, nil
		},
		getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
			return nil, nil
		},
	}

	handler := NewSearchMessagesHandler(mock)
	handlerFunc := handler.HandleFunc()

	if handlerFunc == nil {
		t.Fatal("HandleFunc returned nil")
	}

	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
	})

	result, err := handlerFunc(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Error("expected success result")
	}
}

func TestSearchMessagesHandler_Handle_InvalidCountType(t *testing.T) {
	mock := &mockSlackClient{}
	handler := NewSearchMessagesHandler(mock)

	// Test with string type count (invalid)
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
		"count": "not a number",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for invalid count type")
	}

	// Check error message
	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "count") {
		t.Errorf("error message should mention 'count', got: %s", textContent.Text)
	}
}

func TestSearchMessagesHandler_Handle_ZeroCountUsesMinimum(t *testing.T) {
	var capturedCount int
	mock := &mockSlackClient{
		searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
			capturedCount = count
			return []types.SearchMatch{}, 0, nil
		},
		getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
			return nil, nil
		},
	}

	handler := NewSearchMessagesHandler(mock)

	// Test with zero count - should be normalized to 1
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
		"count": float64(0),
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// Zero count should be normalized to 1 (minimum valid value)
	if capturedCount != 1 {
		t.Errorf("zero count should be normalized to 1, got: %d", capturedCount)
	}
}

func TestSearchMessagesHandler_Handle_NegativeCountUsesMinimum(t *testing.T) {
	var capturedCount int
	mock := &mockSlackClient{
		searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
			capturedCount = count
			return []types.SearchMatch{}, 0, nil
		},
		getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
			return nil, nil
		},
	}

	handler := NewSearchMessagesHandler(mock)

	// Test with negative count - should be normalized to 1
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
		"count": float64(-10),
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// Negative count should be normalized to 1 (minimum valid value)
	if capturedCount != 1 {
		t.Errorf("negative count should be normalized to 1, got: %d", capturedCount)
	}
}

func TestSearchMessagesHandler_Handle_CountExceedsMaximum(t *testing.T) {
	var capturedCount int
	mock := &mockSlackClient{
		searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
			capturedCount = count
			return []types.SearchMatch{}, 0, nil
		},
		getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
			return nil, nil
		},
	}

	handler := NewSearchMessagesHandler(mock)

	// Test with count exceeding max (100) - should be capped at 100
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
		"count": float64(500),
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// Count exceeding max should be capped at 100
	if capturedCount != 100 {
		t.Errorf("count exceeding max should be capped at 100, got: %d", capturedCount)
	}
}

func TestSearchMessagesHandler_Handle_DefaultCount(t *testing.T) {
	var capturedCount int
	mock := &mockSlackClient{
		searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
			capturedCount = count
			return []types.SearchMatch{}, 0, nil
		},
		getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
			return nil, nil
		},
	}

	handler := NewSearchMessagesHandler(mock)

	// Test with no count specified - should use default of 20
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %+v", result.Content)
	}

	// No count specified should use default of 20
	if capturedCount != 20 {
		t.Errorf("default count should be 20, got: %d", capturedCount)
	}
}

func TestSearchMessagesHandler_Handle_SortParameter(t *testing.T) {
	tests := []struct {
		name     string
		sortArg  interface{}
		wantSort string
	}{
		{
			name:     "sort by score (default)",
			sortArg:  nil,
			wantSort: "score",
		},
		{
			name:     "sort by score explicitly",
			sortArg:  "score",
			wantSort: "score",
		},
		{
			name:     "sort by timestamp",
			sortArg:  "timestamp",
			wantSort: "timestamp",
		},
		{
			name:     "invalid sort value uses default",
			sortArg:  "invalid",
			wantSort: "score",
		},
		{
			name:     "non-string sort value uses default",
			sortArg:  12345,
			wantSort: "score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedSort string
			mock := &mockSlackClient{
				searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
					capturedSort = sort
					return []types.SearchMatch{}, 0, nil
				},
				getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
					return nil, nil
				},
			}

			handler := NewSearchMessagesHandler(mock)
			args := map[string]interface{}{
				"query": "test",
			}
			if tt.sortArg != nil {
				args["sort"] = tt.sortArg
			}
			request := createSearchMessagesRequest(args)

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError {
				t.Fatalf("expected success, got error: %+v", result.Content)
			}

			if capturedSort != tt.wantSort {
				t.Errorf("sort = %q, want %q", capturedSort, tt.wantSort)
			}
		})
	}
}

func TestSearchMessagesHandler_Handle_SlackErrors(t *testing.T) {
	tests := []struct {
		name           string
		errorCode      string
		wantErrContain string
	}{
		{
			name:           "user token not configured",
			errorCode:      types.ErrCodeUserTokenNotConfigured,
			wantErrContain: "SLACK_USER_TOKEN not configured",
		},
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
			name:           "permission denied",
			errorCode:      types.ErrCodePermissionDenied,
			wantErrContain: "Permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSlackClient{
				searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
					return nil, 0, types.NewSlackError(tt.errorCode, "mock error")
				},
			}
			handler := NewSearchMessagesHandler(mock)
			request := createSearchMessagesRequest(map[string]interface{}{
				"query": "test",
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

func TestSearchMessagesHandler_Handle_GenericError(t *testing.T) {
	mock := &mockSlackClient{
		searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
			return nil, 0, types.NewSlackError("unknown_error", "something went wrong")
		},
	}

	handler := NewSearchMessagesHandler(mock)
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "Failed to search messages") {
		t.Errorf("error message should contain 'Failed to search messages', got: %s", textContent.Text)
	}
}

func TestSearchMessagesHandler_Handle_CurrentUserGracefulDegradation(t *testing.T) {
	// Test that failure to get current user doesn't fail the whole request
	mock := &mockSlackClient{
		searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
			return []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "U12345678",
					Text:        "Test message",
					Timestamp:   "1355517523.000008",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000008",
				},
			}, 1, nil
		},
		getUserInfo: func(ctx context.Context, userID string) (*types.UserInfo, error) {
			return &types.UserInfo{
				ID:   userID,
				Name: "testuser",
			}, nil
		},
		getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
			// Simulate failure to get current user
			return nil, types.NewSlackError("user_lookup_failed", "failed to get current user")
		},
	}

	handler := NewSearchMessagesHandler(mock)
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed despite current user lookup failure
	if result.IsError {
		t.Fatalf("expected success despite current user failure, got error: %+v", result.Content)
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var searchResult types.SearchMessagesResult
	if err := json.Unmarshal([]byte(textContent.Text), &searchResult); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	// Verify matches are returned
	if len(searchResult.Matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(searchResult.Matches))
	}

	// Verify current_user is nil (graceful degradation)
	if searchResult.CurrentUser != nil {
		t.Error("expected CurrentUser to be nil due to graceful degradation")
	}
}

func TestSearchMessagesHandler_Handle_UserResolutionError(t *testing.T) {
	// Test that failure to resolve a user doesn't fail the whole request
	mock := &mockSlackClient{
		searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
			return []types.SearchMatch{
				{
					ChannelID:   "C01234567",
					ChannelName: "general",
					User:        "U12345678",
					Text:        "Test message",
					Timestamp:   "1355517523.000008",
					Permalink:   "https://slack.com/archives/C01234567/p1355517523000008",
				},
			}, 1, nil
		},
		getUserInfo: func(ctx context.Context, userID string) (*types.UserInfo, error) {
			// Simulate failure to get user info
			return nil, types.NewSlackError("user_not_found", "user not found")
		},
		getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
			return nil, nil
		},
	}

	handler := NewSearchMessagesHandler(mock)
	request := createSearchMessagesRequest(map[string]interface{}{
		"query": "test",
	})

	result, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed despite user resolution failure
	if result.IsError {
		t.Fatalf("expected success despite user resolution failure, got error: %+v", result.Content)
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var searchResult types.SearchMessagesResult
	if err := json.Unmarshal([]byte(textContent.Text), &searchResult); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	// Verify matches are returned
	if len(searchResult.Matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(searchResult.Matches))
	}

	// Verify user name fields are empty (graceful degradation)
	if searchResult.Matches[0].UserName != "" {
		t.Errorf("expected empty UserName due to graceful degradation, got %q", searchResult.Matches[0].UserName)
	}
}

// TestSearchMessagesHandler_Handle_CountValidation tests various count boundary conditions.
func TestSearchMessagesHandler_Handle_CountValidation(t *testing.T) {
	tests := []struct {
		name         string
		requestCount float64
		wantCount    int
	}{
		{
			name:         "count exactly 1 passed through",
			requestCount: 1,
			wantCount:    1,
		},
		{
			name:         "count exactly 100 passed through",
			requestCount: 100,
			wantCount:    100,
		},
		{
			name:         "count 101 capped at 100",
			requestCount: 101,
			wantCount:    100,
		},
		{
			name:         "count 50 passed through",
			requestCount: 50,
			wantCount:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedCount int
			mock := &mockSlackClient{
				searchMessages: func(ctx context.Context, query string, count int, sort string) ([]types.SearchMatch, int, error) {
					capturedCount = count
					return []types.SearchMatch{}, 0, nil
				},
				getCurrentUser: func(ctx context.Context) (*types.UserInfo, error) {
					return nil, nil
				},
			}

			handler := NewSearchMessagesHandler(mock)
			request := createSearchMessagesRequest(map[string]interface{}{
				"query": "test",
				"count": tt.requestCount,
			})

			result, err := handler.Handle(context.Background(), request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError {
				t.Fatalf("expected success, got error: %+v", result.Content)
			}

			if capturedCount != tt.wantCount {
				t.Errorf("count passed to SearchMessages = %d, want %d", capturedCount, tt.wantCount)
			}
		})
	}
}
