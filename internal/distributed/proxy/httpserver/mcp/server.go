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
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/milvus-io/milvus/internal/proxy"
	"github.com/milvus-io/milvus/internal/types"
	"github.com/milvus-io/milvus/pkg/v2/log"
	"github.com/milvus-io/milvus/pkg/v2/util"
	"github.com/milvus-io/milvus/pkg/v2/util/funcutil"
	"go.uber.org/zap"
)

// McpServer represents the MCP server
type McpServer struct {
	catalog *ToolsCatalog
	enabled bool
}

// NewMcpServer creates a new MCP server
func NewMcpServer(proxy types.ProxyComponent) *McpServer {
	return &McpServer{
		catalog: NewToolsCatalog(proxy),
		enabled: true,
	}
}

// RegisterRoutes registers MCP routes to gin router
func (s *McpServer) RegisterRoutes(router gin.IRouter) {
	if !s.enabled {
		return
	}

	// MCP uses a single JSON-RPC endpoint
	// Support both with and without trailing slash
	router.POST("", s.handleMcpRequest)  // Handles /mcp
	router.POST("/", s.handleMcpRequest) // Handles /mcp/
}

// handleMcpRequest is the main entry point for all MCP requests
func (s *McpServer) handleMcpRequest(c *gin.Context) {
	ctx := c.Request.Context()

	var req McpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// For parse errors, we can't get the request ID
		response := &McpResponse{
			Jsonrpc: "2.0",
			ID:      nil,
			Error: &McpError{
				Code:    ErrorCodeParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
		c.JSON(http.StatusOK, response)
		return
	}

	log.Ctx(ctx).Debug("MCP request received",
		zap.String("method", req.Method),
		zap.String("id", s.getRequestID(&req)))

	// Route to appropriate handler based on method
	switch req.Method {
	case "initialize":
		s.handleInitialize(c, &req)
	case "tools/list":
		s.handleToolsList(c, &req)
	case "tools/call":
		s.handleToolsCall(c, &req)
	case "prompts/list":
		s.handlePromptsList(c, &req)
	case "resources/list":
		s.handleResourcesList(c, &req)
	case "resources/templates/list":
		s.handleResourceTemplatesList(c, &req)
	case "ping":
		s.handlePing(c, &req)
	case "notifications/initialized":
		s.handleNotificationsInitialized(c, &req)
	default:
		log.Ctx(ctx).Error("MCP method not supported", zap.String("method", req.Method))
		s.returnError(c, &req, ErrorCodeMethodNotFound,
			"Method not found: "+req.Method, nil)
	}
}

// Protocol handlers

func (s *McpServer) handleInitialize(c *gin.Context, req *McpRequest) {
	ctx := c.Request.Context()
	requestedVersion, _ := req.Params[ParamProtocolVersion].(string)
	headerVersion := c.GetHeader(MCPHeaderProtocolVersion)
	log.Ctx(ctx).Info(LogEvtInitStart,
		zap.String("id", s.getRequestID(req)),
		zap.String("requested_version", requestedVersion),
		zap.String("header_version", headerVersion))

	// Determine protocol version to return based on client request
	returnVersion := s.getProtocolVersion(c, req)

	// Validate the requested version if explicitly provided
	if protocolVersion, ok := req.Params[ParamProtocolVersion].(string); ok {
		if protocolVersion != "2025-03-26" && protocolVersion != "2025-06-18" {
			// Only support 2025-03-26 and 2025-06-18
			s.returnError(c, req, ErrorCodeInvalidParams,
				"Unsupported protocol version",
				map[string]string{"supported": "2025-06-18, 2025-03-26"})
			return
		}
	}

	result := McpInitializeResult{
		ProtocolVersion: returnVersion,
		Capabilities: McpCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Resources: &ResourcesCapability{
				Subscribe:   false,
				ListChanged: false,
			},
			Prompts: &PromptsCapability{
				ListChanged: false,
			},
			Logging: &LoggingCapability{},
			Experimental: &ExperimentalCapability{
				StructuredOutput: true,
			},
		},
		ServerInfo: McpServerInfo{
			Name:    "milvus-mcp-server",
			Version: returnVersion,
			Title:   "Milvus Vector Database MCP Server",
		},
	}

	s.returnSuccess(c, req, result)
	c.Header(MCPHeaderProtocolVersion, returnVersion)
	log.Ctx(ctx).Info(LogEvtInitDone,
		zap.String("id", s.getRequestID(req)),
		zap.String("protocol_version", returnVersion))
}

