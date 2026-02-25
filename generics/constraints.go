package main

import "fmt"

// ── any ───────────────────────────────────────────────────────────────────────
// any is an alias for interface{}. Use it when T can be literally anything —
// no operations other than assign and pass are allowed on T.

func Identity[T any](v T) T { return v }

// ── comparable ────────────────────────────────────────────────────────────────
// comparable means T supports == and !=.
// Comparable types: numbers, strings, booleans, pointers, channels,
//   arrays, structs whose fields are all comparable.
// NOT comparable: slices, maps, functions.

func Equal[T comparable](a, b T) bool { return a == b }

// ── Ordered — union constraint ────────────────────────────────────────────────
// The | operator creates a union: T must be one of the listed types.
// Used whenever you need < > <= >= (e.g. sorting, min/max).

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// ── ~T — underlying type constraint ──────────────────────────────────────────
// ~float64 means "any type whose underlying type is float64",
// including user-defined types like Celsius or Fahrenheit.
//
// Without ~, defined types would NOT satisfy the constraint:
//   float64  → only the named type float64
//   ~float64 → float64 AND any type defined as `type X float64`

type Celsius float64
type Fahrenheit float64

// Temperature accepts both Celsius and Fahrenheit because both have
// underlying type float64.
type Temperature interface{ ~float64 }

func AbsDiff[T Temperature](a, b T) T {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}

// ── Method constraint ─────────────────────────────────────────────────────────
// Constraints can require methods, just like regular interfaces.
// A type satisfies the constraint only if it implements all listed methods.

type Stringer interface{ String() string }

func PrintAll[T Stringer](items []T) {
	for _, v := range items {
		fmt.Println(" ", v.String())
	}
}

type Point struct{ X, Y float64 }

func (p Point) String() string { return fmt.Sprintf("(%.1f, %.1f)", p.X, p.Y) }

// ── Number — combining union types ────────────────────────────────────────────
// Constraints can combine multiple types into a reusable interface.
// Note: a constraint with a union can ONLY be used as a type parameter;
// you cannot declare `var x Number` — that is a compile error.

type Number interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

func Sum[T Number](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}

func demoConstraints() {
	fmt.Println("  any — Identity:")
	fmt.Println("  ", Identity(42), Identity("hello"), Identity(true))

	fmt.Println("\n  comparable — Equal:")
	fmt.Println("  Equal(1, 1)     =", Equal(1, 1))
	fmt.Println("  Equal(\"a\",\"b\") =", Equal("a", "b"))

	fmt.Println("\n  Ordered — Min / Max:")
	fmt.Println("  Min(3, 5)              =", Min(3, 5))
	fmt.Println("  Max(3.14, 2.71)        =", Max(3.14, 2.71))
	fmt.Println("  Min(\"apple\",\"banana\") =", Min("apple", "banana"))

	fmt.Println("\n  ~float64 — defined types satisfy ~T constraint:")
	fmt.Println("  AbsDiff(100°C, 20°C)   =", AbsDiff(Celsius(100), Celsius(20)))
	fmt.Println("  AbsDiff(212°F,  32°F)  =", AbsDiff(Fahrenheit(212), Fahrenheit(32)))

	fmt.Println("\n  Method constraint (Stringer):")
	PrintAll([]Point{{1, 2}, {3, 4}, {5, 6}})

	fmt.Println("\n  Number union — Sum:")
	fmt.Println("  Sum([]int{1..5})        =", Sum([]int{1, 2, 3, 4, 5}))
	fmt.Println("  Sum([]float64{...})     =", Sum([]float64{1.1, 2.2, 3.3}))
}
