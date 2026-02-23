# Race Conditions en Go — Guía completa con ejemplos

## ¿Qué es una race condition?

Una race condition ocurre cuando el resultado de un programa depende del orden
no determinístico en que los goroutines acceden a memoria compartida.

```
goroutine A: LOAD counter → 5
goroutine B: LOAD counter → 5   ← lee el mismo valor que A
goroutine A: STORE counter ← 6
goroutine B: STORE counter ← 6  ← sobrescribe a A: se perdió un incremento
```

---

## El race detector

Go incluye un race detector basado en ThreadSanitizer. Detecta accesos concurrentes
sin sincronización en tiempo de ejecución:

```bash
go run -race .
go test -race ./...
go build -race -o app .
```

Cuando detecta una race imprime el stack trace de ambos accesos y termina el proceso:

```
WARNING: DATA RACE
Write at 0x... by goroutine 7:
  main.demoCounterRace.func1()
      counter.go:38

Previous write at 0x... by goroutine 6:
  main.demoCounterRace.func1()
      counter.go:38
```

El race detector tiene ~2-20× overhead de CPU y memoria. Úsalo siempre en tests;
en producción solo si el overhead es aceptable.

---

## Archivos del módulo

```
race-conditions/
├── go.mod
├── main.go       — ejecuta todos los demos en orden
├── counter.go    — data race en contador + 3 fixes
├── map.go        — acceso concurrente a map + 2 fixes
├── checkact.go   — check-then-act (TOCTOU) + fix
└── publish.go    — publication hazard + 2 fixes
```

---

## Cómo correrlo

```bash
# Normal — el contador muestra lost updates visibles
go run .

# Con race detector — detecta cada acceso sin sincronización
go run -race .
```

---

## Ejemplos

### Data race en contador (`counter.go`)

`counter++` compila a tres instrucciones (LOAD, ADD, STORE). Si dos goroutines
intercalan sus instrucciones, ambas leen el mismo valor y una escritura se pierde.

```go
const (
	goroutines = 100
	increments = 10_000
	expected   = goroutines * increments // 1_000_000
)

// demoCounterRace: 100 goroutines × 10.000 incrementos = 1.000.000 esperado.
// Sin sincronización el resultado es siempre menor por lost updates.
// Run with -race to have the race detector flag every unsynchronized access.
func demoCounterRace() {
	var counter int
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				counter++ // DATA RACE: read-modify-write is not atomic
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  expected: %d  got: %d  lost updates: %d\n",
		expected, counter, expected-counter)
}
```

#### Fix 1 — `sync.Mutex`

```go
func demoCounterMutex() {
	var (
		counter int
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				mu.Lock()
				counter++ // protected: only one goroutine here at a time
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  expected: %d  got: %d  ✓\n", expected, counter)
}
```

#### Fix 2 — `sync/atomic`

```go
func demoCounterAtomic() {
	var counter atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				counter.Add(1) // atomic: single indivisible instruction
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  expected: %d  got: %d  ✓\n", expected, counter.Load())
}
```

#### Fix 3 — Channel (actor model)

Un único goroutine es dueño del contador. Los demás envían peticiones vía canal.
Sin memoria compartida → sin race por construcción.

```go
func demoCounterChannel() {
	inc := make(chan struct{}, 512) // buffer absorbs bursts
	done := make(chan int)

	// Actor: sole owner of the counter.
	go func() {
		counter := 0
		for range inc {
			counter++
		}
		done <- counter
	}()

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				inc <- struct{}{}
			}
		}()
	}

	wg.Wait()
	close(inc)   // signal actor: no more increments
	counter := <-done
	fmt.Printf("  expected: %d  got: %d  ✓\n", expected, counter)
}
```

---

### Race en map (`map.go`)

A diferencia del contador, un acceso concurrente a un map no es solo una race —
es un error fatal que el runtime detecta y que **no puede recuperarse con `recover`**:

```
fatal error: concurrent map read and map write
```

