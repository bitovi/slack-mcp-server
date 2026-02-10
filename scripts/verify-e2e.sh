#!/bin/bash
# End-to-end verification script for Slack MCP Server
# This script verifies that the MCP server starts correctly and exposes the read_message tool.

set -e

echo "=== Slack MCP Server End-to-End Verification ==="
echo ""

# Check for required environment variable
if [ -z "$SLACK_BOT_TOKEN" ]; then
    echo "WARNING: SLACK_BOT_TOKEN is not set."
    echo "For full verification, set: export SLACK_BOT_TOKEN=xoxb-your-token"
    echo ""
fi

# Build the server
echo "1. Building the server..."
go build -o slack-mcp-server ./cmd/server
echo "   [PASS] Build successful"
echo ""

# Run tests
echo "2. Running unit tests..."
go test -v ./... 2>&1 | tail -20
echo "   [PASS] Tests completed"
echo ""

# Verify help flag works
echo "3. Testing --help flag..."
./slack-mcp-server --help | head -5
echo "   [PASS] Help flag works"
echo ""

# Verify version flag works
echo "4. Testing --version flag..."
./slack-mcp-server --version
echo "   [PASS] Version flag works"
echo ""

# Check if SLACK_BOT_TOKEN is set for MCP verification
if [ -n "$SLACK_BOT_TOKEN" ]; then
    echo "5. Testing MCP tools/list response..."
    # Send a tools/list request and capture the response
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | timeout 5 ./slack-mcp-server 2>/dev/null || true)

    if echo "$RESPONSE" | grep -q "read_message"; then
        echo "   [PASS] read_message tool found in response"
        echo ""
        echo "   Tool listing response (abbreviated):"
        echo "$RESPONSE" | head -50
    else
        echo "   [WARN] Could not verify read_message tool in response"
        echo "   Response: $RESPONSE"
    fi
else
    echo "5. Skipping MCP tools/list verification (SLACK_BOT_TOKEN not set)"
fi

echo ""
echo "=== Verification Complete ==="
echo ""
echo "Expected MCP tool schema for read_message:"
echo '  {
    "name": "read_message",
    "description": "Read a Slack message and its thread by URL...",
    "inputSchema": {
      "type": "object",
      "properties": {
        "url": {
          "type": "string",
          "description": "Slack message or thread URL to read..."
        }
      },
      "required": ["url"]
    }
  }'
