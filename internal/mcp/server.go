package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// Server implements an MCP server
type Server struct {
	name        string
	version     string
	tools       []MCPTool
	resources   []Resource
	resourceTpls []ResourceTemplate
	prompts     []Prompt
	capabilities *ServerCapabilities
	
	mu sync.RWMutex
	
	toolHandlers      map[string]ToolHandlerFunc
	resourceReaders   map[string]ResourceReaderFunc
	promptHandlers    map[string]PromptHandlerFunc
	
	logLevel Level
	clientID string
}

// ToolHandlerFunc handles tool invocations
type ToolHandlerFunc func(ctx *Server, args map[string]interface{}) (*CallToolResult, error)

// ResourceReaderFunc reads resource content
type ResourceReaderFunc func(ctx *Server) (*ReadResourceResult, error)

// PromptHandlerFunc handles prompt requests
type PromptHandlerFunc func(ctx *Server, args map[string]interface{}) (*GetPromptResult, error)

// NewServer creates a new MCP server
func NewServer(name, version string) *Server {
	return &Server{
		name:        name,
		version:     version,
		tools:       make([]MCPTool, 0),
		resources:   make([]Resource, 0),
		resourceTpls: make([]ResourceTemplate, 0),
		prompts:     make([]Prompt, 0),
		capabilities: &ServerCapabilities{
			Logging:   &LoggingCapability{},
			Prompts:   &PromptCapability{},
			Resources: &ResourceCapability{},
			Tools:     &ToolCapability{},
		},
		toolHandlers:    make(map[string]ToolHandlerFunc),
		resourceReaders: make(map[string]ResourceReaderFunc),
		promptHandlers:  make(map[string]PromptHandlerFunc),
		logLevel:        LevelInfo,
	}
}

