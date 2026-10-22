package video

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yash-sojitra-20/content-engine/internal/content"
)

// Minimal valid MP4 header (ftyp box with isom brand)
var minimalMP4Bytes = []byte{
	0x00, 0x00, 0x00, 0x20, // size 32
	'f', 't', 'y', 'p', // type "ftyp"
	'i', 's', 'o', 'm', // major brand "isom"
	0x00, 0x00, 0x02, 0x00, // minor version
	'i', 's', 'o', 'm', 'i', 's', 'o', '2', 'a', 'v', 'c', '1', 'm', 'p', '4', '1',
	0x00, 0x00, 0x00, 0x08, // mdat box size 8
	'm', 'd', 'a', 't', // type "mdat"
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, // payload
}

type mockJob struct {
	id          string
	submitTime  time.Time
	pollCount   int
	downloadURL string
}

// MockVideoProvider simulates an asynchronous video generation backend.
type MockVideoProvider struct {
	mu           sync.Mutex
	jobs         map[string]*mockJob
	pollsToDone  int
	SubmitErr    error
	PollErr      error
	DownloadErr  error
}

// NewMockVideoProvider creates a MockVideoProvider that completes after specified poll cycles.
func NewMockVideoProvider(pollsToComplete int) *MockVideoProvider {
	if pollsToComplete <= 0 {
		pollsToComplete = 2
	}
	return &MockVideoProvider{
		jobs:        make(map[string]*mockJob),
		pollsToDone: pollsToComplete,
	}
}

func (m *MockVideoProvider) ProviderName() string {
	return "mock"
}

func (m *MockVideoProvider) SubmitGeneration(ctx context.Context, brief *content.ContentBrief) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SubmitErr != nil {
		return "", m.SubmitErr
	}

	jobID := "mock-job-" + uuid.New().String()[:8]
	m.jobs[jobID] = &mockJob{
		id:          jobID,
		submitTime:  time.Now(),
		pollCount:   0,
		downloadURL: fmt.Sprintf("https://mock.storage.local/videos/%s.mp4", jobID),
	}

	return jobID, nil
}

func (m *MockVideoProvider) PollStatus(ctx context.Context, jobID string) (*JobStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.PollErr != nil {
		return nil, m.PollErr
	}

	job, exists := m.jobs[jobID]
	if !exists {
		return &JobStatus{
			ID:           jobID,
			State:        StateFailed,
			ErrorMessage: "job not found",
		}, nil
	}

	job.pollCount++

	if job.pollCount < m.pollsToDone {
		return &JobStatus{
			ID:       jobID,
			State:    StateProcessing,
			Progress: (job.pollCount * 100) / m.pollsToDone,
		}, nil
	}

	return &JobStatus{
		ID:          jobID,
		State:       StateCompleted,
		DownloadURL: job.downloadURL,
		Progress:    100,
	}, nil
}

func (m *MockVideoProvider) DownloadVideo(ctx context.Context, downloadURL string, destinationFilePath string) (int64, error) {
	if m.DownloadErr != nil {
		return 0, m.DownloadErr
	}

	if err := os.MkdirAll(filepath.Dir(destinationFilePath), 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory for mock video download: %w", err)
	}

	file, err := os.Create(destinationFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer file.Close()

	n, err := file.Write(minimalMP4Bytes)
	if err != nil {
		return 0, fmt.Errorf("failed writing mock video bytes: %w", err)
	}

	return int64(n), nil
}
