package main

import "fmt"

// ── Stack[T] ──────────────────────────────────────────────────────────────────
// LIFO stack backed by a slice. The zero value is ready to use.

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

func (s *Stack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int     { return len(s.items) }
func (s *Stack[T]) IsEmpty() bool { return len(s.items) == 0 }

// ── Queue[T] ──────────────────────────────────────────────────────────────────
// FIFO queue backed by a slice. The zero value is ready to use.
// Note: Dequeue is O(n) due to slice re-slice; use a ring buffer for O(1).

type Queue[T any] struct {
	items []T
}

func (q *Queue[T]) Enqueue(v T) { q.items = append(q.items, v) }

func (q *Queue[T]) Dequeue() (T, bool) {
	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	front := q.items[0]
	q.items = q.items[1:]
	return front, true
}

func (q *Queue[T]) Peek() (T, bool) {
	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	return q.items[0], true
}

func (q *Queue[T]) Len() int     { return len(q.items) }
func (q *Queue[T]) IsEmpty() bool { return len(q.items) == 0 }

// ── Set[T comparable] ─────────────────────────────────────────────────────────
// Unordered collection of unique values. T must be comparable (map key).

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable](vals ...T) *Set[T] {
	s := &Set[T]{m: make(map[T]struct{})}
	for _, v := range vals {
		s.Add(v)
	}
	return s
}

func (s *Set[T]) Add(v T)           { s.m[v] = struct{}{} }
func (s *Set[T]) Remove(v T)        { delete(s.m, v) }
func (s *Set[T]) Contains(v T) bool { _, ok := s.m[v]; return ok }
func (s *Set[T]) Len() int          { return len(s.m) }

// Union returns a new set with all elements from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for v := range s.m {
		result.Add(v)
	}
	for v := range other.m {
		result.Add(v)
	}
	return result
}

// Intersection returns elements present in both sets.
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for v := range s.m {
		if other.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

// Difference returns elements in s that are not in other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for v := range s.m {
		if !other.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

func (s *Set[T]) Slice() []T {
	out := make([]T, 0, len(s.m))
	for v := range s.m {
		out = append(out, v)
	}
	return out
}

func demoDataStructs() {
	fmt.Println("  Stack[int] (LIFO):")
	var st Stack[int]
	for _, v := range []int{1, 2, 3} {
		st.Push(v)
		fmt.Printf("    push %d  len=%d\n", v, st.Len())
	}
	for !st.IsEmpty() {
		v, _ := st.Pop()
		fmt.Printf("    pop → %d\n", v)
	}

	fmt.Println("\n  Stack[string]:")
	var ss Stack[string]
	ss.Push("hello")
	ss.Push("world")
	top, _ := ss.Peek()
	fmt.Printf("    peek=%q  len=%d\n", top, ss.Len())

	fmt.Println("\n  Queue[int] (FIFO):")
	var q Queue[int]
	for _, v := range []int{10, 20, 30} {
		q.Enqueue(v)
	}
	for !q.IsEmpty() {
		v, _ := q.Dequeue()
		fmt.Printf("    dequeue → %d\n", v)
	}

	fmt.Println("\n  Set[string]:")
	a := NewSet("go", "rust", "zig")
	b := NewSet("go", "python", "rust")
	fmt.Println("    a            =", a.Slice())
	fmt.Println("    b            =", b.Slice())
	fmt.Println("    union len    =", a.Union(b).Len())
	fmt.Println("    intersection =", a.Intersection(b).Slice())
	fmt.Println("    a - b        =", a.Difference(b).Slice())
}
