// Package urlparser provides functionality for parsing Slack message URLs.
package urlparser

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/slack-mcp-server/slack-mcp-server/pkg/types"
)

// slackURLPattern matches Slack message URLs.
// Format: https://{workspace}.slack.com/archives/{channel_id}/p{timestamp}
var slackURLPattern = regexp.MustCompile(`^https://[^/]+\.slack\.com/archives/([A-Z0-9]+)/p(\d+)$`)

// Parse extracts channel ID and timestamps from a Slack message URL.
// It handles both regular message URLs and thread URLs with query parameters.
//
// URL formats supported:
//   - Message URL: https://workspace.slack.com/archives/C01234567/p1234567890123456
//   - Thread URL: https://workspace.slack.com/archives/C01234567/p1234567890123456?thread_ts=1234567890.123456&cid=C01234567
//
// Returns a ParsedURL struct with extracted components, or an error if the URL is invalid.
func Parse(slackURL string) (*types.ParsedURL, error) {
	if slackURL == "" {
		return nil, types.NewSlackError(types.ErrCodeInvalidURL, "URL cannot be empty")
	}

	// Parse the URL to extract query parameters
	parsedURL, err := url.Parse(slackURL)
	if err != nil {
		return nil, types.NewSlackError(types.ErrCodeInvalidURL, fmt.Sprintf("failed to parse URL: %v", err))
	}

	// Validate it's a Slack URL
	if !strings.HasSuffix(parsedURL.Host, ".slack.com") {
		return nil, types.NewSlackError(types.ErrCodeInvalidURL, "URL must be a slack.com URL")
	}

	// Build the base URL without query parameters for regex matching
	baseURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)

	// Match against the Slack URL pattern
	matches := slackURLPattern.FindStringSubmatch(baseURL)
	if matches == nil {
		return nil, types.NewSlackError(types.ErrCodeInvalidURL,
			"invalid Slack message URL format. Expected: https://workspace.slack.com/archives/{channel_id}/p{timestamp}")
	}

	channelID := matches[1]
	rawTimestamp := matches[2]

	// Convert the URL timestamp to API format
	timestamp, err := convertTimestamp(rawTimestamp)
	if err != nil {
		return nil, types.NewSlackError(types.ErrCodeInvalidURL, err.Error())
	}

	result := &types.ParsedURL{
		ChannelID: channelID,
		Timestamp: timestamp,
	}

	// Check for thread_ts query parameter (indicates a thread URL)
	query := parsedURL.Query()
	threadTS := query.Get("thread_ts")
	if threadTS != "" {
		result.ThreadTS = threadTS
		result.IsThread = true
	}

	return result, nil
}

// convertTimestamp converts a Slack URL timestamp to API format.
// URL format: 1355517523000008 (no 'p' prefix here, just the digits)
// API format: 1355517523.000008 (insert '.' after 10th digit)
//
// The URL path contains 'p' + 16 digits, where the first 10 are seconds
// and the remaining 6 are microseconds.
func convertTimestamp(urlTimestamp string) (string, error) {
	// Timestamp should be exactly 16 digits (10 seconds + 6 microseconds)
	if len(urlTimestamp) != 16 {
		return "", fmt.Errorf("invalid timestamp format: expected 16 digits, got %d", len(urlTimestamp))
	}

	// Validate all characters are digits
	for _, c := range urlTimestamp {
		if c < '0' || c > '9' {
			return "", fmt.Errorf("invalid timestamp format: contains non-digit characters")
		}
	}

	// Insert '.' after the 10th digit
	// Example: 1355517523000008 -> 1355517523.000008
	return urlTimestamp[:10] + "." + urlTimestamp[10:], nil
}

// ConvertTimestamp is an exported wrapper for testing purposes.
// It converts a Slack URL timestamp to API format.
func ConvertTimestamp(urlTimestamp string) (string, error) {
	return convertTimestamp(urlTimestamp)
}

// IsValidSlackURL checks if a URL appears to be a valid Slack message URL
// without fully parsing it. This can be used for quick validation.
func IsValidSlackURL(slackURL string) bool {
	if slackURL == "" {
		return false
	}

	parsedURL, err := url.Parse(slackURL)
	if err != nil {
		return false
	}

	if !strings.HasSuffix(parsedURL.Host, ".slack.com") {
		return false
	}

	baseURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)
	return slackURLPattern.MatchString(baseURL)
}
