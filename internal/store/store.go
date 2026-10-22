package store

import (
	"context"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/content"
)

// RunStatus represents the state of a single pipeline execution.
type RunStatus string

const (
	StatusStarted                  RunStatus = "started"
	StatusBriefGenerated           RunStatus = "brief_generated"
	StatusVideoGenerationSubmitted RunStatus = "video_generation_submitted"
	StatusVideoGenerating          RunStatus = "video_generating"
	StatusVideoDownloaded          RunStatus = "video_downloaded"
	StatusVideoUploaded            RunStatus = "video_uploaded"
	StatusSucceeded                RunStatus = "succeeded"
	StatusFailed                   RunStatus = "failed"
)

// Run tracks the metadata, stages, and outputs of a single pipeline run.
type Run struct {
	ID             string                `json:"id"`
	Status         RunStatus             `json:"status"`
	StartedAt      time.Time             `json:"started_at"`
	CompletedAt    *time.Time            `json:"completed_at,omitempty"`
	Brief          *content.ContentBrief `json:"brief,omitempty"`
	ProviderJobID  string                `json:"provider_job_id,omitempty"`
	LocalVideoPath string                `json:"local_video_path,omitempty"`
	DriveFileID    string                `json:"drive_file_id,omitempty"`
	DriveFileLink  string                `json:"drive_file_link,omitempty"`
	ErrorStage     string                `json:"error_stage,omitempty"`
	ErrorMessage   string                `json:"error_message,omitempty"`
	RetryCount     int                   `json:"retry_count"`
}

// RunStore defines the persistence interface for pipeline run history.
type RunStore interface {
	CreateRun(ctx context.Context, run *Run) error
	UpdateRun(ctx context.Context, run *Run) error
	GetRun(ctx context.Context, id string) (*Run, error)
	ListRuns(ctx context.Context, limit int) ([]*Run, error)
	Close() error
}
