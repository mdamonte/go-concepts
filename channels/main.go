package main

import "fmt"

func main() {
	section("Unbuffered channel")
	demoUnbuffered()

	section("Buffered channel")
	demoBuffered()

	section("Directional channels")
	demoDirectional()

	section("Close + range")
	demoCloseRange()

	section("Select")
	demoSelect()

	section("Select: default (non-blocking)")
	demoSelectDefault()

	section("Select: nil channel")
	demoSelectNil()

	section("Select: timeout")
	demoSelectTimeout()

	section("Pipeline")
	demoPipeline()

	section("Fan-out")
	demoFanOut()

	section("Fan-in (merge)")
	demoFanIn()

	section("Worker pool")
	demoWorkerPool()

	section("Semaphore")
	demoSemaphore()

	section("Done channel")
	demoDone()

	section("Or-done channel")
	demoOrDone()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
