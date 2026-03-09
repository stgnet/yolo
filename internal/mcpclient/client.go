// Package mcpclient provides a client implementation for connecting to Model Context Protocol servers.
package mcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Protocol versions
const (
	ProtocolVersion = "2024-11-05"
)

// Client represents an MCP client connection to a server
type Client struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	scanner   *bufio.Scanner
	idCounter int64
	mu        sync.Mutex
	tools     []Tool
	prompts   []Prompt
	resources []Resource
	handlers  map[int64]chan Response
	done      chan struct{}
}

// Tool represents an MCP tool
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
	MCPClient   *Client         `json:"-"` // Reference back to client for calling
}

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument defines an argument for a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// JSON-RPC types
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      int64           `json:"id,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Initialize request/response
type InitializeRequestParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ServerInformation  `json:"clientInfo"`
}

type ClientCapabilities struct {
	Tools     *struct{ ListChanged bool } `json:"tools,omitempty"`
	Prompts   *struct{ ListChanged bool } `json:"prompts,omitempty"`
	Resources *struct{ ListChanged bool } `json:"resources,omitempty"`
}

type ServerInformation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResultParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInformation  `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools     *struct{ ListChanged bool } `json:"tools,omitempty"`
	Prompts   *struct{ ListChanged bool } `json:"prompts,omitempty"`
	Resources *struct{ ListChanged bool } `json:"resources,omitempty"`
}

// Tools list request/response
type ListToolsRequestParams struct{}
type ListToolsResultParams struct {
	Tools []Tool `json:"tools"`
}

// Tool call request/response
type CallToolRequestParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}
type CallToolResultParams struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content can be text or other types
type Content struct {
	Type string          `json:"type"`
	Text string          `json:"text,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// NewClient creates a new MCP client connection to a server process
func NewServer(ctx context.Context, command string, args []string, env []string) (*Client, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if len(env) > 0 {
		cmd.Env = env
	} else {
		cmd.Env = os.Environ()
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	client := &Client{
		cmd:      cmd,
		stdin:    stdin,
		scanner:  bufio.NewScanner(stdout),
		handlers: make(map[int64]chan Response),
		done:     make(chan struct{}),
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Start reading responses in background
	go client.readResponses()

	// Initialize the connection
	if err := client.initialize(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	// Discover tools, prompts, and resources
	client.discoverCapabilities()

	return client, nil
}

// Close closes the MCP client connection
func (c *Client) Close() error {
	close(c.done)
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// GetTools returns the list of available tools
func (c *Client) GetTools() []Tool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tools
}

// CallTool calls a tool with the given arguments
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*CallToolResultParams, error) {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/call",
	}

	params := CallToolRequestParams{Name: name, Arguments: args}
	req.Params = mustMarshal(params)

	result, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var callResult CallToolResultParams
	if err := json.Unmarshal(result.Result, &callResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &callResult, nil
}

// GetPrompts returns the list of available prompts
func (c *Client) GetPrompts() []Prompt {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.prompts
}

// GetResources returns the list of available resources
func (c *Client) GetResources() []Resource {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resources
}

// nextID generates the next request ID
func (c *Client) nextID() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.idCounter++
	return c.idCounter
}

// sendRequest sends a request and waits for the response
func (c *Client) sendRequest(ctx context.Context, req Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Add newline terminator
	data = append(data, '\n')

	// Create a channel for the response
	ch := make(chan Response, 1)

	c.mu.Lock()
	c.handlers[req.ID] = ch
	c.mu.Unlock()

	// Send the request
	if _, err := c.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Wait for response with timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		c.mu.Lock()
		delete(c.handlers, req.ID)
		c.mu.Unlock()

		if resp.Error != nil {
			return &resp, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return &resp, nil
	case <-c.done:
		return nil, fmt.Errorf("connection closed")
	}
}

// readResponses reads JSON-RPC responses from stdout
func (c *Client) readResponses() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		if !c.scanner.Scan() {
			return
		}

		line := c.scanner.Bytes()
		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			continue // Skip invalid lines
		}

		c.mu.Lock()
		handler, ok := c.handlers[resp.ID]
		c.mu.Unlock()

		if ok {
			select {
			case handler <- resp:
			default:
				// Handler not ready, drop response
			}
		}
	}
}

// initialize sends the initialization request
func (c *Client) initialize(ctx context.Context) error {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "initialize",
	}

	params := InitializeRequestParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ClientCapabilities{
			Tools:     &struct{ ListChanged bool }{},
			Prompts:   &struct{ ListChanged bool }{},
			Resources: &struct{ ListChanged bool }{},
		},
		ClientInfo: ServerInformation{
			Name:    "yolo",
			Version: "1.0.0",
		},
	}
	req.Params = mustMarshal(params)

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return err
	}

	var result InitializeResultParams
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}

	return nil
}

// discoverCapabilities discovers available tools, prompts, and resources
func (c *Client) discoverCapabilities() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// List tools
	toolsResp, err := c.listTools(ctx)
	if err == nil && toolsResp != nil {
		for i := range toolsResp.Tools {
			toolsResp.Tools[i].MCPClient = c
		}
		c.mu.Lock()
		c.tools = toolsResp.Tools
		c.mu.Unlock()
	}

	// List prompts (if supported)
	promptsResp, err := c.listPrompts(ctx)
	if err == nil && promptsResp != nil {
		c.mu.Lock()
		c.prompts = promptsResp.Prompts
		c.mu.Unlock()
	}

	// List resources (if supported)
	resourcesResp, err := c.listResources(ctx)
	if err == nil && resourcesResp != nil {
		c.mu.Lock()
		c.resources = resourcesResp.Resources
		c.mu.Unlock()
	}
}

func (c *Client) listTools(ctx context.Context) (*ListToolsResultParams, error) {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/list",
		Params:  mustMarshal(ListToolsRequestParams{}),
	}

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result ListToolsResultParams
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
	}

	return &result, nil
}

func (c *Client) listPrompts(ctx context.Context) (*struct{ Prompts []Prompt }, error) {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "prompts/list",
		Params:  mustMarshal(struct{}{}),
	}

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result struct{ Prompts []Prompt }
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompts: %w", err)
	}

	return &result, nil
}

func (c *Client) listResources(ctx context.Context) (*struct{ Resources []Resource }, error) {
	req := Request{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "resources/list",
		Params:  mustMarshal(struct{}{}),
	}

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result struct{ Resources []Resource }
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	return &result, nil
}

// mustMarshal marshals data to JSON, panicking on error
func mustMarshal(data any) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(bytes)
}
