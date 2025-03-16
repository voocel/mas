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

type HTTPRequestParams struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

type HTTPResponseResult struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

func NewHTTPRequestTool() Tool {
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

	return NewBaseTool(
		"http_request",
		"Make an HTTP request to a specified URL",
		schema,
		executeHTTPRequest,
	)
}

func executeHTTPRequest(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var requestParams HTTPRequestParams
	if err := json.Unmarshal(params, &requestParams); err != nil {
		return nil, ErrInvalidParameters.WithDetails(err.Error())
	}

	if requestParams.URL == "" {
		return nil, ErrInvalidParameters.WithDetails("URL is required")
	}
	if requestParams.Method == "" {
		requestParams.Method = "GET"
	}
	timeout := 30
	if requestParams.Timeout > 0 && requestParams.Timeout <= 60 {
		timeout = requestParams.Timeout
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	var reqBody io.Reader
	if requestParams.Body != nil && (requestParams.Method == "POST" || requestParams.Method == "PUT" || requestParams.Method == "PATCH") {
		bodyBytes, err := json.Marshal(requestParams.Body)
		if err != nil {
			return nil, ErrInvalidParameters.WithDetails(fmt.Sprintf("Invalid body: %s", err.Error()))
		}
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, requestParams.Method, requestParams.URL, reqBody)
	if err != nil {
		return nil, ErrExecutionFailed.WithDetails(err.Error())
	}

	req.Header.Set("User-Agent", "Agent-Framework/1.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	for key, value := range requestParams.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrExecutionFailed.WithDetails(err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrExecutionFailed.WithDetails(fmt.Sprintf("Failed to read response body: %s", err.Error()))
	}

	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	result := HTTPResponseResult{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(body),
	}

	return result, nil
}
