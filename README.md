# Go Concurrency — Ejemplos

Colección de módulos independientes que ilustran los conceptos de concurrencia
y programación de sistemas en Go, desde lo más básico hasta patrones de producción.

---

## Estructura del proyecto

```
concurrency/
├── main.go          — fan-out con context y timeout (punto de entrada raíz)
│
├── deadlock/        — todos los estados de bloqueo: chan, select, IO wait, semacquire, running
├── stack-vs-heap/   — escape analysis: dónde vive cada variable
├── interfaces/      — interfaces: declaración, composición, type switch
│
├── goroutines/      — goroutines: lifecycle, leaks, panics, patrones
├── channels/        — channels: buffered, select, pipeline, fan-out, semáforo
├── sync/            — sync: Mutex, WaitGroup, Once, Cond, Pool, Map, atomic
├── context/         — context: cancel, timeout, deadline, value, HTTP
├── race-conditions/ — data race, map race, TOCTOU, publication hazard
├── timers/          — Timer, Ticker, time.After, debounce, rate limit, backoff
├── atomic/          — Int64, Bool, Value, Pointer, CAS, lock-free patterns
├── errors/          — sentinel, tipos custom, wrapping %w, Is/As, Join, panic vs error
├── generics/        — constraints, funciones genéricas, Stack/Queue/Set, patterns
├── slices/          — header {ptr,len,cap}, append, 3-index, nil vs empty, operations
│
└── worker-pool/     — worker pool de producción con shutdown graceful y métricas
```

---

## Módulos

### [`deadlock/`](deadlock/README.md) — Deadlock & Goroutine States

Ejemplos de cada estado de bloqueo visible en un goroutine dump, más el deadlock
AB clásico. Cada demo llama a `runtime.Stack` para mostrar la etiqueta de estado
directamente en el terminal — sin necesidad de pprof.

```go
// channel.go — [chan receive]: bloqueado en <-ch sin sender
func demoChanReceive() {
	ch := make(chan int) // unbuffered — no sender will ever write
	go func() {
		v := <-ch // ← blocked here, shows as [chan receive]
		fmt.Println("received", v) // unreachable
	}()
	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
}

// channel.go — [chan send]: bloqueado en ch <- v sin receiver
func demoChanSend() {
	ch := make(chan int) // unbuffered — no receiver will ever read
	go func() {
		ch <- 42 // ← blocked here, shows as [chan send]
	}()
	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
}

// channel.go — [select]: todos los cases bloqueados
func demoSelect() {
	ch1 := make(chan int)
	ch2 := make(chan string)
	go func() {
		select {
		case v := <-ch1: fmt.Println(v) // ← blocked — shows [select], NOT [chan receive]
		case s := <-ch2: fmt.Println(s) // ← blocked
		}
	}()
	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
}

// io.go — [IO wait]: bloqueado en el poller del OS
// net.Pipe() muestra [select]; se necesita un socket TCP real.
func demoIOWait() {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	conn, _ := net.Dial("tcp", ln.Addr().String())
	go func() {
		buf := make([]byte, 1)
		conn.Read(buf) // ← blocked inside OS poller, shows as [IO wait]
	}()
	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
	conn.Close(); ln.Close()
}

// running.go — [running] / [runnable]: goroutine activo
func demoRunning() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop: return
			default:
				runtime.Gosched() // yields — shows [running] or [runnable]
			}
		}
	}()
	time.Sleep(80 * time.Millisecond)
	dumpGoroutines() // the goroutine calling this shows [running]
	close(stop); <-done
}

// mutex.go — [semacquire] / [sync.Mutex.Lock]: esperando adquirir un mutex
func demoSemacquire() {
	var mu sync.Mutex
	holderReady  := make(chan struct{})
	releaseHolder := make(chan struct{})
	go func() {              // holder: acquires and holds
		mu.Lock()
		close(holderReady)
		<-releaseHolder
		mu.Unlock()
	}()
	<-holderReady
	go func() {
		mu.Lock() // ← blocked here — shows as [sync.Mutex.Lock]
		mu.Unlock()
	}()
	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
	close(releaseHolder)
}

// mutex.go — deadlock AB: orden de locks inconsistente
var muA, muB sync.Mutex
func goroutine1(wg *sync.WaitGroup) {
	defer wg.Done()
	muA.Lock(); time.Sleep(50*time.Millisecond)
	muB.Lock() // ← blocks forever: goroutine2 holds muB
	defer muB.Unlock(); defer muA.Unlock()
}
func goroutine2(wg *sync.WaitGroup) {
	defer wg.Done()
	muB.Lock(); time.Sleep(50*time.Millisecond)
	muA.Lock() // ← blocks forever: goroutine1 holds muA
	defer muA.Unlock(); defer muB.Unlock()
}
// fatal error: all goroutines are asleep - deadlock!
```

