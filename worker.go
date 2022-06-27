package workers

import (
	"context"
	"sync"
)

// Worker for controlling one thread task queue.
// Note:
//	* This worker is blocking.
type Worker[T any] struct {
	wg        *sync.WaitGroup
	startOnce sync.Once
	closeOnce sync.Once
	done      chan struct{}
	jobs      <-chan Job[T]
}

// NewWorker build and returns *Worker[T].
// It doesn't start process work.
// So before sending tasks, you must call Worker.Start.
func NewWorker[T any](jobs <-chan Job[T], opts ...WorkerOptions[T]) *Worker[T] {
	w := &Worker[T]{
		wg:        &sync.WaitGroup{},
		startOnce: sync.Once{},
		closeOnce: sync.Once{},
		done:      make(chan struct{}),
		jobs:      jobs,
	}

	for i := range opts {
		opts[i](w)
	}

	return w
}

// Close closes this worker.
// It waits until worker process is down.
// It's idempotent method.
func (w *Worker[T]) Close() {
	w.closeOnce.Do(func() {
		close(w.done)
	})
	w.wg.Wait()
}

// Start starts worker process.
// It's idempotent method.
func (w *Worker[T]) Start(ctx context.Context) {
	w.startOnce.Do(func() {
		w.wg.Add(1)
		go w.process(ctx)
	})
}

func (w *Worker[T]) process(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case job, ok := <-w.jobs:
			if !ok {
				return
			}

			w.handle(ctx, job)
		}
	}
}

func (w *Worker[T]) handle(ctx context.Context, job Job[T]) {
	val, err := job.Do(ctx)
	res := Result[T]{
		Value: val,
		Err:   err,
	}

	select {
	case <-ctx.Done():
	case <-w.done:
	case job.Result() <- res:
	}
}
