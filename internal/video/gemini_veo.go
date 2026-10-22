package video

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/content"
	"github.com/yash-sojitra-20/content-engine/internal/retry"
)

// GeminiVeoClient implements VideoProvider for Google's Veo video generation API.
type GeminiVeoClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

// NewGeminiVeoClient creates a new GeminiVeoClient.
func NewGeminiVeoClient(apiKey, model string, timeout time.Duration) *GeminiVeoClient {
	if model == "" {
		model = "veo-2.0-generate-001"
	}
	return &GeminiVeoClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (g *GeminiVeoClient) ProviderName() string {
	return "gemini_veo"
}

type veoPredictRequest struct {
	Instances []struct {
		Prompt string `json:"prompt"`
	} `json:"instances"`
	Parameters struct {
		SampleCount int    `json:"sampleCount"`
		AspectRatio string `json:"aspectRatio,omitempty"`
	} `json:"parameters"`
}

type veoOperationResponse struct {
	Name     string `json:"name"` // operations/<operation_id>
	Done     bool   `json:"done"`
	Error    *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Response *struct {
		GeneratedSamples []struct {
			Video struct {
				URI          string `json:"uri,omitempty"`
				BytesBase64  string `json:"bytesBase64Encoded,omitempty"`
			} `json:"video"`
		} `json:"generatedSamples"`
	} `json:"response,omitempty"`
}

func (g *GeminiVeoClient) SubmitGeneration(ctx context.Context, brief *content.ContentBrief) (string, error) {
	if g.apiKey == "" {
		return "", retry.MarkNonRetryable(fmt.Errorf("gemini/veo api key is required"))
	}

	url := fmt.Sprintf("%s/models/%s:predictLongRunning?key=%s", g.baseURL, g.model, g.apiKey)

	reqPayload := veoPredictRequest{}
	reqPayload.Instances = append(reqPayload.Instances, struct {
		Prompt string `json:"prompt"`
	}{
		Prompt: brief.GenerationPrompt,
	})
	reqPayload.Parameters.SampleCount = 1
	reqPayload.Parameters.AspectRatio = brief.AspectRatio

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", retry.MarkNonRetryable(fmt.Errorf("failed to marshal veo request: %w", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to build veo request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("veo submit request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read veo submit response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("veo api submit returned status %d: %s", resp.StatusCode, string(respBytes))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest {
			return "", retry.MarkNonRetryable(fmt.Errorf("%s", errMsg))
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	var opResp veoOperationResponse
	if err := json.Unmarshal(respBytes, &opResp); err != nil {
		return "", fmt.Errorf("failed to decode veo operation response: %w", err)
	}

	if opResp.Name == "" {
		return "", fmt.Errorf("veo did not return operation name: %s", string(respBytes))
	}

	return opResp.Name, nil
}

func (g *GeminiVeoClient) PollStatus(ctx context.Context, operationName string) (*JobStatus, error) {
	// operationName format: "operations/12345" or raw ID
	name := strings.TrimPrefix(operationName, "/")
	url := fmt.Sprintf("%s/%s?key=%s", g.baseURL, name, g.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build veo poll request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("veo poll request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read veo poll response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("veo poll returned HTTP %d: %s", resp.StatusCode, string(respBytes))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, retry.MarkNonRetryable(fmt.Errorf("%s", errMsg))
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	var opResp veoOperationResponse
	if err := json.Unmarshal(respBytes, &opResp); err != nil {
		return nil, fmt.Errorf("failed to decode veo poll response: %w", err)
	}

	status := &JobStatus{
		ID: operationName,
	}

	if !opResp.Done {
		status.State = StateProcessing
		status.Progress = 50
		return status, nil
	}

	if opResp.Error != nil {
		status.State = StateFailed
		status.ErrorMessage = fmt.Sprintf("code %d: %s", opResp.Error.Code, opResp.Error.Message)
		return status, nil
	}

	status.State = StateCompleted
	status.Progress = 100

	if opResp.Response != nil && len(opResp.Response.GeneratedSamples) > 0 {
		sample := opResp.Response.GeneratedSamples[0]
		if sample.Video.URI != "" {
			status.DownloadURL = sample.Video.URI
		} else if sample.Video.BytesBase64 != "" {
			// Inline data URI
			status.DownloadURL = "data:video/mp4;base64," + sample.Video.BytesBase64
		}
	}

	return status, nil
}

func (g *GeminiVeoClient) DownloadVideo(ctx context.Context, downloadURL string, destinationFilePath string) (int64, error) {
	if strings.HasPrefix(downloadURL, "data:video/mp4;base64,") {
		b64 := strings.TrimPrefix(downloadURL, "data:video/mp4;base64,")
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return 0, fmt.Errorf("failed to decode base64 video data: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(destinationFilePath), 0755); err != nil {
			return 0, err
		}
		if err := os.WriteFile(destinationFilePath, data, 0644); err != nil {
			return 0, err
		}
		return int64(len(data)), nil
	}

	return StreamDownloadToFile(ctx, g.httpClient, downloadURL, destinationFilePath)
}
