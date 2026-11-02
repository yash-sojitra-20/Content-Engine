package social

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMockPublisherSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "social-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	videoPath := filepath.Join(tempDir, "video.mp4")
	if err := os.WriteFile(videoPath, []byte("fake mp4 content"), 0644); err != nil {
		t.Fatalf("failed to write test video: %v", err)
	}

	pub := NewMockPublisher()
	if pub.PlatformName() != "mock" {
		t.Errorf("expected platform 'mock', got %s", pub.PlatformName())
	}

	req := PublishRequest{
		RunID:         "run-123",
		FilePath:      videoPath,
		Title:         "Amazing AI Video",
		Description:   "Description of video",
		Tags:          []string{"ai", "tech"},
		PrivacyStatus: "public",
		Category:      "28",
		IsShort:       true,
	}

	res, err := pub.Publish(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	if res.Platform != "mock" {
		t.Errorf("expected platform 'mock', got %s", res.Platform)
	}
	if !strings.Contains(res.PostURL, "https://youtube.com/shorts/") {
		t.Errorf("expected Shorts URL, got %s", res.PostURL)
	}
	if len(pub.Calls) != 1 {
		t.Errorf("expected 1 recorded call, got %d", len(pub.Calls))
	}
}

func TestMockPublisherNonShortURL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "social-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	videoPath := filepath.Join(tempDir, "video.mp4")
	_ = os.WriteFile(videoPath, []byte("fake video data"), 0644)

	pub := NewMockPublisher()
	req := PublishRequest{
		RunID:    "run-456",
		FilePath: videoPath,
		Title:    "Longform Tech Video",
		IsShort:  false,
	}

	res, err := pub.Publish(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	if !strings.Contains(res.PostURL, "https://youtu.be/") {
		t.Errorf("expected standard youtube URL, got %s", res.PostURL)
	}
}

func TestMockPublisherFailure(t *testing.T) {
	pub := NewMockPublisher()
	pub.FailNext = errors.New("simulated upload network timeout")

	req := PublishRequest{
		RunID:    "run-fail",
		FilePath: "nonexistent.mp4",
	}

	_, err := pub.Publish(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "simulated upload network timeout") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMockPublisherValidation(t *testing.T) {
	pub := NewMockPublisher()

	// Missing file path
	_, err := pub.Publish(context.Background(), PublishRequest{})
	if err == nil {
		t.Error("expected error for empty file path")
	}

	// Nonexistent file
	_, err = pub.Publish(context.Background(), PublishRequest{FilePath: "does_not_exist.mp4"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	// 0-byte file
	tempDir, _ := os.MkdirTemp("", "social-test-empty-*")
	defer os.RemoveAll(tempDir)
	emptyPath := filepath.Join(tempDir, "empty.mp4")
	_ = os.WriteFile(emptyPath, []byte{}, 0644)

	_, err = pub.Publish(context.Background(), PublishRequest{FilePath: emptyPath})
	if err == nil {
		t.Error("expected error for 0-byte file")
	}
}

func TestFormatShortsTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		isShort  bool
		expected string
	}{
		{
			name:     "Empty title default",
			input:    "",
			isShort:  true,
			expected: "Autonomous Video Short #Shorts",
		},
		{
			name:     "Appends #Shorts when not present",
			input:    "Top 5 AI Tools",
			isShort:  true,
			expected: "Top 5 AI Tools #Shorts",
		},
		{
			name:     "Does not duplicate existing #shorts",
			input:    "Top 5 AI Tools #shorts",
			isShort:  true,
			expected: "Top 5 AI Tools #shorts",
		},
		{
			name:     "Does not append #Shorts when isShort is false",
			input:    "Top 5 AI Tools",
			isShort:  false,
			expected: "Top 5 AI Tools",
		},
		{
			name:     "Truncates long titles to 100 characters",
			input:    strings.Repeat("Long Title Text ", 10), // ~160 chars
			isShort:  false,
			expected: strings.Repeat("Long Title Text ", 10)[:97] + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := formatShortsTitle(tt.input, tt.isShort)
			if len(res) > 100 {
				t.Errorf("title exceeded 100 chars: len=%d, text=%q", len(res), res)
			}
			if tt.name == "Appends #Shorts when not present" && !strings.HasSuffix(res, "#Shorts") {
				t.Errorf("expected suffix '#Shorts', got %q", res)
			}
			if tt.name == "Does not duplicate existing #shorts" && strings.Count(strings.ToLower(res), "#shorts") != 1 {
				t.Errorf("duplicate #shorts detected: %q", res)
			}
		})
	}
}

func TestNewYouTubePublisherValidation(t *testing.T) {
	// No credentials
	_, err := NewYouTubePublisher(context.Background(), YouTubeConfig{})
	if err == nil {
		t.Error("expected error when no credentials provided")
	}

	// Nonexistent credentials file
	_, err = NewYouTubePublisher(context.Background(), YouTubeConfig{
		CredentialsFile: "nonexistent_creds.json",
	})
	if err == nil {
		t.Error("expected error when credentials file does not exist")
	}
}
