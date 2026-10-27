package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config represents all application configuration parameters loaded from environment variables.
type Config struct {
	// Execution mode: "once" or "schedule"
	RunMode string

	// Scheduling parameters
	CronSchedule string
	CronTimezone string
	CronLocation *time.Location

	// LLM settings
	LLMProvider        string
	GeminiAPIKey       string
	GeminiModel        string
	ContentTopicPrompt string

	// Video generation settings
	VideoProvider     string
	VideoAPIKey       string
	VideoModel        string
	VideoPollInterval time.Duration
	VideoMaxWait      time.Duration

	// Storage settings
	StorageProvider       string
	LocalStorageDir       string
	GDriveCredentialsFile string
	GDriveFolderID        string

	// Run history persistence settings
	DBType      string // "sqlite", "postgres", "jsonl"
	SQLitePath  string
	DatabaseURL string
	JSONLPath   string

	// Reliability, timeouts & retry parameters
	HTTPTimeout          time.Duration
	MaxRetries           int
	RetryInitialInterval time.Duration
	RetryMaxInterval     time.Duration
	TmpDir               string
	ShutdownGracePeriod  time.Duration

	// Observability & logging
	LogLevel  string // "debug", "info", "warn", "error"
	LogFormat string // "text", "json"
}

// Load reads configuration from the environment and optional .env file, applying sensible defaults.
func Load() (*Config, error) {
	// Attempt to load .env file if it exists, ignore error if missing
	_ = godotenv.Load()

	cfg := &Config{
		RunMode:              getEnvOrDefault("RUN_MODE", "once"),
		CronSchedule:         getEnvOrDefault("CRON_SCHEDULE", "0 */6 * * *"),
		CronTimezone:         getEnvOrDefault("CRON_TIMEZONE", "UTC"),
		LLMProvider:          strings.ToLower(getEnvOrDefault("LLM_PROVIDER", "gemini")),
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModel:          getEnvOrDefault("GEMINI_MODEL", "gemini-1.5-flash"),
		ContentTopicPrompt:   getEnvOrDefault("CONTENT_TOPIC_PROMPT", "Create a captivating 10-second tech or AI micro-fact video suitable for social reels and shorts."),
		VideoProvider:        strings.ToLower(getEnvOrDefault("VIDEO_PROVIDER", "mock")),
		VideoAPIKey:          os.Getenv("VIDEO_API_KEY"),
		VideoModel:           getEnvOrDefault("VIDEO_MODEL", "veo-2.0-generate-001"),
		VideoPollInterval:    getDurationOrDefault("VIDEO_POLL_INTERVAL", 10*time.Second),
		VideoMaxWait:         getDurationOrDefault("VIDEO_MAX_WAIT", 10*time.Minute),
		StorageProvider:      strings.ToLower(getEnvOrDefault("STORAGE_PROVIDER", "local")),
		LocalStorageDir:      getEnvOrDefault("LOCAL_STORAGE_DIR", "data/videos"),
		GDriveCredentialsFile: os.Getenv("GDRIVE_CREDENTIALS_FILE"),
		GDriveFolderID:       os.Getenv("GDRIVE_FOLDER_ID"),
		DBType:               strings.ToLower(getEnvOrDefault("DB_TYPE", "sqlite")),
		SQLitePath:           getEnvOrDefault("SQLITE_PATH", "data/runs.db"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		JSONLPath:            getEnvOrDefault("JSONL_PATH", "data/runs.jsonl"),
		HTTPTimeout:          getDurationOrDefault("HTTP_TIMEOUT", 60*time.Second),
		MaxRetries:           getIntOrDefault("MAX_RETRIES", 3),
		RetryInitialInterval: getDurationOrDefault("RETRY_INITIAL_INTERVAL", 2*time.Second),
		RetryMaxInterval:     getDurationOrDefault("RETRY_MAX_INTERVAL", 30*time.Second),
		TmpDir:               getEnvOrDefault("TMP_DIR", "tmp"),
		ShutdownGracePeriod:  getDurationOrDefault("SHUTDOWN_GRACE_PERIOD", 30*time.Second),
		LogLevel:             strings.ToLower(getEnvOrDefault("LOG_LEVEL", "info")),
		LogFormat:            strings.ToLower(getEnvOrDefault("LOG_FORMAT", "text")),
	}

	loc, err := time.LoadLocation(cfg.CronTimezone)
	if err != nil {
		return nil, fmt.Errorf("invalid CRON_TIMEZONE %q: %w", cfg.CronTimezone, err)
	}
	cfg.CronLocation = loc

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks whether the configuration is logically sound and required secrets exist.
func (c *Config) Validate() error {
	var errs []string

	// Validate RunMode
	if c.RunMode != "once" && c.RunMode != "schedule" {
		errs = append(errs, fmt.Sprintf("RUN_MODE must be 'once' or 'schedule', got %q", c.RunMode))
	}

	// Validate Schedule
	if c.RunMode == "schedule" && strings.TrimSpace(c.CronSchedule) == "" {
		errs = append(errs, "CRON_SCHEDULE cannot be empty when RUN_MODE is 'schedule'")
	}

	// Validate LLM Provider
	switch c.LLMProvider {
	case "gemini":
		if strings.TrimSpace(c.GeminiAPIKey) == "" {
			errs = append(errs, "GEMINI_API_KEY is required when LLM_PROVIDER is 'gemini'")
		}
	case "mock":
		// Mock doesn't require keys
	default:
		errs = append(errs, fmt.Sprintf("unsupported LLM_PROVIDER %q (supported: 'gemini', 'mock')", c.LLMProvider))
	}

	// Validate Video Provider
	switch c.VideoProvider {
	case "gemini", "veo":
		// For gemini/veo video, fallback to GEMINI_API_KEY if VIDEO_API_KEY is unset
		if strings.TrimSpace(c.VideoAPIKey) == "" && strings.TrimSpace(c.GeminiAPIKey) != "" {
			c.VideoAPIKey = c.GeminiAPIKey
		}
		if strings.TrimSpace(c.VideoAPIKey) == "" {
			errs = append(errs, "VIDEO_API_KEY (or GEMINI_API_KEY) is required when VIDEO_PROVIDER is 'gemini'")
		}
	case "luma":
		if strings.TrimSpace(c.VideoAPIKey) == "" {
			errs = append(errs, "VIDEO_API_KEY is required when VIDEO_PROVIDER is 'luma'")
		}
	case "mock":
		// Mock doesn't require keys
	default:
		errs = append(errs, fmt.Sprintf("unsupported VIDEO_PROVIDER %q (supported: 'gemini', 'luma', 'mock')", c.VideoProvider))
	}

	// Validate Storage Provider
	switch c.StorageProvider {
	case "gdrive":
		if strings.TrimSpace(c.GDriveCredentialsFile) == "" {
			errs = append(errs, "GDRIVE_CREDENTIALS_FILE is required when STORAGE_PROVIDER is 'gdrive'")
		}
		if strings.TrimSpace(c.GDriveFolderID) == "" {
			errs = append(errs, "GDRIVE_FOLDER_ID is required when STORAGE_PROVIDER is 'gdrive'")
		}
	case "local", "mock":
		// No cloud credentials needed
	default:
		errs = append(errs, fmt.Sprintf("unsupported STORAGE_PROVIDER %q (supported: 'gdrive', 'local', 'mock')", c.StorageProvider))
	}

	// Validate DB Type
	switch c.DBType {
	case "sqlite":
		if strings.TrimSpace(c.SQLitePath) == "" {
			errs = append(errs, "SQLITE_PATH cannot be empty when DB_TYPE is 'sqlite'")
		}
	case "postgres":
		if strings.TrimSpace(c.DatabaseURL) == "" {
			errs = append(errs, "DATABASE_URL is required when DB_TYPE is 'postgres'")
		}
	case "jsonl":
		if strings.TrimSpace(c.JSONLPath) == "" {
			errs = append(errs, "JSONL_PATH cannot be empty when DB_TYPE is 'jsonl'")
		}
	default:
		errs = append(errs, fmt.Sprintf("unsupported DB_TYPE %q (supported: 'sqlite', 'postgres', 'jsonl')", c.DBType))
	}

	// Validate limits
	if c.HTTPTimeout <= 0 {
		errs = append(errs, "HTTP_TIMEOUT must be greater than zero")
	}
	if c.MaxRetries < 0 {
		errs = append(errs, "MAX_RETRIES cannot be negative")
	}
	if c.VideoPollInterval <= 0 {
		errs = append(errs, "VIDEO_POLL_INTERVAL must be greater than zero")
	}
	if c.VideoMaxWait <= 0 {
		errs = append(errs, "VIDEO_MAX_WAIT must be greater than zero")
	}

	if len(errs) > 0 {
		return errors.New("configuration validation failed:\n - " + strings.Join(errs, "\n - "))
	}
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

func getDurationOrDefault(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return d
}

func getIntOrDefault(key string, fallback int) int {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
