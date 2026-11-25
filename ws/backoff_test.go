package ws

import (
	"testing"
	"time"
)

func fixedRand(v int64) func(int64) int64 { return func(n int64) int64 { return min(v, n-1) } }

func TestBackoffSchedule(t *testing.T) {
	b := newBackoff(Reconnect{WaitMinimum: 100 * time.Millisecond, RandomRange: 50 * time.Millisecond, RepeatTimes: 3})
	b.rand = fixedRand(7e6) // 7ms
	var got []time.Duration
	for i := 0; i < 7; i++ {
		got = append(got, b.next())
	}
	ms := time.Millisecond
	want := []time.Duration{107 * ms, 207 * ms, 407 * ms, 807 * ms, 807 * ms, 807 * ms, 807 * ms}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("wait %d = %v, want %v", i, got[i], want[i])
		}
	}
	b.reset()
	if w := b.next(); w != 107*ms {
		t.Errorf("after reset: %v", w)
	}
}

func TestBackoffEdgeCases(t *testing.T) {
	// No random range, no doubling.
	b := newBackoff(Reconnect{WaitMinimum: time.Second})
	for i := 0; i < 3; i++ {
		if w := b.next(); w != time.Second {
			t.Errorf("constant wait: %v", w)
		}
	}
	// Zero value selects the defaults.
	if d := newBackoff(Reconnect{}); d.cfg != DefaultReconnect {
		t.Errorf("defaults: %+v", d.cfg)
	}
	// Random part stays within bounds, using the real random source.
	r := newBackoff(Reconnect{WaitMinimum: time.Second, RandomRange: 500 * time.Millisecond})
	for i := 0; i < 200; i++ {
		if w := r.next(); w < time.Second || w > 1500*time.Millisecond {
			t.Fatalf("out of range: %v", w)
		}
		r.reset()
	}
	// Doubling never overflows.
	o := newBackoff(Reconnect{WaitMinimum: time.Hour, RepeatTimes: 200})
	var last time.Duration
	for i := 0; i < 250; i++ {
		w := o.next()
		if w <= 0 || w < last {
			t.Fatalf("overflow at step %d: %v after %v", i, w, last)
		}
		last = w
	}
	if last != maxWait {
		t.Errorf("should saturate at %v, got %v", maxWait, last)
	}
}
