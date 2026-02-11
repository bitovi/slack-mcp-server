# Slack MCP Server

An MCP (Model Context Protocol) server that enables AI agents to read Slack messages and threads by providing a Slack message URL.

## Overview

This Go-based MCP server integrates with the Slack API to fetch message content. It uses Stdio transport, making it suitable for use with CLI-based AI tools like Claude Code and other MCP-compatible agents.

### Features

- **Read Slack Messages**: Fetch any message from public or private channels, DMs, and group DMs
- **List Channel Messages**: Retrieve recent messages from any channel with pagination support
- **Search Messages**: Search across the entire Slack workspace for messages matching a query
- **Thread Support**: Automatically retrieves entire threads when the message has replies
- **URL-Based Retrieval**: Simply provide a Slack message URL to fetch its content
- **User Resolution**: Automatically resolves user IDs to names and builds user mappings for mentions
- **MCP Protocol**: Standard MCP protocol support for seamless AI agent integration

## Prerequisites

- **Go 1.21+**: Required for building the server
- **Slack Workspace**: Access to a Slack workspace where you can create an app
- **Slack Bot Token**: A bot token with appropriate permissions (see setup below)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/Bitovi/slack-mcp-server.git
cd slack-mcp-server

# Build the server
go build -o slack-mcp-server ./cmd/server

# Or use make (if available)
make build
```

### Verify Installation

```bash
# Check version
./slack-mcp-server --version

# View help
./slack-mcp-server --help
```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `SLACK_BOT_TOKEN` | Slack bot token for API authentication (starts with `xoxb-`) | Yes |
| `SLACK_USER_TOKEN` | Slack user token for search functionality (starts with `xoxp-`) | No* |

*\* `SLACK_USER_TOKEN` is only required if you want to use the `search_messages` tool. The server will start without it, but search operations will fail with a helpful error message.*

### Setting Up a Slack App

1. **Create a Slack App**
   - Go to [Slack API Apps](https://api.slack.com/apps)
   - Click "Create New App" and choose "From scratch"
   - Give your app a name and select your workspace

2. **Configure OAuth Scopes**

   Navigate to **OAuth & Permissions** and add the following scopes:

   **Bot Token Scopes** (required for `read_message` and `list_channel_messages`):

   | Scope | Description |
   |-------|-------------|
   | `channels:history` | Read messages from public channels |
   | `groups:history` | Read messages from private channels |
   | `im:history` | Read direct messages |
   | `mpim:history` | Read group direct messages |

   **User Token Scopes** (required for `search_messages`):

   | Scope | Description |
   |-------|-------------|
   | `search:read` | Search messages in the workspace |

3. **Install the App**
   - Click "Install to Workspace" under **OAuth & Permissions**
   - Authorize the app for your workspace

4. **Copy the Tokens**
   - After installation, copy the **Bot User OAuth Token** (starts with `xoxb-`)
   - This is your `SLACK_BOT_TOKEN`
   - If you added User Token Scopes (for search), also copy the **User OAuth Token** (starts with `xoxp-`)
   - This is your `SLACK_USER_TOKEN`

5. **Invite the Bot to Channels**
   - For private channels, invite the bot: `/invite @your-bot-name`
   - The bot must be a member of private channels to read their messages

### Export the Tokens

```bash
export SLACK_BOT_TOKEN=xoxb-your-bot-token-here

# Optional: Only needed for search_messages
export SLACK_USER_TOKEN=xoxp-your-user-token-here
```

For persistent configuration, add these to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.).

## Usage

### Starting the Server

```bash
# Ensure the bot token is set (required)
export SLACK_BOT_TOKEN=xoxb-your-bot-token-here

# Optional: Set user token for search functionality
export SLACK_USER_TOKEN=xoxp-your-user-token-here

# Run the server
./slack-mcp-server
```

The server uses Stdio transport and will wait for MCP requests on stdin.

### MCP Tools

#### `read_message`

Reads a Slack message and its thread by URL.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "url": {
      "type": "string",
      "description": "Slack message or thread URL to read"
    }
  },
  "required": ["url"]
}
```

**Example Request:**
```json
{
  "name": "read_message",
  "arguments": {
    "url": "https://myworkspace.slack.com/archives/C01234567/p1234567890123456"
  }
}
```

