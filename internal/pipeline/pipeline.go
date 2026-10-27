package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/yash-sojitra-20/content-engine/internal/config"
	"github.com/yash-sojitra-20/content-engine/internal/content"
	"github.com/yash-sojitra-20/content-engine/internal/observability"
	"github.com/yash-sojitra-20/content-engine/internal/retry"
	"github.com/yash-sojitra-20/content-engine/internal/storage"
	"github.com/yash-sojitra-20/content-engine/internal/store"
	"github.com/yash-sojitra-20/content-engine/internal/video"
)

// Pipeline coordinates the end-to-end autonomous content generation workflow.
type Pipeline struct {
	cfg         *config.Config
	llm         content.LLMProvider
	video       video.VideoProvider
	storage     storage.StorageProvider
	store       store.RunStore
	retryPolicy retry.Policy
}

// NewPipeline creates and configures a new Pipeline instance.
func NewPipeline(
	cfg *config.Config,
	llm content.LLMProvider,
	vid video.VideoProvider,
	stor storage.StorageProvider,
	st store.RunStore,
) *Pipeline {
	return &Pipeline{
		cfg:     cfg,
		llm:     llm,
		video:   vid,
		storage: stor,
		store:   st,
		retryPolicy: retry.Policy{
			MaxAttempts:     cfg.MaxRetries,
			InitialInterval: cfg.RetryInitialInterval,
			MaxInterval:     cfg.RetryMaxInterval,
			Multiplier:      2.0,
			JitterFraction:  0.2,
		},
	}
}

// Execute runs the complete content-engine generation pipeline once.
func (p *Pipeline) Execute(ctx context.Context) (*store.Run, error) {
	runID := uuid.New().String()
	now := time.Now().UTC()

	run := &store.Run{
		ID:        runID,
		Status:    store.StatusStarted,
		StartedAt: now,
	}

	slog.Info("Starting pipeline execution",
		observability.RunIDAttr(runID),
		observability.StageAttr("init"),
	)

	// Persist initial run state
	if err := p.store.CreateRun(ctx, run); err != nil {
		slog.Error("Failed to persist initial run state",
			observability.RunIDAttr(runID),
			observability.ErrorAttr(err),
		)
		return nil, fmt.Errorf("failed creating run in store: %w", err)
	}

	// -------------------------------------------------------------------------
	// Stage 1: Generate Content Brief via LLM
	// -------------------------------------------------------------------------
	slog.Info("Generating content brief",
		observability.RunIDAttr(runID),
		observability.StageAttr("brief_generation"),
		observability.ProviderAttr(p.llm.ProviderName()),
	)

	var brief *content.ContentBrief
	err := retry.Do(ctx, p.retryPolicy, func(attempt int) error {
		var genErr error
		brief, genErr = p.llm.GenerateContentBrief(ctx, p.cfg.ContentTopicPrompt)
		if genErr != nil {
			slog.Warn("Brief generation attempt failed",
				observability.RunIDAttr(runID),
				observability.AttemptAttr(attempt),
				observability.ErrorAttr(genErr),
			)
			run.RetryCount++
			return genErr
		}
		return nil
	})

	if err != nil {
		return p.failRun(ctx, run, "brief_generation", fmt.Sprintf("failed generating content brief: %v", err))
	}

	run.Brief = brief
	run.Status = store.StatusBriefGenerated
	if err := p.store.UpdateRun(ctx, run); err != nil {
		slog.Warn("Failed to update run status after brief generation", observability.RunIDAttr(runID), observability.ErrorAttr(err))
	}

	slog.Info("Content brief generated successfully",
		observability.RunIDAttr(runID),
		slog.String("title", brief.Title),
		slog.String("aspect_ratio", brief.AspectRatio),
		slog.Int("duration_seconds", brief.DurationSeconds),
	)

	// -------------------------------------------------------------------------
	// Stage 2: Submit Video Generation Job
	// -------------------------------------------------------------------------
	slog.Info("Submitting video generation job",
		observability.RunIDAttr(runID),
		observability.StageAttr("video_generation_submission"),
		observability.ProviderAttr(p.video.ProviderName()),
	)

	var jobID string
	err = retry.Do(ctx, p.retryPolicy, func(attempt int) error {
		var subErr error
		jobID, subErr = p.video.SubmitGeneration(ctx, brief)
		if subErr != nil {
			slog.Warn("Video generation submission attempt failed",
				observability.RunIDAttr(runID),
				observability.AttemptAttr(attempt),
				observability.ErrorAttr(subErr),
			)
			run.RetryCount++
			return subErr
		}
		return nil
	})

	if err != nil {
		return p.failRun(ctx, run, "video_generation_submission", fmt.Sprintf("failed submitting video generation: %v", err))
	}

	run.ProviderJobID = jobID
	run.Status = store.StatusVideoGenerationSubmitted
	if err := p.store.UpdateRun(ctx, run); err != nil {
		slog.Warn("Failed to update run status after submission", observability.RunIDAttr(runID), observability.ErrorAttr(err))
	}

	slog.Info("Video generation job submitted successfully",
		observability.RunIDAttr(runID),
		slog.String("job_id", jobID),
	)

	// -------------------------------------------------------------------------
	// Stage 3: Poll Provider Job Status
	// -------------------------------------------------------------------------
	slog.Info("Polling video generation status",
		observability.RunIDAttr(runID),
		observability.StageAttr("video_generating"),
		slog.String("job_id", jobID),
	)

	run.Status = store.StatusVideoGenerating
	_ = p.store.UpdateRun(ctx, run)

	downloadURL, err := p.pollVideoCompletion(ctx, runID, jobID)
	if err != nil {
		return p.failRun(ctx, run, "video_generating", err.Error())
	}

	// -------------------------------------------------------------------------
	// Stage 4: Download Completed Video
	// -------------------------------------------------------------------------
	slog.Info("Downloading generated video",
		observability.RunIDAttr(runID),
		observability.StageAttr("video_download"),
	)

	if err := os.MkdirAll(p.cfg.TmpDir, 0755); err != nil {
		return p.failRun(ctx, run, "video_download", fmt.Sprintf("failed to create tmp dir: %v", err))
	}

	destPath := filepath.Join(p.cfg.TmpDir, fmt.Sprintf("%s.mp4", runID))

	err = retry.Do(ctx, p.retryPolicy, func(attempt int) error {
		bytesWritten, dlErr := p.video.DownloadVideo(ctx, downloadURL, destPath)
		if dlErr != nil {
			slog.Warn("Video download attempt failed",
				observability.RunIDAttr(runID),
				observability.AttemptAttr(attempt),
				observability.ErrorAttr(dlErr),
			)
			run.RetryCount++
			return dlErr
		}
		slog.Info("Video downloaded successfully",
			observability.RunIDAttr(runID),
			slog.Int64("bytes", bytesWritten),
			slog.String("path", destPath),
		)
		return nil
	})

	if err != nil {
		return p.failRun(ctx, run, "video_download", fmt.Sprintf("failed downloading video: %v", err))
	}

	run.LocalVideoPath = destPath
	run.Status = store.StatusVideoDownloaded
	_ = p.store.UpdateRun(ctx, run)

	// -------------------------------------------------------------------------
	// Stage 5: Upload Video to Storage (Google Drive / Local)
	// -------------------------------------------------------------------------
	slog.Info("Uploading video to storage",
		observability.RunIDAttr(runID),
		observability.StageAttr("video_upload"),
		observability.ProviderAttr(p.storage.ProviderName()),
	)

	uploadMeta := storage.UploadMetadata{
		RunID:       runID,
		Title:       brief.Title,
		Filename:    fmt.Sprintf("content_%s.mp4", runID),
		ContentType: "video/mp4",
		CreatedAt:   time.Now().UTC(),
	}

	var storedFile *storage.StoredFile
	err = retry.Do(ctx, p.retryPolicy, func(attempt int) error {
		var upErr error
		storedFile, upErr = p.storage.Upload(ctx, destPath, uploadMeta)
		if upErr != nil {
			slog.Warn("Storage upload attempt failed",
				observability.RunIDAttr(runID),
				observability.AttemptAttr(attempt),
				observability.ErrorAttr(upErr),
			)
			run.RetryCount++
			return upErr
		}
		return nil
	})

	if err != nil {
		return p.failRun(ctx, run, "video_upload", fmt.Sprintf("failed uploading video to storage: %v", err))
	}

	run.DriveFileID = storedFile.FileID
	run.DriveFileLink = storedFile.WebViewLink
	run.Status = store.StatusVideoUploaded
	_ = p.store.UpdateRun(ctx, run)

	slog.Info("Video stored successfully",
		observability.RunIDAttr(runID),
		slog.String("file_id", storedFile.FileID),
		slog.String("link", storedFile.WebViewLink),
		slog.Int64("size_bytes", storedFile.Size),
	)

	// -------------------------------------------------------------------------
	// Stage 6: Cleanup Temp File & Mark Succeeded
	// -------------------------------------------------------------------------
	_ = os.Remove(destPath)

	completedAt := time.Now().UTC()
	run.CompletedAt = &completedAt
	run.Status = store.StatusSucceeded

	if err := p.store.UpdateRun(ctx, run); err != nil {
		slog.Warn("Failed to persist final run status", observability.RunIDAttr(runID), observability.ErrorAttr(err))
	}

	slog.Info("Pipeline execution completed successfully",
		observability.RunIDAttr(runID),
		slog.Duration("duration", completedAt.Sub(run.StartedAt)),
		slog.String("file_link", run.DriveFileLink),
	)

	return run, nil
}

