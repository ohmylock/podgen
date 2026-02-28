package proc

import (
	"context"
	"sync"
)

// RunParallel executes functions with bounded concurrency.
// Each function receives a context and returns an error.
// All errors are collected and returned.
func RunParallel(ctx context.Context, workers int, tasks []func(ctx context.Context) error) []error {
	if workers <= 0 {
		workers = 1
	}
	if len(tasks) == 0 {
		return nil
	}

	errs := make([]error, len(tasks))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, fn func(ctx context.Context) error) {
			defer wg.Done()
			defer func() { <-sem }()
			errs[i] = fn(ctx)
		}(i, task)
	}

	wg.Wait()
	return errs
}
