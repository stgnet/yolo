package requests

// Request represents a server request
type Request struct {
	Method      string `json:"method"`
	URL         string `json:"url"`
	ContentType string `json:"content_type,omitempty"`
}

// Response represents a server response
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}
