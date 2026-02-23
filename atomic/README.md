# atomic

Ejemplos del paquete `sync/atomic`: la API tipada (Go 1.19+), CAS, `atomic.Value`,
`atomic.Pointer` y los patrones lock-free más comunes.

## Ejecutar

```bash
go run .
```

## Estructura

| Archivo | Contenido |
|---------|-----------|
| `primitives.go` | `Int64`, `Uint64`, `Bool`, CAS loop |
| `value.go` | `atomic.Value` — hot-reload de configuración |
| `pointer.go` | `atomic.Pointer[T]` — publicación de structs inmutables |
| `patterns.go` | contador lock-free, shutdown flag, copy-on-write |

---

## Cuándo usar atomics vs Mutex

| Criterio | atomic | Mutex |
|----------|--------|-------|
| Una sola variable numérica o puntero | ✓ ideal | funciona pero más caro |
| Múltiples variables relacionadas | ✗ no es suficiente | ✓ necesario |
| Read-modify-write con lógica compleja | ✗ CAS loop frágil | ✓ más claro |
| Hot path, latencia crítica | ✓ lock-free | aceptable |
| Configuración read-mostly con pocos writes | ✓ `atomic.Value` | funciona |

---

## Int64 — Add / Load / Store / Swap

La API tipada (`atomic.Int64`, etc.) es preferible a las funciones legacy
(`atomic.AddInt64`, …): el zero value está listo para usar y no puedes
pasar un puntero desalineado por error.

Todos los métodos garantizan consistencia secuencial: sin lecturas rotas,
sin escrituras perdidas, sin reordenamiento entre la operación y el llamador.

```go
// primitives.go
func demoInt64() {
	var n atomic.Int64 // zero value is 0, no init needed

	// Add returns the NEW value after the addition.
	fmt.Println("  Add(10):", n.Add(10))  // 10
	fmt.Println("  Add(5): ", n.Add(5))   // 15
	fmt.Println("  Add(-3):", n.Add(-3))  // 12
	fmt.Println("  Load():  ", n.Load())  // 12

	n.Store(100)
	fmt.Println("  Store(100), Load():", n.Load()) // 100

	old := n.Swap(42)
	fmt.Printf("  Swap(42): old=%d new=%d\n", old, n.Load()) // old=100, new=42

	// Concurrent usage: 100 goroutines each add 1 → expect exactly 100.
	var counter atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()
	fmt.Printf("  concurrent Add ×100 → %d (expected 100)\n", counter.Load())
}
```

---

## Uint64 — contadores sin signo

Ideal para sequence numbers, event counters, byte counters.
El overflow hace wrap-around (semántica normal de `uint64`).

```go
// primitives.go
func demoUint64() {
	var seq atomic.Uint64

	for i := range 5 {
		id := seq.Add(1) // pre-increment: returns new value
		fmt.Printf("  sequence id %d (i=%d)\n", id, i)
	}

	// Wrap-around: max uint64 + 1 == 0.
	var wrap atomic.Uint64
	wrap.Store(^uint64(0)) // max uint64
	wrap.Add(1)
	fmt.Printf("  max uint64 + 1 = %d (wrapped)\n", wrap.Load())
}
```

---

## Bool — flag atómica

El tipo más sencillo. Úsalo para señales de un solo bit: "¿está iniciado?",
"¿hay que parar?", "¿se ejecutó ya?".

```go
// primitives.go
func demoBool() {
	var started atomic.Bool

	fmt.Println("  started:", started.Load()) // false

	started.Store(true)
	fmt.Println("  started:", started.Load()) // true

	// Swap returns the old value.
	old := started.Swap(false)
	fmt.Printf("  Swap(false): old=%v, new=%v\n", old, started.Load())
}
```

---

## CompareAndSwap (CAS)

CAS es la operación atómica fundamental de los algoritmos lock-free.

```
CAS(ptr, old, new):
  if *ptr == old → *ptr = new, return true
  else           → no-op,      return false
```