```bash
cd deadlock && go run .   # muestra todos los estados y sale con exit 1
```

---

### [`stack-vs-heap/`](stack-vs-heap/README.md) — Stack vs Heap

El compilador decide dónde vive cada variable con *escape analysis*.
Retornar `&x` fuerza a `x` al heap; retornar el valor lo deja en el stack.

```go
// onStack: x is allocated on the stack.
// The compiler knows x doesn't outlive this call.
func onStack() int {
    x := 42
    return x // copied out; x is gone when the frame pops
}

// onHeap: returning &x forces the compiler to allocate x on the heap,
// because its address must remain valid after the function returns.
func onHeap() *int {
    x := 42
    return &x // x escapes to heap
}

// closureCapture: x is captured by the returned closure.
// Since the closure can outlive the function, x escapes to the heap.
func closureCapture() func() int {
    x := 0
    return func() int {
        x++
        return x
    }
}
```

```bash
cd stack-vs-heap
go run .
go build -gcflags="-m" .      # ver escape analysis
go test -bench=. -benchmem .  # benchmarks con allocs/op
```

---

### [`interfaces/`](interfaces/README.md) — Interfaces

Satisfacción implícita, composición de interfaces, polimorfismo, type assertion y type switch.

```go
type Shape interface {
    Area() float64
    Perimeter() float64
}

type Stringer interface {
    String() string
}

// Describer composes Shape and Stringer into a single interface.
type Describer interface {
    Shape
    Stringer
}

// Circle satisfies Shape and Stringer implicitly — no `implements` keyword.
type Circle struct{ Radius float64 }

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }
func (c Circle) String() string     { return fmt.Sprintf("Circle(r=%.2f)", c.Radius) }

// Type switch: branch on the runtime concrete type.
for _, shape := range shapes {
    switch v := shape.(type) {
    case Circle:
        fmt.Printf("Circle — radius: %.2f\n", v.Radius)
    case Rectangle:
        fmt.Printf("Rectangle — %dx%d\n", int(v.Width), int(v.Height))
    case Triangle:
        fmt.Printf("Triangle — sides: %.0f, %.0f, %.0f\n", v.A, v.B, v.C)
    }
}
```

```bash
cd interfaces && go run main.go
```

---

### [`goroutines/`](goroutines/README.md) — Goroutines

Ciclo de vida, leaks, panics y los patrones más importantes.

```go
// LEAK: goroutine queda bloqueada para siempre — nadie lee del canal.
func leak() {
    ch := make(chan int)
    go func() {
        ch <- 42 // blocks forever
    }()
}

// FIX: dar siempre una salida alternativa vía ctx.Done().
func fixed(ctx context.Context) {
    ch := make(chan int, 1)
    go func() {
        select {
        case ch <- 42:
        case <-ctx.Done(): // salida limpia
        }
    }()
}

// safeGo: wrapper que convierte un panic en error en lugar de derribar el proceso.
func safeGo(wg *sync.WaitGroup, errs chan<- error, fn func()) {
    go func() {
        defer wg.Done()
        defer func() {
            if r := recover(); r != nil {
                errs <- fmt.Errorf("panic: %v", r)
            }
        }()
        fn()
    }()
}
```

```bash
cd goroutines
go run .
go run -race .   # con race detector
```

---

### [`channels/`](channels/README.md) — Channels

Todos los patrones de comunicación entre goroutines.

```go
// Pipeline: etapas conectadas por canales.
// generate → filterPrimes → square → print
func generate(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            out <- n
        }
    }()
    return out
}

func square(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            out <- n * n
        }
    }()
    return out
}

// Fan-in: combinar N canales en uno.
func merge(cs ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    wg.Add(len(cs))
    for _, ch := range cs {
        go func(ch <-chan int) {
            defer wg.Done()
            for v := range ch { out <- v }
        }(ch)
    }
    go func() { wg.Wait(); close(out) }()
    return out
}

// Select con timeout: patrón canónico.
select {
case v := <-ch:
    fmt.Println("got:", v)
case <-time.After(100 * time.Millisecond):
    fmt.Println("timeout")
}
```

