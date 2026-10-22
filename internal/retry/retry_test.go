package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySuccessAfterTransientFailures(t *testing.T) {
	policy := Policy{
		MaxAttempts:     3,
		InitialInterval: 5 * time.Millisecond,
		MaxInterval:     20 * time.Millisecond,
		Multiplier:      2.0,
		JitterFraction:  0,
	}

	attempts := 0
	err := Do(context.Background(), policy, func(attempt int) error {
		attempts++
		if attempt < 2 {
			return errors.New("temporary 503 error")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryNonRetryableError(t *testing.T) {
	policy := Policy{
		MaxAttempts:     5,
		InitialInterval: 5 * time.Millisecond,
		MaxInterval:     20 * time.Millisecond,
	}

	attempts := 0
	err := Do(context.Background(), policy, func(attempt int) error {
		attempts++
		return MarkNonRetryable(errors.New("401 Unauthorized"))
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetryContextCancellation(t *testing.T) {
	policy := Policy{
		MaxAttempts:     5,
		InitialInterval: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := Do(ctx, policy, func(attempt int) error {
		return errors.New("network error")
	})

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context deadline/canceled error, got: %v", err)
	}
}
