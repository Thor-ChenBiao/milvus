// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcp

import (
	"context"
	"fmt"
)

const (
	McpProtocolVersion = "2025-06-18"

	DefaultSearchLimit = 10
)

// McpRequest represents an MCP protocol request
type McpRequest struct {
	Jsonrpc string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// McpResponse represents an MCP protocol response
type McpResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *McpError   `json:"error,omitempty"`
}

// McpError represents an MCP protocol error
type McpError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard MCP error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
)

// McpCapabilities describes server capabilities
type McpCapabilities struct {
	Tools        *ToolsCapability        `json:"tools,omitempty"`
	Resources    *ResourcesCapability    `json:"resources,omitempty"`
	Prompts      *PromptsCapability      `json:"prompts,omitempty"`
	Logging      *LoggingCapability      `json:"logging,omitempty"`
	Experimental *ExperimentalCapability `json:"experimental,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type LoggingCapability struct{}

type ExperimentalCapability struct {
	StructuredOutput bool `json:"structuredOutput"`
}

// McpServerInfo describes the server
type McpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Title   string `json:"title"`
}

// McpInitializeResult is returned by initialize method
type McpInitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    McpCapabilities `json:"capabilities"`
	ServerInfo      McpServerInfo   `json:"serverInfo"`
}

// McpToolsListResult is returned by tools/list method
type McpToolsListResult struct {
	Tools      []McpToolDescription `json:"tools"`
	NextCursor string               `json:"nextCursor,omitempty"` // 用于分页
}

// McpToolDescription describes a tool for listing
type McpToolDescription struct {
	Name         string      `json:"name"`
	Title        string      `json:"title,omitempty"`
	Description  string      `json:"description"`
	InputSchema  *ToolSchema `json:"inputSchema"`            // MCP tool input schema
	OutputSchema *ToolSchema `json:"outputSchema,omitempty"` // MCP tool output schema (optional)
}

// McpToolResult is the result of a tool execution
type McpToolResult struct {
	Content           []McpContent `json:"content"`
	IsError           bool         `json:"isError,omitempty"`
	StructuredContent interface{}  `json:"structuredContent,omitempty"`
}

// McpContent represents content in a tool result
type McpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// PrivilegeRequirement defines the privilege needed to execute a tool
type PrivilegeRequirement struct {
	ObjectType      string
	ObjectPrivilege string
	ObjectNameField string
}

// Tool represents an executable MCP tool
type Tool struct {
	Name               string
	Title              string
	Description        string
	Execute            ExecuteFunc
	InputSchema        *ToolSchema // Input schema for the tool
	OutputSchema       *ToolSchema // Output schema for the tool (optional)
	RequiredPrivileges []PrivilegeRequirement
}

// ToolSchema represents a JSON schema for MCP tool input/output
// Based on MCP 2025-06-18 specification - can be used for both inputSchema and outputSchema
type ToolSchema struct {
	Type       string                  `json:"type"`                 // Must be "object" for tool schemas
	Properties map[string]*SchemaParam `json:"properties,omitempty"` // Schema parameters
	Required   []string                `json:"required,omitempty"`   // Required parameters
}

// SchemaParam represents a single parameter in MCP tool schema
// Based on MCP 2025-06-18: supports StringSchema, NumberSchema, BooleanSchema, EnumSchema
// Can be used for both input and output schema parameters
type SchemaParam struct {
	// Core fields (all types)
	Type        string      `json:"type"` // "string", "number", "integer", "boolean"
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`

	// For enum type
	Enum []interface{} `json:"enum,omitempty"`

	// For numeric types (number, integer)
	Minimum *float64 `json:"minimum,omitempty"`
	Maximum *float64 `json:"maximum,omitempty"`

	// For string type
	MinLength *int   `json:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Format    string `json:"format,omitempty"` // e.g., "date", "email", "uri"

	// For array type (though not in basic MCP spec, useful for Milvus)
	Items *SchemaParam `json:"items,omitempty"`
}

// ToolArgs represents structured arguments for tool execution
type ToolArgs map[string]interface{}

// GetString gets a string from args with default value
func (args ToolArgs) GetString(key, defaultValue string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultValue
}

// GetInt gets an int from args with default value
func (args ToolArgs) GetInt(key string, defaultValue int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return defaultValue
	}
}

// GetBool gets a bool from args with default value
func (args ToolArgs) GetBool(key string, defaultValue bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultValue
}

// Require validates that required parameters exist
func (args ToolArgs) Require(keys ...string) error {
	for _, key := range keys {
		if _, exists := args[key]; !exists {
			return fmt.Errorf("required parameter '%s' is missing", key)
		}
	}
	return nil
}

// ToolResult represents the result of tool execution in MCP format
type ToolResult struct {
	Content           []McpContent `json:"content"`                     // Required: tool result content
	IsError           bool         `json:"isError,omitempty"`           // Optional: indicates error
	StructuredContent interface{}  `json:"structuredContent,omitempty"` // Optional: structured data
}

// NewToolResult creates a new tool result with text content
func NewToolResult(text string) *ToolResult {
	return &ToolResult{
		Content: []McpContent{{
			Type: "text",
			Text: text,
		}},
		IsError: false,
	}
}

// NewToolResultWithData creates a tool result with both text and structured data
func NewToolResultWithData(text string, data interface{}) *ToolResult {
	return &ToolResult{
		Content: []McpContent{{
			Type: "text",
			Text: text,
		}},
		StructuredContent: data,
		IsError:           false,
	}
}

// AsError marks the result as an error
func (r *ToolResult) AsError() *ToolResult {
	r.IsError = true
	return r
}

// WithStructuredContent adds structured data
func (r *ToolResult) WithStructuredContent(data interface{}) *ToolResult {
	r.StructuredContent = data
	return r
}

// ExecuteFunc is the function signature for tool execution
type ExecuteFunc func(ctx context.Context, args ToolArgs) (*ToolResult, error)

// NewToolSchema creates a new tool schema
func NewToolSchema() *ToolSchema {
	return &ToolSchema{
		Type:       "object",
		Properties: make(map[string]*SchemaParam),
	}
}

// AddParameter adds a parameter to the schema
func (s *ToolSchema) AddParameter(name string, param *SchemaParam) *ToolSchema {
	if s.Properties == nil {
		s.Properties = make(map[string]*SchemaParam)
	}
	s.Properties[name] = param
	return s
}

// AddRequired adds required parameters
func (s *ToolSchema) AddRequired(params ...string) *ToolSchema {
	s.Required = append(s.Required, params...)
	return s
}

// GetStringSlice gets a string slice from args
func GetStringSlice(args map[string]interface{}, key string) []string {
	if v, ok := args[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// GetFloatSlice gets a float slice from args
func GetFloatSlice(args map[string]interface{}, key string) []float32 {
	if v, ok := args[key].([]interface{}); ok {
		result := make([]float32, 0, len(v))
		for _, item := range v {
			switch val := item.(type) {
			case float64:
				result = append(result, float32(val))
			case float32:
				result = append(result, val)
			}
		}
		return result
	}
	return nil
}
