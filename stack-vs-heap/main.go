package main

import "fmt"

// ── Stack ────────────────────────────────────────────────────────────────────

// returnValue devuelve una copia del valor.
// El compilador mantiene x en el stack frame de esta función.
// Cuando la función retorna, el frame desaparece y x con él.
//
// Escape analysis: x does NOT escape.
func returnValue() int {
	x := 42
	return x // copia; nadie guarda la dirección de x
}

// sumArray trabaja con un array de tamaño fijo y conocido en tiempo de compilación.
// Go puede reservarlo entero en el stack.
//
// Escape analysis: arr does NOT escape.
func sumArray() int {
	arr := [5]int{1, 2, 3, 4, 5}
	total := 0
	for _, v := range arr {
		total += v
	}
	return total
}

// ── Heap ─────────────────────────────────────────────────────────────────────

// returnPointer devuelve la dirección de x.
// La dirección debe seguir siendo válida después de que la función retorne,
// así que el compilador mueve x al heap (escape analysis lo detecta).
//
// Escape analysis: x escapes to heap.
func returnPointer() *int {
	x := 42
	return &x // &x escapa: su vida útil supera al frame de la función
}

// closureCapture devuelve un closure que captura x.
// x debe sobrevivir al frame que lo creó → escapa al heap.
//
// Escape analysis: x escapes to heap.
func closureCapture() func() int {
	x := 0
	return func() int {
		x++ // x vive en el heap; el closure lo referencia
		return x
	}
}

// interfaceBox recibe un valor concreto como `any` (interface{}).
// Para satisfacer la interfaz, Go necesita guardar el valor en el heap
// junto con un puntero a su tipo (itab).
//
// Escape analysis: v escapes to heap.
func interfaceBox(v any) string {
	return fmt.Sprintf("%v", v)
}

// makeSlice crea un slice con make.
// Los slices viven en el heap porque su tamaño puede no conocerse
// en tiempo de compilación y/o pueden crecer.
//
// Escape analysis: slice escapes to heap.
func makeSlice(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i * 2
	}
	return s
}

// ── main ─────────────────────────────────────────────────────────────────────

func main() {
	// --- Stack ---
	fmt.Println("=== STACK ===")

	v := returnValue()
	fmt.Printf("returnValue()  → %d  (copia en stack)\n", v)

	sum := sumArray()
	fmt.Printf("sumArray()     → %d  (array fijo en stack)\n", sum)

	// --- Heap ---
	fmt.Println("\n=== HEAP ===")

	p := returnPointer()
	fmt.Printf("returnPointer() → %d  addr=%p  (x escapó al heap)\n", *p, p)

	counter := closureCapture()
	fmt.Printf("closureCapture() → %d, %d  (x capturado en heap)\n", counter(), counter())

	msg := interfaceBox(99)
	fmt.Printf("interfaceBox(99) → %q  (valor boxeado en heap)\n", msg)

	s := makeSlice(4)
	fmt.Printf("makeSlice(4)    → %v  (slice en heap)\n", s)
}