func (s *McpServer) handleToolsList(c *gin.Context, req *McpRequest) {
	ctx := c.Request.Context()
	log.Ctx(ctx).Info(LogEvtToolsListStart, zap.String("id", s.getRequestID(req)))

	username := c.GetString("username")
	if username == "" {
		username = "anonymous"
	}

	tools := s.catalog.List()
	filteredTools := s.filterToolsByPermission(username, tools)
	result := McpToolsListResult{Tools: filteredTools}
	log.Ctx(ctx).Info(LogEvtToolsListDone,
		zap.String("user", username),
		zap.Int("total_tools", len(tools)),
		zap.Int("accessible_tools", len(filteredTools)))

	protocolVersion := s.getProtocolVersion(c, req)
	s.returnSuccess(c, req, result)
	c.Header(MCPHeaderProtocolVersion, protocolVersion)
}

func (s *McpServer) handleToolsCall(c *gin.Context, req *McpRequest) {
	ctx := c.Request.Context()
	startTime := time.Now()

	toolName, ok := req.Params["name"].(string)
	if !ok || toolName == "" {
		s.returnError(c, req, ErrorCodeInvalidParams, "Tool name is required", nil)
		return
	}

	arguments, _ := req.Params["arguments"].(map[string]interface{})
	if arguments == nil {
		arguments = make(map[string]interface{})
	}
	toolArgs := ToolArgs(arguments)

	// log only argument keys to avoid verbose/sensitive logs
	argKeys := make([]string, 0, len(arguments))
	for k := range arguments {
		argKeys = append(argKeys, k)
	}
	log.Ctx(ctx).Info(LogEvtToolsCallStart,
		zap.String("id", s.getRequestID(req)),
		zap.String("tool", toolName),
		zap.Strings("arg_keys", argKeys))

	// Get tool from catalog
	tool, exists := s.catalog.Get(toolName)
	if !exists {
		s.returnError(c, req, ErrorCodeMethodNotFound, "Tool not found: "+toolName, nil)
		return
	}

	// Get user info from global auth middleware (if auth is enabled)
	username := c.GetString("username")
	if username == "" {
		username = "anonymous"
	}
	log.Ctx(ctx).Debug(LogEvtToolsCallUser, zap.String("user", username))

	// Check tool permission
	if err := s.checkToolPermission(username, tool, toolArgs); err != nil {
		log.Ctx(ctx).Warn(LogEvtToolsCallDenied,
			zap.String("tool", toolName),
			zap.String("user", username),
			zap.Error(err))

		s.returnToolResult(c, req, McpToolResult{
			Content: []McpContent{{
				Type: "text",
				Text: "Permission denied: " + err.Error(),
			}},
			IsError: true,
		})
		return
	}

	// Execute tool
	result, err := tool.Execute(ctx, toolArgs)
	duration := time.Since(startTime)

	if err != nil {
		log.Ctx(ctx).Error(LogEvtToolsCallFailed,
			zap.String("tool", toolName),
			zap.Error(err),
			zap.Duration("duration", duration))

		s.returnToolResult(c, req, McpToolResult{
			Content: []McpContent{{
				Type: "text",
				Text: "Error: " + err.Error(),
			}},
			IsError: true,
		})
		return
	}

	log.Ctx(ctx).Info(LogEvtToolsCallDone,
		zap.String("tool", toolName),
		zap.Duration("duration", duration),
		zap.Bool("is_error", result.IsError))

	// Return the tool result including structured content
	mcpResult := McpToolResult{
		Content:           result.Content,
		IsError:           result.IsError,
		StructuredContent: result.StructuredContent,
	}
	s.returnToolResult(c, req, mcpResult)

	protocolVersion := s.getProtocolVersion(c, req)
	c.Header(MCPHeaderProtocolVersion, protocolVersion)
}

func (s *McpServer) handlePing(c *gin.Context, req *McpRequest) {
	ctx := c.Request.Context()
	log.Ctx(ctx).Debug("MCP ping request", zap.String("id", s.getRequestID(req)))

	protocolVersion := s.getProtocolVersion(c, req)
	result := map[string]interface{}{
		"status":          "healthy",
		"timestamp":       time.Now().Format(time.RFC3339),
		"protocolVersion": protocolVersion,
		"serverVersion":   protocolVersion,
		"toolsCount":      len(s.catalog.tools),
	}

	s.returnSuccess(c, req, result)
	c.Header(MCPHeaderProtocolVersion, protocolVersion)
}

func (s *McpServer) handleNotificationsInitialized(c *gin.Context, req *McpRequest) {
	ctx := c.Request.Context()
	log.Ctx(ctx).Debug("MCP notifications/initialized", zap.String("id", s.getRequestID(req)))

	result := map[string]interface{}{
		"acknowledged": true,
	}

	protocolVersion := s.getProtocolVersion(c, req)
	s.returnSuccess(c, req, result)
	c.Header(MCPHeaderProtocolVersion, protocolVersion)
}

func (s *McpServer) handlePromptsList(c *gin.Context, req *McpRequest) {
	ctx := c.Request.Context()
	log.Ctx(ctx).Debug("MCP prompts/list request", zap.String("id", s.getRequestID(req)))

	// Return empty prompts list
	result := map[string]interface{}{
		"prompts": []interface{}{},
	}

	protocolVersion := s.getProtocolVersion(c, req)
	s.returnSuccess(c, req, result)
	c.Header(MCPHeaderProtocolVersion, protocolVersion)
}

