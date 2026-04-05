package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DetectionResult 表示the result of AI content detection。
type DetectionResult struct {
	Score      float64        `json:"score"` // 0-1, higher = more likely AI
	Provider   string         `json:"provider"`
	DetectedAt string         `json:"detectedAt"`
	Raw        map[string]any `json:"raw,omitempty"`
}

// DetectionConfig 表示detection configuration。
type DetectionConfig struct {
	Provider  string `json:"provider"` // "gptzero", "originality", "custom"
	APIURL    string `json:"apiUrl"`
	APIKeyEnv string `json:"apiKeyEnv"` // Environment variable name for API key
}

// DetectAIContent 检测AI-generated content by calling an external detection API。
func DetectAIContent(ctx context.Context, config DetectionConfig, content string) (*DetectionResult, error) {
	apiKey := os.Getenv(config.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("detection API key not found. Set %s in your environment", config.APIKeyEnv)
	}

	detectedAt := time.Now().Format(time.RFC3339)

	switch config.Provider {
	case "gptzero":
		return detectGPTZero(ctx, config.APIURL, apiKey, content, detectedAt)
	case "originality":
		return detectOriginality(ctx, config.APIURL, apiKey, content, detectedAt)
	case "custom":
		return detectCustom(ctx, config.APIURL, apiKey, content, detectedAt)
	default:
		return nil, fmt.Errorf("unknown detection provider: %s", config.Provider)
	}
}

func detectGPTZero(ctx context.Context, apiURL, apiKey, content, detectedAt string) (*DetectionResult, error) {
	payload := map[string]string{
		"document": content,
	}

	resp, err := callDetectionAPI(ctx, apiURL, apiKey, "X-Api-Key", payload)
	if err != nil {
		return nil, err
	}

	// Parse response
	documents, ok := resp["documents"].([]any)
	score := 0.0
	if ok && len(documents) > 0 {
		if doc, ok := documents[0].(map[string]any); ok {
			if prob, ok := doc["completely_generated_prob"].(float64); ok {
				score = prob
			}
		}
	}

	return &DetectionResult{
		Score:      score,
		Provider:   "gptzero",
		DetectedAt: detectedAt,
		Raw:        resp,
	}, nil
}

func detectOriginality(ctx context.Context, apiURL, apiKey, content, detectedAt string) (*DetectionResult, error) {
	payload := map[string]string{
		"content": content,
	}

	resp, err := callDetectionAPI(ctx, apiURL, apiKey, "Authorization", payload)
	if err != nil {
		return nil, err
	}

	// Parse response
	score := 0.0
	if scoreObj, ok := resp["score"].(map[string]any); ok {
		if aiScore, ok := scoreObj["ai"].(float64); ok {
			score = aiScore
		}
	}

	return &DetectionResult{
		Score:      score,
		Provider:   "originality",
		DetectedAt: detectedAt,
		Raw:        resp,
	}, nil
}

func detectCustom(ctx context.Context, apiURL, apiKey, content, detectedAt string) (*DetectionResult, error) {
	payload := map[string]string{
		"content": content,
	}

	resp, err := callDetectionAPI(ctx, apiURL, apiKey, "Authorization", payload)
	if err != nil {
		return nil, err
	}

	// Custom endpoint must return { score: number }
	score := 0.0
	if s, ok := resp["score"].(float64); ok {
		score = s
	}

	return &DetectionResult{
		Score:      score,
		Provider:   "custom",
		DetectedAt: detectedAt,
		Raw:        resp,
	}, nil
}

func callDetectionAPI(ctx context.Context, apiURL, apiKey, authHeader string, payload any) (map[string]any, error) {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if authHeader == "Authorization" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		req.Header.Set(authHeader, apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("detection API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("detection API failed: %d %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
