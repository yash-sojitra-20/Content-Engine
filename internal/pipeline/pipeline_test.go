package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/config"
	"github.com/yash-sojitra-20/content-engine/internal/content"
	"github.com/yash-sojitra-20/content-engine/internal/social"
	"github.com/yash-sojitra-20/content-engine/internal/storage"
	"github.com/yash-sojitra-20/content-engine/internal/store"
	"github.com/yash-sojitra-20/content-engine/internal/video"
)

func setupTestPipeline(t *testing.T) (*Pipeline, *content.MockLLMClient, *video.MockVideoProvider, *social.MockPublisher, *config.Config, string) {
	tempDir, err := os.MkdirTemp("", "pipe-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cfg := &config.Config{
		RunMode:              "once",
		ContentTopicPrompt:   "Testing Pipeline Flow",
		VideoPollInterval:    10 * time.Millisecond,
		VideoMaxWait:         500 * time.Millisecond,
		TmpDir:               filepath.Join(tempDir, "tmp"),
		MaxRetries:           2,
		RetryInitialInterval: 5 * time.Millisecond,
		RetryMaxInterval:     20 * time.Millisecond,
		YouTubePrivacyStatus: "public",
		YouTubeCategoryID:    "28",
		YouTubeDefaultTags:   "shorts,tech",
	}

	llm := content.NewMockLLMClient()
	vid := video.NewMockVideoProvider(1)
	stor := storage.NewLocalStorageProvider(filepath.Join(tempDir, "output"))
	pub := social.NewMockPublisher()
	st, err := store.NewSQLiteStore(filepath.Join(tempDir, "runs.db"))
	if err != nil {
		t.Fatalf("failed to init test store: %v", err)
	}

	p := NewPipeline(cfg, llm, vid, stor, pub, st)
	return p, llm, vid, pub, cfg, tempDir
}

func TestPipelineExecuteSuccess(t *testing.T) {
	p, _, _, _, _, tempDir := setupTestPipeline(t)
	defer os.RemoveAll(tempDir)

	run, err := p.Execute(context.Background())
	if err != nil {
		t.Fatalf("expected pipeline to succeed, got error: %v", err)
	}

	if run.Status != store.StatusSucceeded {
		t.Errorf("expected status 'succeeded', got %s", run.Status)
	}
	if run.Brief == nil {
		t.Errorf("expected brief to be set")
	}
	if run.ProviderJobID == "" {
		t.Errorf("expected provider job ID to be set")
	}
	if run.DriveFileID == "" {
		t.Errorf("expected stored file ID to be set")
	}
	if run.DriveFileLink == "" {
		t.Errorf("expected stored file link to be set")
	}
	if run.CompletedAt == nil {
		t.Errorf("expected completed_at to be recorded")
	}

	// Verify temp video was cleaned up
	if run.LocalVideoPath != "" {
		if _, err := os.Stat(run.LocalVideoPath); !os.IsNotExist(err) {
			t.Errorf("expected temp video file to be cleaned up, but it still exists: %s", run.LocalVideoPath)
		}
	}
}

func TestPipelineExecuteWithSocialPublishing(t *testing.T) {
	p, _, _, pub, cfg, tempDir := setupTestPipeline(t)
	defer os.RemoveAll(tempDir)

	cfg.SocialPublishEnabled = true
	cfg.SocialPlatform = "mock"

	run, err := p.Execute(context.Background())
	if err != nil {
		t.Fatalf("expected pipeline to succeed with social publishing, got error: %v", err)
	}

	if run.Status != store.StatusSucceeded {
		t.Errorf("expected status 'succeeded', got %s", run.Status)
	}
	if run.SocialPlatform != "mock" {
		t.Errorf("expected social platform 'mock', got %s", run.SocialPlatform)
	}
	if run.SocialPostID == "" {
		t.Errorf("expected social post ID to be populated")
	}
	if run.SocialPostURL == "" {
		t.Errorf("expected social post URL to be populated")
	}
	if len(pub.Calls) != 1 {
		t.Errorf("expected 1 social publish call, got %d", len(pub.Calls))
	}
}

func TestPipelineExecuteSocialPublishingFailure(t *testing.T) {
	p, _, _, pub, cfg, tempDir := setupTestPipeline(t)
	defer os.RemoveAll(tempDir)

	cfg.SocialPublishEnabled = true
	cfg.SocialPlatform = "mock"
	pub.SimulateErr = errors.New("simulated social platform quota exceeded")

	run, err := p.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if run.Status != store.StatusFailed {
		t.Errorf("expected status 'failed', got %s", run.Status)
	}
	if run.ErrorStage != "social_publishing" {
		t.Errorf("expected error stage 'social_publishing', got %s", run.ErrorStage)
	}
}

func TestPipelineExecuteBriefFailure(t *testing.T) {
	p, llm, _, _, _, tempDir := setupTestPipeline(t)
	defer os.RemoveAll(tempDir)

	llm.SimulateErr = errors.New("simulated LLM API quota exceeded")

	run, err := p.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if run.Status != store.StatusFailed {
		t.Errorf("expected status 'failed', got %s", run.Status)
	}
	if run.ErrorStage != "brief_generation" {
		t.Errorf("expected error stage 'brief_generation', got %s", run.ErrorStage)
	}
}

func TestPipelineExecuteVideoSubmitFailure(t *testing.T) {
	p, _, vid, _, _, tempDir := setupTestPipeline(t)
	defer os.RemoveAll(tempDir)

	vid.SubmitErr = errors.New("simulated video provider connection timeout")

	run, err := p.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if run.Status != store.StatusFailed {
		t.Errorf("expected status 'failed', got %s", run.Status)
	}
	if run.ErrorStage != "video_generation_submission" {
		t.Errorf("expected error stage 'video_generation_submission', got %s", run.ErrorStage)
	}
}