func (s *McpServer) handleResourcesList(c *gin.Context, req *McpRequest) {
	ctx := c.Request.Context()
	log.Ctx(ctx).Debug("MCP resources/list request", zap.String("id", s.getRequestID(req)))

	// Return empty resources list
	result := map[string]interface{}{
		"resources": []interface{}{},
	}

	protocolVersion := s.getProtocolVersion(c, req)
	s.returnSuccess(c, req, result)
	c.Header(MCPHeaderProtocolVersion, protocolVersion)
}

func (s *McpServer) handleResourceTemplatesList(c *gin.Context, req *McpRequest) {
	ctx := c.Request.Context()
	log.Ctx(ctx).Debug("MCP resources/templates/list request", zap.String("id", s.getRequestID(req)))

	// Return empty resource templates list
	result := map[string]interface{}{
		"resourceTemplates": []interface{}{},
	}

	protocolVersion := s.getProtocolVersion(c, req)
	s.returnSuccess(c, req, result)
	c.Header(MCPHeaderProtocolVersion, protocolVersion)
}

// Helper methods

func (s *McpServer) returnSuccess(c *gin.Context, req *McpRequest, result interface{}) {
	response := &McpResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  result,
	}
	c.JSON(http.StatusOK, response)
}

func (s *McpServer) returnError(c *gin.Context, req *McpRequest, code int, message string, data interface{}) {
	response := &McpResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Error: &McpError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	c.JSON(http.StatusOK, response)
}

func (s *McpServer) returnToolResult(c *gin.Context, req *McpRequest, result McpToolResult) {
	s.returnSuccess(c, req, result)
}

func (s *McpServer) getRequestID(req *McpRequest) string {
	if id, ok := req.ID.(string); ok {
		return id
	}
	if id, ok := req.ID.(float64); ok {
		return string(rune(int(id)))
	}
	return "unknown"
}

// getProtocolVersion determines the protocol version to use based on request context
func (s *McpServer) getProtocolVersion(c *gin.Context, req *McpRequest) string {
	// First check if protocol version is in request params (for initialize)
	if protocolVersion, ok := req.Params[ParamProtocolVersion].(string); ok && protocolVersion == "2025-03-26" {
		return "2025-03-26"
	}

	// Then check MCP-Protocol-Version header
	if headerVersion := c.GetHeader("MCP-Protocol-Version"); headerVersion == "2025-03-26" {
		return "2025-03-26"
	}

	// Default to latest version
	return McpProtocolVersion
}

func (s *McpServer) isProtocolVersionSupported(version string) bool {
	supportedVersions := []string{
		"2025-06-18",
		"2025-03-26",
		"2024-11-05",
	}
	for _, supported := range supportedVersions {
		if supported == version {
			return true
		}
	}
	return false
}

func (s *McpServer) checkToolPermission(username string, tool *Tool, args ToolArgs) error {
	if !proxy.Params.CommonCfg.AuthorizationEnabled.GetAsBool() {
		return nil
	}

	if len(tool.RequiredPrivileges) == 0 {
		return nil
	}

	roleNames, err := proxy.GetRole(username)
	if err != nil {
		return err
	}
	roleNames = append(roleNames, util.RolePublic)

	dbName := args.GetString("database", util.DefaultDBName)

	for _, privilege := range tool.RequiredPrivileges {
		objectName := args.GetString(privilege.ObjectNameField, "*")
		if privilege.ObjectType == "Database" && objectName == "*" {
			objectName = dbName
		}

		objectResource := funcutil.PolicyForResource(dbName, privilege.ObjectType, objectName)

		hasPermission := false
		for _, roleName := range roleNames {
			isPermit, cached, version := proxy.GetPrivilegeCache(roleName, objectResource, privilege.ObjectPrivilege)
			if cached && isPermit {
				hasPermission = true
				break
			}

			if !cached {
				proxy.SetPrivilegeCache(roleName, objectResource, privilege.ObjectPrivilege, false, version)
			}
		}

		if !hasPermission {
			return fmt.Errorf("permission denied: %s privilege required on %s:%s",
				privilege.ObjectPrivilege, privilege.ObjectType, objectName)
		}
	}

	return nil
}

func (s *McpServer) filterToolsByPermission(username string, tools []McpToolDescription) []McpToolDescription {
	if !proxy.Params.CommonCfg.AuthorizationEnabled.GetAsBool() {
		return tools
	}

	filtered := make([]McpToolDescription, 0)
	for _, toolDesc := range tools {
		tool, exists := s.catalog.Get(toolDesc.Name)
		if !exists {
			continue
		}

		args := ToolArgs{"database": util.DefaultDBName}
		if s.checkToolPermission(username, tool, args) == nil {
			filtered = append(filtered, toolDesc)
		}
	}

	return filtered
}