```bash
cd channels && go run .
```

---

### [`sync/`](sync/README.md) — sync & atomic

Todas las primitivas del paquete `sync` y `sync/atomic`.

```go
// Mutex: exclusión mutua sobre una sección crítica.
var (
    mu      sync.Mutex
    counter int
)
go func() {
    mu.Lock()
    defer mu.Unlock()
    counter++
}()

// Once: inicialización lazy thread-safe ejecutada exactamente una vez.
var (
    instance *DB
    once     sync.Once
)
func GetDB() *DB {
    once.Do(func() {
        instance = &DB{dsn: "postgres://..."}
    })
    return instance
}

// atomic.Int64: contador lock-free.
var counter atomic.Int64
counter.Add(1)
counter.CompareAndSwap(0, 1) // CAS: escribe solo si el valor actual == 0

// atomic.Value: config hot-reload sin mutex.
var cfg atomic.Value
cfg.Store(Config{MaxConns: 10})
c := cfg.Load().(Config)
```

```bash
cd sync
go run .
go run -race .
```

---

### [`context/`](context/README.md) — context.Context

Todos los constructores y patrones de uso de `context.Context`.

```go
// WithCancel: cancelación manual.
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    tick := time.NewTicker(40 * time.Millisecond)
    defer tick.Stop()
    for {
        select {
        case <-tick.C:
            fmt.Println("worker: tick")
        case <-ctx.Done():
            fmt.Println("worker: done, reason:", ctx.Err())
            return
        }
    }
}()

// WithCancelCause (Go 1.20): adjuntar el motivo específico de cancelación.
ctx, cancel := context.WithCancelCause(context.Background())
cancel(ErrRateLimit)
fmt.Println(ctx.Err())         // context.Canceled
fmt.Println(context.Cause(ctx)) // rate limit exceeded

// HTTP client: NewRequestWithContext es la API correcta.
ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
defer cancel()
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err := http.DefaultClient.Do(req)

// HTTP server: r.Context() se cancela cuando el cliente desconecta.
func handler(w http.ResponseWriter, r *http.Request) {
    result, err := db.QueryContext(r.Context(), "SELECT ...")
}
```

```bash
cd context && go run .
```

---

### [`race-conditions/`](race-conditions/README.md) — Race Conditions

Las cuatro familias de race conditions más comunes. El contador demuestra
lost updates visibles en el output (~85% de incrementos perdidos sin sincronización).

```go
// ── counter.go: DATA RACE ────────────────────────────────────────────────────
// counter++ compila a LOAD → ADD → STORE. Si dos goroutines intercalan,
// ambas leen el mismo valor y una escritura se pierde.
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
	fmt.Printf("expected: %d  got: %d  lost: %d\n", expected, counter, expected-counter)
}

// Fix — actor model: un solo goroutine es dueño del contador.
func demoCounterChannel() {
	inc := make(chan struct{}, 512)
	done := make(chan int)
	go func() { // actor: único propietario
		n := 0
		for range inc { n++ }
		done <- n
	}()
	// ... senders ... close(inc)
	fmt.Printf("expected: %d  got: %d  ✓\n", expected, <-done)
}

// ── checkact.go: CHECK-THEN-ACT (TOCTOU) ────────────────────────────────────
// La condición cambia entre el check y el act.
func (a *account) withdrawRacy(amount int) bool {
	if a.balance >= amount { // CHECK
		// ← otro goroutine puede correr aquí y también pasar el check
		a.balance -= amount  // ACT — balance puede quedar negativo
		return true
	}
	return false
}

// Fix: el lock debe abarcar tanto el check como el act.
func (a *safeAccount) withdraw(amount int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.balance >= amount {
		a.balance -= amount // nadie puede entrar entre estas dos líneas
		return true
	}
	return false
}

// ── publish.go: PUBLICATION HAZARD ──────────────────────────────────────────
// Anti-patrón: otro goroutine puede ver instance != nil con campos en cero.
func getConfigRacy() *config {
	if instance != nil { // unsynchronized read — DATA RACE
		return instance   // puede devolver struct parcialmente inicializado
	}
	// ...
}

// Fix: sync.Once garantiza happens-before para todos los campos.
func getConfigOnce() *config {
	cfgOnce.Do(func() {
		cfgInstance = &config{host: "localhost", port: 5432, timeout: 30 * time.Second}
	})
	return cfgInstance
}
```

