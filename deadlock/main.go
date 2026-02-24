package main

import "fmt"

// Each demo spawns goroutines into a specific blocking state, prints the
// goroutine dump so you can see the state label, then cleans up if possible.
//
// The final demo (demoMutexDeadlock) triggers the real runtime panic:
//   fatal error: all goroutines are asleep - deadlock!
//
// To inspect live goroutine states before the crash you can also run:
//   go tool pprof http://localhost:6062/debug/pprof/goroutine?debug=2
func main() {
	section("[chan receive] — blocked waiting to receive from a channel")
	demoChanReceive()

	section("[chan send]    — blocked waiting to send to a channel")
	demoChanSend()

	section("[select]       — blocked in select with all cases blocking")
	demoSelect()

	section("[IO wait]      — blocked on network I/O (no data from peer)")
	demoIOWait()

	section("[running]      — goroutine actively executing (busy loop)")
	demoRunning()

	section("[semacquire] / [sync.Mutex.Lock] — blocked waiting to acquire a mutex")
	demoSemacquire()

	section("[semacquire]   — AB deadlock: inconsistent lock ordering")
	fmt.Println("  Shows complete dump with all accumulated states, then exits with code 1.")
	fmt.Println("  On a net-free program the runtime itself would print the fatal error.\n")
	demoMutexDeadlock()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
