package router

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response is a simple HTTP response helper
type Response struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// HelloHandler handles /hello endpoint
func HelloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := Response{
		Status:    "success",
		Message:   "Hello from YOLO!",
		Timestamp: time.Now(),
	}
	json.NewEncoder(w).Encode(resp)
}

// HealthHandler handles /health endpoint
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := Response{
		Status:    "ok",
		Message:   "Service is healthy",
		Timestamp: time.Now(),
	}
	json.NewEncoder(w).Encode(resp)
}

// SetupRoutes configures all HTTP routes
func SetupRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", HelloHandler)
	mux.HandleFunc("/health", HealthHandler)
	return mux
}