```bash
cd race-conditions
go run .       # el contador muestra lost updates visibles (~85% perdidos)
go run -race . # el race detector señala cada acceso sin sincronización
```

---

### [`timers/`](timers/README.md) — Timers & Tickers

`time.Timer` (one-shot), `time.Ticker` (periódico), atajos `time.After`/`time.Tick`
y los patrones más útiles: debounce, rate limiter, retry con exponential backoff
y tarea periódica cancelable.

```go
// timer.go — stop-and-drain antes de Reset (Go < 1.23)
if !timer.Stop() {
    select {
    case <-timer.C:
    default:
    }
}
timer.Reset(newDuration)

// ticker.go — siempre Stop(); un ticker sin Stop() vive para siempre
ticker := time.NewTicker(40 * time.Millisecond)
defer ticker.Stop()

// patterns.go — debounce: actuar solo tras un período de silencio
timer := time.NewTimer(debounce)
for {
    select {
    case e := <-eventCh:
        if !timer.Stop() { select { case <-timer.C: default: } }
        timer.Reset(debounce) // reiniciar en cada evento
        _ = e
    case <-timer.C:
        fmt.Println("debounced action fired") // solo aquí actuamos
    }
}

// patterns.go — exponential backoff con jitter
delay := baseDelay
for attempt := 1; ; attempt++ {
    if err := tryOperation(); err == nil { return }
    jitter := time.Duration(rand.Int63n(int64(delay / 2)))
    wait := min(delay+jitter, maxDelay)
    <-time.NewTimer(wait).C
    delay *= 2
}

// patterns.go — tarea periódica cancelable
ticker := time.NewTicker(60 * time.Millisecond)
for {
    select {
    case <-ticker.C:
        doWork()
    case <-done: // o ctx.Done()
        ticker.Stop()
        return
    }
}
```

```bash
cd timers && go run .
```

---

### [`atomic/`](atomic/README.md) — Atomic Operations

`sync/atomic` completo: la API tipada (`atomic.Int64`, `atomic.Bool`, `atomic.Pointer[T]`),
Compare-And-Swap, `atomic.Value` para hot-reload de configuración y los patrones
lock-free más comunes.

```go
// primitives.go — Add devuelve el NUEVO valor; zero value listo para usar
var n atomic.Int64
fmt.Println(n.Add(10))  // 10
fmt.Println(n.Add(5))   // 15
fmt.Println(n.Add(-3))  // 12
n.Store(100)
old := n.Swap(42) // old=100, n=42

// primitives.go — CAS: escribe solo si el valor actual == old
swapped := val.CompareAndSwap(10, 20) // true  — 10 → 20
swapped  = val.CompareAndSwap(10, 99) // false — ya no es 10

// primitives.go — CAS loop: read-modify-write lock-free
for {
	old := counter.Load()
	if counter.CompareAndSwap(old, old+1) {
		return // ganamos la carrera
	}
	// otro goroutine modificó el valor — reintentar
}

// value.go — hot-reload de config: muchos lectores, un escritor
var cfgVal atomic.Value
cfgVal.Store(&Config{MaxConns: 10, Timeout: 5 * time.Second, Feature: "v1"})
cfg := cfgVal.Load().(*Config) // snapshot consistente, lock-free
cfgVal.Store(&Config{MaxConns: 50, Timeout: 10 * time.Second, Feature: "v2"})

// pointer.go — atomic.Pointer[T]: nil válido, sin type assertion ni boxing
var ptr atomic.Pointer[Node]
fmt.Println(ptr.Load()) // <nil> — nil es válido, a diferencia de atomic.Value
ptr.Store(&Node{Value: 1, Label: "initial"})
old2 := ptr.Swap(&Node{Value: 2, Label: "updated"}) // retorna el anterior
current := ptr.Load()
ptr.CompareAndSwap(current, &Node{Value: 3, Label: "cas"})

// patterns.go — copy-on-write: lectores lock-free, escritores clonan + CAS
appendItem := func(item string) {
	for {
		old := snap.Load()
		newItems := make([]string, len(old.Items)+1)
		copy(newItems, old.Items)
		newItems[len(old.Items)] = item
		if snap.CompareAndSwap(old, &SliceSnapshot{Items: newItems}) {
			return // ganamos la carrera
		}
		// otro writer ganó — reintentar con el snapshot más reciente
	}
}

// patterns.go — shutdown flag para workers creados dinámicamente
var shutdown atomic.Bool
go func() {
	for tick := range 20 {
		if shutdown.Load() { return }
		time.Sleep(10 * time.Millisecond)
		_ = tick
	}
}()
time.Sleep(35 * time.Millisecond)
shutdown.Store(true)
```

