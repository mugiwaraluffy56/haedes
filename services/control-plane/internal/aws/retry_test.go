package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/smithy-go"
)

func TestRetryRetriesOnlyTransientErrorsWithBoundedJitter(t *testing.T) {
	var attempts int
	var delays []time.Duration
	policy := RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    15 * time.Millisecond,
		Jitter:      func(max int64) int64 { return max / 2 },
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	}
	err := Retry(context.Background(), policy, func(context.Context) error {
		attempts++
		if attempts < 3 {
			return &smithy.GenericAPIError{Code: "ThrottlingException", Fault: smithy.FaultServer}
		}
		return nil
	})
	if err != nil || attempts != 3 {
		t.Fatalf("retry result = %v, attempts = %d", err, attempts)
	}
	if len(delays) != 2 || delays[0] != 15*time.Millisecond || delays[1] != 15*time.Millisecond {
		t.Fatalf("delays = %v", delays)
	}
}

func TestRetryDoesNotRetryConflictsOrValidationErrors(t *testing.T) {
	for _, err := range []error{
		&smithy.GenericAPIError{Code: "ConditionalCheckFailedException"},
		&smithy.GenericAPIError{Code: "ValidationException"},
	} {
		attempts := 0
		got := Retry(context.Background(), RetryPolicy{Sleep: func(context.Context, time.Duration) error { return nil }}, func(context.Context) error {
			attempts++
			return err
		})
		if !errors.Is(got, err) || attempts != 1 {
			t.Fatalf("error = %v, attempts = %d", got, attempts)
		}
	}
}
