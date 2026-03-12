package proc

import (
	"context"
	"sync"

	"podgen/internal/app/podgen/podcast"
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

// UploadTask represents a single upload job to be processed by a worker.
type UploadTask struct {
	Index     int
	PodcastID string
	Folder    string
	Episode   *podcast.Episode
}

// UploadTaskResult holds the outcome of a single upload task.
type UploadTaskResult struct {
	Index    int
	Episode  *podcast.Episode
	Location string
	Err      error
}

// UploadFn is the function signature for processing a single upload task.
// workerID is stable for the lifetime of the worker goroutine (0, 1, 2, ...).
type UploadFn func(ctx context.Context, workerID int, task UploadTask) UploadTaskResult

// RunWorkerPool starts a pool of workers that process tasks from the tasks channel.
// Each worker has a stable workerID (0..workers-1) for progress reporting.
// Results are sent to the returned channel, which is closed when all tasks are done.
// Context cancellation stops workers after their current task completes.
func RunWorkerPool(ctx context.Context, workers int, tasks <-chan UploadTask, uploadFn UploadFn) <-chan UploadTaskResult {
	if workers <= 0 {
		workers = 1
	}

	results := make(chan UploadTaskResult, workers)

	var wg sync.WaitGroup
	for workerID := 0; workerID < workers; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-tasks:
					if !ok {
						return
					}
					result := uploadFn(ctx, workerID, task)
					// Always send the result after upload completes to prevent S3/DB inconsistency.
					// If upload succeeded, we must update DB even if context was canceled.
					results <- result
				}
			}
		}(workerID)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
