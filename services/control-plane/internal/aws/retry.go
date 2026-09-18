package aws

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"time"

	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

var ErrInvalidRetryOperation = errors.New("retry operation is required")

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Sleep       func(context.Context, time.Duration) error
	Jitter      func(int64) int64
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
	}
}

func Retry(ctx context.Context, policy RetryPolicy, operation func(context.Context) error) error {
	if operation == nil {
		return ErrInvalidRetryOperation
	}
	if ctx == nil {
		ctx = context.Background()
	}
	policy = policy.withDefaults()

	var lastErr error
	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = operation(ctx)
		if lastErr == nil || !IsRetryable(lastErr) || attempt == policy.MaxAttempts-1 {
			return lastErr
		}
		if err := policy.Sleep(ctx, policy.delay(attempt)); err != nil {
			return err
		}
	}
	return lastErr
}

func IsRetryable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var responseError *smithyhttp.ResponseError
	if errors.As(err, &responseError) {
		switch responseError.HTTPStatusCode() {
		case 408, 429, 500, 502, 503, 504:
			return true
		}
	}

	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "Throttling", "ThrottlingException", "TooManyRequestsException",
			"ProvisionedThroughputExceededException", "RequestLimitExceeded", "SlowDown",
			"RequestTimeout", "ServiceUnavailableException", "InternalError",
			"InternalServerError", "ServerException":
			return true
		}
		return apiError.ErrorFault() == smithy.FaultServer
	}

	var networkError net.Error
	return errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary())
}

func (policy RetryPolicy) withDefaults() RetryPolicy {
	defaults := DefaultRetryPolicy()
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = defaults.MaxAttempts
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = defaults.BaseDelay
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = defaults.MaxDelay
	}
	if policy.MaxDelay < policy.BaseDelay {
		policy.MaxDelay = policy.BaseDelay
	}
	if policy.Sleep == nil {
		policy.Sleep = sleep
	}
	if policy.Jitter == nil {
		policy.Jitter = func(max int64) int64 {
			if max <= 0 {
				return 0
			}
			return rand.Int63n(max)
		}
	}
	return policy
}

func (policy RetryPolicy) delay(attempt int) time.Duration {
	delay := policy.BaseDelay
	for index := 0; index < attempt && delay < policy.MaxDelay; index++ {
		delay *= 2
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}
	jitter := time.Duration(policy.Jitter(int64(delay)))
	if delay+jitter > policy.MaxDelay {
		return policy.MaxDelay
	}
	return delay + jitter
}

func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
