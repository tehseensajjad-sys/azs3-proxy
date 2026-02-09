package server

import (
	"context"
	"sync"
	"time"
)

// adaptiveConcurrencyLimiter adjusts allowable concurrent requests based on observed latency and errors.
// It uses a simple additive-increase/decrease strategy guided by an EMA of latency relative to a target.
type adaptiveConcurrencyLimiter struct {
	mu       sync.Mutex
	cond     *sync.Cond
	limit    int
	min      int
	max      int
	inFlight int

	target   time.Duration
	emaNanos float64
	alpha    float64
}

func newAdaptiveConcurrencyLimiter(min, max int, target time.Duration) *adaptiveConcurrencyLimiter {
	if min < 1 {
		min = 1
	}
	if max < min {
		max = min
	}
	if target <= 0 {
		target = 200 * time.Millisecond
	}
	l := &adaptiveConcurrencyLimiter{
		limit:  min,
		min:    min,
		max:    max,
		target: target,
		alpha:  0.2,
	}
	l.cond = sync.NewCond(&l.mu)
	return l
}

func (l *adaptiveConcurrencyLimiter) Acquire(ctx context.Context) error {
	l.mu.Lock()

	for l.inFlight >= l.limit {
		if ctx.Err() != nil {
			l.mu.Unlock()
			return ctx.Err()
		}

		if done := ctx.Done(); done != nil {
			l.mu.Unlock()
			select {
			case <-done:
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
				l.mu.Lock()
				continue
			}
		}

		l.cond.Wait()
	}

	l.inFlight++
	l.mu.Unlock()
	return nil
}

func (l *adaptiveConcurrencyLimiter) Release(duration time.Duration, success bool) {
	l.mu.Lock()

	if l.inFlight > 0 {
		l.inFlight--
	}

	if duration > 0 {
		d := float64(duration)
		if l.emaNanos == 0 {
			l.emaNanos = d
		} else {
			l.emaNanos = l.alpha*d + (1-l.alpha)*l.emaNanos
		}
	}

	target := float64(l.target)
	if target <= 0 {
		target = float64(200 * time.Millisecond)
	}

	switch {
	case !success:
		if l.limit > l.min {
			l.limit--
		}
	case l.emaNanos > 0 && l.emaNanos > 1.5*target:
		if l.limit > l.min {
			l.limit--
		}
	case l.emaNanos > 0 && l.emaNanos < 0.7*target:
		if l.limit < l.max {
			l.limit++
		}
	}

	l.cond.Broadcast()
	l.mu.Unlock()
}

func (l *adaptiveConcurrencyLimiter) CurrentLimit() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

func (l *adaptiveConcurrencyLimiter) InFlight() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inFlight
}