**Example Response:**
```json
{
  "message": {
    "user": "U01234567",
    "text": "Hello, this is the parent message",
    "timestamp": "1234567890.123456",
    "thread_ts": "1234567890.123456"
  },
  "thread": [
    {
      "user": "U01234567",
      "text": "Hello, this is the parent message",
      "timestamp": "1234567890.123456"
    },
    {
      "user": "U09876543",
      "text": "This is a reply",
      "timestamp": "1234567891.123456"
    }
  ],
  "channel_id": "C01234567"
}
```

#### `list_channel_messages`

Lists recent messages from a Slack channel by channel ID.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "channel_id": {
      "type": "string",
      "description": "Slack channel ID (e.g., C01234567)"
    },
    "limit": {
      "type": "number",
      "description": "Number of messages to retrieve (default: 100, max: 200)"
    },
    "oldest": {
      "type": "string",
      "description": "Only return messages after this Unix timestamp"
    },
    "latest": {
      "type": "string",
      "description": "Only return messages before this Unix timestamp"
    }
  },
  "required": ["channel_id"]
}
```

**Example Request:**
```json
{
  "name": "list_channel_messages",
  "arguments": {
    "channel_id": "C01234567",
    "limit": 50
  }
}
```

**Example Response:**
```json
{
  "messages": [
    {
      "user": "U01234567",
      "user_name": "jsmith",
      "display_name": "John Smith",
      "real_name": "John Smith",
      "text": "Here's the latest update on the project",
      "timestamp": "1234567892.123456",
      "reply_count": 3
    },
    {
      "user": "U09876543",
      "user_name": "mjones",
      "display_name": "Mary Jones",
      "real_name": "Mary Jones",
      "text": "Thanks for the update! <@U01234567>",
      "timestamp": "1234567891.123456"
    }
  ],
  "channel_id": "C01234567",
  "has_more": true,
  "current_user": {
    "id": "U11111111",
    "name": "mybot",
    "display_name": "My Bot",
    "real_name": "My Bot",
    "is_bot": true
  },
  "user_mapping": {
    "U01234567": {
      "id": "U01234567",
      "name": "jsmith",
      "display_name": "John Smith",
      "real_name": "John Smith",
      "is_bot": false
    }
  }
}
```

#### `search_messages`

Searches for messages across the Slack workspace. **Requires `SLACK_USER_TOKEN`** with `search:read` scope.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Search query string. Supports Slack search modifiers (in:#channel, from:@user)"
    },
    "count": {
      "type": "number",
      "description": "Number of results to return (default: 20, max: 100)"
    },
    "sort": {
      "type": "string",
      "description": "Sort order: 'score' (relevance) or 'timestamp' (default: score)"
    }
  },
  "required": ["query"]
}
```

**Example Request:**
```json
{
  "name": "search_messages",
  "arguments": {
    "query": "project update in:#engineering",
    "count": 10,
    "sort": "timestamp"
  }
}
```

**Example Response:**
```json
{
  "query": "project update in:#engineering",
  "total": 42,
  "matches": [
    {
      "channel_id": "C01234567",
      "channel_name": "engineering",
      "user": "U01234567",
      "user_name": "jsmith",
      "display_name": "John Smith",
      "real_name": "John Smith",
      "text": "Here's the project update for this week",
      "timestamp": "1234567890.123456",
      "permalink": "https://myworkspace.slack.com/archives/C01234567/p1234567890123456"
    },
    {
      "channel_id": "C01234567",
      "channel_name": "engineering",
      "user": "U09876543",
      "user_name": "mjones",
      "display_name": "Mary Jones",
      "real_name": "Mary Jones",
      "text": "Thanks for the project update!",
      "timestamp": "1234567891.123456",
      "permalink": "https://myworkspace.slack.com/archives/C01234567/p1234567891123456"
    }
  ],
  "current_user": {
    "id": "U11111111",
    "name": "jsmith",
    "display_name": "John Smith",
    "real_name": "John Smith",
    "is_bot": false
  }
}
```

**Search Modifiers:**

The `query` parameter supports Slack's search modifiers:

| Modifier | Description | Example |
|----------|-------------|---------|
| `in:#channel` | Search in a specific channel | `bug in:#engineering` |
| `from:@user` | Search messages from a user | `report from:@jsmith` |
| `before:date` | Messages before a date | `meeting before:2024-01-01` |
| `after:date` | Messages after a date | `release after:2024-01-01` |
| `has:link` | Messages containing links | `documentation has:link` |
| `has:reaction` | Messages with reactions | `announcement has:reaction` |

### Slack URL Formats

The server supports these Slack URL formats:

**Single Message:**
```
https://workspace.slack.com/archives/C01234567/p1234567890123456
```

**Thread Message:**
```
https://workspace.slack.com/archives/C01234567/p1234567890123456?thread_ts=1234567890.123456&cid=C01234567
```

### Integration with Claude Code

Add the server to your Claude Code MCP configuration:

```json
{
  "mcpServers": {
    "slack": {
      "command": "/path/to/slack-mcp-server",
      "env": {
        "SLACK_BOT_TOKEN": "xoxb-your-bot-token-here",
        "SLACK_USER_TOKEN": "xoxp-your-user-token-here"
      }
    }
  }
}
```

**Note:** `SLACK_USER_TOKEN` is optional. If omitted, the `search_messages` tool will not be available, but `read_message` and `list_channel_messages` will work normally.

## Development

### Project Structure

```
slack-mcp-server/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── server/
│   │   └── server.go         # MCP server setup and tool registration
│   ├── slack/
│   │   ├── client.go         # Slack API client wrapper
│   │   └── errors.go         # Error types and handling
│   ├── urlparser/
│   │   ├── parser.go         # Slack URL parsing logic
│   │   └── parser_test.go    # URL parser tests
│   └── tools/
│       ├── read_message.go   # read_message tool implementation
│       ├── read_message_test.go
│       ├── list_channel_messages.go      # list_channel_messages tool implementation
│       ├── list_channel_messages_test.go
│       ├── search_messages.go            # search_messages tool implementation
│       └── search_messages_test.go
├── pkg/
│   └── types/
│       └── types.go          # Shared type definitions
├── go.mod
├── go.sum
└── README.md
```

### Building

```bash
# Build the binary
go build -o slack-mcp-server ./cmd/server

# Build with version information
go build -ldflags "-X main.version=1.0.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o slack-mcp-server ./cmd/server
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Linting

```bash
# Format code
go fmt ./...

# Run linter (requires golangci-lint)
golangci-lint run
```

## Error Handling

The server provides descriptive error messages for common issues:

| Error | Description |
|-------|-------------|
| Invalid URL format | The provided URL is not a valid Slack message URL |
| Message not found | The message doesn't exist or was deleted |
| Channel not found | The channel ID in the URL is invalid |
| Not in channel | The bot needs to be invited to the private channel |
| Rate limit exceeded | Slack API rate limit reached (wait before retrying) |
| Invalid token | The `SLACK_BOT_TOKEN` or `SLACK_USER_TOKEN` is invalid or expired |
| User token not configured | `SLACK_USER_TOKEN` not set when calling `search_messages` |

## Troubleshooting

### "channel_not_found" Error
- Verify the channel ID in the URL is correct
- For private channels, ensure the bot has been invited

### "not_in_channel" Error
- Invite the bot to the private channel: `/invite @your-bot-name`

### "invalid_auth" Error
- Verify your `SLACK_BOT_TOKEN` is correct and starts with `xoxb-`
- Verify your `SLACK_USER_TOKEN` is correct and starts with `xoxp-` (if using search)
- Regenerate the token if necessary

### "user_token_not_configured" Error (search_messages)
- Set the `SLACK_USER_TOKEN` environment variable
- Ensure the user token has the `search:read` scope
- The user token starts with `xoxp-` (not `xoxb-`)

### Rate Limiting
- The Slack API has rate limits (typically 1 request/second for Tier 2 methods)
- The server will return rate limit errors when exceeded
- Wait before retrying

## Dependencies

- [mcp-go](https://github.com/mark3labs/mcp-go) v0.20.1 - MCP protocol implementation for Go
- [slack-go/slack](https://github.com/slack-go/slack) v0.17.3 - Slack API client for Go

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
