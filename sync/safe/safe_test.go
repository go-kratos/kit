package safe

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo_Normal(t *testing.T) {
	err := Do(func() {})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestDo_PanicBecomesError(t *testing.T) {
	err := Do(func() { panic("oops") })
	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}
	if err.Error() != "oops" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestDo_PanicErrorType(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := Do(func() { panic(sentinel) })
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// fmt.Errorf wraps via %v; check message at minimum
	if err.Error() != sentinel.Error() {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGo_RunsFn(t *testing.T) {
	var ran atomic.Bool
	Go(func() { ran.Store(true) })
	time.Sleep(50 * time.Millisecond)
	if !ran.Load() {
		t.Fatal("expected fn to run")
	}
}

func TestGo_PanicDoesNotCrash(t *testing.T) {
	// Should not crash the test process.
	Go(func() { panic("safe Go panic") })
	time.Sleep(50 * time.Millisecond)
}

func TestGoWithRecover_CallsOnRecover(t *testing.T) {
	var got any
	var done atomic.Bool
	GoWithRecover(func() {
		panic("boom")
	}, func(r any) {
		got = r
		done.Store(true)
	})
	time.Sleep(50 * time.Millisecond)
	if !done.Load() {
		t.Fatal("onRecover was not called")
	}
	if got != "boom" {
		t.Fatalf("expected 'boom', got %v", got)
	}
}

func TestGoWithRecover_NoPanic(t *testing.T) {
	var recovered atomic.Bool
	GoWithRecover(func() {}, func(r any) { recovered.Store(true) })
	time.Sleep(50 * time.Millisecond)
	if recovered.Load() {
		t.Fatal("onRecover should not be called when fn does not panic")
	}
}
