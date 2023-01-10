package pulsar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoff_NextMinValue(t *testing.T) {
	backoff := &BackoffPolicy{
		min: 100 * time.Millisecond,
	}
	delay := backoff.Next()
	assert.GreaterOrEqual(t, int64(delay), int64(100*time.Millisecond))
	assert.LessOrEqual(t, int64(delay), int64(120*time.Millisecond))
}

func TestBackoff_NextExponentialBackoff(t *testing.T) {
	backoff := &BackoffPolicy{
		min: 100 * time.Millisecond,
		max: 3 * time.Second,
	}

	previousDelay := backoff.Next()
	for {
		delay := backoff.Next()
		// the jitter introduces at most 20% difference so at least delay is 1.6=(1-0.2)*2 bigger
		assert.GreaterOrEqual(t, int64(delay), int64(1.6*float64(previousDelay)))
		// the jitter introduces at most 20% difference so delay is less than twice the previous value
		assert.LessOrEqual(t, int64(float64(delay)*.8), int64(2*float64(previousDelay)))
		previousDelay = delay
		if previousDelay > 2*time.Second {
			break
		}
		assert.Equal(t, false, backoff.IsMaxBackoffReached())
	}
	assert.Equal(t, true, backoff.IsMaxBackoffReached())
}

func TestBackoff_NextMaxValue(t *testing.T) {
	backoff := &BackoffPolicy{
		max: 10 * time.Second,
	}
	var delay time.Duration
	for {
		delay = backoff.Next()
		if delay >= backoff.max {
			break
		}
	}

	cappedDelay := backoff.Next()
	assert.GreaterOrEqual(t, int64(cappedDelay), int64(backoff.max))
	assert.Equal(t, true, backoff.IsMaxBackoffReached())
	// max value is 60 seconds + 20% jitter = 72 seconds
	assert.LessOrEqual(t, int64(cappedDelay), int64(72*time.Second))
}
