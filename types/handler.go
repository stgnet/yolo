package types

// HTTPHandler represents an HTTP request handler
type HTTPHandler func(method string, path string) ([]byte, int)
