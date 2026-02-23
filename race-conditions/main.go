package main

import "fmt"

func main() {
	section("Counter race — lost updates")
	demoCounterRace()

	section("Counter fix — sync.Mutex")
	demoCounterMutex()

	section("Counter fix — sync/atomic")
	demoCounterAtomic()

	section("Counter fix — channel (actor)")
	demoCounterChannel()

	section("Map race — fatal concurrent access")
	demoMapRace()

	section("Map fix — sync.RWMutex")
	demoMapRWMutex()

	section("Map fix — sync.Map")
	demoMapSyncMap()

	section("Check-then-act race (TOCTOU)")
	demoCheckActRace()

	section("Check-then-act fix — lock the whole operation")
	demoCheckActFixed()

	section("Publication hazard — partially visible struct")
	demoPublishRace()

	section("Publication fix — sync.Once")
	demoPublishFixed()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
