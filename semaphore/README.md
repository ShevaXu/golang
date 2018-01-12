# Semaphore

For [Bounding resource use](https://github.com/golang/go/wiki/BoundingResourceUse).

```go
s := semaphore.NewSemaphore(10)

go func() {
    ctx := context.Background() // for cancellation
    if s.Obtain(ctx) {
        defer s.Release()
        // do whatever 
    }
}()
```

For *weighted* semaphore, see [this implementation](https://github.com/golang/sync/blob/master/semaphore/semaphore.go).
