package main

import (
	"fmt"
	"sync"
	"time"
)

type job struct {
	id    int
	value int
}

type result struct {
	jobID  int
	output int
}

// demoWorkerPool shows the worker pool pattern:
// a fixed number of goroutines pull work from a shared jobs channel and
// send results to a shared results channel.
//
// Benefits over one-goroutine-per-task:
//   - Bounds the number of concurrent goroutines (memory, CPU).
//   - Backpressure: the sender blocks when the pool is saturated.
func demoWorkerPool() {
	const numWorkers = 3
	const numJobs = 9

	jobs := make(chan job, numJobs)
	results := make(chan result, numJobs)

	// Start workers. They block on jobs until the channel is closed.
	var wg sync.WaitGroup
	for id := 1; id <= numWorkers; id++ {
		wg.Add(1)
		go poolWorker(id, jobs, results, &wg)
	}

	// Enqueue all jobs and close the channel to signal no more work.
	for i := 1; i <= numJobs; i++ {
		jobs <- job{id: i, value: i * 10}
	}
	close(jobs) // workers will exit their range loop after draining

	// Close results once all workers finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results.
	for r := range results {
		fmt.Printf("  job %d → %d\n", r.jobID, r.output)
	}
}

func poolWorker(id int, jobs <-chan job, results chan<- result, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		// Simulate variable processing time.
		time.Sleep(10 * time.Millisecond)
		fmt.Printf("  worker%d processing job%d\n", id, j.id)
		results <- result{jobID: j.id, output: j.value * j.value}
	}
}
