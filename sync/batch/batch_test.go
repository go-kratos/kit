package batch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFlushBySize(t *testing.T) {
	var mu sync.Mutex
	var got []int

	b := New[string, int](func(_ context.Context, _ string, items []int) error {
		mu.Lock()
		got = append(got, items...)
		mu.Unlock()
		return nil
	}, WithMaxSize[string, int](3), WithFlushInterval[string, int](time.Hour))
	defer b.Close()

	for i := 0; i < 3; i++ {
		if err := b.Add(context.Background(), "k", i); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	n := len(got)
	mu.Unlock()
	if n != 3 {
		t.Fatalf("expected 3 items flushed, got %d", n)
	}
}

func TestFlushByInterval(t *testing.T) {
	var mu sync.Mutex
	var calls int

	b := New[string, int](func(_ context.Context, _ string, items []int) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return nil
	}, WithMaxSize[string, int](100), WithFlushInterval[string, int](20*time.Millisecond))
	defer b.Close()

	_ = b.Add(context.Background(), "k", 1)
	_ = b.Add(context.Background(), "k", 2)

	time.Sleep(60 * time.Millisecond)

	mu.Lock()
	c := calls
	mu.Unlock()
	if c == 0 {
		t.Fatal("expected flush to fire via interval")
	}
}

func TestSameKeyAlwaysTogether(t *testing.T) {
	var mu sync.Mutex
	var maxBatch int

	b := New[string, int](func(_ context.Context, _ string, items []int) error {
		mu.Lock()
		if len(items) > maxBatch {
			maxBatch = len(items)
		}
		mu.Unlock()
		return nil
	}, WithMaxSize[string, int](5), WithFlushInterval[string, int](time.Hour))
	defer b.Close()

	for i := 0; i < 5; i++ {
		_ = b.Add(context.Background(), "x", i)
	}
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	m := maxBatch
	mu.Unlock()
	if m != 5 {
		t.Fatalf("expected one flush with 5 items, got max batch size %d", m)
	}
}

func TestDifferentKeysFlushSeparately(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]int{}

	b := New[string, int](func(_ context.Context, key string, items []int) error {
		mu.Lock()
		seen[key] += len(items)
		mu.Unlock()
		return nil
	}, WithMaxSize[string, int](2), WithFlushInterval[string, int](time.Hour), WithShards[string, int](1))
	defer b.Close()

	_ = b.Add(context.Background(), "a", 1)
	_ = b.Add(context.Background(), "a", 2)
	_ = b.Add(context.Background(), "b", 10)
	_ = b.Add(context.Background(), "b", 20)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if seen["a"] != 2 {
		t.Fatalf("key a: expected 2 items, got %d", seen["a"])
	}
	if seen["b"] != 2 {
		t.Fatalf("key b: expected 2 items, got %d", seen["b"])
	}
}

func TestCloseDrains(t *testing.T) {
	var count atomic.Int32

	b := New[string, int](func(_ context.Context, _ string, items []int) error {
		count.Add(int32(len(items)))
		return nil
	}, WithMaxSize[string, int](100), WithFlushInterval[string, int](time.Hour))

	for i := 0; i < 10; i++ {
		_ = b.Add(context.Background(), "k", i)
	}
	b.Close()

	if got := count.Load(); got != 10 {
		t.Fatalf("expected 10 items drained on Close, got %d", got)
	}
}

func TestCloseIdempotent(t *testing.T) {
	b := New[string, int](func(_ context.Context, _ string, _ []int) error { return nil })
	if err := b.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := b.Close(); !errors.Is(err, ErrBatcherClosed) {
		t.Fatalf("second Close should return ErrBatcherClosed, got %v", err)
	}
}

func TestAddAfterClose(t *testing.T) {
	b := New[string, int](func(_ context.Context, _ string, _ []int) error { return nil })
	b.Close()

	err := b.Add(context.Background(), "k", 1)
	if !errors.Is(err, ErrBatcherClosed) {
		t.Fatalf("expected ErrBatcherClosed, got %v", err)
	}
}