```go
// primitives.go
func demoCAS() {
	var val atomic.Int64
	val.Store(10)

	// Simple CAS: succeeds because current value == 10.
	swapped := val.CompareAndSwap(10, 20)
	fmt.Printf("  CAS(10→20): swapped=%v, val=%d\n", swapped, val.Load())

	// Fails: current value is now 20, not 10.
	swapped = val.CompareAndSwap(10, 99)
	fmt.Printf("  CAS(10→99): swapped=%v, val=%d\n", swapped, val.Load())

	// CAS loop: safely increment without Add (shows the pattern).
	// Useful when you need read-modify-write with arbitrary logic.
	var counter atomic.Int64
	counter.Store(0)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				old := counter.Load()
				// Only proceed if we win the CAS; otherwise retry.
				if counter.CompareAndSwap(old, old+1) {
					return
				}
			}
		}()
	}
	wg.Wait()
	fmt.Printf("  CAS-loop ×50 → %d (expected 50)\n", counter.Load())
}
```

> ⚠ **ABA problem**: si el valor vuelve a `old` entre el Load y el CAS,
> el CAS tiene éxito aunque otro goroutine haya modificado el valor.
> Para la mayoría de los contadores esto es inofensivo; para estructuras
> más complejas usa versioned pointers o un Mutex.

---

## atomic.Value — hot-reload de configuración

`atomic.Value` almacena un `interface{}` de forma atómica. El tipo
concreto debe ser **siempre el mismo** en cada `Store`. No acepta `nil`.

```go
// value.go
type Config struct {
	MaxConns int
	Timeout  time.Duration
	Feature  string
}

func demoValue() {
	var cfgVal atomic.Value

	// Initial configuration — stored as a pointer to allow nil-check on Load.
	cfgVal.Store(&Config{MaxConns: 10, Timeout: 5 * time.Second, Feature: "v1"})

	var wg sync.WaitGroup

	// Simulate 5 readers running concurrently with a writer.
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cfg := cfgVal.Load().(*Config) // always a consistent snapshot
			fmt.Printf("  reader %d: maxConns=%d feature=%s\n", id, cfg.MaxConns, cfg.Feature)
		}(i)
	}

	// Writer: replaces the config atomically.
	// Readers that already loaded the old pointer keep a valid copy;
	// new readers will see the updated config.
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Millisecond) // let readers start first
		cfgVal.Store(&Config{MaxConns: 50, Timeout: 10 * time.Second, Feature: "v2"})
		fmt.Println("  writer: config reloaded to v2")
	}()

	wg.Wait()

	// Confirm the final value.
	final := cfgVal.Load().(*Config)
	fmt.Printf("  final config: maxConns=%d feature=%s\n", final.MaxConns, final.Feature)
}
```

---

## atomic.Pointer[T] — publicación de structs (Go 1.19+)

`atomic.Pointer[T]` es la alternativa tipada y genérica a `atomic.Value`
para punteros. Ventajas: acepta `nil`, no requiere type assertion, sin boxing.

```go
// pointer.go
type Node struct {
	Value int
	Label string
}

func demoPointer() {
	var ptr atomic.Pointer[Node]

	// ptr.Load() returns nil before any Store.
	fmt.Println("  before Store:", ptr.Load()) // <nil>

	ptr.Store(&Node{Value: 1, Label: "initial"})
	n := ptr.Load()
	fmt.Printf("  after Store:  value=%d label=%s\n", n.Value, n.Label)

	// Swap: replace and get the old pointer.
	old := ptr.Swap(&Node{Value: 2, Label: "updated"})
	fmt.Printf("  Swap: old={%d %s} new={%d %s}\n",
		old.Value, old.Label,
		ptr.Load().Value, ptr.Load().Label,
	)

	// CompareAndSwap on the pointer.
	current := ptr.Load()
	swapped := ptr.CompareAndSwap(current, &Node{Value: 3, Label: "cas"})
	fmt.Printf("  CAS: swapped=%v val=%d label=%s\n", swapped, ptr.Load().Value, ptr.Load().Label)

	// Concurrent publish pattern: one writer, many readers.
	var latest atomic.Pointer[Node]
	latest.Store(&Node{Value: 0, Label: "start"})

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			// Replace the shared pointer with a fresh immutable struct.
			latest.Store(&Node{Value: v, Label: fmt.Sprintf("gen-%d", v)})
		}(i)
	}
	wg.Wait()

	snap := latest.Load()
	fmt.Printf("  final snapshot: value=%d label=%s\n", snap.Value, snap.Label)
}
```

