package video

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/yash-sojitra-20/content-engine/internal/content"
)

// JobState represents the current lifecycle phase of a video generation task.
type JobState string

const (
	StatePending    JobState = "pending"
	StateProcessing JobState = "processing"
	StateCompleted  JobState = "completed"
	StateFailed     JobState = "failed"
)

// JobStatus contains the provider's asynchronous generation job status.
type JobStatus struct {
	ID           string   `json:"id"`
	State        JobState `json:"state"`
	DownloadURL  string   `json:"download_url,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
	Progress     int      `json:"progress,omitempty"`
}

// VideoProvider defines the interface for submitting, polling, and retrieving videos from an AI video provider.
type VideoProvider interface {
	ProviderName() string
	SubmitGeneration(ctx context.Context, brief *content.ContentBrief) (jobID string, err error)
	PollStatus(ctx context.Context, jobID string) (*JobStatus, error)
	DownloadVideo(ctx context.Context, downloadURL string, destinationFilePath string) (bytesWritten int64, err error)
}

// StreamDownloadToFile streams an HTTP download URL directly to destination file without buffering into memory.
func StreamDownloadToFile(ctx context.Context, httpClient *http.Client, downloadURL, destPath string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory for download: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build download request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("download server returned HTTP status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create local file %s: %w", destPath, err)
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("failed while streaming video to disk: %w", err)
	}

	if n == 0 {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("downloaded video file is 0 bytes")
	}

	return n, nil
}
