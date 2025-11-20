package subscriber

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/aws/smithy-go"
)

const (
	cloudWatchAPIMaxAttempts = 10
	cloudWatchRetryBaseDelay = 300 * time.Millisecond
	cloudWatchRetryMaxDelay  = 10 * time.Second
)

func isRetryableCloudWatchError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ThrottlingException",
			"ThrottledException",
			"TooManyRequestsException",
			"ServiceUnavailableException",
			"RequestLimitExceeded",
			"OperationAbortedException":
			return true
		}
	}
	return false
}

func sleepWithBackoff(ctx context.Context, attempt int) error {
	delay := cloudWatchRetryBaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= cloudWatchRetryMaxDelay {
			delay = cloudWatchRetryMaxDelay
			break
		}
	}

	// Add jitter to avoid synchronized retries across workers/invocations.
	jitter := 0.5 + rand.Float64() // [0.5, 1.5)
	delay = time.Duration(float64(delay) * jitter)
	if delay > cloudWatchRetryMaxDelay {
		delay = cloudWatchRetryMaxDelay
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