```bash
cd atomic && go run .
```

---

### [`errors/`](errors/README.md) — Error Handling

Manejo de errores idiomático en Go: centinelas, tipos custom, wrapping con `%w`,
`errors.Is/As`, métodos `Is()/As()` personalizados, `errors.Join` y el patrón
panic vs error.

```go
// sentinel.go — errores centinela y errors.Is recorriendo la cadena
var (
	ErrNotFound   = errors.New("not found")
	ErrPermission = errors.New("permission denied")
)

func findUser(id int) (string, error) {
	switch id {
	case 1:  return "alice", nil
	case 2:  return "", ErrPermission
	default: return "", fmt.Errorf("findUser %d: %w", id, ErrNotFound)
	}
}

// errors.Is recorre toda la cadena de Unwrap — no es una comparación ==
wrapped := fmt.Errorf("service: %w", fmt.Errorf("repo: %w", ErrNotFound))
fmt.Println(errors.Is(wrapped, ErrNotFound)) // true

// types.go — tipos custom y errors.As
type ValidationError struct{ Field, Message string }
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %q: %s", e.Field, e.Message)
}

var valErr *ValidationError
if errors.As(err, &valErr) {    // extrae el tipo concreto de la cadena
	fmt.Println(valErr.Field)   // acceso a los campos
}

// wrapping.go — %w vs %v: %w mantiene la cadena, %v la rompe
dbErr   := errors.New("connection refused")
svcErr  := fmt.Errorf("service.GetUser id=42: %w",
              fmt.Errorf("repo.FindUser: %w", dbErr))
fmt.Println(errors.Is(svcErr, dbErr))                    // true  — %w
opaque  := fmt.Errorf("something went wrong: %v", dbErr) // %v rompe la cadena
fmt.Println(errors.Is(opaque, dbErr))                    // false

// custom_is_as.go — Is() por igualdad semántica (ignora el mensaje)
type StatusError struct{ Code int; Message string }
func (e *StatusError) Is(target error) bool {
	var t *StatusError
	return errors.As(target, &t) && e.Code == t.Code
}
sentinel := &StatusError{Code: 404}
err1     := &StatusError{Code: 404, Message: "user not found"}
fmt.Println(errors.Is(err1, sentinel)) // true — mismo código, mensaje diferente

// custom_is_as.go — As() para buscar dentro de un tipo contenedor
type MultiError struct{ Errors []error }
func (m *MultiError) As(target any) bool {
	for _, err := range m.Errors {
		if errors.As(err, target) { return true }
	}
	return false
}

// join.go — errors.Join (Go 1.20+): combinar múltiples errores
joined := errors.Join(
	errors.New("database timeout"),
	errors.New("cache miss"),
)
fmt.Println(joined)                          // database timeout\ncache miss
fmt.Println(errors.Is(joined, ErrNotFound))  // false

var errs []error
for _, f := range fields {
	if f.value == "" {
		errs = append(errs, &ValidationError{Field: f.name, Message: "required"})
	}
}
if combined := errors.Join(errs...); combined != nil {
	fmt.Println(combined) // todos los errores de validación de una vez
}

// patterns.go — panic vs error
safeDiv := func(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("safeDiv: panic: %v", r)
		}
	}()
	return a / b, nil // panic si b == 0 — convertido en error en el boundary
}
```

```bash
cd errors && go run .
```

---

### [`generics/`](generics/README.md) — Generics (Go 1.18+)

Parámetros de tipo, constraints, funciones genéricas y estructuras de datos
parametrizadas. Incluye todas las limitaciones que suelen aparecer en entrevistas.

