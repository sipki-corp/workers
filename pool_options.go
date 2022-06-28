package workers

// PoolOptions for modification Pool.
type PoolOptions[T any] func(*Pool[T])

func WithPoolInitWorkerCount[T any](initWorkerCount int) PoolOptions[T] {
	return func(p *Pool[T]) {
		p.initWorkerCount = initWorkerCount
	}
}

func WithPoolMinWorkers[T any](minWorkers int) PoolOptions[T] {
	return func(p *Pool[T]) {
		p.minWorkers = minWorkers
	}
}

func WithPoolIMaxWorkers[T any](maxWorkers int) PoolOptions[T] {
	return func(p *Pool[T]) {
		p.maxWorkers = maxWorkers
	}
}
