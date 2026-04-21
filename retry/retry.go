package retry

import (
	"context"
	"time"
)

// defaultRetry is a retry configuration with the default values.
var defaultRetry = New(2)

// Option is retry option.
type Option func(*Retry)

// WithRetryable with retryable.
func WithRetryable(r Retryable) Option {
	return func(o *Retry) {
		o.retryable = r
	}
}

// WithBaseDelay overrides the initial wait duration.
func WithBaseDelay(d time.Duration) Option {
	return func(o *Retry) {
		if d > 0 {
			o.backoff.baseDelay = d
		}
	}
}

// WithMaxDelay overrides the max wait duration.
func WithMaxDelay(d time.Duration) Option {
	return func(o *Retry) {
		if d > 0 {
			o.backoff.maxDelay = d
		}
	}
}

// WithMultiplier overrides the exponential factor.
func WithMultiplier(m float64) Option {
	return func(o *Retry) {
		if m > 0 {
			o.backoff.mult = m
		}
	}
}

// WithJitter overrides the jitter factor.
func WithJitter(j float64) Option {
	return func(o *Retry) {
		if j >= 0 {
			o.backoff.jitter = j
		}
	}
}

// Retryable is used to judge whether an error is retryable or not.
type Retryable func(err error) bool

// Retry config.
type Retry struct {
	backoff   backoffConfig
	retryable Retryable
	attempts  int
}

// New new a retry with backoff.
func New(attempts int, opts ...Option) *Retry {
	r := &Retry{
		attempts:  attempts,
		retryable: func(err error) bool { return true },
		backoff:   defaultBackoff(),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Do wraps func with a backoff to retry.
func (r *Retry) Do(ctx context.Context, fn func(context.Context) error) error {
	var (
		err     error
		retries int
	)
	for {
		if err = ctx.Err(); err != nil {
			break
		}
		if err = fn(ctx); err == nil {
			break
		}
		if !r.retryable(err) {
			break
		}
		retries++
		if r.attempts > 0 && retries >= r.attempts {
			break
		}
		time.Sleep(r.backoff.duration(retries))
	}
	return err
}

// Do wraps func with a backoff to retry.
func Do(ctx context.Context, fn func(context.Context) error) error {
	return defaultRetry.Do(ctx, fn)
}

// Infinite wraps func with a backoff to retry.
func Infinite(ctx context.Context, fn func(context.Context) error) error {
	r := New(-1)
	return r.Do(ctx, fn)
}