func (p *Pipeline) pollVideoCompletion(ctx context.Context, runID, jobID string) (string, error) {
	deadline := time.Now().Add(p.cfg.VideoMaxWait)
	ticker := time.NewTicker(p.cfg.VideoPollInterval)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("video generation timed out after %v", p.cfg.VideoMaxWait)
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			status, err := p.video.PollStatus(ctx, jobID)
			if err != nil {
				slog.Warn("Error during video status polling",
					observability.RunIDAttr(runID),
					slog.String("job_id", jobID),
					observability.ErrorAttr(err),
				)
				continue
			}

			slog.Debug("Polled video generation status",
				observability.RunIDAttr(runID),
				slog.String("state", string(status.State)),
				slog.Int("progress", status.Progress),
			)

			switch status.State {
			case video.StateCompleted:
				if status.DownloadURL == "" {
					return "", fmt.Errorf("provider reported job completed but returned empty download URL")
				}
				return status.DownloadURL, nil
			case video.StateFailed:
				return "", fmt.Errorf("provider generation job failed: %s", status.ErrorMessage)
			case video.StatePending, video.StateProcessing:
				// continue polling
			}
		}
	}
}

func (p *Pipeline) failRun(ctx context.Context, run *store.Run, stage, message string) (*store.Run, error) {
	now := time.Now().UTC()
	run.CompletedAt = &now
	run.Status = store.StatusFailed
	run.ErrorStage = stage
	run.ErrorMessage = observability.Sanitize(message)

	slog.Error("Pipeline execution failed",
		observability.RunIDAttr(run.ID),
		observability.StageAttr(stage),
		slog.String("message", run.ErrorMessage),
	)

	_ = p.store.UpdateRun(ctx, run)
	return run, fmt.Errorf("pipeline failed at stage [%s]: %s", stage, message)
}
