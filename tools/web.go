package tools

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/voocel/mas"
)

// WebSearch creates a web search tool
func WebSearch() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"query":    mas.StringProperty("Search query"),
			"engine":   mas.EnumProperty("Search engine to use", []string{"google", "bing", "duckduckgo"}),
			"limit":    mas.NumberProperty("Maximum number of results (default: 10)"),
			"language": mas.StringProperty("Language code (e.g., 'en', 'zh', default: 'en')"),
		},
		Required: []string{"query"},
	}

	return mas.NewTool(
		"web_search",
		"Searches the web for information using various search engines",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			query, ok := params["query"].(string)
			if !ok {
				return nil, fmt.Errorf("query parameter is required")
			}

			engine := "google"
			if engineParam, exists := params["engine"]; exists {
				if e, ok := engineParam.(string); ok {
					engine = e
				}
			}

			limit := 10
			if limitParam, exists := params["limit"]; exists {
				if l, ok := limitParam.(float64); ok {
					limit = int(l)
				}
			}

			language := "en"
			if langParam, exists := params["language"]; exists {
				if l, ok := langParam.(string); ok {
					language = l
				}
			}

			// Since we can't actually call search APIs without API keys,
			// we'll simulate search results based on the query
			results := simulateSearchResults(query, engine, limit, language)

			return map[string]interface{}{
				"query":    query,
				"engine":   engine,
				"language": language,
				"count":    len(results),
				"results":  results,
			}, nil
		},
	)
}

// URLShortener creates a URL shortening tool
func URLShortener() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"url":    mas.StringProperty("URL to shorten"),
			"custom": mas.StringProperty("Custom short code (optional)"),
		},
		Required: []string{"url"},
	}

	return mas.NewTool(
		"url_shortener",
		"Shortens long URLs and provides analytics",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			longURL, ok := params["url"].(string)
			if !ok {
				return nil, fmt.Errorf("url parameter is required")
			}

			// Validate URL
			_, err := url.Parse(longURL)
			if err != nil {
				return nil, fmt.Errorf("invalid URL: %v", err)
			}

			custom := ""
			if customParam, exists := params["custom"]; exists {
				if c, ok := customParam.(string); ok {
					custom = c
				}
			}

			// Generate a mock short URL
			shortCode := custom
			if shortCode == "" {
				shortCode = generateShortCode(longURL)
			}

			shortURL := fmt.Sprintf("https://short.ly/%s", shortCode)

			return map[string]interface{}{
				"original_url": longURL,
				"short_url":    shortURL,
				"short_code":   shortCode,
				"created_at":   time.Now().Format(time.RFC3339),
				"clicks":       0,
			}, nil
		},
	)
}

// DomainInfo creates a domain information lookup tool
func DomainInfo() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"domain": mas.StringProperty("Domain name to lookup"),
			"info":   mas.EnumProperty("Type of information", []string{"whois", "dns", "ssl", "all"}),
		},
		Required: []string{"domain"},
	}

	return mas.NewTool(
		"domain_info",
		"Retrieves information about domain names including WHOIS, DNS, and SSL data",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			domain, ok := params["domain"].(string)
			if !ok {
				return nil, fmt.Errorf("domain parameter is required")
			}

			infoType := "all"
			if infoParam, exists := params["info"]; exists {
				if i, ok := infoParam.(string); ok {
					infoType = i
				}
			}

			// Clean domain name
			domain = strings.TrimSpace(domain)
			domain = strings.TrimPrefix(domain, "http://")
			domain = strings.TrimPrefix(domain, "https://")
			domain = strings.TrimPrefix(domain, "www.")

			// Mock domain information
			result := map[string]interface{}{
				"domain": domain,
				"query":  infoType,
			}

			if infoType == "whois" || infoType == "all" {
				result["whois"] = mockWhoisInfo(domain)
			}

			if infoType == "dns" || infoType == "all" {
				result["dns"] = mockDNSInfo(domain)
			}

			if infoType == "ssl" || infoType == "all" {
				result["ssl"] = mockSSLInfo(domain)
			}

			return result, nil
		},
	)
}

// Helper functions

// simulateSearchResults creates mock search results
func simulateSearchResults(query, engine string, limit int, language string) []map[string]interface{} {
	// This is a simulation - in a real implementation, you would:
	// 1. Call actual search APIs (Google Custom Search, Bing Search API, etc.)
	// 2. Handle API keys and rate limits
	// 3. Parse real search results

	baseResults := []map[string]interface{}{
		{
			"title":       fmt.Sprintf("Information about %s - Wikipedia", query),
			"url":         fmt.Sprintf("https://en.wikipedia.org/wiki/%s", url.QueryEscape(query)),
			"description": fmt.Sprintf("Learn about %s on Wikipedia, the free encyclopedia.", query),
			"engine":      engine,
		},
		{
			"title":       fmt.Sprintf("%s - Official Website", query),
			"url":         fmt.Sprintf("https://www.%s.com", strings.ToLower(strings.ReplaceAll(query, " ", ""))),
			"description": fmt.Sprintf("Official website for %s with comprehensive information and resources.", query),
			"engine":      engine,
		},
		{
			"title":       fmt.Sprintf("Latest news about %s", query),
			"url":         fmt.Sprintf("https://news.google.com/search?q=%s", url.QueryEscape(query)),
			"description": fmt.Sprintf("Stay updated with the latest news and developments about %s.", query),
			"engine":      engine,
		},
	}

	// Limit results
	if limit < len(baseResults) {
		baseResults = baseResults[:limit]
	}

	return baseResults
}

// generateShortCode generates a short code for URL shortening
func generateShortCode(url string) string {
	// Simple hash-based short code generation
	hash := 0
	for _, char := range url {
		hash = hash*31 + int(char)
		if hash < 0 {
			hash = -hash
		}
	}

	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := ""
	for i := 0; i < 6; i++ {
		code += string(chars[hash%len(chars)])
		hash = hash / len(chars)
	}

	return code
}

// mockWhoisInfo creates mock WHOIS information
func mockWhoisInfo(domain string) map[string]interface{} {
	return map[string]interface{}{
		"registrar":     "Mock Registrar Inc.",
		"creation_date": "2020-01-15",
		"expiry_date":   "2025-01-15",
		"status":        "active",
		"nameservers": []string{
			"ns1.example.com",
			"ns2.example.com",
		},
	}
}

// mockDNSInfo creates mock DNS information
func mockDNSInfo(domain string) map[string]interface{} {
	return map[string]interface{}{
		"a_records": []string{
			"192.168.1.1",
			"192.168.1.2",
		},
		"mx_records": []map[string]interface{}{
			{"priority": 10, "server": "mail1." + domain},
			{"priority": 20, "server": "mail2." + domain},
		},
		"ns_records": []string{
			"ns1." + domain,
			"ns2." + domain,
		},
		"txt_records": []string{
			"v=spf1 include:_spf.google.com ~all",
		},
	}
}

// mockSSLInfo creates mock SSL certificate information
func mockSSLInfo(domain string) map[string]interface{} {
	return map[string]interface{}{
		"valid":        true,
		"issuer":       "Let's Encrypt Authority X3",
		"subject":      domain,
		"valid_from":   "2024-01-15T00:00:00Z",
		"valid_until":  "2025-01-15T00:00:00Z",
		"fingerprint":  "AB:CD:EF:12:34:56:78:90",
		"key_size":     2048,
		"signature":    "SHA256withRSA",
	}
}