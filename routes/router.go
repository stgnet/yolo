package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"time"

	"yolo/types"
)

var (
	requestCount atomic.Int64
	lastRequest  time.Time
)

// Handler wraps the router and provides HTTP handlers
type Handler struct {
	config *Config
}

// NewHandler creates a new HTTP handler
func NewHandler(config *Config) *Handler {
	return &Handler{config: config}
}

// ServeHTTP is the main entry point for all requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Track request stats
	requestCount.Add(1)
	lastRequest = time.Now()

	// Log request
	log.Printf("[%s] %s %s", r.RemoteAddr, r.Method, r.URL.Path)

	// Route to appropriate handler
	switch r.URL.Path {
	case "/":
		h.rootHandler(w, r)
	case "/status":
		h.statusHandler(w, r)
	case "/health":
		h.healthHandler(w, r)
	case "/agent/status":
		h.agentStatusHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

// rootHandler serves the main page
func (h *Handler) rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>YOLO Agent</title></head>
<body>
<h1>YOLO Agent Server</h1>
<p>Available endpoints:</p>
<ul>
<li><a href="/status">/status - Server status</a></li>
<li><a href="/health">/health - Health check</a></li>
<li><a href="/agent/status">/agent/status - Agent status</a></li>
</ul>
</body>
</html>`)
}

// statusHandler returns server statistics
func (h *Handler) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data := types.Status{
		Request:       "status",
		Response:      "ok",
		Code:          200,
		Uptime:        time.Since(lastRequest).String(),
		RequestsTotal: requestCount.Load(),
		ServerTime:    lastRequest,
		Version:       "1.0.0",
	}

	json.NewEncoder(w).Encode(data)
}

// healthHandler performs a simple health check
func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// agentStatusHandler returns the status of the YOLO agent
func (h *Handler) agentStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := types.AgentStatus{
		Status:  "running",
		Version: "1.0.0",
		Message: "Agent is operational",
	}

	json.NewEncoder(w).Encode(status)
}

// Config holds the router configuration
type Config struct {
	Port int `json:"port"`
}

func main() {
	config := &Config{Port: 8080}

	h := NewHandler(config)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Println("Starting YOLO server on port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
		os.Exit(1)
	}
}
