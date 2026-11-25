package ws

import (
	"math"
	"math/rand/v2"
	"time"
)

// Reconnect configures the reconnect back-off of a Client, following OCPP-J
// 2.0.1 §5.4. The field names mirror the standard configuration variables.
//
// The first wait is WaitMinimum plus a random value of up to RandomRange.
// After each failed attempt the wait doubles, at most RepeatTimes times, and a
// new random value is added each time. After a successful connection the
// schedule starts over.
type Reconnect struct {
	// WaitMinimum is RetryBackOffWaitMinimum.
	WaitMinimum time.Duration
	// RandomRange is RetryBackOffRandomRange.
	RandomRange time.Duration
	// RepeatTimes is RetryBackOffRepeatTimes: how many times the wait doubles.
	RepeatTimes int
}

// DefaultReconnect is used when ClientConfig.Reconnect is the zero value.
var DefaultReconnect = Reconnect{WaitMinimum: time.Second, RandomRange: 2 * time.Second, RepeatTimes: 6}

// maxWait stops the doubling from overflowing.
const maxWait = time.Duration(math.MaxInt64 / 4)

type backoff struct {
	cfg     Reconnect
	attempt int
	rand    func(n int64) int64 // uniform in [0, n)
}

func newBackoff(cfg Reconnect) *backoff {
	if cfg == (Reconnect{}) {
		cfg = DefaultReconnect
	}
	return &backoff{cfg: cfg, rand: rand.Int64N}
}

// next returns how long to wait before the next connection attempt.
func (b *backoff) next() time.Duration {
	wait := b.cfg.WaitMinimum
	for i := 0; i < b.attempt && i < b.cfg.RepeatTimes; i++ {
		if wait > maxWait/2 {
			wait = maxWait
			break
		}
		wait *= 2
	}
	if b.attempt < b.cfg.RepeatTimes {
		b.attempt++
	}
	if b.cfg.RandomRange > 0 {
		wait += time.Duration(b.rand(int64(b.cfg.RandomRange) + 1))
	}
	return wait
}

// reset starts the schedule over after a successful connection.
func (b *backoff) reset() { b.attempt = 0 }
