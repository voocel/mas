package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/voocel/mas/tools"
)

// FetchTool fetches and processes web content.
type FetchTool struct {
	*tools.BaseTool
	client      *http.Client
	maxBodySize int64
}

type FetchRequest struct {
	URL     string `json:"url" description:"URL to fetch content from"`
	Format  string `json:"format" description:"Output format: text, markdown, or html"`
	Timeout int    `json:"timeout,omitempty" description:"Timeout in seconds (max 120)"`
}

type FetchResponse struct {
	Success   bool   `json:"success"`
	Content   string `json:"content"`
	URL       string `json:"url"`
	Format    string `json:"format"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
}

func NewFetchTool(maxBodySize int64) *FetchTool {
	if maxBodySize <= 0 {
		maxBodySize = 5 * 1024 * 1024 // Default: 5MB.
	}

	schema := tools.CreateToolSchema(
		"Fetch and process content from URLs with format conversion support",
		map[string]interface{}{
			"url": tools.StringProperty("The URL to fetch content from"),
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Output format: text (plain text), markdown (converted from HTML), or html (raw HTML body)",
				"enum":        []string{"text", "markdown", "html"},
			},
			"timeout": tools.NumberProperty("Optional timeout in seconds (max 120, default 30)"),
		},
		[]string{"url", "format"},
	)

	baseTool := tools.NewBaseTool("fetch", "Fetch and process content from URLs", schema).
		WithCapabilities(tools.CapabilityNetwork)

	return &FetchTool{
		BaseTool: baseTool,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		maxBodySize: maxBodySize,
	}
}

// Execute performs the fetch.
func (t *FetchTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var req FetchRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return t.errorResponse("Failed to parse fetch parameters: " + err.Error())
	}

	if req.URL == "" {
		return t.errorResponse("URL parameter is required")
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		return t.errorResponse("URL must start with http:// or https://")
	}

	format := strings.ToLower(req.Format)
	if format != "text" && format != "markdown" && format != "html" {
		return t.errorResponse("Format must be one of: text, markdown, html")
	}

	reqCtx := ctx
	if req.Timeout > 0 {
		maxTimeout := 120 // 2 minutes.
		if req.Timeout > maxTimeout {
			req.Timeout = maxTimeout
		}
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
		defer cancel()
	}

	httpReq, err := http.NewRequestWithContext(reqCtx, "GET", req.URL, nil)
	if err != nil {
		return t.errorResponse(fmt.Sprintf("Failed to create request: %v", err))
	}

	httpReq.Header.Set("User-Agent", "MAS-Fetch/1.0")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return t.errorResponse(fmt.Sprintf("Failed to fetch URL: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return t.errorResponse(fmt.Sprintf("Request failed with status code: %d", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, t.maxBodySize))
	if err != nil {
		return t.errorResponse(fmt.Sprintf("Failed to read response body: %v", err))
	}

	content := string(body)

	if !utf8.ValidString(content) {
		return t.errorResponse("Response content is not valid UTF-8")
	}

	contentType := resp.Header.Get("Content-Type")
	truncated := false

	switch format {
	case "text":
		if strings.Contains(contentType, "text/html") {
			text, err := extractTextFromHTML(content)
			if err != nil {
				return t.errorResponse(fmt.Sprintf("Failed to extract text from HTML: %v", err))
			}
			content = text
		}

	case "markdown":
		if strings.Contains(contentType, "text/html") {
			markdown, err := convertHTMLToMarkdown(content)
			if err != nil {
				return t.errorResponse(fmt.Sprintf("Failed to convert HTML to Markdown: %v", err))
			}
			content = markdown
		}
		// Return markdown as-is, not wrapped in code blocks

	case "html":
		if strings.Contains(contentType, "text/html") {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
			if err != nil {
				return t.errorResponse(fmt.Sprintf("Failed to parse HTML: %v", err))
			}
			body, err := doc.Find("body").Html()
			if err != nil {
				return t.errorResponse(fmt.Sprintf("Failed to extract body from HTML: %v", err))
			}
			if body == "" {
				return t.errorResponse("No body content found in HTML")
			}
			content = "<html>\n<body>\n" + body + "\n</body>\n</html>"
		}
	}

	contentSize := int64(len(content))
	if contentSize > t.maxBodySize {
		content = content[:t.maxBodySize]
		content += fmt.Sprintf("\n\n[Content truncated to %d bytes]", t.maxBodySize)
		truncated = true
	}

	resp2 := FetchResponse{
		Success:   true,
		Content:   content,
		URL:       req.URL,
		Format:    format,
		Size:      contentSize,
		Truncated: truncated,
	}

	return json.Marshal(resp2)
}

func (t *FetchTool) errorResponse(errorMsg string) (json.RawMessage, error) {
	resp := FetchResponse{
		Success: false,
		Error:   errorMsg,
	}
	return json.Marshal(resp)
}

// ExecuteAsync executes asynchronously.
func (t *FetchTool) ExecuteAsync(ctx context.Context, input json.RawMessage) (<-chan tools.ToolResult, error) {
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

func extractTextFromHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	text := doc.Find("body").Text()
	text = strings.Join(strings.Fields(text), " ")

	return text, nil
}

func convertHTMLToMarkdown(html string) (string, error) {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", err
	}
	return markdown, nil
}
