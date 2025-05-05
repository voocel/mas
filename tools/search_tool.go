package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SearchResponse represents search results
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// SearchResult represents a single search result item
type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

// NewSearchTool creates a new search tool
func NewSearchTool() Tool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query string"
			},
			"limit": {
				"type": "integer",
				"minimum": 1,
				"maximum": 10,
				"default": 5,
				"description": "Maximum number of results to return"
			}
		},
		"required": ["query"],
		"additionalProperties": false
	}`)

	return NewTool(
		"search",
		"Search for information on the internet",
		schema,
		executeSearch,
	)
}

// executeSearch handles search execution
func executeSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract required parameters
	query, ok := params["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("%w: query parameter must be provided and cannot be empty", ErrInvalidParameters)
	}

	// Extract optional parameters
	limit := 5
	if l, ok := params["limit"].(float64); ok && l > 0 && l <= 10 {
		limit = int(l)
	}

	// TODO: Implement actual search functionality, possibly calling a third-party search API
	// Return mock data for now
	results := make([]SearchResult, 0, limit)
	for i := 0; i < limit; i++ {
		results = append(results, SearchResult{
			Title:   fmt.Sprintf("Search Result %d Title", i+1),
			Snippet: fmt.Sprintf("This is a summary of search result %d about %s...", i+1, query),
			URL:     fmt.Sprintf("https://example.com/result/%d", i+1),
		})
	}

	return SearchResponse{
		Results: results,
	}, nil
} 