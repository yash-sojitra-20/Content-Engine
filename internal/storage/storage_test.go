package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalStorageUploadSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcFile := filepath.Join(tempDir, "source.mp4")
	if err := os.WriteFile(srcFile, []byte("fake mp4 video bytes"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	destDir := filepath.Join(tempDir, "stored_videos")
	provider := NewLocalStorageProvider(destDir)

	meta := UploadMetadata{
		RunID:       "run-1234",
		Title:       "Sample Run",
		Filename:    "video_run-1234.mp4",
		ContentType: "video/mp4",
		CreatedAt:   time.Now(),
	}

	result, err := provider.Upload(context.Background(), srcFile, meta)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if result.Filename != "video_run-1234.mp4" {
		t.Errorf("expected filename video_run-1234.mp4, got %s", result.Filename)
	}
	if result.Size != int64(len("fake mp4 video bytes")) {
		t.Errorf("expected size %d, got %d", len("fake mp4 video bytes"), result.Size)
	}

	// Verify file exists in destDir
	if _, err := os.Stat(filepath.Join(destDir, "video_run-1234.mp4")); err != nil {
		t.Errorf("destination file does not exist: %v", err)
	}
}

func TestLocalStorageUploadEmptyFileFails(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcFile := filepath.Join(tempDir, "empty.mp4")
	if err := os.WriteFile(srcFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write empty test file: %v", err)
	}

	provider := NewLocalStorageProvider(filepath.Join(tempDir, "output"))
	_, err = provider.Upload(context.Background(), srcFile, UploadMetadata{RunID: "run-000"})
	if err == nil {
		t.Fatal("expected error on empty file upload, got nil")
	}
}
