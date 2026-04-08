package safe

import (
	"fmt"
	"runtime"
)

// Do calls fn and recovers any panic, returning it wrapped as an error.
// Returns nil if fn completes without panicking.
func Do(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	fn()
	return nil
}

// DoWithRecover calls fn and recovers any panic, passing the raw panic value
// to onRecover. onRecover is only called if fn panics.
func DoWithRecover(fn func(), onRecover func(r any)) {
	defer func() {
		if r := recover(); r != nil {
			onRecover(r)
		}
	}()
	fn()
}

// DefaultPanicHandler prints the panic value and stack trace to stderr.
func DefaultPanicHandler(r any) {
	buf := make([]byte, 64<<10)
	buf = buf[:runtime.Stack(buf, false)]
	fmt.Printf("panic recovered: %v\n%s\n", r, buf)
}

// Go launches fn in a new goroutine. If fn panics, the panic value and
// stack trace are printed to stderr.
func Go(fn func()) {
	GoWithRecover(fn, DefaultPanicHandler)
}

// GoWithRecover launches fn in a new goroutine. If fn panics, onRecover
// is called with the recovered value.
func GoWithRecover(fn func(), onRecover func(r any)) {
	go DoWithRecover(fn, onRecover)
}