```go
func demoMapRace() {
	fmt.Println("  racy map code shown below — not executed to avoid fatal crash:")
	fmt.Println(`
  m := make(map[string]int)
  go func() { m["a"]++ }()   // writer goroutine
  go func() { _ = m["b"] }() // reader goroutine
  // → fatal error: concurrent map read and map write`)
}
```

#### Fix 1 — `sync.RWMutex`

```go
func demoMapRWMutex() {
	var mu sync.RWMutex
	m := make(map[string]int)
	var wg sync.WaitGroup

	// 5 concurrent writers.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			mu.Lock()
			m[key] = id * 10 // exclusive write
			mu.Unlock()
		}(i)
	}

	// 5 concurrent readers.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			mu.RLock()
			_ = m[key] // shared read — safe alongside other RLocks
			mu.RUnlock()
		}(i)
	}

	wg.Wait()

	mu.RLock()
	fmt.Printf("  map has %d entries  ✓\n", len(m))
	mu.RUnlock()
}
```

#### Fix 2 — `sync.Map`

```go
func demoMapSyncMap() {
	var m sync.Map
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			m.Store(fmt.Sprintf("key%d", id), id*10) // no external lock needed
		}(i)
	}
	wg.Wait()

	count := 0
	m.Range(func(_, _ any) bool { count++; return true })
	fmt.Printf("  sync.Map has %d entries  ✓\n", count)
}
```

| Fix | Cuándo usarlo |
|---|---|
| `map + RWMutex` | Control total sobre el tipo; escrituras frecuentes |
| `sync.Map` | Escritura única + muchas lecturas; claves disjuntas por goroutine |

---

### Check-then-act / TOCTOU (`checkact.go`)

La condición cambia entre el momento en que se comprueba y el momento en que se actúa.

```
goroutine A: balance=100 → check (100 >= 100) ✓ → ...
goroutine B: balance=100 → check (100 >= 100) ✓ → balance -= 100 → balance=0
goroutine A: ...                               → balance -= 100 → balance=-100
```

```go
type account struct {
	balance int
}

// withdrawRacy reads the balance and deducts in two separate steps.
// Between the check and the deduct, another goroutine can run its own check
// and see the same (unmodified) balance — both withdrawals succeed.
func (a *account) withdrawRacy(amount int) bool {
	if a.balance >= amount { // CHECK — balance looks sufficient
		// ← another goroutine can run here and also pass the check
		a.balance -= amount // ACT — balance is now negative
		return true
	}
	return false
}

// demoCheckActRace: 10 goroutines each try to withdraw 100 from balance=100.
// Without synchronization, multiple withdrawals succeed → negative balance.
func demoCheckActRace() {
	a := &account{balance: 100}
	var wg sync.WaitGroup
	successes := 0
	var mu sync.Mutex // only to safely count successes for printing

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if a.withdrawRacy(100) { // DATA RACE on a.balance
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  balance: %d  successful withdrawals: %d  (expected balance ≥ 0)\n",
		a.balance, successes)
}
```

#### Fix — mantener el lock durante toda la operación

```go
type safeAccount struct {
	mu      sync.Mutex
	balance int
}

// withdraw holds the lock across the entire check-and-act sequence.
// No other goroutine can sneak in between the read and the write.
func (a *safeAccount) withdraw(amount int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.balance >= amount { // check
		a.balance -= amount  // act — atomic with respect to other goroutines
		return true
	}
	return false
}

func demoCheckActFixed() {
	a := &safeAccount{balance: 100}
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if a.withdraw(100) {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  balance: %d  successful withdrawals: %d  ✓\n",
		a.balance, successes) // balance: 0, withdrawals: 1
}
```

La regla: **todo el check-then-act debe ocurrir bajo el mismo lock**. Liberar
el lock entre la comprobación y la acción vuelve a abrir la ventana de race.

---

### Publication hazard (`publish.go`)

Una goroutine publica un puntero (lo hace no-nil) antes de que todas las escrituras
a los campos del struct sean visibles para otras goroutines. Los CPUs y compiladores
pueden reordenar escrituras para optimizar.

Este es el caso más sutil: `-race` puede no detectarlo porque no siempre hay un
interleaving observable — el problema es el **ordering de memoria**, no la exclusión mutua.

```go
type config struct {
	host    string
	port    int
	timeout time.Duration
}

