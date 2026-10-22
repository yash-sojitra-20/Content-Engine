package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/content"
)

func TestSQLiteStoreCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "runs.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	run := &Run{
		ID:        "run-test-001",
		Status:    StatusStarted,
		StartedAt: now,
		Brief: &content.ContentBrief{
			Title:            "Test Brief",
			Topic:            "Test Topic",
			GenerationPrompt: "Test Prompt",
			DurationSeconds:  10,
			AspectRatio:      "9:16",
		},
	}

	// 1. Create Run
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	// 2. Get Run
	fetched, err := store.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if fetched.ID != run.ID || fetched.Status != StatusStarted {
		t.Errorf("mismatch on fetched run: %+v", fetched)
	}
	if fetched.Brief == nil || fetched.Brief.Title != "Test Brief" {
		t.Errorf("brief not deserialized properly: %+v", fetched.Brief)
	}

	// 3. Update Run
	completedAt := now.Add(1 * time.Minute)
	fetched.Status = StatusSucceeded
	fetched.CompletedAt = &completedAt
	fetched.DriveFileID = "drive-file-999"
	fetched.DriveFileLink = "https://drive.google.com/file/d/drive-file-999/view"

	if err := store.UpdateRun(ctx, fetched); err != nil {
		t.Fatalf("UpdateRun failed: %v", err)
	}

	updated, err := store.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after update failed: %v", err)
	}
	if updated.Status != StatusSucceeded {
		t.Errorf("expected status 'succeeded', got %v", updated.Status)
	}
	if updated.DriveFileID != "drive-file-999" {
		t.Errorf("expected DriveFileID drive-file-999, got %v", updated.DriveFileID)
	}

	// 4. List Runs
	runs, err := store.ListRuns(ctx, 10)
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected 1 run in list, got %d", len(runs))
	}
}

func TestJSONLStoreCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jsonl-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonlPath := filepath.Join(tempDir, "runs.jsonl")
	store, err := NewJSONLStore(jsonlPath)
	if err != nil {
		t.Fatalf("NewJSONLStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	run := &Run{
		ID:        "run-jsonl-001",
		Status:    StatusStarted,
		StartedAt: now,
		Brief: &content.ContentBrief{
			Title:            "JSONL Brief",
			Topic:            "JSONL Topic",
			GenerationPrompt: "JSONL Prompt",
			DurationSeconds:  10,
			AspectRatio:      "9:16",
		},
	}

	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	fetched, err := store.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if fetched.ID != run.ID {
		t.Errorf("mismatch on fetched run: %+v", fetched)
	}

	fetched.Status = StatusSucceeded
	if err := store.UpdateRun(ctx, fetched); err != nil {
		t.Fatalf("UpdateRun failed: %v", err)
	}

	updated, err := store.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun after update failed: %v", err)
	}
	if updated.Status != StatusSucceeded {
		t.Errorf("expected status 'succeeded', got %v", updated.Status)
	}
}
