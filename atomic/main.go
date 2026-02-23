package main

import "fmt"

func main() {
	section("Int64 — Add / Load / Store / Swap")
	demoInt64()

	section("Uint64 — contadores sin signo")
	demoUint64()

	section("Bool — flag atómica")
	demoBool()

	section("CompareAndSwap — CAS loop")
	demoCAS()

	section("atomic.Value — hot-reload de configuración")
	demoValue()

	section("atomic.Pointer — intercambio de structs")
	demoPointer()

	section("Patrón: contador lock-free vs Mutex")
	demoLockFreeCounter()

	section("Patrón: flag de cierre (shutdown flag)")
	demoShutdownFlag()

	section("Patrón: referencia compartida (copy-on-write)")
	demoCopyOnWrite()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
