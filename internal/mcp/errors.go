package mcp

import (
	"encoding/json"
	"fmt"
)

// JSONRPCError interface for typed errors
type JSONRPCError interface {
	error
	Code() int
	Message() string
	Data() interface{}
}

// rpcError implements JSONRPCError
type rpcError struct {
	code    int
	message string
	data    interface{}
}

func (e *rpcError) Code() int       { return e.code }
func (e *rpcError) Message() string { return e.message }
func (e *rpcError) Data() interface{} { return e.data }
func (e *rpcError) Error() string {
	if e.data != nil {
		return fmt.Sprintf("%s: %v", e.message, e.data)
	}
	return e.message
}

// createError creates a JSON-RPC error
func createError(code int, message string, data interface{}) JSONRPCError {
	return &rpcError{code: code, message: message, data: data}
}

// createErrorResponse creates a JSON-RPC error response
func createErrorResponse(id RequestID, code int, message string, data interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorData{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// createSuccessResponse creates a JSON-RPC success response
func createSuccessResponse(id RequestID, result json.RawMessage) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}
