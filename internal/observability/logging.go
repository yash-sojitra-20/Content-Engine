package observability

import (
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

var (
	apiKeyPattern = regexp.MustCompile(`(?i)(api[-_]?key|token|secret|password|bearer\s+)([=:\s])([^\s&,;]{4})[^\s&,;]*([^\s&,;]{4})`)
)

// SetupLogger initializes and registers the default slog.Logger based on level and format.
func SetupLogger(levelStr, formatStr string) *slog.Logger {
	return NewLogger(levelStr, formatStr, os.Stdout)
}

// NewLogger creates a new slog.Logger writing to the given writer.
func NewLogger(levelStr, formatStr string, out io.Writer) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Redact any potentially sensitive keys or values
			if isSensitiveKey(a.Key) {
				return slog.Attr{Key: a.Key, Value: slog.StringValue("[REDACTED]")}
			}
			if a.Value.Kind() == slog.KindString {
				return slog.Attr{Key: a.Key, Value: slog.StringValue(Sanitize(a.Value.String()))}
			}
			return a
		},
	}

	var handler slog.Handler
	if strings.ToLower(strings.TrimSpace(formatStr)) == "json" {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "key") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "password") ||
		strings.Contains(lower, "auth") ||
		strings.Contains(lower, "credential")
}

// Sanitize masks recognizable secrets inside text.
func Sanitize(s string) string {
	if s == "" {
		return s
	}
	// Mask occurrences of API keys, bearer tokens
	return apiKeyPattern.ReplaceAllString(s, `${1}${2}${3}...[REDACTED]...${4}`)
}

// Context attributes helpers
func RunIDAttr(id string) slog.Attr {
	return slog.String("run_id", id)
}

func StageAttr(stage string) slog.Attr {
	return slog.String("stage", stage)
}

func ProviderAttr(provider string) slog.Attr {
	return slog.String("provider", provider)
}

func AttemptAttr(attempt int) slog.Attr {
	return slog.Int("attempt", attempt)
}

func ErrorAttr(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.String("error", Sanitize(err.Error()))
}
