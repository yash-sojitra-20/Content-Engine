package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError bool
	}{
		{
			name: "Valid default config with mock providers",
			cfg: Config{
				RunMode:           "once",
				LLMProvider:       "mock",
				VideoProvider:     "mock",
				StorageProvider:   "local",
				DBType:            "sqlite",
				SQLitePath:        "data/runs.db",
				HTTPTimeout:       30 * time.Second,
				MaxRetries:        3,
				VideoPollInterval: 5 * time.Second,
				VideoMaxWait:      1 * time.Minute,
			},
			wantError: false,
		},
		{
			name: "Invalid RunMode",
			cfg: Config{
				RunMode:           "infinite",
				LLMProvider:       "mock",
				VideoProvider:     "mock",
				StorageProvider:   "local",
				DBType:            "sqlite",
				SQLitePath:        "data/runs.db",
				HTTPTimeout:       30 * time.Second,
				VideoPollInterval: 5 * time.Second,
				VideoMaxWait:      1 * time.Minute,
			},
			wantError: true,
		},
		{
			name: "Schedule mode without schedule expression",
			cfg: Config{
				RunMode:           "schedule",
				CronSchedule:      "",
				LLMProvider:       "mock",
				VideoProvider:     "mock",
				StorageProvider:   "local",
				DBType:            "sqlite",
				SQLitePath:        "data/runs.db",
				HTTPTimeout:       30 * time.Second,
				VideoPollInterval: 5 * time.Second,
				VideoMaxWait:      1 * time.Minute,
			},
			wantError: true,
		},
		{
			name: "Gemini LLM without API key",
			cfg: Config{
				RunMode:           "once",
				LLMProvider:       "gemini",
				GeminiAPIKey:      "",
				VideoProvider:     "mock",
				StorageProvider:   "local",
				DBType:            "sqlite",
				SQLitePath:        "data/runs.db",
				HTTPTimeout:       30 * time.Second,
				VideoPollInterval: 5 * time.Second,
				VideoMaxWait:      1 * time.Minute,
			},
			wantError: true,
		},
		{
			name: "Luma video without API key",
			cfg: Config{
				RunMode:           "once",
				LLMProvider:       "mock",
				VideoProvider:     "luma",
				VideoAPIKey:       "",
				StorageProvider:   "local",
				DBType:            "sqlite",
				SQLitePath:        "data/runs.db",
				HTTPTimeout:       30 * time.Second,
				VideoPollInterval: 5 * time.Second,
				VideoMaxWait:      1 * time.Minute,
			},
			wantError: true,
		},
		{
			name: "GDrive storage without folder or credentials",
			cfg: Config{
				RunMode:           "once",
				LLMProvider:       "mock",
				VideoProvider:     "mock",
				StorageProvider:   "gdrive",
				DBType:            "sqlite",
				SQLitePath:        "data/runs.db",
				HTTPTimeout:       30 * time.Second,
				VideoPollInterval: 5 * time.Second,
				VideoMaxWait:      1 * time.Minute,
			},
			wantError: true,
		},
		{
			name: "Postgres without DATABASE_URL",
			cfg: Config{
				RunMode:           "once",
				LLMProvider:       "mock",
				VideoProvider:     "mock",
				StorageProvider:   "local",
				DBType:            "postgres",
				DatabaseURL:       "",
				HTTPTimeout:       30 * time.Second,
				VideoPollInterval: 5 * time.Second,
				VideoMaxWait:      1 * time.Minute,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	// Set minimal env to satisfy validation
	os.Setenv("LLM_PROVIDER", "mock")
	os.Setenv("VIDEO_PROVIDER", "mock")
	os.Setenv("STORAGE_PROVIDER", "local")
	os.Setenv("DB_TYPE", "sqlite")
	defer func() {
		os.Unsetenv("LLM_PROVIDER")
		os.Unsetenv("VIDEO_PROVIDER")
		os.Unsetenv("STORAGE_PROVIDER")
		os.Unsetenv("DB_TYPE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.RunMode != "once" {
		t.Errorf("expected RunMode 'once', got %q", cfg.RunMode)
	}
	if cfg.CronTimezone != "UTC" || cfg.CronLocation == nil {
		t.Errorf("expected UTC location, got %v", cfg.CronLocation)
	}
}
