package main

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// nil slice vs empty slice — the most common "trick question" about slices.
//
//   var s []int        → nil slice  : s == nil → true,  len=0, cap=0
//   s := []int{}       → empty slice: s == nil → false, len=0, cap=0
//   s := make([]int,0) → empty slice: s == nil → false, len=0, cap=0
//
// For most operations they are identical. The differences matter for:
//   • JSON marshaling (null vs [])
//   • reflect.DeepEqual
//   • explicit nil checks in APIs

func demoNil() {
	var nilSlice []int
	emptyLiteral := []int{}
	emptyMake := make([]int, 0)

	// ── Declaration and nil check ─────────────────────────────────────────────
	fmt.Println("  Declaration:")
	fmt.Printf("  var s []int       nil=%-5v len=%d cap=%d\n",
		nilSlice == nil, len(nilSlice), cap(nilSlice))
	fmt.Printf("  s := []int{}      nil=%-5v len=%d cap=%d\n",
		emptyLiteral == nil, len(emptyLiteral), cap(emptyLiteral))
	fmt.Printf("  make([]int, 0)    nil=%-5v len=%d cap=%d\n",
		emptyMake == nil, len(emptyMake), cap(emptyMake))

	// ── Both work the same for common operations ──────────────────────────────
	fmt.Println("\n  All three behave identically for range, len, cap, append:")
	for range nilSlice {
	} // zero iterations, no panic
	nilSlice = append(nilSlice, 1, 2, 3)
	fmt.Println("  append to nil:", nilSlice) // perfectly valid

	// ── JSON marshaling — the key difference ─────────────────────────────────
	// nil slice  → marshals as JSON null
	// empty slice → marshals as JSON []
	// This distinction is critical in API responses.
	fmt.Println("\n  JSON marshaling (critical for API responses):")
	type Resp struct {
		Items []int `json:"items"`
	}
	nilJSON, _ := json.Marshal(Resp{Items: nil})
	emptyJSON, _ := json.Marshal(Resp{Items: []int{}})
	fmt.Printf("  nil slice   → %s\n", nilJSON)   // {"items":null}
	fmt.Printf("  empty slice → %s\n", emptyJSON) // {"items":[]}

	// ── reflect.DeepEqual ─────────────────────────────────────────────────────
	// DeepEqual distinguishes nil from empty — this can surprise test writers.
	fmt.Println("\n  reflect.DeepEqual treats nil and empty as different:")
	fmt.Println("  DeepEqual(nil, []int{})  =", reflect.DeepEqual([]int(nil), []int{}))
	fmt.Println("  DeepEqual([]int{}, []int{}) =", reflect.DeepEqual([]int{}, []int{}))

	// ── Slices are NOT comparable with == ────────────────────────────────────
	// Two non-nil slices cannot be compared directly — it's a compile error.
	// The only valid == comparison for a slice is against nil.
	fmt.Println("\n  Slice comparison — == only works against nil:")
	s := []int{1, 2, 3}
	fmt.Println("  s == nil:", s == nil) // only valid use of == on a slice
	// s == []int{1,2,3}  ← compile error: invalid operation

	// Compare two slices: use reflect.DeepEqual or slices.Equal (Go 1.21+)
	fmt.Println("  DeepEqual([1,2,3],[1,2,3]):", reflect.DeepEqual(s, []int{1, 2, 3}))

	// ── Convention ───────────────────────────────────────────────────────────
	fmt.Println("\n  Convention:")
	fmt.Println("  • Return nil (not []int{}) to represent 'no results' — idiomatic Go")
	fmt.Println("  • Use []int{} or make([]int,0) only when JSON [] is required,")
	fmt.Println("    or when the caller must distinguish 'empty' from 'uninitialized'")

	// ── Gotcha: nil map vs nil slice ─────────────────────────────────────────
	// A nil map panics on write; a nil slice does not panic on append.
	fmt.Println("\n  nil slice vs nil map behavior:")
	fmt.Println("  append to nil slice → safe (returns new slice)")
	fmt.Println("  write to nil map    → panic: assignment to entry in nil map")
}
