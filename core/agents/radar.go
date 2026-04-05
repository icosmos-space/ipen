package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/llm"
)

// RadarResult 表示radar result。
type RadarResult struct {
	Recommendations []RadarRecommendation `json:"recommendations"`
	MarketSummary   string                `json:"marketSummary"`
	Timestamp       string                `json:"timestamp"`
}

// RadarRecommendation 表示radar recommendation。
type RadarRecommendation struct {
	Platform        string   `json:"platform"`
	Genre           string   `json:"genre"`
	Concept         string   `json:"concept"`
	Confidence      float64  `json:"confidence"`
	Reasoning       string   `json:"reasoning"`
	BenchmarkTitles []string `json:"benchmarkTitles"`
}

// RadarAgent 表示the radar agent。
type RadarAgent struct {
	*BaseAgent
	Sources []RadarSource
}

// RadarSource 表示a radar source。
type RadarSource interface {
	Fetch(ctx context.Context) (*PlatformRankings, error)
}

// PlatformRankings 表示platform rankings。
type PlatformRankings struct {
	Platform string         `json:"platform"`
	Entries  []RankingEntry `json:"entries"`
}

// RankingEntry 表示a ranking entry。
type RankingEntry struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Category string `json:"category"`
	Extra    string `json:"extra"`
}

// NewRadarAgent 创建新的radar agent。
func NewRadarAgent(ctx AgentContext, sources ...RadarSource) *RadarAgent {
	if len(sources) == 0 {
		sources = []RadarSource{}
	}

	return &RadarAgent{
		BaseAgent: NewBaseAgent(ctx),
		Sources:   sources,
	}
}

// Name 返回the agent name。
func (r *RadarAgent) Name() string {
	return "radar"
}

// Scan 执行a radar scan。
func (r *RadarAgent) Scan(ctx context.Context) (*RadarResult, error) {
	var allRankings []PlatformRankings
	for _, source := range r.Sources {
		rankings, err := source.Fetch(ctx)
		if err != nil {
			r.Log().Warn("failed to fetch rankings", map[string]any{"error": err.Error()})
			continue
		}
		allRankings = append(allRankings, *rankings)
	}

	rankingsText := formatRankings(allRankings)
	systemPrompt := r.buildRadarSystemPrompt(rankingsText)
	userPrompt := "请基于实时榜单分析当前网文市场热度，并给出开书建议。"

	messages := []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := r.Chat(ctx, messages, &llm.ChatOptions{
		Temperature: 0.6,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("radar chat failed: %w", err)
	}

	return r.parseRadarResult(response.Content), nil
}

func (r *RadarAgent) buildRadarSystemPrompt(rankingsText string) string {
	return fmt.Sprintf(`你是专业的网络小说市场分析师。以下是实时榜单数据，请基于这些数据进行分析。
## 实时榜单数据
%s

请输出 JSON：{
  "recommendations": [
    {
      "platform": "平台名",
      "genre": "题材",
      "concept": "一句话概念",
      "confidence": 0.0,
      "reasoning": "推荐依据",
      "benchmarkTitles": ["对标作品1", "对标作品2"]
    }
  ],
  "marketSummary": "市场总结"
}`,
		rankingsText,
	)
}

func (r *RadarAgent) parseRadarResult(content string) *RadarResult {
	result := &RadarResult{
		Recommendations: []RadarRecommendation{},
		MarketSummary:   strings.TrimSpace(content),
		Timestamp:       time.Now().Format(time.RFC3339),
	}

	jsonBlock := extractFirstJSONObject(content)
	if jsonBlock == "" {
		if result.MarketSummary == "" {
			result.MarketSummary = "未返回可解析的 JSON 结果。"
		}
		return result
	}

	type radarPayload struct {
		Recommendations []RadarRecommendation `json:"recommendations"`
		MarketSummary   string                `json:"marketSummary"`
	}

	var payload radarPayload
	if err := json.Unmarshal([]byte(jsonBlock), &payload); err != nil {
		if result.MarketSummary == "" {
			result.MarketSummary = "雷达结果 JSON 解析失败。"
		}
		return result
	}

	if payload.Recommendations != nil {
		for _, recommendation := range payload.Recommendations {
			if recommendation.Confidence < 0 {
				recommendation.Confidence = 0
			}
			if recommendation.Confidence > 1 {
				recommendation.Confidence = 1
			}
			if recommendation.BenchmarkTitles == nil {
				recommendation.BenchmarkTitles = []string{}
			}
			result.Recommendations = append(result.Recommendations, recommendation)
		}
	}

	if strings.TrimSpace(payload.MarketSummary) != "" {
		result.MarketSummary = strings.TrimSpace(payload.MarketSummary)
	}

	if result.MarketSummary == "" {
		result.MarketSummary = "市场分析完成。"
	}

	return result
}

func formatRankings(rankings []PlatformRankings) string {
	var sections []string
	for _, ranking := range rankings {
		if len(ranking.Entries) == 0 {
			continue
		}

		var lines []string
		for _, entry := range ranking.Entries {
			line := fmt.Sprintf("- %s", entry.Title)
			if entry.Author != "" {
				line += fmt.Sprintf(" (%s)", entry.Author)
			}
			if entry.Category != "" {
				line += fmt.Sprintf(" [%s]", entry.Category)
			}
			if strings.TrimSpace(entry.Extra) != "" {
				line += " " + strings.TrimSpace(entry.Extra)
			}
			lines = append(lines, line)
		}

		sections = append(sections, fmt.Sprintf("### %s\n%s", ranking.Platform, strings.Join(lines, "\n")))
	}

	if len(sections) == 0 {
		return "（未获取到实时排行数据）"
	}

	return strings.Join(sections, "\n\n")
}
