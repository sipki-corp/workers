package workers

import "context"

// Job interface for using any type jobs.
type Job[T any] interface {
	// Do call job action.
	Do(ctx context.Context) (T, error)
	// Result returns channel for sending job result.
	// Blocking send.
	Result() chan<- Result[T]
}
