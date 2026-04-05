package agents

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TextRadarSource wraps free-form text as one ranking entry.
type TextRadarSource struct {
	Name string
	Text string
}

// NewTextRadarSource 创建a text radar source。
func NewTextRadarSource(text, name string) *TextRadarSource {
	if strings.TrimSpace(name) == "" {
		name = "external"
	}
	return &TextRadarSource{Name: name, Text: text}
}

// Fetch 返回one synthetic ranking entry。
func (s *TextRadarSource) Fetch(ctx context.Context) (*PlatformRankings, error) {
	_ = ctx
	return &PlatformRankings{
		Platform: s.Name,
		Entries: []RankingEntry{
			{Title: s.Text, Author: "", Category: "", Extra: "[external-analysis]"},
		},
	}, nil
}

var fanqieRankTypes = []struct {
	SideType int
	Label    string
}{
	{SideType: 10, Label: "hot"},
	{SideType: 13, Label: "rising"},
}

// FanqieRadarSource 拉取rankings from fanqie public endpoints。
type FanqieRadarSource struct {
	Client *http.Client
}

// Fetch pulls ranking entries from fanqie endpoint.
func (s *FanqieRadarSource) Fetch(ctx context.Context) (*PlatformRankings, error) {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	entries := []RankingEntry{}
	for _, rank := range fanqieRankTypes {
		url := "https://api-lf.fanqiesdk.com/api/novel/channel/homepage/rank/rank_list/v2/?aid=13&limit=15&offset=0&side_type=" + strconvI(rank.SideType)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; iPen/0.1)")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}

		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			continue
		}
		data, _ := parsed["data"].(map[string]any)
		result, _ := data["result"].([]any)
		for _, item := range result {
			rec, _ := item.(map[string]any)
			entries = append(entries, RankingEntry{
				Title:    anyString(rec["book_name"]),
				Author:   anyString(rec["author"]),
				Category: anyString(rec["category"]),
				Extra:    "[" + rank.Label + "]",
			})
		}
	}

	return &PlatformRankings{Platform: "fanqie", Entries: entries}, nil
}

// QidianRadarSource 拉取simple top titles from qidian rank page HTML。
type QidianRadarSource struct {
	Client *http.Client
}

// Fetch scrapes title anchors from qidian ranking page.
func (s *QidianRadarSource) Fetch(ctx context.Context) (*PlatformRankings, error) {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.qidian.com/rank/", nil)
	if err != nil {
		return &PlatformRankings{Platform: "qidian", Entries: []RankingEntry{}}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return &PlatformRankings{Platform: "qidian", Entries: []RankingEntry{}}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &PlatformRankings{Platform: "qidian", Entries: []RankingEntry{}}, nil
	}

	htmlBytes, _ := io.ReadAll(resp.Body)
	html := string(htmlBytes)

	pattern := regexp.MustCompile(`<a[^>]*href="//book\.qidian\.com/info/\d+"[^>]*>([^<]+)</a>`)
	matches := pattern.FindAllStringSubmatch(html, -1)
	seen := map[string]struct{}{}
	entries := []RankingEntry{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		title := strings.TrimSpace(match[1])
		if title == "" || len(title) < 2 || len(title) > 80 {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		entries = append(entries, RankingEntry{Title: title, Extra: "[qidian-rank]"})
		if len(entries) >= 20 {
			break
		}
	}

	return &PlatformRankings{Platform: "qidian", Entries: entries}, nil
}

func anyString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func strconvI(v int) string {
	return strconv.Itoa(v)
}