var (
	instance *config
	initMu   sync.Mutex
)

// getConfigRacy illustrates the anti-pattern.
// The outer check (`instance != nil`) is an unsynchronized read.
// Even if instance is non-nil, its fields may be zero.
func getConfigRacy() *config {
	if instance != nil { // unsynchronized read — DATA RACE
		return instance
	}
	initMu.Lock()
	defer initMu.Unlock()
	if instance == nil {
		instance = &config{
			host:    "localhost",
			port:    5432,
			timeout: 30 * time.Second,
		}
		// The pointer store and the field stores can be reordered by the CPU.
		// Another goroutine calling getConfigRacy may observe instance != nil
		// while host/port/timeout still contain their zero values.
	}
	return instance
}
```

#### Fix 1 — `sync.Once` (recomendado para singletons)

`sync.Once` garantiza que todos los stores del inicializador son visibles para
cualquier goroutine que observe el Once como "done" (happens-before guarantee).

```go
var (
	cfgOnce     sync.Once
	cfgInstance *config
)

func getConfigOnce() *config {
	cfgOnce.Do(func() {
		cfgInstance = &config{
			host:    "localhost",
			port:    5432,
			timeout: 30 * time.Second,
		}
	})
	return cfgInstance
}

func demoPublishFixed() {
	var wg sync.WaitGroup
	results := make([]*config, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results[id] = getConfigOnce() // safe: Once provides happens-before
		}(i)
	}
	wg.Wait()

	// All goroutines must receive the same pointer with fully visible fields.
	first := results[0]
	allSame := true
	for _, r := range results {
		if r != first || r.host == "" || r.port == 0 {
			allSame = false
		}
	}
	fmt.Printf("  all goroutines got same config: %v  host=%s port=%d  ✓\n",
		allSame, first.host, first.port)
}
```

#### Fix 2 — `atomic.Pointer[T]` (para hot-reload)

Para configs que se reemplazan en runtime, `atomic.Pointer` provee publish-and-replace
seguro sin mutex. `Store` y `Load` garantizan el mismo ordering que `sync/atomic`.

```go
var atomicCfg atomic.Pointer[config]

func publishConfig(c *config) {
	atomicCfg.Store(c) // atomic store: all fields visible to any Load after this
}

func readConfig() *config {
	return atomicCfg.Load() // atomic load: sees fully initialized struct or nil
}
```

---

## Cuándo usar cada fix

| Race | Fix recomendado | Alternativa |
|---|---|---|
| Contador simple | `atomic.Int64` | `sync.Mutex` |
| Struct con múltiples campos | `sync.Mutex` | canal (actor) |
| Map concurrente | `map + RWMutex` | `sync.Map` |
| Check-then-act | `sync.Mutex` span completo | CAS con `atomic` |
| Inicialización lazy | `sync.Once` | canal cerrado como señal |
| Config hot-reload | `atomic.Pointer[T]` | `sync.RWMutex` |

---

## Reglas prácticas

| Regla | Motivo |
|---|---|
| Correr tests con `-race` siempre | El race detector es el único que ve races sutiles |
| No leer variables compartidas fuera de un lock | Una lectura concurrente con una escritura es una race |
| El lock debe cubrir todo el check-then-act | Liberar entre check y act reabre la ventana |
| Nunca escribir en un map sin sincronización | Fatal error no recuperable |
| Usar `sync.Once` para publicar singletons | Garantiza ordering además de ejecución única |
| `atomic` solo para tipos simples | Para structs, siempre `sync.Mutex` |
