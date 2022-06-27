package workers

// WorkerOptions for modification Worker.
type WorkerOptions[T any] func(*Worker[T])
