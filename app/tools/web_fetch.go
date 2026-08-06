//go:build windows || darwin

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/auth"
)

type WebFetch struct{}

type FetchRequest struct {
	URL string `json:"url"`
}

type FetchResponse struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Links   []string `json:"links"`
}

// webFetchTool is the definition the model receives. See webSearchTool for why
// it is declared here rather than as a JSON schema string.
var webFetchTool = api.ToolFunction{
	Name:        "web_fetch",
	Description: "Crawl and extract text content from web pages",
	Parameters: api.ToolFunctionParameters{
		Type:     "object",
		Required: []string{"url"},
		Properties: toolProperties([]namedProperty{
			{"url", api.ToolProperty{
				Type:        api.PropertyType{"string"},
				Description: "URL to crawl and extract content from",
			}},
		}),
	},
}

func (w *WebFetch) Name() string {
	return webFetchTool.Name
}

func (w *WebFetch) Description() string {
	return webFetchTool.Description
}

func (w *WebFetch) ToolFunction() api.ToolFunction {
	return webFetchTool
}

func (w *WebFetch) Schema() map[string]any {
	return schemaOf(webFetchTool)
}

func (w *WebFetch) Prompt() string {
	return ""
}

func (w *WebFetch) Execute(ctx context.Context, args map[string]any) (any, string, error) {
	urlRaw, ok := args["url"]
	if !ok {
		return nil, "", fmt.Errorf("url parameter is required")
	}
	urlStr, ok := urlRaw.(string)
	if !ok || strings.TrimSpace(urlStr) == "" {
		return nil, "", fmt.Errorf("url must be a non-empty string")
	}
	if !allowedDirectURL(ctx, urlStr) {
		return nil, "", fmt.Errorf("web fetch is only allowed for URLs provided by the user")
	}

	result, err := performWebFetch(ctx, urlStr)
	if err != nil {
		return nil, "", err
	}
	for _, link := range result.Links {
		addAllowedDirectURL(ctx, link)
	}

	return result, "", nil
}

func performWebFetch(ctx context.Context, targetURL string) (*FetchResponse, error) {
	if err := ensureCloudEnabledForTool(ctx, "web fetch is unavailable"); err != nil {
		return nil, err
	}

	reqBody := FetchRequest{URL: targetURL}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	crawlURL, err := url.Parse("https://ollama.com/api/web_fetch")
	if err != nil {
		return nil, fmt.Errorf("failed to parse fetch URL: %w", err)
	}

	query := crawlURL.Query()
	query.Add("ts", strconv.FormatInt(time.Now().Unix(), 10))
	crawlURL.RawQuery = query.Encode()

	data := fmt.Appendf(nil, "%s,%s", http.MethodPost, crawlURL.RequestURI())
	signature, err := auth.Sign(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, crawlURL.String(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signature))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute fetch request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch API error (status %d)", resp.StatusCode)
	}

	var result FetchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
