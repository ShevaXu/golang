package semaphore_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ShevaXu/golang/assert"
	"github.com/ShevaXu/golang/semaphore"
)

func TestNewSemaphore(t *testing.T) {
	a := assert.NewAssert(t)
	const n = 5

	sema := semaphore.NewSemaphore(n)
	a.Equal(0, sema.Count(), "No one queued")
	a.Equal(n, sema.Capacity(), "Full cap")

	a.True(!sema.Closed(), "It is not closed")
	sema.Close()
	a.True(sema.Closed(), "It is closed now")
}

func TestSemaphore_ObtainRelease(t *testing.T) {
	a := assert.NewAssert(t)
	const n = 2 // easier to test

	sema := semaphore.NewSemaphore(n)

	a.True(!sema.Release(), "Release on full returns false immediately")

	// obtain
	bc := context.Background()
	a.True(sema.Obtain(bc), "Obtained immediately")
	a.Equal(1, sema.Count(), "Now has one queued")

	ctx, cancel := context.WithTimeout(bc, 10*time.Millisecond)
	defer cancel()

	sema.Obtain(bc) // obtain one more
	a.True(!sema.Obtain(ctx), "Should fail")

	a.Equal(sema.Capacity(), sema.Count(), "Now it is full")

	// release
	a.True(sema.Release(), "Release succeed")
	a.Equal(1, sema.Count(), "Now has one queued again")
	a.True(sema.Obtain(bc), "Can obtain again")

	sema.Close()
	a.True(!sema.Obtain(bc), "Should fail immediately when closed")
}

func TestSemaphore_Sync(t *testing.T) {
	// TODO: better cases?
	a := assert.NewAssert(t)
	const n = 2 // easier to test
	const m = 5 // #workers
	wg := sync.WaitGroup{}
	ctx := context.Background()

	sema := semaphore.NewSemaphore(n)
	for i := 0; i < m; i++ {
		go func() {
			wg.Add(1)
			sema.Obtain(ctx)
			wg.Done()
		}()
	}

	time.Sleep(10 * time.Millisecond)
	a.Equal(n, sema.Count(), "Full and overflowed")

	for i := 0; i < m-n; i++ {
		sema.Release()
	}

	wg.Wait()
	a.Equal(n, sema.Count(), "Still full but buffered")
}
