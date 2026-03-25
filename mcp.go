package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─── MCP (Model Context Protocol) Client ─────────────────────────────
//
// Implements the MCP client specification (JSON-RPC 2.0) with support
// for stdio and HTTP/SSE transports. MCP servers are declared in
// .yolo/mcp.json and their tools are dynamically registered alongside
// native YOLO tools.

// ─── Configuration ───────────────────────────────────────────────────

// MCPConfig is the top-level structure for .yolo/mcp.json.
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig defines how to connect to a single MCP server.
type MCPServerConfig struct {
	// Transport type: "stdio" (default) or "http"
	Transport string `json:"transport,omitempty"`

	// Stdio transport fields
	Command string            `json:"command,omitempty"` // executable to spawn
	Args    []string          `json:"args,omitempty"`    // command arguments
	Env     map[string]string `json:"env,omitempty"`     // extra environment variables

	// HTTP transport fields
	URL     string            `json:"url,omitempty"`     // base URL for HTTP transport
	Headers map[string]string `json:"headers,omitempty"` // extra HTTP headers

	// Common fields
	Disabled bool `json:"disabled,omitempty"` // skip this server
}

// ─── JSON-RPC 2.0 Types ─────────────────────────────────────────────

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ─── MCP Protocol Types ─────────────────────────────────────────────

// MCPToolInfo represents a tool advertised by an MCP server.
type MCPToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema MCPInputSchema `json:"inputSchema,omitempty"`
}

// MCPInputSchema is the JSON Schema for a tool's input parameters.
type MCPInputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]MCPPropertyDef `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// MCPPropertyDef is a single property in a tool's input schema.
type MCPPropertyDef struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// MCPToolCallResult is the result of calling an MCP tool.
type MCPToolCallResult struct {
	Content []MCPContentBlock `json:"content,omitempty"`
	IsError bool              `json:"isError,omitempty"`
}

// MCPContentBlock is a content item in a tool result.
type MCPContentBlock struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
}

// ─── MCP Transport Interface ────────────────────────────────────────

// mcpTransport abstracts the communication layer with an MCP server.
type mcpTransport interface {
	// Send sends a JSON-RPC request and returns the response.
	Send(ctx context.Context, req *jsonRPCRequest) (*jsonRPCResponse, error)
	// Close shuts down the transport.
	Close() error
}

// ─── Stdio Transport ────────────────────────────────────────────────

type stdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex // serializes request/response pairs
}

func newStdioTransport(command string, args []string, env map[string]string) (*stdioTransport, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	// Discard stderr to prevent blocking
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start MCP server %q: %w", command, err)
	}

	return &stdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}, nil
}

func (t *stdioTransport) Send(ctx context.Context, req *jsonRPCRequest) (*jsonRPCResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Write request line (JSON-RPC over stdio uses newline-delimited JSON)
	if _, err := t.stdin.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("write to MCP server: %w", err)
	}

	// Read response line with context timeout
	type readResult struct {
		line []byte
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		line, err := t.stdout.ReadBytes('\n')
		ch <- readResult{line, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("read from MCP server: %w", r.err)
		}
		var resp jsonRPCResponse
		if err := json.Unmarshal(r.line, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
		return &resp, nil
	}
}

func (t *stdioTransport) Close() error {
	t.stdin.Close()
	// Give the process a moment to exit cleanly, then kill it
	done := make(chan error, 1)
	go func() { done <- t.cmd.Wait() }()
	select {
	case <-time.After(3 * time.Second):
		t.cmd.Process.Kill()
		<-done
	case <-done:
	}
	return nil
}

// ─── HTTP Transport ─────────────────────────────────────────────────

type httpTransport struct {
	baseURL string
	headers map[string]string
	client  *http.Client
}

func newHTTPTransport(baseURL string, headers map[string]string) *httpTransport {
	return &httpTransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		headers: headers,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (t *httpTransport) Send(ctx context.Context, req *jsonRPCRequest) (*jsonRPCResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.baseURL, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request to MCP server: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read MCP response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

func (t *httpTransport) Close() error {
	return nil
}

// ─── MCP Server Connection ─────────────────────────────────────────

// MCPServer represents an active connection to a single MCP server.
type MCPServer struct {
	Name      string        // config key name
	Config    MCPServerConfig
	Transport mcpTransport
	Tools     []MCPToolInfo // tools discovered from this server
	nextID    atomic.Int64
}

// newMCPServer creates and initializes a connection to an MCP server.
func newMCPServer(name string, cfg MCPServerConfig) (*MCPServer, error) {
	s := &MCPServer{
		Name:   name,
		Config: cfg,
	}

	transport := strings.ToLower(cfg.Transport)
	if transport == "" {
		transport = "stdio"
	}

	switch transport {
	case "stdio":
		if cfg.Command == "" {
			return nil, fmt.Errorf("MCP server %q: stdio transport requires 'command'", name)
		}
		t, err := newStdioTransport(cfg.Command, cfg.Args, cfg.Env)
		if err != nil {
			return nil, err
		}
		s.Transport = t

	case "http":
		if cfg.URL == "" {
			return nil, fmt.Errorf("MCP server %q: http transport requires 'url'", name)
		}
		s.Transport = newHTTPTransport(cfg.URL, cfg.Headers)

	default:
		return nil, fmt.Errorf("MCP server %q: unsupported transport %q", name, transport)
	}

	return s, nil
}

// call sends a JSON-RPC request to the server.
func (s *MCPServer) call(ctx context.Context, method string, params any) (*jsonRPCResponse, error) {
	id := s.nextID.Add(1)
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	return s.Transport.Send(ctx, req)
}

// Initialize performs the MCP initialize handshake.
func (s *MCPServer) Initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "yolo",
			"version": "1.0.0",
		},
	}

	resp, err := s.call(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send initialized notification (no response expected for notifications,
	// but we send it as a request with a new ID for compatibility)
	notif := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	// For stdio, we send it but don't wait for response (it's a notification)
	data, _ := json.Marshal(notif)
	if st, ok := s.Transport.(*stdioTransport); ok {
		st.mu.Lock()
		st.stdin.Write(append(data, '\n'))
		st.mu.Unlock()
	}
	// For HTTP we skip the notification — many HTTP servers don't expect it

	return nil
}

// DiscoverTools calls tools/list and populates s.Tools.
func (s *MCPServer) DiscoverTools(ctx context.Context) error {
	resp, err := s.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return fmt.Errorf("tools/list: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	var result struct {
		Tools []MCPToolInfo `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse tools/list result: %w", err)
	}
	s.Tools = result.Tools
	return nil
}

