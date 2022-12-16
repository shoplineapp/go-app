package pulsar

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type BackoffPolicy struct {
	backoff time.Duration
	min     time.Duration
	max     time.Duration
}

const (
	jitterPercentage = 0.2
)

var (
	// From 500ms to 5min with 0.2 jitter
	DefaultBackoffPolicy = &BackoffPolicy{
		min: 500 * time.Millisecond,
		max: 5 * time.Minute,
	}
)

// Next returns the delay to wait before next retry
func (b *BackoffPolicy) Next() time.Duration {
	// Double the delay each time
	b.backoff += b.backoff
	if b.backoff.Nanoseconds() < b.min.Nanoseconds() {
		b.backoff = b.min
	} else if b.backoff.Nanoseconds() > b.max.Nanoseconds() {
		b.backoff = b.max
	}
	jitter := rand.Float64() * float64(b.backoff) * jitterPercentage

	return b.backoff + time.Duration(jitter)
}

// IsMaxBackReached evaluates if the max number of retries is reached
func (b *BackoffPolicy) IsMaxBackoffReached() bool {
	return b.backoff >= b.max
}
