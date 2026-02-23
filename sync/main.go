package main

import "fmt"

func main() {
	section("sync.Mutex")
	demoMutex()

	section("sync.RWMutex")
	demoRWMutex()

	section("sync.WaitGroup")
	demoWaitGroup()

	section("sync.Once")
	demoOnce()

	section("sync.Cond — Signal")
	demoCondSignal()

	section("sync.Cond — Broadcast")
	demoCondBroadcast()

	section("sync.Pool")
	demoPool()

	section("sync.Map")
	demoSyncMap()

	section("sync/atomic — counters & CAS")
	demoAtomic()

	section("sync/atomic — Value")
	demoAtomicValue()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
