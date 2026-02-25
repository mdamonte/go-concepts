package main

import "fmt"

// Each demo covers one aspect of Go slices that appears in technical interviews.
//
// Run:
//
//	go run .
func main() {
	section("Internals — header {ptr, len, cap}, shared backing array, pass-by-value")
	demoInternals()

	section("Append — in-place vs reallocation, subslice gotcha, 3-index slice")
	demoAppend()

	section("Operations — copy, delete, insert, filter, reverse, dedup")
	demoOperations()

	section("Nil vs empty — JSON, reflect.DeepEqual, comparison gotcha")
	demoNil()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
