package main

import "encoding/json"

// successJSON returns a successful MCP result with JSON-encoded data as text content.
func successJSON(data any) *MCPToolResult {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return errorResult("failed to marshal response: " + err.Error())
	}
	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: string(b)}},
	}
}

// errorResult returns an MCP error result.
func errorResult(msg string) *MCPToolResult {
	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: msg}},
		IsError: true,
	}
}