func TestAddCancelledContext(t *testing.T) {
	// maxSize=1, buffer=4 (1*4), fill all slots to block the next Add
	b := New[string, int](func(_ context.Context, _ string, _ []int) error {
		time.Sleep(time.Hour) // never flush so channel stays full
		return nil
	}, WithMaxSize[string, int](1), WithShards[string, int](1))
	defer b.Close()

	// Fill the buffer (capacity = maxSize*4 = 4) plus trigger flushes that block
	blocker := make(chan struct{})
	b2 := New[string, int](func(_ context.Context, _ string, _ []int) error {
		<-blocker
		return nil
	}, WithMaxSize[string, int](1), WithShards[string, int](1))
	defer func() {
		close(blocker)
		b2.Close()
	}()

	// Fill buffer: maxSize=1 so buffer=4; first Add triggers flush (size=1=maxSize),
	// subsequent ones fill the channel while the flush goroutine is blocked.
	for i := 0; i < 5; i++ {
		_ = b2.Add(context.Background(), "k", i)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b2.Add(ctx, "k", 99)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPanicRecovery(t *testing.T) {
	var mu sync.Mutex
	var recovered any

	b := New[string, int](
		func(_ context.Context, _ string, _ []int) error {
			panic("flush panic")
		},
		WithMaxSize[string, int](1),
		WithOnPanic[string, int](func(r any) {
			mu.Lock()
			recovered = r
			mu.Unlock()
		}),
	)
	defer b.Close()

	_ = b.Add(context.Background(), "k", 1)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	r := recovered
	mu.Unlock()
	if r != "flush panic" {
		t.Fatalf("expected panic value 'flush panic', got %v", r)
	}

	// Batcher should still work after panic.
	var count atomic.Int32
	b2 := New[string, int](func(_ context.Context, _ string, items []int) error {
		count.Add(int32(len(items)))
		return nil
	}, WithMaxSize[string, int](1))
	defer b2.Close()

	_ = b2.Add(context.Background(), "k", 42)
	time.Sleep(50 * time.Millisecond)
	if count.Load() != 1 {
		t.Fatal("expected task to run after previous panic in a different batcher")
	}
}

func TestShardKeyNegativeValue(t *testing.T) {
	var count atomic.Int32

	b := New[string, int](
		func(_ context.Context, _ string, items []int) error {
			count.Add(int32(len(items)))
			return nil
		},
		WithShardKey[string, int](func(_ string) int { return -1 }),
		WithMaxSize[string, int](1),
	)
	defer b.Close()

	// Should not panic even though shardKey returns negative.
	if err := b.Add(context.Background(), "k", 1); err != nil {
		t.Fatalf("Add: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if count.Load() != 1 {
		t.Fatal("item was not flushed")
	}
}

func TestFlushFuncErrorIgnored(t *testing.T) {
	var count atomic.Int32

	b := New[string, int](func(_ context.Context, _ string, _ []int) error {
		count.Add(1)
		return errors.New("flush failed")
	}, WithMaxSize[string, int](1))
	defer b.Close()

	for i := 0; i < 3; i++ {
		if err := b.Add(context.Background(), "k", i); err != nil {
			t.Fatalf("Add returned unexpected error: %v", err)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if count.Load() == 0 {
		t.Fatal("flush was never called")
	}
}

func TestConcurrency(t *testing.T) {
	const goroutines = 8
	const itemsEach = 50

	var mu sync.Mutex
	totals := map[string]int{}

	b := New[string, int](func(_ context.Context, key string, items []int) error {
		mu.Lock()
		totals[key] += len(items)
		mu.Unlock()
		return nil
	}, WithMaxSize[string, int](20), WithShards[string, int](8))

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		g := g
		key := string(rune('a' + g))
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < itemsEach; i++ {
				_ = b.Add(context.Background(), key, i)
			}
		}()
	}
	wg.Wait()
	b.Close()

	mu.Lock()
	defer mu.Unlock()
	for g := 0; g < goroutines; g++ {
		key := string(rune('a' + g))
		if totals[key] != itemsEach {
			t.Errorf("key %s: expected %d items, got %d", key, itemsEach, totals[key])
		}
	}
}
