package social

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/retry"
)

// MockPublisher is an in-memory test and offline simulator for social media publishing.
type MockPublisher struct {
	mu           sync.Mutex
	Calls        []PublishRequest
	SimulateErr  error
	FailNext     error
	CustomPostID string
}

// NewMockPublisher creates a new MockPublisher instance.
func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		Calls: make([]PublishRequest, 0),
	}
}

func (m *MockPublisher) PlatformName() string {
	return "mock"
}

func (m *MockPublisher) Publish(ctx context.Context, req PublishRequest) (*PublishResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, req)

	if m.SimulateErr != nil {
		return nil, m.SimulateErr
	}

	if m.FailNext != nil {
		err := m.FailNext
		m.FailNext = nil
		return nil, err
	}

	// Basic validation of local video file
	if req.FilePath == "" {
		return nil, retry.MarkNonRetryable(fmt.Errorf("file path is required"))
	}
	stat, err := os.Stat(req.FilePath)
	if err != nil {
		return nil, retry.MarkNonRetryable(fmt.Errorf("video file not found at %s: %w", req.FilePath, err))
	}
	if stat.Size() == 0 {
		return nil, retry.MarkNonRetryable(fmt.Errorf("refusing to publish empty video file (0 bytes)"))
	}

	postID := m.CustomPostID
	if postID == "" {
		postID = fmt.Sprintf("mock_post_%s", req.RunID)
	}

	postURL := fmt.Sprintf("https://youtube.com/shorts/%s", postID)
	if !req.IsShort {
		postURL = fmt.Sprintf("https://youtu.be/%s", postID)
	}

	return &PublishResult{
		Platform:    "mock",
		PostID:      postID,
		PostURL:     postURL,
		PublishedAt: time.Now().UTC(),
		Metadata: map[string]string{
			"title":          req.Title,
			"privacy_status": req.PrivacyStatus,
			"is_short":       fmt.Sprintf("%t", req.IsShort),
		},
	}, nil
}
