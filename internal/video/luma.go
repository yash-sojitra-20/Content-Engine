package video

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/content"
	"github.com/yash-sojitra-20/content-engine/internal/retry"
)

// LumaClient implements VideoProvider for Luma Dream Machine API.
type LumaClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewLumaClient creates a new LumaClient.
func NewLumaClient(apiKey string, timeout time.Duration) *LumaClient {
	return &LumaClient{
		apiKey:  apiKey,
		baseURL: "https://api.lumalabs.ai/dream-machine/v1",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (l *LumaClient) ProviderName() string {
	return "luma"
}

type lumaCreateRequest struct {
	Prompt      string `json:"prompt"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Loop        bool   `json:"loop"`
}

type lumaGenerationResponse struct {
	ID            string `json:"id"`
	State         string `json:"state"` // "queued", "dreaming", "completed", "failed"
	FailureReason string `json:"failure_reason,omitempty"`
	Assets        *struct {
		Video string `json:"video,omitempty"`
	} `json:"assets,omitempty"`
}

func (l *LumaClient) SubmitGeneration(ctx context.Context, brief *content.ContentBrief) (string, error) {
	if l.apiKey == "" {
		return "", retry.MarkNonRetryable(fmt.Errorf("luma api key is required"))
	}

	url := fmt.Sprintf("%s/generations", l.baseURL)

	payload := lumaCreateRequest{
		Prompt:      brief.GenerationPrompt,
		AspectRatio: brief.AspectRatio,
		Loop:        false,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", retry.MarkNonRetryable(fmt.Errorf("failed to marshal luma request: %w", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to build luma request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("luma submit request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read luma submit response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("luma api submit returned status %d: %s", resp.StatusCode, string(respBytes))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest {
			return "", retry.MarkNonRetryable(fmt.Errorf("%s", errMsg))
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	var genResp lumaGenerationResponse
	if err := json.Unmarshal(respBytes, &genResp); err != nil {
		return "", fmt.Errorf("failed to decode luma response: %w", err)
	}

	if genResp.ID == "" {
		return "", fmt.Errorf("luma generation ID not returned in response")
	}

	return genResp.ID, nil
}

func (l *LumaClient) PollStatus(ctx context.Context, jobID string) (*JobStatus, error) {
	url := fmt.Sprintf("%s/generations/%s", l.baseURL, jobID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build luma poll request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("luma poll request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read luma poll response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("luma poll returned HTTP %d: %s", resp.StatusCode, string(respBytes))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, retry.MarkNonRetryable(fmt.Errorf("%s", errMsg))
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	var genResp lumaGenerationResponse
	if err := json.Unmarshal(respBytes, &genResp); err != nil {
		return nil, fmt.Errorf("failed to decode luma poll response: %w", err)
	}

	status := &JobStatus{
		ID: jobID,
	}

	switch genResp.State {
	case "queued":
		status.State = StatePending
		status.Progress = 10
	case "dreaming":
		status.State = StateProcessing
		status.Progress = 50
	case "completed":
		status.State = StateCompleted
		status.Progress = 100
		if genResp.Assets != nil {
			status.DownloadURL = genResp.Assets.Video
		}
		if status.DownloadURL == "" {
			return nil, fmt.Errorf("luma job completed but video URL is missing")
		}
	case "failed":
		status.State = StateFailed
		status.ErrorMessage = genResp.FailureReason
	default:
		status.State = StateProcessing
	}

	return status, nil
}

func (l *LumaClient) DownloadVideo(ctx context.Context, downloadURL string, destinationFilePath string) (int64, error) {
	return StreamDownloadToFile(ctx, l.httpClient, downloadURL, destinationFilePath)
}
