package builtin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// HTTPClientTool issues HTTP requests
type HTTPClientTool struct {
	*tools.BaseTool
	client      *http.Client
	maxBodySize int64
}

// HTTPRequest contains HTTP request parameters
type HTTPRequest struct {
	Method  string            `json:"method" description:"HTTP method: GET, POST, PUT, DELETE, etc."`
	URL     string            `json:"url" description:"Request URL"`
	Headers map[string]string `json:"headers,omitempty" description:"Request headers"`
	Body    string            `json:"body,omitempty" description:"Request body (JSON string or plain text)"`
	Timeout int               `json:"timeout,omitempty" description:"Timeout in seconds (default 30)"`
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	Success    bool              `json:"success"`
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Size       int64             `json:"size"`
	Duration   string            `json:"duration"`
	Error      string            `json:"error,omitempty"`
}

// NewHTTPClientTool constructs an HTTP client tool
func NewHTTPClientTool(maxBodySize int64) *HTTPClientTool {
	if maxBodySize <= 0 {
		maxBodySize = 5 * 1024 * 1024 // Default 5MB
	}

	schema := tools.CreateToolSchema(
		"HTTP client tool for fetching network resources",
		map[string]interface{}{
			"method":  tools.StringProperty("HTTP method: GET, POST, PUT, DELETE, etc."),
			"url":     tools.StringProperty("Request URL"),
			"headers": tools.ObjectProperty("Request header key-value pairs", map[string]interface{}{}),
			"body":    tools.StringProperty("Request body content"),
			"timeout": tools.NumberProperty("Timeout in seconds (default 30)"),
		},
		[]string{"method", "url"},
	)

	baseTool := tools.NewBaseTool("http_client", "HTTP client tool for fetching network resources", schema)

	return &HTTPClientTool{
		BaseTool: baseTool,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxBodySize: maxBodySize,
	}
}

// Execute performs the HTTP request
func (t *HTTPClientTool) Execute(ctx runtime.Context, input json.RawMessage) (json.RawMessage, error) {
	var httpReq HTTPRequest
	if err := json.Unmarshal(input, &httpReq); err != nil {
		return nil, schema.NewToolError(t.Name(), "parse_input", err)
	}

	// Validate input
	if httpReq.Method == "" {
		return t.errorResponse("HTTP method cannot be empty")
	}
	if httpReq.URL == "" {
		return t.errorResponse("URL cannot be empty")
	}

	// Normalize the HTTP method
	httpReq.Method = strings.ToUpper(httpReq.Method)

	// Configure timeout
	if httpReq.Timeout > 0 {
		t.client.Timeout = time.Duration(httpReq.Timeout) * time.Second
	}

	startTime := time.Now()

	// Construct the request
	var bodyReader io.Reader
	if httpReq.Body != "" {
		bodyReader = strings.NewReader(httpReq.Body)
	}

	req, err := http.NewRequest(httpReq.Method, httpReq.URL, bodyReader)
	if err != nil {
		return t.errorResponse(fmt.Sprintf("failed to create request: %v", err))
	}

	// Apply headers
	if httpReq.Headers != nil {
		for key, value := range httpReq.Headers {
			req.Header.Set(key, value)
		}
	}

	// Auto-detect Content-Type when a body is provided without one
	if httpReq.Body != "" && req.Header.Get("Content-Type") == "" {
		if t.isJSON(httpReq.Body) {
			req.Header.Set("Content-Type", "application/json")
		} else {
			req.Header.Set("Content-Type", "text/plain")
		}
	}

	// Send the request
	resp, err := t.client.Do(req)
	if err != nil {
		return t.errorResponse(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Limit the response body size
	limitedReader := io.LimitReader(resp.Body, t.maxBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return t.errorResponse(fmt.Sprintf("failed to read response body: %v", err))
	}

	// Extract response headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0] // Use the first value
		}
	}

	// Build the response
	httpResp := HTTPResponse{
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       string(body),
		Size:       int64(len(body)),
		Duration:   duration.String(),
	}

	return json.Marshal(httpResp)
}

// errorResponse builds an error response
func (t *HTTPClientTool) errorResponse(errorMsg string) (json.RawMessage, error) {
	resp := HTTPResponse{
		Success: false,
		Error:   errorMsg,
	}
	return json.Marshal(resp)
}

// isJSON checks whether the string is JSON
func (t *HTTPClientTool) isJSON(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}

// ExecuteAsync performs the request asynchronously
func (t *HTTPClientTool) ExecuteAsync(ctx runtime.Context, input json.RawMessage) (<-chan tools.ToolResult, error) {
	resultChan := make(chan tools.ToolResult, 1)

	go func() {
		defer close(resultChan)

		result, err := t.Execute(ctx, input)
		if err != nil {
			resultChan <- tools.ToolResult{
				Success: false,
				Error:   err.Error(),
			}
			return
		}

		resultChan <- tools.ToolResult{
			Success: true,
			Data:    result,
		}
	}()

	return resultChan, nil
}
