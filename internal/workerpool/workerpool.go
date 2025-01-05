// package workerpool contains a single goroutine worker pool that executes arbitrary
// encapsulated functions.
package workerpool

import (
	"sync"
)

// Pool is a single goroutine worker pool.
type Pool struct {
	num int
	q   chan func()
	wg  sync.WaitGroup
}

// New returns a new goroutine workerpool.
func New(num, queueSize int) *Pool {
	return &Pool{
		num: num,
		q:   make(chan func(), queueSize),
		wg:  sync.WaitGroup{},
	}
}

// Run initializes the goroutine worker pool.
func (w *Pool) Run() {
	for i := 0; i < w.num; i++ {
		w.wg.Add(1)
		go func() {
			for f := range w.q {
				f()
			}
			w.wg.Done()
		}()
	}
}

// Push pushes a job to the worker queue to execute.
func (w *Pool) Push(f func()) {
	w.q <- f
}

func (w *Pool) Close() {
	close(w.q)
	w.wg.Wait()
}
