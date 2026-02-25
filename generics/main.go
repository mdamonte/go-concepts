package main

import "fmt"

// Each demo covers one aspect of Go generics (introduced in Go 1.18).
//
// Run:
//
//	go run .
func main() {
	section("Constraints — any, comparable, ~T, union, method")
	demoConstraints()

	section("Functions — Map, Filter, Reduce, Contains, Keys/Values, Must")
	demoFunctions()

	section("Data structures — Stack[T], Queue[T], Set[T comparable]")
	demoDataStructs()

	section("Patterns — inference, multiple params, zero value, Result[T], limitations")
	demoPatterns()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
