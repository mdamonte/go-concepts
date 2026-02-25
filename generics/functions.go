package main

import "fmt"

// Map transforms every element of s using f.
// T → input type, U → output type (can differ).
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}

// Filter returns elements of s for which f returns true.
func Filter[T any](s []T, f func(T) bool) []T {
	var out []T
	for _, v := range s {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

// Reduce folds s into a single value, applying f left-to-right.
// acc starts at init; T is element type, U is accumulator type.
func Reduce[T, U any](s []T, init U, f func(U, T) U) U {
	acc := init
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

// Contains reports whether v appears in s.
// T must be comparable to support ==.
func Contains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Keys returns all keys of m in unspecified order.
func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Values returns all values of m in unspecified order.
func Values[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// Must unwraps (value, error), panicking if err != nil.
// Useful for initialization paths that should never fail.
//
//	cfg := Must(os.ReadFile("config.json"))
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func demoFunctions() {
	nums := []int{1, 2, 3, 4, 5}

	fmt.Println("  Map[int, string] — squares:")
	strs := Map(nums, func(n int) string { return fmt.Sprintf("%d²=%d", n, n*n) })
	for _, s := range strs {
		fmt.Println("   ", s)
	}

	fmt.Println("\n  Filter — evens:")
	fmt.Println("  ", Filter(nums, func(n int) bool { return n%2 == 0 }))

	fmt.Println("\n  Reduce[int, int] — sum:")
	fmt.Println("  sum =", Reduce(nums, 0, func(acc, n int) int { return acc + n }))

	fmt.Println("\n  Reduce[string, string] — join:")
	joined := Reduce([]string{"go", "rust", "zig"}, "", func(acc, v string) string {
		if acc == "" {
			return v
		}
		return acc + ", " + v
	})
	fmt.Println("  joined =", joined)

	fmt.Println("\n  Contains[int]:")
	fmt.Println("  Contains(nums, 3) =", Contains(nums, 3))
	fmt.Println("  Contains(nums, 9) =", Contains(nums, 9))

	fmt.Println("\n  Keys / Values:")
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	fmt.Println("  len(Keys(m))   =", len(Keys(m)))
	fmt.Println("  len(Values(m)) =", len(Values(m)))

	fmt.Println("\n  Must — unwrap (value, error):")
	fmt.Println("  Must(42, nil)  =", Must(42, nil))
}
