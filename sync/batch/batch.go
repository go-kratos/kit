package batch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kit/sync/safe"
)

// ErrBatcherClosed is returned by Add when the Batcher has been closed.
var ErrBatcherClosed = errors.New("batch: add on closed batcher")

// options holds the configuration for a Batcher.
type options[K comparable, V any] struct {
	maxSize       int
	flushInterval time.Duration
	shards        int
	shardKey      func(K) int
	onPanic       func(r any)
	flushFn       func(ctx context.Context, key K, items []V) error
}

// Option configures a Batcher.
type Option[K comparable, V any] func(*options[K, V])

// WithMaxSize sets the maximum number of items per key before a flush is triggered (default: 100).
func WithMaxSize[K comparable, V any](n int) Option[K, V] {
	return func(o *options[K, V]) {
		if n > 0 {
			o.maxSize = n
		}
	}
}

// WithFlushInterval sets the interval at which each shard flushes all pending items (default: 10ms).
func WithFlushInterval[K comparable, V any](d time.Duration) Option[K, V] {
	return func(o *options[K, V]) {
		if d > 0 {
			o.flushInterval = d
		}
	}
}

// WithShards sets the number of shards (default: 16).
func WithShards[K comparable, V any](n int) Option[K, V] {
	return func(o *options[K, V]) {
		if n > 0 {
			o.shards = n
		}
	}
}

// WithShardKey sets a custom shard-selection function.
// The function receives a key and returns an integer; the batcher maps it to a shard via abs(n) % numShards.
// Default: FNV-32a hash of the key.
func WithShardKey[K comparable, V any](fn func(K) int) Option[K, V] {
	return func(o *options[K, V]) {
		if fn != nil {
			o.shardKey = fn
		}
	}
}

// WithOnPanic sets a hook called when a flush function panics.
// r is the recovered panic value.
func WithOnPanic[K comparable, V any](fn func(r any)) Option[K, V] {
	return func(o *options[K, V]) {
		if fn != nil {
			o.onPanic = fn
		}
	}
}

// WithFlushFunc overrides the flush function set via New.
func WithFlushFunc[K comparable, V any](fn func(context.Context, K, []V) error) Option[K, V] {
	return func(o *options[K, V]) {
		if fn != nil {
			o.flushFn = fn
		}
	}
}

type item[K comparable, V any] struct {
	ctx context.Context
	key K
	val V
}

type shard[K comparable, V any] struct {
	ch chan item[K, V]
}

// Batcher groups items by key and flushes them together to a user-supplied function.
// Items with the same key always land in the same shard and the same flush call.
// Flush is triggered either when a key's item count reaches MaxSize or when the
// flush interval elapses. Close gracefully drains all pending items before stopping.
type Batcher[K comparable, V any] struct {
	shards []*shard[K, V]
	opts   options[K, V]
	closed atomic.Bool
	wg     sync.WaitGroup
}

// New creates a new Batcher. flushFn is required; it is called with (ctx, key, items)
// whenever a batch is ready. Panics if flushFn is nil after all options are applied.
func New[K comparable, V any](flushFn func(context.Context, K, []V) error, opts ...Option[K, V]) *Batcher[K, V] {
	o := options[K, V]{
		maxSize:       100,
		flushInterval: 10 * time.Millisecond,
		shards:        16,
		shardKey:      defaultShardKey[K],
		onPanic:       safe.DefaultPanicHandler,
		flushFn:       flushFn,
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.flushFn == nil {
		panic("batch: flushFn must not be nil")
	}
	b := &Batcher[K, V]{
		shards: make([]*shard[K, V], o.shards),
		opts:   o,
	}
	for i := range b.shards {
		b.shards[i] = &shard[K, V]{
			ch: make(chan item[K, V], o.maxSize*4),
		}
	}
	b.wg.Add(o.shards)
	for i := range b.shards {
		go b.runShard(i)
	}
	return b
}

// Add enqueues val under key to be flushed by a shard goroutine.
// Returns ErrBatcherClosed if the Batcher is closed.
// Returns ctx.Err() if ctx is already done before the item is enqueued.
func (b *Batcher[K, V]) Add(ctx context.Context, key K, val V) error {
	if b.closed.Load() {
		return ErrBatcherClosed
	}
	s := b.shardFor(key)
	select {
	case s.ch <- item[K, V]{ctx: ctx, key: key, val: val}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the Batcher from accepting new items, drains all pending items,
// and waits for all shard goroutines to finish. It is safe to call multiple times.
func (b *Batcher[K, V]) Close() error {
	if !b.closed.CompareAndSwap(false, true) {
		return ErrBatcherClosed
	}
	for _, s := range b.shards {
		close(s.ch)
	}
	b.wg.Wait()
	return nil
}

func (b *Batcher[K, V]) shardFor(key K) *shard[K, V] {
	idx := b.opts.shardKey(key)
	if idx < 0 {
		idx = -idx
	}
	return b.shards[idx%len(b.shards)]
}

func (b *Batcher[K, V]) runShard(idx int) {
	defer b.wg.Done()
	s := b.shards[idx]
	pending := make(map[K][]V)
	ctxFor := make(map[K]context.Context)
	ticker := time.NewTicker(b.opts.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case it, ok := <-s.ch:
			if !ok {
				b.flushAll(pending, ctxFor)
				return
			}
			pending[it.key] = append(pending[it.key], it.val)
			ctxFor[it.key] = it.ctx
			if len(pending[it.key]) >= b.opts.maxSize {
				b.flushKey(ctxFor[it.key], it.key, pending[it.key])
				delete(pending, it.key)
				delete(ctxFor, it.key)
			}
		case <-ticker.C:
			b.flushAll(pending, ctxFor)
			pending = make(map[K][]V)
			ctxFor = make(map[K]context.Context)
		}
	}
}

func (b *Batcher[K, V]) flushAll(pending map[K][]V, ctxFor map[K]context.Context) {
	for k, items := range pending {
		b.flushKey(ctxFor[k], k, items)
	}
}

func (b *Batcher[K, V]) flushKey(ctx context.Context, key K, items []V) {
	safe.DoWithRecover(func() {
		_ = b.opts.flushFn(ctx, key, items)
	}, b.opts.onPanic)
}
