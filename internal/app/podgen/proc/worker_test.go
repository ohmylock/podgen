package proc_test

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"podgen/internal/app/podgen/podcast"
	"podgen/internal/app/podgen/proc"
)

func makeTask(index int) proc.UploadTask {
	return proc.UploadTask{
		Index:     index,
		PodcastID: "pod1",
		Folder:    "/tmp",
		Episode:   &podcast.Episode{Filename: "ep.mp3", Size: 1024},
	}
}

func sendTasks(tasks []proc.UploadTask) <-chan proc.UploadTask {
	ch := make(chan proc.UploadTask, len(tasks))
	for _, t := range tasks {
		ch <- t
	}
	close(ch)
	return ch
}

func TestRunWorkerPool_AllTasksProcessed(t *testing.T) {
	const n = 10
	tasks := make([]proc.UploadTask, n)
	for i := range tasks {
		tasks[i] = makeTask(i)
	}

	uploadFn := func(_ context.Context, _ int, task proc.UploadTask) proc.UploadTaskResult {
		return proc.UploadTaskResult{Index: task.Index, Episode: task.Episode, Location: "s3://bucket/ep.mp3"}
	}

	results := proc.RunWorkerPool(context.Background(), 3, sendTasks(tasks), uploadFn)

	got := make([]int, 0, n)
	for r := range results {
		require.NoError(t, r.Err)
		got = append(got, r.Index)
	}
	sort.Ints(got)

	expected := make([]int, n)
	for i := range expected {
		expected[i] = i
	}
	assert.Equal(t, expected, got)
}

func TestRunWorkerPool_StableWorkerIDs(t *testing.T) {
	const workers = 3
	const n = 15

	tasks := make([]proc.UploadTask, n)
	for i := range tasks {
		tasks[i] = makeTask(i)
	}

	var mu sync.Mutex
	seenIDs := map[int]struct{}{}

	uploadFn := func(_ context.Context, workerID int, task proc.UploadTask) proc.UploadTaskResult {
		mu.Lock()
		seenIDs[workerID] = struct{}{}
		mu.Unlock()
		return proc.UploadTaskResult{Index: task.Index, Episode: task.Episode}
	}

	results := proc.RunWorkerPool(context.Background(), workers, sendTasks(tasks), uploadFn)
	for range results {
	}

	mu.Lock()
	defer mu.Unlock()
	for id := range seenIDs {
		assert.GreaterOrEqual(t, id, 0)
		assert.Less(t, id, workers)
	}
}

func TestRunWorkerPool_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Unbuffered channel — tasks are sent one by one.
	taskCh := make(chan proc.UploadTask)

	var processed atomic.Int32
	uploadFn := func(_ context.Context, _ int, task proc.UploadTask) proc.UploadTaskResult {
		processed.Add(1)
		return proc.UploadTaskResult{Index: task.Index, Episode: task.Episode}
	}

	results := proc.RunWorkerPool(ctx, 2, taskCh, uploadFn)

	// Send one task and let it complete.
	taskCh <- makeTask(0)

	// Cancel before sending more tasks.
	cancel()

	// Drain results; the channel must eventually close.
	done := make(chan struct{})
	go func() {
		for range results {
		}
		close(done)
	}()

	select {
	case <-done:
		// OK — pool shut down after cancellation.
	case <-time.After(2 * time.Second):
		t.Fatal("worker pool did not shut down after context cancellation")
	}
}

func TestRunWorkerPool_EmptyTasks(t *testing.T) {
	ch := make(chan proc.UploadTask)
	close(ch)

	uploadFn := func(_ context.Context, _ int, task proc.UploadTask) proc.UploadTaskResult {
		return proc.UploadTaskResult{Index: task.Index}
	}

	results := proc.RunWorkerPool(context.Background(), 3, ch, uploadFn)
	var count int
	for range results {
		count++
	}
	assert.Equal(t, 0, count)
}

func TestRunWorkerPool_ErrorPropagation(t *testing.T) {
	tasks := []proc.UploadTask{makeTask(0), makeTask(1), makeTask(2)}

	uploadFn := func(_ context.Context, _ int, task proc.UploadTask) proc.UploadTaskResult {
		if task.Index == 1 {
			return proc.UploadTaskResult{Index: task.Index, Err: assert.AnError}
		}
		return proc.UploadTaskResult{Index: task.Index, Episode: task.Episode, Location: "s3://ok"}
	}

	results := proc.RunWorkerPool(context.Background(), 2, sendTasks(tasks), uploadFn)

	var errCount int
	for r := range results {
		if r.Err != nil {
			errCount++
		}
	}
	assert.Equal(t, 1, errCount)
}

func TestRunWorkerPool_ZeroWorkersDefaultsToOne(t *testing.T) {
	tasks := []proc.UploadTask{makeTask(0)}
	uploadFn := func(_ context.Context, _ int, task proc.UploadTask) proc.UploadTaskResult {
		return proc.UploadTaskResult{Index: task.Index, Episode: task.Episode}
	}

	results := proc.RunWorkerPool(context.Background(), 0, sendTasks(tasks), uploadFn)
	var count int
	for range results {
		count++
	}
	assert.Equal(t, 1, count)
}
