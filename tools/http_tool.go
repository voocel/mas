package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPResponse represents the result of an HTTP request
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// NewHTTPTool creates a new HTTP request tool
func NewHTTPTool() Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "The URL to make the request to"
			},
			"method": {
				"type": "string",
				"enum": ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"],
				"default": "GET",
				"description": "The HTTP method to use"
			},
			"headers": {
				"type": "object",
				"additionalProperties": {
					"type": "string"
				},
				"description": "HTTP headers to include in the request"
			},
			"body": {
				"type": "object",
				"description": "The request body (for POST, PUT, PATCH requests)"
			},
			"timeout": {
				"type": "integer",
				"minimum": 1,
				"maximum": 60,
				"default": 30,
				"description": "Request timeout in seconds"
			}
		},
		"required": ["url"],
		"additionalProperties": false
	}`)

	return NewTool(
		"http",
		"Make an HTTP request to a specified URL",
		schema,
		executeHTTPRequest,
	)
}

// executeHTTPRequest handles HTTP request execution
func executeHTTPRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract required parameters
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("%w: url is required", ErrInvalidParameters)
	}

	// Extract optional parameters with defaults
	method := "GET"
	if m, ok := params["method"].(string); ok && m != "" {
		method = m
	}

	timeout := 30
	if t, ok := params["timeout"].(float64); ok && t > 0 && t <= 60 {
		timeout = int(t)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Prepare request body if needed
	var reqBody io.Reader
	if body, ok := params["body"]; ok && (method == "POST" || method == "PUT" || method == "PATCH") {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid body: %s", ErrInvalidParameters, err.Error())
		}
		reqBody = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExecutionFailed, err.Error())
	}

	// Set default headers
	req.Header.Set("User-Agent", "MAS-Agent/1.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	// Add custom headers if provided
	if headers, ok := params["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrExecutionFailed, err.Error())
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %s", ErrExecutionFailed, err.Error())
	}

	// Extract headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Return formatted response
	return HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(body),
	}, nil
}
