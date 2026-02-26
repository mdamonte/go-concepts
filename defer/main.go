package main

import "fmt"

// Each demo covers one aspect of defer that appears in technical interviews.
// defer is the #1 source of "what does this print?" trick questions in Go.
//
// Run:
//
//	go run .
func main() {
	section("Basics — LIFO, argument evaluation, closures")
	demoBasics()

	section("Named returns — defer can modify the value a function returns")
	demoReturns()

	section("Loops — the resource-leak gotcha and three fixes")
	demoLoops()

	section("Panic & recover — defer runs during unwind, recover rules")
	demoPanic()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
