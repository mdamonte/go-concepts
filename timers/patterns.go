package main

import (
	"fmt"
	"math/rand"
	"time"
)

// demoDebounce shows a debounce pattern: ignore rapid-fire events and only
// act after a quiet period. Classic use case: typing in a search box.
//
// Each new event resets the timer; the action fires only once the stream
// of events has been silent for the debounce window.
func demoDebounce() {
	events := []time.Duration{0, 30, 60, 90, 250, 280} // ms after start

	debounce := 120 * time.Millisecond
	timer := time.NewTimer(debounce)
	defer timer.Stop()

	start := time.Now()
	fired := 0

	// Simulate events arriving on a channel.
	eventCh := make(chan string, len(events))
	go func() {
		for i, d := range events {
			time.Sleep(d*time.Millisecond - time.Since(start))
			eventCh <- fmt.Sprintf("event-%d", i+1)
		}
		close(eventCh)
	}()

	fmt.Printf("  debounce window: %v\n", debounce)
	for {
		select {
		case e, ok := <-eventCh:
			if !ok {
				// No more events — wait for final debounce to fire.
				eventCh = nil
				continue
			}
			fmt.Printf("  received %s at +%v — resetting timer\n", e, time.Since(start).Round(time.Millisecond))
			// Reset the debounce timer on each event.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(debounce)

		case <-timer.C:
			fired++
			fmt.Printf("  debounced action fired at +%v (fired %d time(s))\n",
				time.Since(start).Round(time.Millisecond), fired)
			if eventCh == nil {
				return
			}
		}
	}
}

// demoRateLimit shows a token-bucket–style rate limiter using a Ticker:
// at most one request is processed per tick interval.
func demoRateLimit() {
	requests := make(chan int, 8)
	for i := 1; i <= 8; i++ {
		requests <- i
	}
	close(requests)

	// Allow one request every 50 ms.
	limiter := time.NewTicker(50 * time.Millisecond)
	defer limiter.Stop()

	fmt.Println("  processing 8 requests at max 1 per 50 ms:")
	for req := range requests {
		<-limiter.C // wait for the next token
		fmt.Printf("    request %d processed at %s\n", req, time.Now().Format("15:04:05.000"))
	}
}

// demoRetryBackoff shows exponential backoff with jitter for retrying a
// failing operation. The delay doubles on each failure, capped at maxDelay.
//
// Adding random jitter avoids the "thundering herd" problem where many
// clients retry in lockstep after a shared failure.
func demoRetryBackoff() {
	const (
		maxAttempts = 5
		baseDelay   = 20 * time.Millisecond
		maxDelay    = 200 * time.Millisecond
		failUntil   = 3 // succeed on attempt 4
	)

	attempt := 0
	delay := baseDelay

	for {
		attempt++
		fmt.Printf("  attempt %d...", attempt)

		// Simulate an operation that fails for the first N attempts.
		if attempt < failUntil {
			fmt.Println(" failed")
			if attempt >= maxAttempts {
				fmt.Println("  giving up")
				return
			}

			// Jitter: add up to 50 % of delay as random noise.
			jitter := time.Duration(rand.Int63n(int64(delay / 2)))
			wait := delay + jitter
			if wait > maxDelay {
				wait = maxDelay
			}
			fmt.Printf("  retrying in %v\n", wait.Round(time.Millisecond))

			timer := time.NewTimer(wait)
			<-timer.C
			timer.Stop()

			delay *= 2 // exponential back-off
		} else {
			fmt.Println(" success")
			return
		}
	}
}

// demoPeriodic shows a cancellable periodic task pattern: work runs on a
// fixed interval and stops cleanly when a done channel is closed.
//
// Key difference from a plain ticker loop: passing an explicit done channel
// (or context.Done()) makes the goroutine stoppable from outside.
func demoPeriodic() {
	done := make(chan struct{})
	ticker := time.NewTicker(60 * time.Millisecond)

	// Stop the periodic task after 250 ms.
	go func() {
		time.Sleep(250 * time.Millisecond)
		close(done)
	}()

	fmt.Println("  periodic task running (interval 60 ms, stops after ~250 ms):")
	count := 0
	for {
		select {
		case t := <-ticker.C:
			count++
			fmt.Printf("    tick %d at %s\n", count, t.Format("15:04:05.000"))
		case <-done:
			ticker.Stop()
			fmt.Printf("  cancelled after %d ticks\n", count)
			return
		}
	}
}