// CallTool invokes a tool on the MCP server.
func (s *MCPServer) CallTool(ctx context.Context, toolName string, arguments map[string]any) (string, error) {
	params := map[string]any{
		"name":      toolName,
		"arguments": arguments,
	}

	resp, err := s.call(ctx, "tools/call", params)
	if err != nil {
		return "", fmt.Errorf("tools/call %q: %w", toolName, err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("tools/call %q error: %s", toolName, resp.Error.Message)
	}

	var result MCPToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		// If we can't parse as structured result, return raw JSON
		return string(resp.Result), nil
	}

	// Concatenate text content blocks
	var texts []string
	for _, block := range result.Content {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}

	output := strings.Join(texts, "\n")
	if result.IsError {
		return "Error: " + output, nil
	}
	return output, nil
}

// Close shuts down the transport.
func (s *MCPServer) Close() error {
	return s.Transport.Close()
}

// ─── MCP Manager ────────────────────────────────────────────────────

// MCPManager manages all MCP server connections and their tools.
type MCPManager struct {
	yoloDir    string
	servers    []*MCPServer
	toolMap    map[string]*MCPServer // maps "servername:toolname" to server
	rawToolMap map[string]*MCPServer // maps MCP original tool name to server (for dispatch)
	toolDefs   []ToolDef             // converted tool definitions for Ollama
	mu         sync.RWMutex
}

// NewMCPManager creates a manager that loads config from .yolo/mcp.json.
func NewMCPManager(yoloDir string) *MCPManager {
	return &MCPManager{
		yoloDir:    yoloDir,
		toolMap:    make(map[string]*MCPServer),
		rawToolMap: make(map[string]*MCPServer),
	}
}

// configPath returns the path to mcp.json.
func (m *MCPManager) configPath() string {
	return filepath.Join(m.yoloDir, "mcp.json")
}

// LoadConfig reads .yolo/mcp.json. Returns nil if file doesn't exist.
func (m *MCPManager) LoadConfig() (*MCPConfig, error) {
	data, err := os.ReadFile(m.configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read mcp.json: %w", err)
	}
	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse mcp.json: %w", err)
	}
	return &cfg, nil
}

