package workers

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/powerman/chanq"
)

// Result contains job payload.
type Result[T any] struct {
	Value T
	Err   error
}

type (
	reqGetJobCount struct {
		resp chan int
	}
	reqGetWorkerCount struct {
		resp chan int
	}
)

// Pool for controlling N workers at same time.
type Pool[T any] struct {
	initWorkerCount   int
	maxWorkers        int
	minWorkers        int
	newJob            chan Job[T]
	exec              chan Job[T]
	resize            chan int
	reqGetJobCount    chan reqGetJobCount
	reqGetWorkerCount chan reqGetWorkerCount
	done              chan struct{}
	startOnce         sync.Once
	closeOnce         sync.Once
	wg                *sync.WaitGroup
}

const (
	defaultInitWorkerCount = 1
	defaultMaxWorkers      = 100
	minWorkers             = 1
)

var ErrInvalidArgument = errors.New("invalid argument")

// NewPool build and returns *Pool[T].
// It doesn't start process work.
// So before sending tasks, you must call Pool.Start.
func NewPool[T any](opts ...PoolOptions[T]) (*Pool[T], error) {
	pool := &Pool[T]{
		initWorkerCount:   defaultInitWorkerCount,
		maxWorkers:        defaultMaxWorkers,
		minWorkers:        minWorkers,
		newJob:            make(chan Job[T]),
		exec:              make(chan Job[T]),
		resize:            make(chan int),
		reqGetJobCount:    make(chan reqGetJobCount),
		reqGetWorkerCount: make(chan reqGetWorkerCount),
		done:              make(chan struct{}),
		startOnce:         sync.Once{},
		closeOnce:         sync.Once{},
		wg:                &sync.WaitGroup{},
	}

	for i := range opts {
		opts[i](pool)
	}

	switch {
	case pool.minWorkers < minWorkers:
		return nil, fmt.Errorf("%w: min workers can't be less than %d", ErrInvalidArgument, minWorkers)
	case pool.initWorkerCount < minWorkers:
		return nil, fmt.Errorf("%w: init workers count can't be less than %d", ErrInvalidArgument, minWorkers)
	}

	return pool, nil
}

// Max returns the maximum number of concurrent workers.
func (p *Pool[T]) Max() int {
	return p.maxWorkers
}

// Send sends new job to worker pool.
// Not-blocking send.
func (p *Pool[T]) Send(j Job[T]) {
	p.newJob <- j
}

// Resize send event for resizing worker pool.
func (p *Pool[T]) Resize(i int) {
	p.resize <- i
}

// WorkerSize returns current worker size.
func (p *Pool[T]) WorkerSize(ctx context.Context) (int, error) {
	resp := make(chan int)
	req := reqGetWorkerCount{
		resp: resp,
	}

	select {
	case p.reqGetWorkerCount <- req:
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	select {
	case res := <-resp:
		return res, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// JobBufferSize returns current jub queue size.
func (p *Pool[T]) JobBufferSize(ctx context.Context) (int, error) {
	resp := make(chan int)
	req := reqGetJobCount{
		resp: resp,
	}

	select {
	case p.reqGetJobCount <- req:
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	select {
	case res := <-resp:
		return res, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// Start worker processes.
func (p *Pool[T]) Start(ctx context.Context) {
	p.startOnce.Do(func() {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go p.process(ctx, &wg)
		wg.Add(1)
		go p.dispatch(ctx, &wg)
	})
}

// Close worker processes.
func (p *Pool[T]) Close() {
	p.closeOnce.Do(func() {
		close(p.done)
	})
	p.wg.Wait()
}

// controlling for not-blocking job collecting.
func (p *Pool[T]) process(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	q := chanq.NewQueue(p.exec)

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case j := <-p.newJob:
			q.Enqueue(j)
		case q.C <- q.Elem:
			q.Dequeue()
		case req := <-p.reqGetJobCount:
			req.resp <- len(q.Queue)
		}
	}
}

// controlling for job executing.
func (p *Pool[T]) dispatch(ctx context.Context, wg *sync.WaitGroup) {
	jobs := make(chan Job[T])
	workers := make([]*Worker[T], p.initWorkerCount)
	for i := range workers {
		workers[i] = NewWorker(jobs)
		workers[i].Start(ctx)
	}

	defer func() {
		close(jobs)
		wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-p.done:
			return

		case job := <-p.exec:
			select {
			case <-ctx.Done():
				return
			case <-p.done:
				return
			case jobs <- job:
			}

		case req := <-p.reqGetWorkerCount:
			req.resp <- len(workers)

		case resize := <-p.resize:
			switch {
			case resize > 0:

				currentSize := len(workers)
				if currentSize+resize > p.maxWorkers {
					resize = p.maxWorkers - currentSize
				}

				newWorkers := make([]*Worker[T], resize)
				for i := range newWorkers {
					newWorkers[i] = NewWorker(jobs)
					newWorkers[i].Start(ctx)
				}

				workers = append(workers, newWorkers...)

			case resize < 0:

				resize = resize * -1 // TODO: Fixme
				for i := 0; i < resize; i++ {
					if len(workers) == 1 {
						break
					} else if len(workers) <= i {
						break
					}

					workers[i].Close()
					workers[i] = workers[len(workers)-1]
					workers[len(workers)-1] = nil
					workers = workers[:len(workers)-1]
				}

			default:
				continue // If user send 0, we just ignore it.
			}
		}
	}
}
