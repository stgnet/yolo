package types

import "time"

// Status represents server status information with request/response details
type Status struct {
	Request       string    `json:"request"`
	Response      string    `json:"response"`
	Code          int       `json:"code"`
	Uptime        string    `json:"uptime"`
	RequestsTotal int64     `json:"requests_total"`
	ServerTime    time.Time `json:"server_time"`
	Version       string    `json:"version"`
}

// AgentStatus represents the YOLO agent status for agent status management
type AgentStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Message string `json:"message"`
}
