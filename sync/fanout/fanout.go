package fanout

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/go-kratos/kit/sync/safe"
)

// ErrFanoutClosed is returned by Do when the Fanout has been closed.
var ErrFanoutClosed = errors.New("fanout: send on closed fanout")

// options holds the configuration for a Fanout.
type options struct {
	worker  int
	buffer  int
	onPanic func(r any)
}

// Option configures a Fanout.
type Option func(*options)

// WithWorker sets the number of worker goroutines (default: 1).
func WithWorker(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.worker = n
		}
	}
}

// WithBuffer sets the task channel buffer size (default: 1000).
func WithBuffer(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.buffer = n
		}
	}
}

// WithOnPanic sets a hook called when a task panics.
// r is the recovered panic value.
func WithOnPanic(fn func(r any)) Option {
	return func(o *options) {
		o.onPanic = fn
	}
}

type task struct {
	fn  func(context.Context)
	ctx context.Context
}

// Fanout dispatches tasks to a pool of worker goroutines asynchronously.
// Close gracefully drains all pending tasks before stopping workers.
type Fanout struct {
	ch     chan task
	closed atomic.Bool
	wg     sync.WaitGroup
	opts   options
}

// New creates a new Fanout with the given options.
func New(opts ...Option) *Fanout {
	o := options{
		worker:  1,
		buffer:  1000,
		onPanic: safe.DefaultPanicHandler,
	}
	for _, opt := range opts {
		opt(&o)
	}
	f := &Fanout{
		ch:   make(chan task, o.buffer),
		opts: o,
	}
	f.wg.Add(o.worker)
	for i := 0; i < o.worker; i++ {
		go f.proc()
	}
	return f
}

func (f *Fanout) proc() {
	defer f.wg.Done()
	for t := range f.ch {
		f.execute(t)
	}
}

// execute runs the task and recovers from any panic, calling the onPanic hook if set.
func (f *Fanout) execute(t task) {
	safe.DoWithRecover(func() { t.fn(t.ctx) }, f.opts.onPanic)
}

// Do enqueues fn to be executed by a worker goroutine.
// Returns ErrFanoutClosed if the Fanout is closed.
// Returns ctx.Err() if ctx is already done before the task is enqueued.
func (f *Fanout) Do(ctx context.Context, fn func(context.Context)) error {
	if fn == nil {
		return nil
	}
	if f.closed.Load() {
		return ErrFanoutClosed
	}
	select {
	case f.ch <- task{fn: fn, ctx: ctx}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the Fanout from accepting new tasks, drains all pending tasks,
// and waits for all workers to finish. It is safe to call multiple times.
func (f *Fanout) Close() error {
	if !f.closed.CompareAndSwap(false, true) {
		return ErrFanoutClosed
	}
	close(f.ch)
	f.wg.Wait()
	return nil
}
