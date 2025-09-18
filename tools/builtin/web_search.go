package builtin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// WebSearchTool performs web searches
type WebSearchTool struct {
	*tools.BaseTool
	apiKey         string
	searchURL      string
	client         *http.Client
	searchEngineID string
	provider       string
}

// SearchInput describes search parameters
type SearchInput struct {
	Query      string `json:"query" description:"Search query"`
	MaxResults int    `json:"max_results,omitempty" description:"Maximum number of results (default 10)"`
	Language   string `json:"language,omitempty" description:"Language code (e.g., zh-CN, en)"`
}

// SearchResult captures a single search hit
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source,omitempty"`
}

// SearchOutput wraps the search response
type SearchOutput struct {
	Success bool           `json:"success"`
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Message string         `json:"message,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// NewWebSearchTool constructs a web search tool
func NewWebSearchTool(apiKey string) *WebSearchTool {
	schema := tools.CreateToolSchema(
		"Web search tool for retrieving information from the internet",
		map[string]interface{}{
			"query":       tools.StringProperty("Search query"),
			"max_results": tools.NumberProperty("Maximum number of results (default 10)"),
			"language":    tools.StringProperty("Language code (e.g., zh-CN, en)"),
		},
		[]string{"query"},
	)

	baseTool := tools.NewBaseTool("web_search", "Web search tool for retrieving information from the internet", schema)

	return &WebSearchTool{
		BaseTool:  baseTool,
		apiKey:    apiKey,
		searchURL: "https://api.duckduckgo.com/", // Use DuckDuckGo as the default search engine
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewGoogleSearchTool constructs a Google search tool
func NewGoogleSearchTool(apiKey, searchEngineID string) *WebSearchTool {
	tool := NewWebSearchTool(apiKey)
	tool.searchURL = "https://www.googleapis.com/customsearch/v1"
	tool.searchEngineID = searchEngineID
	tool.provider = "google"
	return tool
}

// Execute performs a search
func (t *WebSearchTool) Execute(ctx runtime.Context, input json.RawMessage) (json.RawMessage, error) {
	var searchInput SearchInput
	if err := json.Unmarshal(input, &searchInput); err != nil {
		return nil, schema.NewToolError(t.Name(), "parse_input", err)
	}

	if searchInput.Query == "" {
		output := SearchOutput{
			Success: false,
			Error:   "search query cannot be empty",
		}
		return json.Marshal(output)
	}

	if searchInput.MaxResults <= 0 {
		searchInput.MaxResults = 10
	}
	if searchInput.MaxResults > 50 {
		searchInput.MaxResults = 50 // Cap the maximum number of results
	}

	// Select the search provider based on configuration
	switch t.provider {
	case "google":
		return t.searchGoogle(searchInput)
	default:
		return t.searchDuckDuckGo(searchInput)
	}
}

// searchDuckDuckGo queries DuckDuckGo
func (t *WebSearchTool) searchDuckDuckGo(input SearchInput) (json.RawMessage, error) {
	// DuckDuckGo Instant Answer API
	params := url.Values{}
	params.Set("q", input.Query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")

	searchURL := t.searchURL + "?" + params.Encode()

	resp, err := t.client.Get(searchURL)
	if err != nil {
		output := SearchOutput{
			Success: false,
			Query:   input.Query,
			Error:   fmt.Sprintf("search request failed: %v", err),
		}
		return json.Marshal(output)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		output := SearchOutput{
			Success: false,
			Query:   input.Query,
			Error:   fmt.Sprintf("failed to read response: %v", err),
		}
		return json.Marshal(output)
	}

	// Parse the DuckDuckGo response
	var ddgResponse map[string]interface{}
	if err := json.Unmarshal(body, &ddgResponse); err != nil {
		output := SearchOutput{
			Success: false,
			Query:   input.Query,
			Error:   fmt.Sprintf("failed to parse response: %v", err),
		}
		return json.Marshal(output)
	}

	results := t.parseDuckDuckGoResponse(ddgResponse, input.MaxResults)

	output := SearchOutput{
		Success: true,
		Query:   input.Query,
		Results: results,
		Total:   len(results),
		Message: fmt.Sprintf("found %d results for '%s'", len(results), input.Query),
	}

	return json.Marshal(output)
}

// searchGoogle queries the Google Custom Search API
func (t *WebSearchTool) searchGoogle(input SearchInput) (json.RawMessage, error) {
	if t.apiKey == "" {
		output := SearchOutput{
			Success: false,
			Query:   input.Query,
			Error:   "Google API key not configured",
		}
		return json.Marshal(output)
	}

	if t.searchEngineID == "" {
		output := SearchOutput{
			Success: false,
			Query:   input.Query,
			Error:   "Google search engine ID not configured",
		}
		return json.Marshal(output)
	}

	params := url.Values{}
	params.Set("key", t.apiKey)
	params.Set("cx", t.searchEngineID)
	params.Set("q", input.Query)
	params.Set("num", fmt.Sprintf("%d", input.MaxResults))
	if input.Language != "" {
		params.Set("lr", "lang_"+input.Language)
	}

	searchURL := t.searchURL + "?" + params.Encode()

	resp, err := t.client.Get(searchURL)
	if err != nil {
		output := SearchOutput{
			Success: false,
			Query:   input.Query,
			Error:   fmt.Sprintf("search request failed: %v", err),
		}
		return json.Marshal(output)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		output := SearchOutput{
			Success: false,
			Query:   input.Query,
			Error:   fmt.Sprintf("failed to read response: %v", err),
		}
		return json.Marshal(output)
	}

	// Parse the Google response
	var googleResponse map[string]interface{}
	if err := json.Unmarshal(body, &googleResponse); err != nil {
		output := SearchOutput{
			Success: false,
			Query:   input.Query,
			Error:   fmt.Sprintf("failed to parse response: %v", err),
		}
		return json.Marshal(output)
	}

	results := t.parseGoogleResponse(googleResponse)

	output := SearchOutput{
		Success: true,
		Query:   input.Query,
		Results: results,
		Total:   len(results),
		Message: fmt.Sprintf("found %d results for '%s'", len(results), input.Query),
	}

	return json.Marshal(output)
}

// parseDuckDuckGoResponse parses the DuckDuckGo response
func (t *WebSearchTool) parseDuckDuckGoResponse(response map[string]interface{}, maxResults int) []SearchResult {
	var results []SearchResult

	// Parse the Abstract field
	if abstract, ok := response["Abstract"].(string); ok && abstract != "" {
		if abstractURL, ok := response["AbstractURL"].(string); ok {
			results = append(results, SearchResult{
				Title:   "Abstract",
				URL:     abstractURL,
				Snippet: abstract,
				Source:  "DuckDuckGo",
			})
		}
	}

	// Parse RelatedTopics
	if relatedTopics, ok := response["RelatedTopics"].([]interface{}); ok {
		for _, topic := range relatedTopics {
			if topicMap, ok := topic.(map[string]interface{}); ok {
				if text, ok := topicMap["Text"].(string); ok && text != "" {
					if firstURL, ok := topicMap["FirstURL"].(string); ok {
						results = append(results, SearchResult{
							Title:   t.extractTitle(text),
							URL:     firstURL,
							Snippet: text,
							Source:  "DuckDuckGo",
						})
						if len(results) >= maxResults {
							break
						}
					}
				}
			}
		}
	}

	return results
}

// parseGoogleResponse parses the Google response
func (t *WebSearchTool) parseGoogleResponse(response map[string]interface{}) []SearchResult {
	var results []SearchResult

	if items, ok := response["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				title, _ := itemMap["title"].(string)
				link, _ := itemMap["link"].(string)
				snippet, _ := itemMap["snippet"].(string)

				results = append(results, SearchResult{
					Title:   title,
					URL:     link,
					Snippet: snippet,
					Source:  "Google",
				})
			}
		}
	}

	return results
}

// extractTitle derives a title from the text
func (t *WebSearchTool) extractTitle(text string) string {
	// Simple heuristic: use the first sentence or first 50 characters
	if len(text) > 50 {
		if idx := strings.Index(text, "."); idx > 0 && idx < 50 {
			return text[:idx]
		}
		return text[:50] + "..."
	}
	return text
}

// ExecuteAsync performs the search asynchronously
func (t *WebSearchTool) ExecuteAsync(ctx runtime.Context, input json.RawMessage) (<-chan tools.ToolResult, error) {
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
