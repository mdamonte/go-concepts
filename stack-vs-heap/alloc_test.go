package main

import "testing"

// BenchmarkReturnValue mide el costo de una asignación en stack.
// El valor se copia al retornar; no hay presión sobre el GC.
func BenchmarkReturnValue(b *testing.B) {
	var sink int
	for i := 0; i < b.N; i++ {
		sink = returnValue()
	}
	_ = sink
}

// BenchmarkReturnPointer mide el costo de una asignación en heap.
// Cada llamada pide memoria al runtime y eventualmente el GC la recolecta.
func BenchmarkReturnPointer(b *testing.B) {
	var sink *int
	for i := 0; i < b.N; i++ {
		sink = returnPointer()
	}
	_ = sink
}

// BenchmarkMakeSlice mide la asignación de un slice en heap.
func BenchmarkMakeSlice(b *testing.B) {
	var sink []int
	for i := 0; i < b.N; i++ {
		sink = makeSlice(64)
	}
	_ = sink
}