---

## Patrón: contador lock-free vs Mutex

```go
// patterns.go
func demoLockFreeCounter() {
	const goroutines = 8
	const increments = 100_000

	// ── Atomic ───────────────────────────────────────────────────────────────
	var atomicCount atomic.Int64
	var wg sync.WaitGroup

	start := time.Now()
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range increments {
				atomicCount.Add(1)
			}
		}()
	}
	wg.Wait()
	atomicDur := time.Since(start)

	// ── Mutex ─────────────────────────────────────────────────────────────────
	var mu sync.Mutex
	var mutexCount int64

	start = time.Now()
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range increments {
				mu.Lock()
				mutexCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	mutexDur := time.Since(start)

	expected := int64(goroutines * increments)
	fmt.Printf("  atomic: %d in %v\n", atomicCount.Load(), atomicDur.Round(time.Millisecond))
	fmt.Printf("  mutex:  %d in %v\n", mutexCount, mutexDur.Round(time.Millisecond))
	fmt.Printf("  expected: %d — both correct: %v\n",
		expected, atomicCount.Load() == expected && mutexCount == expected)
}
```

Salida típica:
```
atomic: 800000 in 38ms
mutex:  800000 in 122ms
expected: 800000 — both correct: true
```

---

## Patrón: shutdown flag

Señal de parada para un conjunto dinámico de workers sin conocer cuántos hay.

```go
// patterns.go
func demoShutdownFlag() {
	var shutdown atomic.Bool
	var wg sync.WaitGroup

	for i := range 3 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for tick := range 20 {
				if shutdown.Load() {
					fmt.Printf("  worker %d: saw shutdown at tick %d\n", id, tick)
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Signal after 35 ms — workers typically see it at tick 3–4.
	time.Sleep(35 * time.Millisecond)
	shutdown.Store(true)
	fmt.Println("  shutdown flag set")

	wg.Wait()
	fmt.Println("  all workers stopped")
}
```

> Alternativa: `close(done chan struct{})` — más idiomática cuando todos
> los goroutines son conocidos en el momento del shutdown.
> La flag atómica es útil cuando los workers se crean dinámicamente.

---

## Patrón: copy-on-write (COW)

Los lectores obtienen siempre un snapshot consistente; los escritores
clonan la estructura y reemplazan el puntero con CAS.

```go
// patterns.go
type SliceSnapshot struct {
	Items []string
}

func demoCopyOnWrite() {
	var snap atomic.Pointer[SliceSnapshot]
	snap.Store(&SliceSnapshot{Items: []string{"a", "b", "c"}})

	// append atomically: load, clone, append, CAS-replace.
	appendItem := func(item string) {
		for {
			old := snap.Load()
			// Build a new slice — old is never modified.
			newItems := make([]string, len(old.Items)+1)
			copy(newItems, old.Items)
			newItems[len(old.Items)] = item
			next := &SliceSnapshot{Items: newItems}
			if snap.CompareAndSwap(old, next) {
				return // won the race
			}
			// Lost to another writer — retry with the latest snapshot.
		}
	}

	var wg sync.WaitGroup
	for _, item := range []string{"d", "e", "f"} {
		wg.Add(1)
		go func(v string) {
			defer wg.Done()
			appendItem(v)
		}(item)
	}
	wg.Wait()

	current := snap.Load()
	fmt.Printf("  snapshot after concurrent appends (%d items): %v\n",
		len(current.Items), current.Items)
}
```

Trade-off: escrituras O(n) (clonar), lecturas O(1) lock-free.
Ideal para slices leídas millones de veces y escritas raramente.

---

## Reglas clave

1. **Usa la API tipada** (`atomic.Int64`, `atomic.Bool`, …) sobre las funciones legacy.
2. **`atomic.Value` impone el mismo tipo** en cada `Store` — si mezclas tipos, panic en runtime.
3. **`atomic.Pointer` acepta nil**; `atomic.Value` no (panic).
4. **Los atomics no reemplazan un Mutex** cuando proteges múltiples variables relacionadas.
5. **Cuidado con el ABA problem** en CAS loops sobre punteros de estructuras mutables.
6. **Preferir `context.Done()` o `close(ch)`** sobre una flag bool para shutdown coordinado.
