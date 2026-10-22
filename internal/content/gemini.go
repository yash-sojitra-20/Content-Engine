package content

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/retry"
)

// GeminiClient implements LLMProvider using the Google Gemini API.
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

// NewGeminiClient creates a new GeminiClient.
func NewGeminiClient(apiKey, model string, timeout time.Duration) *GeminiClient {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return &GeminiClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ProviderName returns "gemini".
func (g *GeminiClient) ProviderName() string {
	return "gemini"
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
}

type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

const briefSystemInstruction = `You are an expert viral video producer and AI prompt engineer.
Your job is to generate a structured content brief and a high-fidelity visual prompt for generating an AI video.
You MUST output valid, raw JSON only conforming strictly to this schema:
{
  "title": "Short catchy title",
  "topic": "Core topic or insight",
  "target_audience": "Intended audience (e.g. developers, founders)",
  "script": "Voice-over script or punchy on-screen text",
  "duration_seconds": 10,
  "aspect_ratio": "9:16",
  "visual_style": "Cinematic 3D render, photorealistic, cyberpunk, etc.",
  "generation_prompt": "Highly detailed, cinematic visual prompt describing camera movements, lighting, textures, action, and subject for text-to-video AI generation",
  "scenes": [
    {
      "scene_number": 1,
      "description": "Visual scene description",
      "visual_cue": "Camera pan, neon lighting",
      "narration": "Voiceover line"
    }
  ]
}
`

// GenerateContentBrief calls the Gemini API to generate a video brief.
func (g *GeminiClient) GenerateContentBrief(ctx context.Context, topicPrompt string) (*ContentBrief, error) {
	if g.apiKey == "" {
		return nil, retry.MarkNonRetryable(fmt.Errorf("gemini api key is required"))
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", g.baseURL, g.model, g.apiKey)

	reqPayload := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{
				{Text: briefSystemInstruction},
			},
		},
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: fmt.Sprintf("Generate a complete video brief for the following topic/theme: %s", topicPrompt)},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.7,
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, retry.MarkNonRetryable(fmt.Errorf("failed to marshal gemini request: %w", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("gemini api returned status %d: %s", resp.StatusCode, string(respBytes))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest {
			return nil, retry.MarkNonRetryable(fmt.Errorf("%s", errMsg))
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("gemini error (%d): %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty candidate list in gemini response")
	}

	rawText := geminiResp.Candidates[0].Content.Parts[0].Text
	rawText = cleanJSONMarkdown(rawText)

	var brief ContentBrief
	if err := json.Unmarshal([]byte(rawText), &brief); err != nil {
		return nil, fmt.Errorf("failed to parse content brief JSON (%w): %s", err, rawText)
	}

	if err := brief.Validate(); err != nil {
		return nil, fmt.Errorf("gemini brief validation failed: %w", err)
	}

	return &brief, nil
}

// cleanJSONMarkdown removes ```json and ``` fences if present.
func cleanJSONMarkdown(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}