// Start loads config, connects to all enabled MCP servers, and discovers tools.
// Errors on individual servers are logged but don't prevent other servers from starting.
func (m *MCPManager) Start() error {
	cfg, err := m.LoadConfig()
	if err != nil {
		return err
	}
	if cfg == nil || len(cfg.MCPServers) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for name, serverCfg := range cfg.MCPServers {
		if serverCfg.Disabled {
			cprint(Gray, fmt.Sprintf("  MCP: %s (disabled)", name))
			continue
		}

		server, err := newMCPServer(name, serverCfg)
		if err != nil {
			cprint(Yellow, fmt.Sprintf("  MCP: %s failed to connect: %v", name, err))
			continue
		}

		if err := server.Initialize(ctx); err != nil {
			cprint(Yellow, fmt.Sprintf("  MCP: %s failed to initialize: %v", name, err))
			server.Close()
			continue
		}

		if err := server.DiscoverTools(ctx); err != nil {
			cprint(Yellow, fmt.Sprintf("  MCP: %s failed to discover tools: %v", name, err))
			server.Close()
			continue
		}

		m.mu.Lock()
		m.servers = append(m.servers, server)

		for _, tool := range server.Tools {
			// Use prefixed name to avoid collisions with native tools
			prefixedName := fmt.Sprintf("mcp_%s_%s", name, tool.Name)
			m.toolMap[prefixedName] = server
			m.rawToolMap[prefixedName] = server

			// Convert MCP tool schema to YOLO ToolDef
			td := mcpToolToToolDef(prefixedName, name, tool)
			m.toolDefs = append(m.toolDefs, td)
		}
		m.mu.Unlock()

		cprint(Green, fmt.Sprintf("  MCP: %s connected (%d tools)", name, len(server.Tools)))
	}

	return nil
}

// GetToolDefs returns all MCP tool definitions for Ollama.
func (m *MCPManager) GetToolDefs() []ToolDef {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.toolDefs
}

// GetToolNames returns the prefixed names of all MCP tools.
func (m *MCPManager) GetToolNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.toolMap))
	for name := range m.toolMap {
		names = append(names, name)
	}
	return names
}

// IsMCPTool returns true if the given tool name is handled by an MCP server.
func (m *MCPManager) IsMCPTool(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.rawToolMap[name]
	return ok
}

// ExecuteTool calls an MCP tool and returns the result string.
func (m *MCPManager) ExecuteTool(name string, args map[string]any) string {
	m.mu.RLock()
	server, ok := m.rawToolMap[name]
	m.mu.RUnlock()

	if !ok {
		return errorMessage("MCP tool %q not found", name)
	}

	// Extract the original MCP tool name (strip prefix)
	prefix := fmt.Sprintf("mcp_%s_", server.Name)
	mcpToolName := strings.TrimPrefix(name, prefix)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ToolTimeout)*time.Second)
	defer cancel()

	result, err := server.CallTool(ctx, mcpToolName, args)
	if err != nil {
		return errorMessage("MCP tool %q: %v", name, err)
	}
	return result
}

// ServerStatus returns a summary of connected servers and their tools.
func (m *MCPManager) ServerStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.servers) == 0 {
		return "No MCP servers connected"
	}

	var sb strings.Builder
	for _, s := range m.servers {
		sb.WriteString(fmt.Sprintf("  %s: %d tools", s.Name, len(s.Tools)))
		transport := s.Config.Transport
		if transport == "" {
			transport = "stdio"
		}
		sb.WriteString(fmt.Sprintf(" [%s]", transport))
		sb.WriteString("\n")
		for _, t := range s.Tools {
			sb.WriteString(fmt.Sprintf("    - mcp_%s_%s", s.Name, t.Name))
			if t.Description != "" {
				desc := t.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				sb.WriteString(": " + desc)
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// Close shuts down all MCP server connections.
func (m *MCPManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.servers {
		s.Close()
	}
	m.servers = nil
	m.toolMap = make(map[string]*MCPServer)
	m.rawToolMap = make(map[string]*MCPServer)
	m.toolDefs = nil
}

// ─── Helpers ─────────────────────────────────────────────────────────

// mcpToolToToolDef converts an MCP tool definition to a YOLO ToolDef.
func mcpToolToToolDef(prefixedName, serverName string, tool MCPToolInfo) ToolDef {
	props := make(map[string]ToolParam)
	for pName, pDef := range tool.InputSchema.Properties {
		paramType := pDef.Type
		if paramType == "" {
			paramType = "string"
		}
		props[pName] = ToolParam{
			Type:        paramType,
			Description: pDef.Description,
		}
	}

	desc := tool.Description
	if desc == "" {
		desc = fmt.Sprintf("MCP tool from %s", serverName)
	} else {
		desc = fmt.Sprintf("[MCP:%s] %s", serverName, desc)
	}

	return toolDef(prefixedName, desc, props, tool.InputSchema.Required)
}
