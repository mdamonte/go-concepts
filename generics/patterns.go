package main

import "fmt"

// ── Type inference ────────────────────────────────────────────────────────────
// Go infers type parameters from arguments when unambiguous.
// Explicit syntax is always available: Double[int](21).

func Double[T Number](v T) T { return v * 2 }

// ── Multiple type parameters ──────────────────────────────────────────────────
// Functions and types can have more than one type parameter.

type Pair[A, B any] struct {
	First  A
	Second B
}

func NewPair[A, B any](a A, b B) Pair[A, B] { return Pair[A, B]{a, b} }
func (p Pair[A, B]) String() string          { return fmt.Sprintf("(%v, %v)", p.First, p.Second) }

// ── Zero value of a type parameter ───────────────────────────────────────────
// `var zero T` gives the zero value for any T:
//   int/float → 0, string → "", bool → false, pointer/slice/map → nil.
// This is the idiomatic way to return "nothing" from a generic function.

func First[T any](s []T) (T, bool) {
	if len(s) == 0 {
		var zero T
		return zero, false
	}
	return s[0], true
}

// ── Result[T] — generic result type ──────────────────────────────────────────
// Encapsulates a value OR an error, inspired by Rust's Result<T, E>.
// Useful when passing results through channels or collecting async outcomes.

type Result[T any] struct {
	Value T
	Err   error
}

func Ok[T any](v T) Result[T]      { return Result[T]{Value: v} }
func Err[T any](e error) Result[T] { return Result[T]{Err: e} }

func (r Result[T]) IsOk() bool { return r.Err == nil }

func (r Result[T]) Unwrap() T {
	if r.Err != nil {
		panic(r.Err)
	}
	return r.Value
}

// ── GroupBy — comparable as map key ──────────────────────────────────────────
// K must be comparable to use as a map key.

func GroupBy[T any, K comparable](s []T, key func(T) K) map[K][]T {
	m := make(map[K][]T)
	for _, v := range s {
		k := key(v)
		m[k] = append(m[k], v)
	}
	return m
}

// ── Type switch via any(v) ────────────────────────────────────────────────────
// You cannot directly type-switch on a type parameter (v.(type) is invalid
// when T is not an interface in the current scope).
// Workaround: cast to any first. This loses static type safety — prefer
// interface methods or separate overloads when possible.

func Describe[T any](v T) string {
	switch x := any(v).(type) {
	case int:
		return fmt.Sprintf("int(%d)", x)
	case string:
		return fmt.Sprintf("string(%q)", x)
	case float64:
		return fmt.Sprintf("float64(%g)", x)
	case bool:
		return fmt.Sprintf("bool(%v)", x)
	default:
		return fmt.Sprintf("%T(%v)", x, x)
	}
}

// ── Limitation: no generic methods on non-generic types ──────────────────────
//
// The following does NOT compile:
//
//   type MySlice []int
//   func (s MySlice) Map[U any](f func(int) U) []U { ... }  // INVALID
//
// Methods cannot introduce new type parameters. Workarounds:
//   1. Top-level generic function:  Map(mySlice, f)
//   2. Make the receiver itself generic:  type MySlice[T any] []T

// ── Limitation: no type assertion on unconstrained T ─────────────────────────
//
// The following does NOT compile when T is constrained to a union:
//
//   func f[T Number](v T) {
//       _ = v.(int)  // INVALID — T is not an interface
//   }
//
// Workaround: any(v).(int) — but this is a runtime assertion, not compile-time.

// ── Instantiation — pinning a concrete type ───────────────────────────────────
// A generic type becomes concrete when you provide type arguments.
// You can alias the instantiation for readability.

type IntStack = Stack[int]
type StringQueue = Queue[string]

func demoPatterns() {
	fmt.Println("  Type inference:")
	fmt.Println("  Double(21)       =", Double(21))      // Double[int]
	fmt.Println("  Double(3.14)     =", Double(3.14))    // Double[float64]
	fmt.Println("  Double[int64](7) =", Double[int64](7)) // explicit

	fmt.Println("\n  Multiple type parameters — Pair[A, B]:")
	fmt.Println("  ", NewPair("age", 30))
	fmt.Println("  ", NewPair(true, []int{1, 2, 3}))

	fmt.Println("\n  Zero value of type parameter:")
	v1, ok1 := First([]int{10, 20, 30})
	fmt.Printf("  First([10,20,30]) = %v ok=%v\n", v1, ok1)
	v2, ok2 := First([]string{})
	fmt.Printf("  First([])         = %q ok=%v  ← zero value of string\n", v2, ok2)

	fmt.Println("\n  Result[T]:")
	r1 := Ok(42)
	r2 := Err[int](fmt.Errorf("not found"))
	fmt.Println("  Ok(42).IsOk()    =", r1.IsOk(), "  value =", r1.Unwrap())
	fmt.Println("  Err(...).IsOk()  =", r2.IsOk(), " err   =", r2.Err)

	fmt.Println("\n  GroupBy — words by length:")
	words := []string{"go", "rust", "zig", "java", "c"}
	byLen := GroupBy(words, func(s string) int { return len(s) })
	for _, n := range []int{1, 2, 3, 4} {
		if g, ok := byLen[n]; ok {
			fmt.Printf("  len=%d: %v\n", n, g)
		}
	}

	fmt.Println("\n  Type switch via any(v).(type):")
	fmt.Println("  ", Describe(42))
	fmt.Println("  ", Describe("hello"))
	fmt.Println("  ", Describe(3.14))
	fmt.Println("  ", Describe(true))

	fmt.Println("\n  Instantiation aliases:")
	var si IntStack
	si.Push(100)
	si.Push(200)
	fmt.Println("  IntStack len =", si.Len())
	var sq StringQueue
	sq.Enqueue("hello")
	fmt.Println("  StringQueue len =", sq.Len())
}