```go
// constraints.go — any, comparable, ~T, uniones, métodos

// any — sin restricción de operaciones
func Identity[T any](v T) T { return v }

// comparable — permite == y !=, necesario para claves de map
func Equal[T comparable](a, b T) bool { return a == b }

// ~T — acepta tipos definidos cuyo underlying type sea T
type Temperature interface{ ~float64 }
type Celsius float64    // satisface ~float64
type Fahrenheit float64 // satisface ~float64

func AbsDiff[T Temperature](a, b T) T { ... }

// Union constraint — aritmética sobre todos los tipos numéricos
type Number interface {
    ~int | ~int32 | ~int64 | ~float32 | ~float64
}
func Sum[T Number](s []T) T { ... }
```

```go
// functions.go — Map, Filter, Reduce, Contains, Keys/Values, Must
func Map[T, U any](s []T, f func(T) U) []U
func Filter[T any](s []T, f func(T) bool) []T
func Reduce[T, U any](s []T, init U, f func(U, T) U) U
func Contains[T comparable](s []T, v T) bool
func Keys[K comparable, V any](m map[K]V) []K
func Must[T any](v T, err error) T

squares := Map([]int{1,2,3}, func(n int) string { return fmt.Sprintf("%d²=%d", n, n*n) })
evens   := Filter([]int{1,2,3,4}, func(n int) bool { return n%2 == 0 })
sum     := Reduce([]int{1,2,3}, 0, func(acc, n int) int { return acc + n })
```

```go
// datastructs.go — estructuras de datos genéricas

type Stack[T any] struct{ items []T }
func (s *Stack[T]) Push(v T)
func (s *Stack[T]) Pop() (T, bool)   // zero value + false si vacío

type Queue[T any] struct{ items []T }
func (q *Queue[T]) Enqueue(v T)
func (q *Queue[T]) Dequeue() (T, bool)

type Set[T comparable] struct{ m map[T]struct{} }
func NewSet[T comparable](vals ...T) *Set[T]
func (s *Set[T]) Union(other *Set[T]) *Set[T]
func (s *Set[T]) Intersection(other *Set[T]) *Set[T]
```

```go
// patterns.go — inferencia, múltiples params, zero value, Result[T], limitaciones

// Inferencia — Go deduce el tipo del argumento
Double(21)         // Double[int]
Double(3.14)       // Double[float64]
Double[int64](7)   // explícito

// Múltiples parámetros de tipo
type Pair[A, B any] struct{ First A; Second B }
NewPair("edad", 30)  // Pair[string, int]

// Zero value de T
func First[T any](s []T) (T, bool) {
    if len(s) == 0 {
        var zero T   // 0, "", false, nil — según T
        return zero, false
    }
    return s[0], true
}

// Result[T] — valor o error encapsulado
type Result[T any] struct{ Value T; Err error }
func Ok[T any](v T) Result[T]
func Err[T any](e error) Result[T]

// Limitación: no se pueden definir métodos genéricos en tipos no genéricos
type MySlice []int
// func (s MySlice) Map[U any](f func(int) U) []U { }  // ← INVÁLIDO
// Solución: función top-level Map(mySlice, f)

// Limitación: type assertion requiere cast previo a any
func Describe[T any](v T) string {
    switch x := any(v).(type) {  // any(v) primero, luego .(type)
    case int:    return fmt.Sprintf("int(%d)", x)
    case string: return fmt.Sprintf("string(%q)", x)
    default:     return fmt.Sprintf("%T(%v)", x, x)
    }
}
```

```bash
cd generics && go run .
```

---

### [`slices/`](slices/README.md) — Slices internals & gotchas

Un slice es un header `{ptr, len, cap}` que apunta a un backing array. La mayoría
de los bugs y "trick questions" de entrevista vienen de no entender esto.

```go
// internals.go — header, backing array compartido, pass-by-value

// sizeof([]int) = 24 bytes (ptr + len + cap, 3×8 en 64-bit)

a := []int{1, 2, 3, 4, 5}
b := a[1:4]   // comparte el backing array de a
              // cap(b) = 4 (llega hasta el final de a, no sólo len(b)=3)

b[0] = 99     // escribe en a[1] — a también lo ve
fmt.Println(a) // [1 99 3 4 5]

// El header se copia al pasar a función — el array se comparte
func modifyElement(s []int, i, v int) { s[i] = v }   // ✓ caller lo ve
func appendInside(s []int)             { s = append(s, 99) } // ✗ caller NO lo ve
```