// ServeSSE handles SSE transport (basic implementation)
func (s *Server) ServeSSE(w io.Writer) {
	// For now, we'll implement a simple echo server for SSE
	// Full SSE implementation would require proper event streams
	
	fmt.Fprintf(w, "event: message\ndata: %s\n\n", `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
}

func (s *Server) handleRequest(req *Request) *Response {
	s.mu.RLock()
	_ = s.clientID // will use for logging later
	s.mu.RUnlock()
	
	var result json.RawMessage
	var err error
	
	switch req.Method {
	case "initialize":
		result, err = s.handleInitialize(req)
	case "initialized":
		return nil // Notifications don't get responses
	case "ping":
		result, err = s.handlePing(req)
	case "resources/list":
		result, err = s.handleListResources()
	case "resources/read":
		result, err = s.handleReadResource(req)
	case "tools/list":
		result, err = s.handleListTools()
	case "tools/call":
		result, err = s.handleCallTool(req)
	case "prompts/list":
		result, err = s.handleListPrompts()
	case "prompts/get":
		result, err = s.handleGetPrompt(req)
	case "logging/setLevel":
		result, err = s.handleSetLevel(req)
	case "roots/list":
		result, err = s.handleListRoots()
	default:
		err = createError(MethodNotFound, "Method not found", nil)
	}
	
	if err != nil {
		jsonrpcErr, ok := err.(JSONRPCError)
		if !ok {
			return createErrorResponse(req.ID, InternalError, "Internal error", nil)
		}
		return createErrorResponse(req.ID, jsonrpcErr.Code(), 
			jsonrpcErr.Message(), jsonrpcErr.Data())
	}
	
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) handleInitialize(req *Request) (json.RawMessage, error) {
	var initReq InitializeRequest
	if err := json.Unmarshal(req.Params, &initReq); err != nil {
		return nil, createError(InvalidParams, "Invalid params", nil)
	}
	
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    *s.capabilities,
		ServerInfo: Implementation{
			Name:    s.name,
			Version: s.version,
		},
	}
	
	s.mu.Lock()
	s.clientID = "stdio-0" // Default for stdio transport
	s.mu.Unlock()
	
	return json.Marshal(result)
}

func (s *Server) handlePing(*Request) (json.RawMessage, error) {
	return nil, nil // Empty result for ping
}

func (s *Server) handleListResources() (json.RawMessage, error) {
	s.mu.RLock()
	resources := slices.Clone(s.resources)
	s.mu.RUnlock()
	
	result := ListResourcesResult{
		Resources: resources,
	}
	return json.Marshal(result)
}

func (s *Server) handleReadResource(req *Request) (json.RawMessage, error) {
	var readReq ReadResourceRequest
	if err := json.Unmarshal(req.Params, &readReq); err != nil {
		return nil, createError(InvalidParams, "Invalid params", nil)
	}
	
	s.mu.RLock()
	readerFunc, ok := s.resourceReaders[readReq.URI]
	s.mu.RUnlock()
	
	if !ok {
		return nil, createError(-32001, "Resource not found", readReq.URI)
	}
	
	result, err := readerFunc(s)
	if err != nil {
		return nil, createError(InternalError, "Read failed: "+err.Error(), nil)
	}
	
	return json.Marshal(result)
}

func (s *Server) handleListTools() (json.RawMessage, error) {
	s.mu.RLock()
	tools := slices.Clone(s.tools)
	s.mu.RUnlock()
	
	result := ListToolsResult{
		Tools: tools,
	}
	return json.Marshal(result)
}

func (s *Server) handleCallTool(req *Request) (json.RawMessage, error) {
	var callReq CallToolRequest
	if err := json.Unmarshal(req.Params, &callReq); err != nil {
		return nil, createError(InvalidParams, "Invalid params", nil)
	}
	
	s.mu.RLock()
	handler, ok := s.toolHandlers[callReq.Name]
	s.mu.RUnlock()
	
	if !ok {
		return nil, createError(MethodNotFound, "Tool not found", callReq.Name)
	}
	
	result, err := handler(s, callReq.Arguments)
	if err != nil {
		return nil, createError(InternalError, "Tool execution failed: "+err.Error(), nil)
	}
	
	return json.Marshal(result)
}

func (s *Server) handleListPrompts() (json.RawMessage, error) {
	s.mu.RLock()
	prompts := slices.Clone(s.prompts)
	s.mu.RUnlock()
	
	result := ListPromptsResult{
		Prompts: prompts,
	}
	return json.Marshal(result)
}

func (s *Server) handleGetPrompt(req *Request) (json.RawMessage, error) {
	var getReq GetPromptRequest
	if err := json.Unmarshal(req.Params, &getReq); err != nil {
		return nil, createError(InvalidParams, "Invalid params", nil)
	}
	
	s.mu.RLock()
	handler, ok := s.promptHandlers[getReq.Name]
	s.mu.RUnlock()
	
	if !ok {
		return nil, createError(MethodNotFound, "Prompt not found", getReq.Name)
	}
	
	result, err := handler(s, getReq.Arguments)
	if err != nil {
		return nil, createError(InternalError, "Prompt fetch failed: "+err.Error(), nil)
	}
	
	return json.Marshal(result)
}

func (s *Server) handleSetLevel(req *Request) (json.RawMessage, error) {
	var setReq SetLevelRequest
	if err := json.Unmarshal(req.Params, &setReq); err != nil {
		return nil, createError(InvalidParams, "Invalid params", nil)
	}
	
	// TODO: Implement log level setting
	return nil, nil
}

func (s *Server) handleListRoots() (json.RawMessage, error) {
	// For now, return empty roots
	result := ListRootsResult{
		Roots: []Root{},
	}
	return json.Marshal(result)
}

// Utility functions
func writeResponse(out io.Writer, resp *Response) error {
	if resp == nil {
		return nil
	}
	
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	
	_, err = out.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	
	return nil
}

// RegisterTool registers a tool with the server
func (s *Server) RegisterTool(tool MCPTool, handler ToolHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.toolHandlers[tool.Name]; exists {
		return fmt.Errorf("tool already registered: %s", tool.Name)
	}
	
	s.tools = append(s.tools, tool)
	s.toolHandlers[tool.Name] = handler
	
	return nil
}

// RegisterResource registers a resource with the server
func (s *Server) RegisterResource(resource Resource, reader ResourceReaderFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.resourceReaders[resource.URI]; exists {
		return fmt.Errorf("resource already registered: %s", resource.URI)
	}
	
	s.resources = append(s.resources, resource)
	s.resourceReaders[resource.URI] = reader
	
	return nil
}

// RegisterPrompt registers a prompt with the server
func (s *Server) RegisterPrompt(prompt Prompt, handler PromptHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.promptHandlers[prompt.Name]; exists {
		return fmt.Errorf("prompt already registered: %s", prompt.Name)
	}
	
	s.prompts = append(s.prompts, prompt)
	s.promptHandlers[prompt.Name] = handler
	
	return nil
}

// Helper to create text content
func newTextContent(text string) TextContent {
	return TextContent{Type: "text", Text: text}
}

// Helper to create error tool result
func newErrorToolResult(message string) CallToolResult {
	return CallToolResult{
		Content: []Content{newTextContent("Error: " + message)},
		IsError: true,
	}
}

// Helper to create success tool result
func newTextToolResult(text string) CallToolResult {
	return CallToolResult{
		Content:   []Content{newTextContent(text)},
		IsError:   false,
	}
}

// Format a map of arguments into a readable string
func formatArguments(args map[string]interface{}) string {
	var builder strings.Builder
	for k, v := range args {
		if builder.Len() > 0 {
			builder.WriteString(" ")
		}
		switch val := v.(type) {
		case bool:
			builder.WriteString(fmt.Sprintf("%s=%v", k, val))
		case float64:
			if val == float64(int(val)) {
				builder.WriteString(fmt.Sprintf("%s=%d", k, int(val)))
			} else {
				builder.WriteString(fmt.Sprintf("%s=%.2f", k, val))
			}
		default:
			builder.WriteString(fmt.Sprintf("%s=%q", k, v))
		}
	}
	return builder.String()
}
