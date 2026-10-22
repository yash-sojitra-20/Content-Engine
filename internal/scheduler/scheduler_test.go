package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/config"
	"github.com/yash-sojitra-20/content-engine/internal/content"
	"github.com/yash-sojitra-20/content-engine/internal/pipeline"
	"github.com/yash-sojitra-20/content-engine/internal/storage"
	"github.com/yash-sojitra-20/content-engine/internal/store"
	"github.com/yash-sojitra-20/content-engine/internal/video"
)

func TestSchedulerOverlapPrevention(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sched-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		RunMode:              "schedule",
		ContentTopicPrompt:   "Test Overlap",
		VideoPollInterval:    10 * time.Millisecond,
		VideoMaxWait:         500 * time.Millisecond,
		TmpDir:               filepath.Join(tempDir, "tmp"),
		MaxRetries:           1,
		RetryInitialInterval: 5 * time.Millisecond,
		RetryMaxInterval:     10 * time.Millisecond,
	}

	llm := content.NewMockLLMClient()
	vid := video.NewMockVideoProvider(1)
	stor := storage.NewLocalStorageProvider(filepath.Join(tempDir, "output"))
	st, err := store.NewSQLiteStore(filepath.Join(tempDir, "runs.db"))
	if err != nil {
		t.Fatalf("failed to init test store: %v", err)
	}

	p := pipeline.NewPipeline(cfg, llm, vid, stor, st)

	sched, err := NewScheduler("0 * * * *", time.UTC, p)
	if err != nil {
		t.Fatalf("NewScheduler failed: %v", err)
	}

	// Manually simulate active run
	sched.isBusy.Store(true)

	// Trigger onTick while busy
	sched.onTick()

	// Should still be busy and should not have crashed
	if !sched.IsBusy() {
		t.Errorf("expected scheduler to remain busy")
	}

	sched.isBusy.Store(false)
}

func TestSchedulerRunAndGracefulStop(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sched-stop-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		RunMode:              "schedule",
		ContentTopicPrompt:   "Test Stop",
		VideoPollInterval:    10 * time.Millisecond,
		VideoMaxWait:         500 * time.Millisecond,
		TmpDir:               filepath.Join(tempDir, "tmp"),
		MaxRetries:           1,
		RetryInitialInterval: 5 * time.Millisecond,
		RetryMaxInterval:     10 * time.Millisecond,
	}

	llm := content.NewMockLLMClient()
	vid := video.NewMockVideoProvider(1)
	stor := storage.NewLocalStorageProvider(filepath.Join(tempDir, "output"))
	st, err := store.NewSQLiteStore(filepath.Join(tempDir, "runs.db"))
	if err != nil {
		t.Fatalf("failed to init test store: %v", err)
	}

	p := pipeline.NewPipeline(cfg, llm, vid, stor, st)

	sched, err := NewScheduler("0 0 1 1 *", time.UTC, p)
	if err != nil {
		t.Fatalf("NewScheduler failed: %v", err)
	}

	sched.Start()
	time.Sleep(50 * time.Millisecond)
	sched.Stop(100 * time.Millisecond)

	// Verify we can still run pipeline manually
	run, err := p.Execute(context.Background())
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	if run.Status != store.StatusSucceeded {
		t.Errorf("expected succeeded, got %s", run.Status)
	}
}
