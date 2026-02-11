// Package urlparser provides functionality for parsing Slack message URLs.
package urlparser

import (
	"testing"

	"github.com/Bitovi/slack-mcp-server/pkg/types"
)

func TestParse_ValidMessageURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		channelID string
		timestamp string
		isThread  bool
		threadTS  string
	}{
		{
			name:      "simple message URL",
			url:       "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			channelID: "C01234567",
			timestamp: "1355517523.000008",
			isThread:  false,
			threadTS:  "",
		},
		{
			name:      "message URL with different workspace",
			url:       "https://mycompany.slack.com/archives/C98765432/p1234567890123456",
			channelID: "C98765432",
			timestamp: "1234567890.123456",
			isThread:  false,
			threadTS:  "",
		},
		{
			name:      "message URL with enterprise grid subdomain",
			url:       "https://company-enterprise.slack.com/archives/C11111111/p1111111111111111",
			channelID: "C11111111",
			timestamp: "1111111111.111111",
			isThread:  false,
			threadTS:  "",
		},
		{
			name:      "message in private channel",
			url:       "https://workspace.slack.com/archives/G01234567/p1355517523000008",
			channelID: "G01234567",
			timestamp: "1355517523.000008",
			isThread:  false,
			threadTS:  "",
		},
		{
			name:      "message in DM channel",
			url:       "https://workspace.slack.com/archives/D01234567/p1355517523000008",
			channelID: "D01234567",
			timestamp: "1355517523.000008",
			isThread:  false,
			threadTS:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ChannelID != tt.channelID {
				t.Errorf("ChannelID = %q, want %q", result.ChannelID, tt.channelID)
			}
			if result.Timestamp != tt.timestamp {
				t.Errorf("Timestamp = %q, want %q", result.Timestamp, tt.timestamp)
			}
			if result.IsThread != tt.isThread {
				t.Errorf("IsThread = %v, want %v", result.IsThread, tt.isThread)
			}
			if result.ThreadTS != tt.threadTS {
				t.Errorf("ThreadTS = %q, want %q", result.ThreadTS, tt.threadTS)
			}
		})
	}
}

func TestParse_ValidThreadURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		channelID string
		timestamp string
		threadTS  string
		isThread  bool
	}{
		{
			name:      "thread URL with thread_ts parameter",
			url:       "https://workspace.slack.com/archives/C01234567/p1355517523000008?thread_ts=1355517523.000008&cid=C01234567",
			channelID: "C01234567",
			timestamp: "1355517523.000008",
			threadTS:  "1355517523.000008",
			isThread:  true,
		},
		{
			name:      "thread URL with reply timestamp",
			url:       "https://workspace.slack.com/archives/C01234567/p1355517524000009?thread_ts=1355517523.000008&cid=C01234567",
			channelID: "C01234567",
			timestamp: "1355517524.000009",
			threadTS:  "1355517523.000008",
			isThread:  true,
		},
		{
			name:      "thread URL with only thread_ts (no cid)",
			url:       "https://workspace.slack.com/archives/C01234567/p1355517523000008?thread_ts=1355517523.000008",
			channelID: "C01234567",
			timestamp: "1355517523.000008",
			threadTS:  "1355517523.000008",
			isThread:  true,
		},
		{
			name:      "thread URL in private channel",
			url:       "https://workspace.slack.com/archives/G01234567/p1355517524000009?thread_ts=1355517523.000008",
			channelID: "G01234567",
			timestamp: "1355517524.000009",
			threadTS:  "1355517523.000008",
			isThread:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ChannelID != tt.channelID {
				t.Errorf("ChannelID = %q, want %q", result.ChannelID, tt.channelID)
			}
			if result.Timestamp != tt.timestamp {
				t.Errorf("Timestamp = %q, want %q", result.Timestamp, tt.timestamp)
			}
			if result.ThreadTS != tt.threadTS {
				t.Errorf("ThreadTS = %q, want %q", result.ThreadTS, tt.threadTS)
			}
			if result.IsThread != tt.isThread {
				t.Errorf("IsThread = %v, want %v", result.IsThread, tt.isThread)
			}
		})
	}
}

