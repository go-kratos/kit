# go-kratos/kit

Lightweight, type-safe utility toolkit that provides concurrency-friendly generic container and a simple, configurable retryer.

Included packages:

- container/maps: Type-safe generic Map built on `sync.Map`.
- container/sets: Generic Set implemented on top of Map.
- container/slices: Concurrency-safe slice list guarded by `sync.RWMutex`.
- retry: General-purpose retryer with exponential backoff and configurable retry conditions and backoff parameters.

Standard library only. Easy to integrate into any project.

## Installation

```bash
go get github.com/go-kratos/kit@latest
```

Import subpackages as needed, for example:

```go
import (
    "github.com/go-kratos/kit/container/maps"
    "github.com/go-kratos/kit/container/sets"
    "github.com/go-kratos/kit/container/slices"
    "github.com/go-kratos/kit/retry"
)
```

## Quick Start

### container/sets.Set[T]

```go
package main

import (
    "fmt"
    "github.com/go-kratos/kit/container/sets"
)

func main() {
    s := sets.New[string]("a", "b")
    s.Insert("c").Delete("b")

    fmt.Println(s.Has("a"))          // true
    fmt.Println(s.HasAny("x", "c"))  // true
    fmt.Println(s.HasAll("a", "c"))  // true

    t := s.Clone()
    fmt.Println(t.HasAll("a", "c"))  // true

    fmt.Println(s.ToSlice())           // [a c] (order not guaranteed)
}
```

Common methods: `Insert`, `Delete`, `Has`, `HasAny`, `HasAll`, `Clear`, `Clone`, `ToSlice`.

JSON: Set encodes to an array of elements; decoding fills the set.

### container/maps.Map[K,V]

```go
package main

import (
    "fmt"
    "github.com/go-kratos/kit/container/maps"
)

func main() {
    m := maps.New[string, int]()
    m.Store("a", 1)

    if v, ok := m.Load("a"); ok {
        fmt.Println(v) // 1
    }

    if v, loaded := m.LoadOrStore("b", 2); !loaded {
        fmt.Println(v) // 2 (first insert)
    }

    m.Range(func(k string, v int) bool {
        fmt.Println(k, v)
        return true
    })

    // Copy to a built-in map
    fmt.Println(m.ToMap())

    // Clone into a new concurrent Map
    mm := m.Clone()
    _ = mm
}
```

Common methods: `Store`, `Load`, `LoadOrStore`, `LoadAndDelete`, `Delete`, `Clear`, `Range`, `ToMap`, `Clone`.

JSON: Serializes/deserializes directly as an object (map).

### container/slices.Slice[T]

```go
package main

import (
    "fmt"
    "github.com/go-kratos/kit/container/slices"
)

func main() {
    l := slices.New[int](1, 2, 3)
    l.Append(4, 5)

    if v, ok := l.Get(1); ok {
        fmt.Println(v) // 2
    }

    // Set and remove
    _ = l.Set(1, 99)
    if v, ok := l.RemoveAt(0); ok {
        fmt.Println("removed:", v) // 1
    }

    // Snapshot iteration
    l.Range(func(i int, v int) bool {
        fmt.Printf("%d:%d ", i, v)
        return true
    })
    fmt.Println()

    fmt.Println(l.ToSlice()) // returns a copy
}
```

Common methods: `Append`, `Get`, `Set`, `RemoveAt`, `Range`, `Slice`/`SliceStart`/`SliceEnd`, `ToSlice`, `Clone`, `Len`, `Clear`.

JSON: Serializes/deserializes as an array; internal operations are concurrency-safe.

### retry

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"
    "github.com/go-kratos/kit/retry"
)

func main() {
    // Retry up to 3 times, only when the error matches context.DeadlineExceeded
    r := retry.New(
        3,
        retry.WithRetryable(func(err error) bool { return errors.Is(err, context.DeadlineExceeded) }),
        retry.WithBaseDelay(10*time.Millisecond),
        retry.WithMaxDelay(1*time.Second),
        retry.WithMultiplier(1.6),
        retry.WithJitter(0.2),
    )

    err := r.Do(context.Background(), func(ctx context.Context) error {
        // Your business logic
        return context.DeadlineExceeded
    })

    fmt.Println("done:", err)
}
```

Convenience helpers:

- `retry.Do(ctx, fn)`: Retry with default configuration.
- `retry.Infinite(ctx, fn)`: Retry indefinitely (until success or `ctx` is done).

Tunables: `WithBaseDelay`, `WithMaxDelay`, `WithMultiplier`, `WithJitter`, `WithRetryable`.

## Concurrency & Performance

- slices: Writes are protected with a mutex; `Range` and `ToSlice` use snapshots to avoid long-held locks.
- maps: Type-safe wrapper over `sync.Map`, suitable for read-mostly or cross-goroutine sharing scenarios.
- sets: Built on the concurrent Map; methods are concurrency-safe.

Note: `Slice.Range` copies the underlying slice; consider memory overhead when iterating very large lists. `ToSlice` and `ToMap` return a copy.

## Requirements

Go 1.18+ (generics enabled).

## License

MIT. See `LICENSE`.
