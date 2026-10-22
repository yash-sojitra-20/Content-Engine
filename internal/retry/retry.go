package retry

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// Policy defines bounded exponential backoff parameters.
type Policy struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	JitterFraction  float64
}

// DefaultPolicy returns a standard resilient backoff policy.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:     3,
		InitialInterval: 2 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		JitterFraction:  0.2,
	}
}

// RetryAfterProvider is an interface for errors that specify a Retry-After delay.
type RetryAfterProvider interface {
	RetryAfter() time.Duration
}

// NonRetryableError wraps an error that should not be retried.
type NonRetryableError struct {
	Err error
}

func (e *NonRetryableError) Error() string {
	return e.Err.Error()
}

func (e *NonRetryableError) Unwrap() error {
	return e.Err
}

// MarkNonRetryable wraps an error so the retry runner will stop immediately.
func MarkNonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &NonRetryableError{Err: err}
}

// IsRetryable determines whether an error is eligible for retry.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var nonRetryable *NonRetryableError
	if errors.As(err, &nonRetryable) {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// Do executes operation fn with bounded exponential backoff.
func Do(ctx context.Context, policy Policy, fn func(attempt int) error) error {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}
	if policy.InitialInterval <= 0 {
		policy.InitialInterval = 1 * time.Second
	}
	if policy.MaxInterval <= 0 {
		policy.MaxInterval = 30 * time.Second
	}
	if policy.Multiplier <= 1.0 {
		policy.Multiplier = 2.0
	}

	interval := policy.InitialInterval
	var lastErr error

	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		// Check context before executing
		if err := ctx.Err(); err != nil {
			return err
		}

		err := fn(attempt)
		if err == nil {
			return nil
		}
		lastErr = err

		if !IsRetryable(err) {
			return err
		}

		if attempt >= policy.MaxAttempts {
			break
		}

		// Check if error specifies Retry-After
		sleepDuration := interval
		var ra RetryAfterProvider
		if errors.As(err, &ra) && ra.RetryAfter() > 0 {
			sleepDuration = ra.RetryAfter()
		} else {
			// Apply jitter
			if policy.JitterFraction > 0 {
				jitterRange := float64(sleepDuration) * policy.JitterFraction
				jitterDelta := (rand.Float64()*2 - 1) * jitterRange
				sleepDuration = time.Duration(float64(sleepDuration) + jitterDelta)
			}
			// Compute next interval with exponential growth
			interval = time.Duration(float64(interval) * policy.Multiplier)
			if interval > policy.MaxInterval {
				interval = policy.MaxInterval
			}
		}

		// Sleep or wait for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDuration):
		}
	}

	return lastErr
}

// ParseRetryAfterHeader attempts to parse HTTP Retry-After header (seconds or HTTP date).
func ParseRetryAfterHeader(header string) time.Duration {
	if header == "" {
		return 0
	}
	// Try parsing as integer seconds
	if secs, err := strconv.Atoi(header); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	// Try parsing as RFC1123 HTTP date
	if t, err := http.ParseTime(header); err == nil {
		diff := time.Until(t)
		if diff > 0 {
			return diff
		}
	}
	return 0
}
