package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/voocel/mas"
)

// HTTPRequest creates an HTTP request tool
func HTTPRequest() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"url":     mas.StringProperty("The URL to send the request to"),
			"method":  mas.EnumProperty("HTTP method", []string{"GET", "POST", "PUT", "DELETE", "PATCH"}),
			"headers": mas.StringProperty("JSON string of headers (optional)"),
			"body":    mas.StringProperty("Request body for POST/PUT requests (optional)"),
			"timeout": mas.NumberProperty("Request timeout in seconds (default: 30)"),
		},
		Required: []string{"url", "method"},
	}

	return mas.NewTool(
		"http_request",
		"Makes HTTP requests to web services and APIs",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			url, ok := params["url"].(string)
			if !ok {
				return nil, fmt.Errorf("url parameter is required")
			}

			method, ok := params["method"].(string)
			if !ok {
				return nil, fmt.Errorf("method parameter is required")
			}

			method = strings.ToUpper(method)

			// Create HTTP client with timeout
			timeout := 30 * time.Second
			if timeoutVal, exists := params["timeout"]; exists {
				if timeoutSec, ok := timeoutVal.(float64); ok {
					timeout = time.Duration(timeoutSec) * time.Second
				}
			}

			client := &http.Client{
				Timeout: timeout,
			}

			// Prepare request body
			var bodyReader io.Reader
			if bodyStr, exists := params["body"]; exists && bodyStr != "" {
				if body, ok := bodyStr.(string); ok {
					bodyReader = strings.NewReader(body)
				}
			}

			// Create request
			req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %v", err)
			}

			// Add headers
			if headersStr, exists := params["headers"]; exists && headersStr != "" {
				if headers, ok := headersStr.(string); ok {
					var headerMap map[string]string
					err := json.Unmarshal([]byte(headers), &headerMap)
					if err != nil {
						return nil, fmt.Errorf("invalid headers JSON: %v", err)
					}

					for key, value := range headerMap {
						req.Header.Set(key, value)
					}
				}
			}

			// Set default content type for POST/PUT requests
			if (method == "POST" || method == "PUT" || method == "PATCH") && req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/json")
			}

			// Make the request
			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("request failed: %v", err)
			}
			defer resp.Body.Close()

			// Read response body
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %v", err)
			}

			// Parse response headers
			responseHeaders := make(map[string]string)
			for key, values := range resp.Header {
				if len(values) > 0 {
					responseHeaders[key] = values[0]
				}
			}

			return map[string]interface{}{
				"url":     url,
				"method":  method,
				"status":  resp.StatusCode,
				"headers": responseHeaders,
				"body":    string(bodyBytes),
				"success": resp.StatusCode >= 200 && resp.StatusCode < 300,
			}, nil
		},
	)
}

// WebScraper creates a simple web scraping tool
func WebScraper() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"url":      mas.StringProperty("The URL to scrape"),
			"selector": mas.StringProperty("CSS selector to extract specific content (optional)"),
			"timeout":  mas.NumberProperty("Request timeout in seconds (default: 30)"),
		},
		Required: []string{"url"},
	}

	return mas.NewTool(
		"web_scraper",
		"Scrapes web pages and extracts content",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			url, ok := params["url"].(string)
			if !ok {
				return nil, fmt.Errorf("url parameter is required")
			}

			// Create HTTP client with timeout
			timeout := 30 * time.Second
			if timeoutVal, exists := params["timeout"]; exists {
				if timeoutSec, ok := timeoutVal.(float64); ok {
					timeout = time.Duration(timeoutSec) * time.Second
				}
			}

			client := &http.Client{
				Timeout: timeout,
			}

			// Make request
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %v", err)
			}

			// Set user agent to avoid being blocked
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MAS-Bot/1.0)")

			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			}

			// Read content
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response: %v", err)
			}

			content := string(bodyBytes)

			// Basic content extraction (just return the HTML for now)
			// In a real implementation, you might use a proper HTML parser
			// to extract specific elements based on CSS selectors

			// Extract title
			title := extractTitle(content)

			return map[string]interface{}{
				"url":     url,
				"title":   title,
				"content": truncateContent(content, 5000), // Limit content size
				"length":  len(content),
				"status":  resp.StatusCode,
			}, nil
		},
	)
}

// JSONParser creates a JSON parsing and manipulation tool
func JSONParser() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"json":      mas.StringProperty("JSON string to parse"),
			"path":      mas.StringProperty("JSONPath expression to extract specific data (optional)"),
			"operation": mas.EnumProperty("Operation to perform", []string{"parse", "validate", "extract", "format"}),
		},
		Required: []string{"json", "operation"},
	}

	return mas.NewTool(
		"json_parser",
		"Parses, validates, and manipulates JSON data",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			jsonStr, ok := params["json"].(string)
			if !ok {
				return nil, fmt.Errorf("json parameter is required")
			}

			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter is required")
			}

			switch operation {
			case "parse", "validate":
				var data interface{}
				err := json.Unmarshal([]byte(jsonStr), &data)
				if err != nil {
					return map[string]interface{}{
						"valid":  false,
						"error":  err.Error(),
						"result": nil,
					}, nil
				}

				return map[string]interface{}{
					"valid":  true,
					"error":  nil,
					"result": data,
				}, nil

			case "format":
				var data interface{}
				err := json.Unmarshal([]byte(jsonStr), &data)
				if err != nil {
					return nil, fmt.Errorf("invalid JSON: %v", err)
				}

				formatted, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to format JSON: %v", err)
				}

				return map[string]interface{}{
					"formatted": string(formatted),
					"original":  jsonStr,
				}, nil

			case "extract":
				path, exists := params["path"].(string)
				if !exists {
					return nil, fmt.Errorf("path parameter is required for extract operation")
				}

				var data interface{}
				err := json.Unmarshal([]byte(jsonStr), &data)
				if err != nil {
					return nil, fmt.Errorf("invalid JSON: %v", err)
				}

				// Simple path extraction (just basic dot notation)
				result := extractJSONPath(data, path)

				return map[string]interface{}{
					"path":   path,
					"result": result,
					"found":  result != nil,
				}, nil

			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}
		},
	)
}

// Helper functions

// extractTitle extracts the title from HTML content
func extractTitle(html string) string {
	// Simple title extraction
	start := strings.Index(strings.ToLower(html), "<title>")
	if start == -1 {
		return ""
	}
	start += 7 // len("<title>")

	end := strings.Index(strings.ToLower(html[start:]), "</title>")
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(html[start : start+end])
}

// truncateContent truncates content to a maximum length
func truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "... (truncated)"
}

// extractJSONPath extracts data from JSON using a simple dot notation path
func extractJSONPath(data interface{}, path string) interface{} {
	if path == "" || path == "." {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case []interface{}:
			// Handle array access (simplified)
			if part == "0" && len(v) > 0 {
				current = v[0]
			} else {
				return nil
			}
		default:
			return nil
		}

		if current == nil {
			return nil
		}
	}

	return current
}