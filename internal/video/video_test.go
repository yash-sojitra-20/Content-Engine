package video

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yash-sojitra-20/content-engine/internal/content"
)

func TestMockVideoProviderLifecycle(t *testing.T) {
	mock := NewMockVideoProvider(2)
	ctx := context.Background()

	brief := &content.ContentBrief{
		Title:            "Test Video",
		Topic:            "Test Topic",
		GenerationPrompt: "Camera flies over glowing futuristic city",
		DurationSeconds:  10,
		AspectRatio:      "9:16",
	}

	jobID, err := mock.SubmitGeneration(ctx, brief)
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if jobID == "" {
		t.Fatal("expected non-empty job ID")
	}

	// Poll 1: processing
	status1, err := mock.PollStatus(ctx, jobID)
	if err != nil {
		t.Fatalf("unexpected poll error: %v", err)
	}
	if status1.State != StateProcessing {
		t.Errorf("expected state 'processing', got %v", status1.State)
	}

	// Poll 2: completed
	status2, err := mock.PollStatus(ctx, jobID)
	if err != nil {
		t.Fatalf("unexpected poll error: %v", err)
	}
	if status2.State != StateCompleted {
		t.Errorf("expected state 'completed', got %v", status2.State)
	}
	if status2.DownloadURL == "" {
		t.Errorf("expected non-empty download URL")
	}

	// Test Download
	tmpDir, err := os.MkdirTemp("", "video-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "test.mp4")
	bytesWritten, err := mock.DownloadVideo(ctx, status2.DownloadURL, destPath)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if bytesWritten == 0 {
		t.Errorf("expected bytes written > 0")
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("stat failed on downloaded video: %v", err)
	}
	if info.Size() != bytesWritten {
		t.Errorf("file size %d != bytesWritten %d", info.Size(), bytesWritten)
	}
}

func TestMockVideoProviderPollNotFound(t *testing.T) {
	mock := NewMockVideoProvider(2)
	status, err := mock.PollStatus(context.Background(), "non-existent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.State != StateFailed {
		t.Errorf("expected state failed for unknown job, got %v", status.State)
	}
}

func TestMockVideoProviderSimulatedErrors(t *testing.T) {
	mock := NewMockVideoProvider(1)
	mock.SubmitErr = os.ErrPermission

	_, err := mock.SubmitGeneration(context.Background(), &content.ContentBrief{})
	if err != os.ErrPermission {
		t.Errorf("expected ErrPermission, got: %v", err)
	}
}
