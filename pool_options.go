package workers

// PoolOptions for modification Pool.
type PoolOptions[T any] func(*Pool[T])

func PoolInitWorkerCount[T any](initWorkerCount int) PoolOptions[T] {
	return func(p *Pool[T]) {
		p.initWorkerCount = initWorkerCount
	}
}

func PoolMinWorkers[T any](minWorkers int) PoolOptions[T] {
	return func(p *Pool[T]) {
		p.minWorkers = minWorkers
	}
}

func PoolIMaxWorkers[T any](maxWorkers int) PoolOptions[T] {
	return func(p *Pool[T]) {
		p.maxWorkers = maxWorkers
	}
}
