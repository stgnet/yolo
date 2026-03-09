package mcp

import (
	"encoding/json"
)

// ProtocolVersion is the current MCP protocol version
const ProtocolVersion = "2024-11-05"

// JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// RequestID is an alias for json.RawMessage to represent JSON-RPC request IDs
type RequestID = json.RawMessage

// ErrorData represents a JSON-RPC error in a response
type ErrorData struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error is an alias for ErrorData
type Error = ErrorData

// TextContent represents text content in tool results
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Root represents a filesystem root
type Root struct {
	URI  string `json:"uri"`
	Name string `json:"name,omitempty"`
}

// Request represents an MCP request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents an MCP response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorData      `json:"error,omitempty"`
}

// Notification is an MCP notification (no response expected)
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Implementation contains server/client info
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ProtocolInfo contains protocol version info
type ProtocolInfo struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// InitializeRequest is sent to initialize the connection
type InitializeRequest struct {
	ProtocolInfo
	ClientInfo   Implementation `json:"clientInfo"`
	Capabilities ClientCapabilities `json:"capabilities,omitempty"`
}

// InitializeResult contains server info after initialization
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
}

// ClientCapabilities describes what the client supports
type ClientCapabilities struct {
	Prompts   *PromptCapability   `json:"prompts,omitempty"`
	Resources *ResourceCapability `json:"resources,omitempty"`
	Tools     *ToolCapability     `json:"tools,omitempty"`
	Sampling  *SamplingCapability `json:"sampling,omitempty"`
	Roots     *RootsCapability    `json:"roots,omitempty"`
}

// ServerCapabilities describes what the server supports
type ServerCapabilities struct {
	Logging   *LoggingCapability   `json:"logging,omitempty"`
	Prompts   *PromptCapability   `json:"prompts,omitempty"`
	Resources *ResourceCapability `json:"resources,omitempty"`
	Tools     *ToolCapability     `json:"tools,omitempty"`
}

// LoggingCapability indicates logging support
type LoggingCapability struct{}

// PromptCapability describes prompt support
type PromptCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapability describes resource support
type ResourceCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolCapability describes tool support
type ToolCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability describes sampling support
type SamplingCapability struct{}

// RootsCapability describes roots support
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPTool defines a tool that can be called
type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// Tool is an alias for MCPTool
type Tool = MCPTool

// Resource represents a read-only data source
type Resource struct {
	URI         string `json:"uri"`
	MimeType    string `json:"mimeType,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ResourceTemplate is a template for creating resources
type ResourceTemplate struct {
	URITemplate   string          `json:"uriTemplate"`
	MimeType      string          `json:"mimeType,omitempty"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	ResourceAnchor *Resource       `json:"resourceAnchor,omitempty"`
}

// Prompt represents a prompt template
type Prompt struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument defines an argument for a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ListResourcesResult contains list of resources
type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

// ReadResourceRequest requests a resource by URI
type ReadResourceRequest struct {
	URI string `json:"uri"`
}

// ReadResourceResult contains resource content
type ReadResourceResult struct {
	Contents []Content `json:"contents"`
}

// Content represents resource content
type Content struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // Base64 encoded binary data
}

// ListToolsResult contains list of tools
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolRequest requests to call a tool
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult contains the result of a tool call
type CallToolResult struct {
	Content []interface{} `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ListPromptsResult contains list of prompts
type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

// GetPromptRequest requests a prompt by name
type GetPromptRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// GetPromptResult contains the result of a prompt request
type GetPromptResult struct {
	Messages []Message `json:"messages"`
}

// Message represents a message in a conversation
type Message struct {
	Role    RoleType    `json:"role"`
	Content interface{} `json:"content"`
	Model   string      `json:"model,omitempty"`
}

// RoleType defines the role of a message sender
type RoleType string

const (
	RoleUser     RoleType = "user"
	RoleAssistant RoleType = "assistant"
)

// SetLevelRequest sets the log level
type SetLevelRequest struct {
	Level Level `json:"level"`
}

// SetLevelResult contains result of setLevel
type SetLevelResult struct{}

// ListRootsResult contains list of roots
type ListRootsResult struct {
	Roots []Root `json:"roots"`
}

// Level represents log levels
type Level int

const (
	LevelDebug   Level = iota + 10
	LevelInfo            = 20
	LevelNotice          = 25
	LevelWarn            = 30
	LevelError           = 40
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelNotice:
		return "notice"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

// Type aliases for convenience
const (
	DebugLevel   Level = LevelDebug
	InfoLevel    Level = LevelInfo
	NoticeLevel  Level = LevelNotice
	WarningLevel Level = LevelWarn
	ErrorLevel   Level = LevelError
)

func (l Level) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

func (l *Level) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	
	switch s {
	case "debug":
		*l = LevelDebug
	case "info":
		*l = LevelInfo
	case "notice":
		*l = LevelNotice
	case "warn", "warning":
		*l = LevelWarn
	case "error":
		*l = LevelError
	default:
		*l = LevelInfo
	}
	return nil
}