func TestParse_InvalidURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantErrCode string
		wantErrMsg  string
	}{
		{
			name:        "empty URL",
			url:         "",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "URL cannot be empty",
		},
		{
			name:        "non-Slack URL",
			url:         "https://google.com/archives/C01234567/p1355517523000008",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "URL must be a slack.com URL",
		},
		{
			name:        "Slack URL without archives path",
			url:         "https://workspace.slack.com/messages/C01234567",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "invalid Slack message URL format",
		},
		{
			name:        "Slack URL with missing channel",
			url:         "https://workspace.slack.com/archives/",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "invalid Slack message URL format",
		},
		{
			name:        "Slack URL with missing timestamp",
			url:         "https://workspace.slack.com/archives/C01234567",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "invalid Slack message URL format",
		},
		{
			name:        "Slack URL with invalid timestamp (no p prefix in path)",
			url:         "https://workspace.slack.com/archives/C01234567/1355517523000008",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "invalid Slack message URL format",
		},
		{
			name:        "Slack URL with short timestamp",
			url:         "https://workspace.slack.com/archives/C01234567/p135551752",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "invalid timestamp format: expected 16 digits",
		},
		{
			name:        "Slack URL with long timestamp",
			url:         "https://workspace.slack.com/archives/C01234567/p135551752300000800",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "invalid timestamp format: expected 16 digits",
		},
		{
			name:        "malformed URL",
			url:         "not-a-url",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "URL must be a slack.com URL",
		},
		{
			name:        "http URL (not https)",
			url:         "http://workspace.slack.com/archives/C01234567/p1355517523000008",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "invalid Slack message URL format",
		},
		{
			name:        "lowercase channel ID",
			url:         "https://workspace.slack.com/archives/c01234567/p1355517523000008",
			wantErrCode: types.ErrCodeInvalidURL,
			wantErrMsg:  "invalid Slack message URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.url)
			if err == nil {
				t.Fatalf("expected error, got result: %+v", result)
			}

			slackErr, ok := err.(*types.SlackError)
			if !ok {
				t.Fatalf("expected *types.SlackError, got %T", err)
			}

			if slackErr.Code != tt.wantErrCode {
				t.Errorf("error code = %q, want %q", slackErr.Code, tt.wantErrCode)
			}

			// Check that error message contains expected substring
			if tt.wantErrMsg != "" && !containsSubstring(slackErr.Message, tt.wantErrMsg) {
				t.Errorf("error message = %q, want to contain %q", slackErr.Message, tt.wantErrMsg)
			}
		})
	}
}

func TestConvertTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{
			name:      "standard timestamp",
			input:     "1355517523000008",
			want:      "1355517523.000008",
			wantError: false,
		},
		{
			name:      "timestamp with all zeros in microseconds",
			input:     "1234567890000000",
			want:      "1234567890.000000",
			wantError: false,
		},
		{
			name:      "timestamp with all nines",
			input:     "9999999999999999",
			want:      "9999999999.999999",
			wantError: false,
		},
		{
			name:      "timestamp with mixed digits",
			input:     "1609459200123456",
			want:      "1609459200.123456",
			wantError: false,
		},
		{
			name:      "too short timestamp",
			input:     "135551752300000",
			want:      "",
			wantError: true,
		},
		{
			name:      "too long timestamp",
			input:     "13555175230000089",
			want:      "",
			wantError: true,
		},
		{
			name:      "empty timestamp",
			input:     "",
			want:      "",
			wantError: true,
		},
		{
			name:      "timestamp with non-digit characters",
			input:     "135551752300000a",
			want:      "",
			wantError: true,
		},
		{
			name:      "timestamp with spaces",
			input:     "1355517523 00008",
			want:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertTimestamp(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got result: %q", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("ConvertTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidSlackURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "valid message URL",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008",
			want: true,
		},
		{
			name: "valid thread URL",
			url:  "https://workspace.slack.com/archives/C01234567/p1355517523000008?thread_ts=1355517523.000008",
			want: true,
		},
		{
			name: "valid URL with enterprise subdomain",
			url:  "https://company-enterprise.slack.com/archives/C01234567/p1355517523000008",
			want: true,
		},
		{
			name: "empty string",
			url:  "",
			want: false,
		},
		{
			name: "non-Slack URL",
			url:  "https://google.com/archives/C01234567/p1355517523000008",
			want: false,
		},
		{
			name: "Slack URL without archives",
			url:  "https://workspace.slack.com/messages/C01234567",
			want: false,
		},
		{
			name: "Slack URL without timestamp",
			url:  "https://workspace.slack.com/archives/C01234567",
			want: false,
		},
		{
			name: "malformed URL",
			url:  "not-a-valid-url",
			want: false,
		},
		{
			name: "http instead of https",
			url:  "http://workspace.slack.com/archives/C01234567/p1355517523000008",
			want: false,
		},
		{
			name: "slack.com without subdomain",
			url:  "https://slack.com/archives/C01234567/p1355517523000008",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidSlackURL(tt.url)
			if got != tt.want {
				t.Errorf("IsValidSlackURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestParse_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		channelID string
		timestamp string
		isThread  bool
	}{
		{
			name:      "URL with extra query parameters",
			url:       "https://workspace.slack.com/archives/C01234567/p1355517523000008?thread_ts=1355517523.000008&cid=C01234567&extra=param",
			channelID: "C01234567",
			timestamp: "1355517523.000008",
			isThread:  true,
		},
		{
			name:      "URL with fragment",
			url:       "https://workspace.slack.com/archives/C01234567/p1355517523000008#some-fragment",
			channelID: "C01234567",
			timestamp: "1355517523.000008",
			isThread:  false,
		},
		{
			name:      "channel ID with all numbers",
			url:       "https://workspace.slack.com/archives/C12345678/p1355517523000008",
			channelID: "C12345678",
			timestamp: "1355517523.000008",
			isThread:  false,
		},
		{
			name:      "channel ID with letters and numbers",
			url:       "https://workspace.slack.com/archives/CABCDEF12/p1355517523000008",
			channelID: "CABCDEF12",
			timestamp: "1355517523.000008",
			isThread:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ChannelID != tt.channelID {
				t.Errorf("ChannelID = %q, want %q", result.ChannelID, tt.channelID)
			}
			if result.Timestamp != tt.timestamp {
				t.Errorf("Timestamp = %q, want %q", result.Timestamp, tt.timestamp)
			}
			if result.IsThread != tt.isThread {
				t.Errorf("IsThread = %v, want %v", result.IsThread, tt.isThread)
			}
		})
	}
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
