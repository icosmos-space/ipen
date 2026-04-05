package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// SearchResult 是a web-search result row。
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// SearchWeb 搜索via Tavily API (requires TAVILY_API_KEY)。
func SearchWeb(query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 5
	}
	apiKey := strings.TrimSpace(os.Getenv("TAVILY_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("TAVILY_API_KEY not set")
	}

	payload := map[string]any{
		"api_key":      apiKey,
		"query":        query,
		"max_results":  maxResults,
		"search_depth": "basic",
	}
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	responseBody, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("tavily search failed: %d %s", res.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, err
	}

	result := make([]SearchResult, 0, len(decoded.Results))
	for _, row := range decoded.Results {
		result = append(result, SearchResult{Title: row.Title, URL: row.URL, Snippet: row.Content})
	}
	return result, nil
}

// FetchURL 拉取URL content and returns text/plain-ish body up to maxChars。
func FetchURL(url string, maxChars int) (string, error) {
	if maxChars <= 0 {
		maxChars = 8000
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html, application/json, text/plain")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("fetch failed: %d %s", res.StatusCode, res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	text := string(body)
	contentType := strings.ToLower(res.Header.Get("content-type"))
	if strings.Contains(contentType, "html") {
		text = regexp.MustCompile(`(?is)<script[\s\S]*?</script>`).ReplaceAllString(text, " ")
		text = regexp.MustCompile(`(?is)<style[\s\S]*?</style>`).ReplaceAllString(text, " ")
		text = regexp.MustCompile(`(?is)<[^>]*>`).ReplaceAllString(text, " ")
		text = strings.Join(strings.Fields(text), " ")
	}

	runes := []rune(text)
	if len(runes) > maxChars {
		text = string(runes[:maxChars])
	}
	return text, nil
}
