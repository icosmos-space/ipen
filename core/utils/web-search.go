package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// 预编译正则表达式，提高性能
var (
	reScriptTag  = regexp.MustCompile(`(?is)<script[\s\S]*?</script>`)
	reStyleTag   = regexp.MustCompile(`(?is)<style[\s\S]*?</style>`)
	reHTMLTags   = regexp.MustCompile(`(?is)<[^>]*>`)
	reMultiSpace = regexp.MustCompile(`\s+`)
)

// 常量定义
const (
	defaultMaxResults   = 5
	defaultMaxChars     = 8000
	httpTimeout         = 15 * time.Second
	maxResponseBodySize = 10 * 1024 * 1024 // 10MB 限制
	tavilyAPIURL        = "https://api.tavily.com/search"
	userAgent           = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
)

// SearchResult 是a web-search result row。
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// SearchWeb 搜索via Tavily API (requires TAVILY_API_KEY)。
func SearchWeb(query string, maxResults int) ([]SearchResult, error) {
	// 验证 query 参数
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	// 设置默认的 maxResults
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}

	// 获取并验证 API Key
	apiKey := strings.TrimSpace(os.Getenv("TAVILY_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("TAVILY_API_KEY not set")
	}

	// 构建请求 payload
	payload := map[string]any{
		"api_key":      apiKey,
		"query":        query,
		"max_results":  maxResults,
		"search_depth": "basic",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// 创建 HTTP 客户端
	client := &http.Client{Timeout: httpTimeout}

	// 创建请求
	req, err := http.NewRequest(http.MethodPost, tavilyAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	// 读取响应体，限制最大大小
	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查 HTTP 状态码
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		errMsg := strings.TrimSpace(string(responseBody))
		if len(errMsg) > 500 {
			errMsg = errMsg[:500] + "..."
		}
		return nil, fmt.Errorf("tavily search failed with status %d: %s", res.StatusCode, errMsg)
	}

	// 解析响应
	var decoded struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 构建结果
	result := make([]SearchResult, 0, len(decoded.Results))
	for _, row := range decoded.Results {
		result = append(result, SearchResult{
			Title:   strings.TrimSpace(row.Title),
			URL:     strings.TrimSpace(row.URL),
			Snippet: strings.TrimSpace(row.Content),
		})
	}
	return result, nil
}

// FetchURL 拉取URL content and returns text/plain-ish body up to maxChars。
func FetchURL(urlStr string, maxChars int) (string, error) {
	// 验证 URL 参数
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return "", errors.New("url cannot be empty")
	}

	// 验证 URL 格式
	if _, err := url.ParseRequestURI(urlStr); err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// 设置默认的 maxChars
	if maxChars <= 0 {
		maxChars = defaultMaxChars
	}

	// 创建 HTTP 客户端
	client := &http.Client{Timeout: httpTimeout}

	// 创建请求
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html, application/json, text/plain, */*")

	// 发送请求
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	// 检查 HTTP 状态码
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("fetch failed with status %d: %s", res.StatusCode, res.Status)
	}

	// 读取响应体，限制最大大小
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBodySize))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// 处理内容
	text := string(body)
	contentType := strings.ToLower(res.Header.Get("content-type"))
	if strings.Contains(contentType, "html") {
		text = cleanHTML(text)
	}

	// 截断到指定字符数
	runes := []rune(text)
	if len(runes) > maxChars {
		text = string(runes[:maxChars])
	}
	return text, nil
}

// cleanHTML 清理 HTML 内容，移除脚本、样式和标签
func cleanHTML(html string) string {
	// 移除 script 标签
	text := reScriptTag.ReplaceAllString(html, " ")
	// 移除 style 标签
	text = reStyleTag.ReplaceAllString(text, " ")
	// 移除所有 HTML 标签
	text = reHTMLTags.ReplaceAllString(text, " ")
	// 合并多余空白字符
	text = strings.Join(strings.Fields(text), " ")
	return text
}
