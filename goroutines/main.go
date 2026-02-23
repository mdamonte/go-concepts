package main

import "fmt"

func main() {
	section("Basics: launch styles")
	demoBasics()

	section("Closure capture bug")
	demoClosure()

	section("Goroutine lifecycle & runtime")
	demoLifecycle()

	section("Goroutine leak — blocked send")
	demoLeakSend()

	section("Goroutine leak — blocked receive")
	demoLeakReceive()

	section("Goroutine leak — fixed with context")
	demoLeakFixed()

	section("Panic & recover")
	demoPanic()

	section("Fire and forget")
	demoFireAndForget()

	section("First response wins")
	demoFirstWins()

	section("Bounded concurrency")
	demoBounded()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