```go
// append.go — in-place vs reallocation, el gotcha más común

// len < cap → escribe en el array existente (sin allocación)
// len == cap → aloca nuevo array, copia todo

// EL GOTCHA: append a un subslice puede sobreescribir el original
orig := []int{1, 2, 3, 4, 5}
sub  := orig[1:3]           // cap(sub) = 4 — llega hasta orig[4]
sub   = append(sub, 99)     // cap permite → escribe en orig[3]!
fmt.Println(orig)           // [1 2 3 99 5]  ← sobreescrito silenciosamente

// Fix: slice de 3 índices — fuerza cap == len
safe := orig[1:3:3]         // cap = 3-1 = 2 = len → append debe alocar
safe  = append(safe, 99)    // nuevo backing array
fmt.Println(orig)           // [1 2 3 4 5]  ← intacto
```

```go
// operations.go — patrones de manipulación

// copy: min(len(dst), len(src)) elementos; maneja overlap correctamente
n := copy(dst, src)

// Delete O(n) — preserva orden
s = append(s[:i], s[i+1:]...)

// Delete O(1) — cambia orden (swap con último)
s[i] = s[len(s)-1]; s = s[:len(s)-1]

// Insert en posición i
s = append(s, 0); copy(s[i+1:], s[i:]); s[i] = v

// Filter in-place — zero allocation, reutiliza el backing array
result := s[:0]
for _, v := range s { if keep(v) { result = append(result, v) } }

// stdlib slices package (Go 1.21+)
slices.Sort(s); slices.Contains(s, v); slices.Delete(s, i, j); slices.Compact(sorted)
```

```go
// nil.go — nil vs empty slice

var s []int        // nil slice  : s == nil → true
s := []int{}       // empty slice: s == nil → false

// Para range, len, cap, append: idénticos
// Diferencia crítica: JSON
json.Marshal(Resp{Items: nil})      // → {"items":null}
json.Marshal(Resp{Items: []int{}})  // → {"items":[]}

// reflect.DeepEqual los distingue
reflect.DeepEqual([]int(nil), []int{}) // false

// == solo es válido contra nil
s == nil                // ✓
// s == []int{1,2,3}   // ✗ compile error
slices.Equal(a, b)      // ✓ comparación correcta (Go 1.21+)
```

```bash
cd slices && go run .
```

---

### [`worker-pool/`](worker-pool/README.md) — Worker Pool (producción)

Implementación lista para producción: shutdown graceful, propagación de context,
métricas atómicas y tests con race detector.

```go
// Job es la unidad de trabajo: función que recibe el context del pool.
type Job func(ctx context.Context) error

// Crear el pool.
pool := workerpool.New(workerpool.Config{
    Workers:         4,
    QueueSize:       20,
    ShutdownTimeout: 3 * time.Second,
})

// Submit encola un job; bloquea si la cola está llena.
// Respeta el context del llamador para cancelar la espera.
err := pool.Submit(ctx, func(ctx context.Context) error {
    select {
    case <-time.After(200 * time.Millisecond):
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
})

// Shutdown graceful: drena la cola, cancela si supera el timeout.
if err := pool.Shutdown(); err == workerpool.ErrShutdownTimeout {
    log.Println("forced shutdown")
}

// Métricas atómicas: seguras de leer desde cualquier goroutine.
m := pool.Metrics()
fmt.Printf("submitted=%d succeeded=%d failed=%d\n",
    m.Submitted, m.Succeeded, m.Failed)
```

```bash
cd worker-pool
go run .                         # demo con simulación de órdenes
go test -race ./workerpool/...   # tests con race detector
```

---

## Guía de lectura sugerida

```
1. goroutines/       → qué son y cómo se comportan
2. channels/         → cómo se comunican
3. sync/             → cómo comparten memoria con seguridad
4. context/          → cómo se cancelan y coordinan
5. race-conditions/  → qué sale mal sin sincronización y cómo detectarlo
6. deadlock/         → qué sale mal con sincronización incorrecta
7. stack-vs-heap/    → dónde vive la memoria
8. timers/           → temporización: timers, tickers y patrones
9. atomic/           → operaciones atómicas y patrones lock-free
10. errors/          → manejo de errores idiomático
11. generics/        → parámetros de tipo, constraints y estructuras genéricas
12. slices/          → internals, append, gotchas y operaciones comunes
13. worker-pool/     → todo junto en un componente de producción
```
