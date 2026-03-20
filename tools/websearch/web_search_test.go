package websearch

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestSearch_InputValidation tests input validation in Search method
func TestSearch_InputValidation(t *testing.T) {
	ws := NewWebSearcher()

	tests := []struct {
		name    string
		ctx     context.Context
		query   string
		limit   int
		wantErr bool
	}{
		{"empty query", context.Background(), "", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ws.Search(tt.ctx, tt.query, tt.limit)
			
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNewWebSearcher tests default constructor
func TestNewWebSearcher(t *testing.T) {
	ws := NewWebSearcher()
	
	if ws.DuckDuckGoURL != "https://html.duckduckgo.com/html/" {
		t.Errorf("unexpected DuckDuckGoURL: %q", ws.DuckDuckGoURL)
	}
	
	if ws.WikipediaURL != "https://en.wikipedia.org/api/rest_v1/page/summary/" {
		t.Errorf("unexpected WikipediaURL: %q", ws.WikipediaURL)
	}
	
	if ws.Client == nil {
		t.Error("expected non-nil client")
	}
	
	if ws.RetryCount != 2 {
		t.Errorf("expected retry count 2, got %d", ws.RetryCount)
	}
	
	if ws.BaseTimeout != 30*time.Second {
		t.Errorf("expected base timeout 30s, got %v", ws.BaseTimeout)
	}
	
	if ws.UserAgent != DefaultUserAgent {
		t.Errorf("unexpected UserAgent: %q", ws.UserAgent)
	}
}

// TestNewWebSearcherWithConfig tests custom configuration constructor
func TestNewWebSearcherWithConfig(t *testing.T) {
	customClient := &http.Client{Timeout: 60 * time.Second}
	timeout := 45 * time.Second
	retryCount := 3
	
	ws := NewWebSearcherWithConfig(customClient, retryCount, timeout)
	
	if ws.Client != customClient {
		t.Error("expected custom client")
	}
	
	if ws.RetryCount != retryCount {
		t.Errorf("expected retry count %d, got %d", retryCount, ws.RetryCount)
	}
	
	if ws.BaseTimeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, ws.BaseTimeout)
	}
}

// TestNewWebSearcherWithConfig_NilClient tests nil client handling
func TestNewWebSearcherWithConfig_NilClient(t *testing.T) {
	ws := NewWebSearcherWithConfig(nil, 2, 30*time.Second)
	
	if ws.Client == nil {
		t.Error("expected non-nil client even when passed nil")
	}
	
	if ws.Client.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", ws.Client.Timeout)
	}
}

// TestDefaultUserAgent tests default user agent constant
func TestDefaultUserAgent(t *testing.T) {
	if DefaultUserAgent == "" {
		t.Error("DefaultUserAgent should not be empty")
	}
	
	if !strings.Contains(DefaultUserAgent, "YOLO") {
		t.Error("DefaultUserAgent should contain YOLO")
	}
}
