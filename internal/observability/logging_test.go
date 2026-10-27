package observability

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestLoggerRedaction(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("info", "json", &buf)

	logger.Info("User request", "api_key", "secret1234567890", "message", "Calling API with Bearer mysecretkey9999 successfully")

	output := buf.String()

	// Ensure sensitive key is redacted
	if strings.Contains(output, "secret1234567890") {
		t.Errorf("expected secret to be redacted, got output: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in output, got: %s", output)
	}
}

func TestContextAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("debug", "text", &buf)

	err := errors.New("connection failed with api_key=supersecret123")
	logger.Error("Step failed",
		RunIDAttr("run-abc-123"),
		StageAttr("video_generation"),
		ProviderAttr("gemini"),
		AttemptAttr(2),
		ErrorAttr(err),
	)

	out := buf.String()
	if !strings.Contains(out, "run_id=run-abc-123") {
		t.Errorf("missing run_id: %s", out)
	}
	if !strings.Contains(out, "stage=video_generation") {
		t.Errorf("missing stage: %s", out)
	}
	if !strings.Contains(out, "provider=gemini") {
		t.Errorf("missing provider: %s", out)
	}
	if !strings.Contains(out, "attempt=2") {
		t.Errorf("missing attempt: %s", out)
	}
}
