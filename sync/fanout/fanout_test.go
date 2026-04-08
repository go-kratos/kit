package fanout

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo_Normal(t *testing.T) {
	f := New(WithWorker(1), WithBuffer(10))
	defer f.Close()

	var ran atomic.Bool
	err := f.Do(context.Background(), func(ctx context.Context) {
		ran.Store(true)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Wait for worker to process.
	time.Sleep(50 * time.Millisecond)
	if !ran.Load() {
		t.Fatal("expected task to run")
	}
}

func TestDo_AfterClose(t *testing.T) {
	f := New()
	f.Close()

	err := f.Do(context.Background(), func(ctx context.Context) {})
	if !errors.Is(err, ErrFanoutClosed) {
		t.Fatalf("expected ErrFanoutClosed, got %v", err)
	}
}

func TestDo_NilFunc(t *testing.T) {
	f := New()
	defer f.Close()

	if err := f.Do(context.Background(), nil); err != nil {
		t.Fatalf("nil func should be a no-op, got %v", err)
	}
}

func TestDo_CancelledContext(t *testing.T) {
	// Use buffer=0 to force the send to block so ctx cancellation is observed.
	f := New(WithWorker(1), WithBuffer(1))
	defer f.Close()

	// Fill the buffer so the next Do blocks on send.
	blocker := make(chan struct{})
	_ = f.Do(context.Background(), func(ctx context.Context) {
		<-blocker
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	err := f.Do(ctx, func(ctx context.Context) {})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	close(blocker)
}

func TestClose_Idempotent(t *testing.T) {
	f := New()
	if err := f.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := f.Close(); !errors.Is(err, ErrFanoutClosed) {
		t.Fatalf("second Close should return ErrFanoutClosed, got %v", err)
	}
}

func TestClose_GracefulDrain(t *testing.T) {
	f := New(WithWorker(1), WithBuffer(100))

	const n = 50
	var count atomic.Int32
	for i := 0; i < n; i++ {
		_ = f.Do(context.Background(), func(ctx context.Context) {
			count.Add(1)
		})
	}
	// Close should drain all pending tasks before returning.
	f.Close()

	if got := count.Load(); got != n {
		t.Fatalf("expected %d tasks to run, got %d", n, got)
	}
}

func TestOnPanic_Hook(t *testing.T) {
	var mu sync.Mutex
	var recovered any

	f := New(
		WithOnPanic(func(r any) {
			mu.Lock()
			recovered = r
			mu.Unlock()
		}),
	)
	defer f.Close()

	_ = f.Do(context.Background(), func(ctx context.Context) {
		panic("boom")
	})
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	r := recovered
	mu.Unlock()
	if r != "boom" {
		t.Fatalf("expected panic value 'boom', got %v", r)
	}
}

func TestOnPanic_DefaultNoOp(t *testing.T) {
	f := New()
	defer f.Close()

	// Should not crash the process even without an OnPanic hook.
	_ = f.Do(context.Background(), func(ctx context.Context) {
		panic("silent panic")
	})
	time.Sleep(50 * time.Millisecond)
}

func TestMultipleWorkers(t *testing.T) {
	const workers = 4
	f := New(WithWorker(workers), WithBuffer(200))
	defer f.Close()

	var wg sync.WaitGroup
	var count atomic.Int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		_ = f.Do(context.Background(), func(ctx context.Context) {
			defer wg.Done()
			count.Add(1)
		})
	}
	wg.Wait()
	if got := count.Load(); got != 100 {
		t.Fatalf("expected 100, got %d", got)
	}
}

func TestDo_ContextPassedThrough(t *testing.T) {
	f := New()
	defer f.Close()

	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "hello")

	var got any
	var done atomic.Bool
	_ = f.Do(ctx, func(ctx context.Context) {
		got = ctx.Value(key{})
		done.Store(true)
	})
	time.Sleep(50 * time.Millisecond)
	if !done.Load() {
		t.Fatal("task did not run")
	}
	if got != "hello" {
		t.Fatalf("expected 'hello', got %v", got)
	}
}
