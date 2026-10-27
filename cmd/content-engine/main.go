package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/yash-sojitra-20/content-engine/internal/config"
	"github.com/yash-sojitra-20/content-engine/internal/content"
	"github.com/yash-sojitra-20/content-engine/internal/observability"
	"github.com/yash-sojitra-20/content-engine/internal/pipeline"
	"github.com/yash-sojitra-20/content-engine/internal/scheduler"
	"github.com/yash-sojitra-20/content-engine/internal/storage"
	"github.com/yash-sojitra-20/content-engine/internal/store"
	"github.com/yash-sojitra-20/content-engine/internal/video"
)

func main() {
	modeFlag := flag.String("mode", "", "Run mode: 'once' (execute single pipeline and exit) or 'schedule' (recurring cron)")
	topicFlag := flag.String("topic", "", "Custom topic/prompt override for content brief generation")
	flag.Parse()

	// Load configuration from environment & optional .env
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Apply CLI flags if provided
	if *modeFlag != "" {
		cfg.RunMode = *modeFlag
	}
	if *topicFlag != "" {
		cfg.ContentTopicPrompt = *topicFlag
	}

	// Initialize structured logging
	logger := observability.SetupLogger(cfg.LogLevel, cfg.LogFormat)
	logger.Info("Starting Content-Engine",
		slog.String("run_mode", cfg.RunMode),
		slog.String("llm_provider", cfg.LLMProvider),
		slog.String("video_provider", cfg.VideoProvider),
		slog.String("storage_provider", cfg.StorageProvider),
		slog.String("db_type", cfg.DBType),
	)

	// Initialize Persistence Store
	runStore, err := initStore(cfg)
	if err != nil {
		logger.Error("Failed to initialize persistence store", observability.ErrorAttr(err))
		os.Exit(1)
	}
	defer runStore.Close()

	// Initialize LLM Provider
	llmProvider := initLLM(cfg)

	// Initialize Video Provider
	videoProvider := initVideo(cfg)

	// Initialize Storage Provider
	storageProvider, err := initStorage(context.Background(), cfg)
	if err != nil {
		logger.Error("Failed to initialize storage provider", observability.ErrorAttr(err))
		os.Exit(1)
	}

	// Build Pipeline
	p := pipeline.NewPipeline(cfg, llmProvider, videoProvider, storageProvider, runStore)

	// Handle Run Modes
	if cfg.RunMode == "once" {
		runOnce(p)
		return
	}

	if cfg.RunMode == "schedule" {
		runSchedule(cfg, p)
		return
	}

	logger.Error("Unknown RUN_MODE", slog.String("mode", cfg.RunMode))
	os.Exit(1)
}

func runOnce(p *pipeline.Pipeline) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture interrupt signals even in once mode for graceful cancellation
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Warn("Interrupt received, canceling pipeline execution...")
		cancel()
	}()

	run, err := p.Execute(ctx)
	if err != nil {
		slog.Error("Pipeline execution failed", observability.ErrorAttr(err))
		os.Exit(1)
	}

	fmt.Printf("\n=======================================================\n")
	fmt.Printf(" [SUCCESS] Content-Engine Pipeline Finished\n")
	fmt.Printf(" Run ID:      %s\n", run.ID)
	fmt.Printf(" Status:      %s\n", run.Status)
	if run.Brief != nil {
		fmt.Printf(" Title:       %s\n", run.Brief.Title)
		fmt.Printf(" Topic:       %s\n", run.Brief.Topic)
		fmt.Printf(" Duration:    %ds (%s)\n", run.Brief.DurationSeconds, run.Brief.AspectRatio)
	}
	fmt.Printf(" Provider Job:%s\n", run.ProviderJobID)
	fmt.Printf(" File Link:   %s\n", run.DriveFileLink)
	fmt.Printf("=======================================================\n\n")
}

func runSchedule(cfg *config.Config, p *pipeline.Pipeline) {
	sched, err := scheduler.NewScheduler(cfg.CronSchedule, cfg.CronLocation, p)
	if err != nil {
		slog.Error("Failed to initialize scheduler", observability.ErrorAttr(err))
		os.Exit(1)
	}

	sched.Start()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	slog.Info("Shutdown signal received", slog.String("signal", sig.String()))

	sched.Stop(cfg.ShutdownGracePeriod)
	slog.Info("Content-Engine shutdown completed cleanly")
}

func initStore(cfg *config.Config) (store.RunStore, error) {
	switch cfg.DBType {
	case "postgres":
		return store.NewPostgresStore(cfg.DatabaseURL)
	case "jsonl":
		return store.NewJSONLStore(cfg.JSONLPath)
	default:
		return store.NewSQLiteStore(cfg.SQLitePath)
	}
}

func initLLM(cfg *config.Config) content.LLMProvider {
	switch cfg.LLMProvider {
	case "gemini":
		return content.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.HTTPTimeout)
	default:
		return content.NewMockLLMClient()
	}
}

func initVideo(cfg *config.Config) video.VideoProvider {
	switch cfg.VideoProvider {
	case "gemini", "veo":
		return video.NewGeminiVeoClient(cfg.VideoAPIKey, cfg.VideoModel, cfg.HTTPTimeout)
	case "luma":
		return video.NewLumaClient(cfg.VideoAPIKey, cfg.HTTPTimeout)
	default:
		return video.NewMockVideoProvider(2)
	}
}

func initStorage(ctx context.Context, cfg *config.Config) (storage.StorageProvider, error) {
	switch cfg.StorageProvider {
	case "gdrive":
		return storage.NewGoogleDriveProvider(ctx, cfg.GDriveCredentialsFile, cfg.GDriveFolderID)
	default:
		return storage.NewLocalStorageProvider(cfg.LocalStorageDir), nil
	}
}
