# sync — Guía completa con ejemplos

## ¿Qué provee el paquete `sync`?

Primitivas de sincronización para coordinar goroutines que comparten memoria.
Complementa a los channels: úsalos cuando la comunicación no es el objetivo,
sino proteger acceso concurrente a datos compartidos.

---

## Archivos del módulo

```
sync/
├── go.mod
├── main.go       — ejecuta todos los demos en orden
├── mutex.go      — Mutex, RWMutex
├── waitgroup.go  — WaitGroup
├── once.go       — Once (lazy init, singleton)
├── cond.go       — Cond (Signal y Broadcast)
├── pool.go       — Pool
├── syncmap.go    — sync.Map
└── atomic.go     — sync/atomic (contadores, CAS, Value)
```

---

## Cómo correrlo

```bash
go run .
```

Detectar races:

```bash
go run -race .
```

---

## Ejemplos

### `sync.Mutex` (`mutex.go`)

Protege una sección crítica: solo un goroutine puede estar dentro a la vez.
`defer mu.Unlock()` garantiza la liberación en cualquier camino de retorno.

```go
var (
    mu      sync.Mutex
    counter int
)

go func() {
    mu.Lock()
    defer mu.Unlock() // se ejecuta aunque haya un panic o return temprano
    counter++
}()
```

---

### `sync.RWMutex` (`mutex.go`)

Permite múltiples lectores simultáneos o un escritor exclusivo.
Usa `RWMutex` cuando las lecturas son frecuentes y las escrituras son raras.

```go
var mu sync.RWMutex

// Múltiples goroutines pueden tener RLock al mismo tiempo.
mu.RLock()
defer mu.RUnlock()
_ = cache["key"]

// El escritor espera a que todos los RLock se liberen.
mu.Lock()
defer mu.Unlock()
cache["key"] = "new value"
```

| Operación | Concurrencia permitida |
|---|---|
| `RLock` / `RUnlock` | N lectores simultáneos |
| `Lock` / `Unlock` | 1 escritor exclusivo |

---

### `sync.WaitGroup` (`waitgroup.go`)

Esperar a que un conjunto de goroutines termine.

```go
var wg sync.WaitGroup

for i := 1; i <= 5; i++ {
    wg.Add(1)           // incrementar ANTES de lanzar el goroutine
    go func(id int) {
        defer wg.Done() // decrementar al salir, siempre via defer
        doWork(id)
    }(i)
}

wg.Wait() // bloquea hasta que el contador llegue a cero
```

Reglas:
- `Add(n)` antes de lanzar el goroutine, nunca dentro de él.
- `Done()` siempre en un `defer` para cubrir panics y returns tempranos.
- No reusar un WaitGroup hasta que `Wait()` haya retornado.

---

### `sync.Once` (`once.go`)

Ejecuta una función exactamente una vez, sin importar cuántos goroutines llamen
a `Do` concurrentemente. Ideal para inicialización lazy y singletons.

```go
var once sync.Once

// 10 goroutines llaman Do al mismo tiempo.
// La función init solo se ejecuta en la primera llamada.
for i := 0; i < 10; i++ {
    go func() {
        once.Do(func() {
            fmt.Println("init — solo una vez")
        })
    }()
}
```

Patrón singleton:

```go
var (
    instance *DB
    once     sync.Once
)

func GetDB() *DB {
    once.Do(func() {
        instance = &DB{dsn: "postgres://..."}
    })
    return instance // siempre el mismo puntero
}
```

---

### `sync.Cond` — Signal (`cond.go`)

Variable de condición: un goroutine espera hasta que otro le notifique que el
estado cambió. `Signal` despierta a **uno** de los goroutines en espera.

```go
cond := sync.NewCond(&mu)

// Consumer — espera hasta que haya trabajo.
mu.Lock()
for len(queue) == 0 {   // for, no if: re-verificar tras el wakeup
    cond.Wait()         // libera mu atomicamente; lo re-adquiere al despertar
}
item := queue[0]
mu.Unlock()

// Producer — añade trabajo y despierta a un consumer.
mu.Lock()
queue = append(queue, item)
cond.Signal() // despierta a exactamente un goroutine en espera
mu.Unlock()
```

---

### `sync.Cond` — Broadcast (`cond.go`)

`Broadcast` despierta a **todos** los goroutines en espera. Útil cuando un cambio
de estado es relevante para múltiples waiters (e.g. "servidor listo", "config cargada").

```go
// N workers esperan la señal de inicio.
for i := 0; i < 4; i++ {
    go func() {
        mu.Lock()
        for !ready {
            cond.Wait()
        }
        mu.Unlock()
        startWork()
    }()
}

// Cuando todo está listo, despertar a todos a la vez.
mu.Lock()
ready = true
cond.Broadcast()
mu.Unlock()
```

| Método | Goroutines despertados |
|---|---|
| `Signal()` | 1 (el scheduler elige cuál) |
| `Broadcast()` | Todos los que esperan en este Cond |

---

### `sync.Pool` (`pool.go`)

Cache de objetos temporales reutilizables. Reduce la presión sobre el GC al evitar
allocaciones repetidas de objetos de corta vida (buffers, encoders, slices).

```go
pool := &sync.Pool{
    New: func() any {
        return new(bytes.Buffer) // llamado solo cuando el pool está vacío
    },
}

// Obtener del pool (o crear si está vacío).
buf := pool.Get().(*bytes.Buffer)

buf.WriteString("hello")
fmt.Println(buf.String())

// Devolver al pool — resetear antes para que el próximo uso sea limpio.
buf.Reset()
pool.Put(buf)
```

Notas:
- El GC puede vaciar el pool en cualquier momento: no guardar estado persistente.
- Siempre hacer `Reset()` antes de `Put()` para no contaminar al próximo usuario.
- Pool es seguro para uso concurrente.

---

### `sync.Map` (`syncmap.go`)

Mapa concurrent-safe sin locking externo. Optimizado para dos patrones:
1. Escritura única + lecturas frecuentes (caches, registros).
2. Goroutines que escriben en conjuntos de claves disjuntos.

```go
var m sync.Map

// Store / Load
m.Store("key", "value")
if v, ok := m.Load("key"); ok {
    fmt.Println(v)
}

// LoadOrStore: get-or-set atómico
actual, loaded := m.LoadOrStore("key", "default")
// loaded=true  → "key" ya existía; actual = valor previo
// loaded=false → "key" fue insertada; actual = "default"

// LoadAndDelete: get + delete atómico
if v, ok := m.LoadAndDelete("key"); ok {
    fmt.Println("deleted:", v)
}

// Delete
m.Delete("key")

// Range: iterar (orden no garantizado); retornar false para parar.
m.Range(func(k, v any) bool {
    fmt.Printf("%v → %v\n", k, v)
    return true
})
```

---

### `sync/atomic` — contadores y CAS (`atomic.go`)

Operaciones atómicas sobre tipos primitivos sin mutex. Más barato que un mutex
para contadores y flags simples porque mapea a instrucciones únicas de CPU.

```go
// Typed API (Go 1.19+) — preferida por ser type-safe.
var counter atomic.Int64

counter.Add(1)               // incremento atómico
counter.Store(0)             // escritura incondicional
n := counter.Load()          // lectura atómica
prev := counter.Swap(99)     // escribe y devuelve el valor anterior

// atomic.Bool
var running atomic.Bool
running.Store(true)
if running.Load() { ... }
```

**Compare-And-Swap (CAS):** escribe el nuevo valor solo si el actual coincide con
el esperado. Primitiva fundamental de estructuras lock-free.

```go
var state atomic.Int32 // 0=idle, 1=running, 2=done

// idle → running: éxito (state era 0)
swapped := state.CompareAndSwap(0, 1) // true

// idle → running otra vez: falla (state es 1, no 0)
swapped = state.CompareAndSwap(0, 1)  // false

// running → done: éxito
swapped = state.CompareAndSwap(1, 2)  // true
```

---

### `sync/atomic.Value` (`atomic.go`)

Almacena y carga un valor de tipo arbitrario de forma atómica. Ideal para
configuración recargable en caliente, routing tables y snapshots inmutables.

```go
var cfg atomic.Value

// Store: todos los valores almacenados deben ser del mismo tipo concreto.
cfg.Store(Config{MaxConns: 10, Timeout: 30})

// Load: cualquier goroutine lee sin bloquear.
c := cfg.Load().(Config)

// Swap: escribe y devuelve el valor anterior en una operación atómica.
prev := cfg.Swap(Config{MaxConns: 50}).(Config)

// CompareAndSwap: reemplaza solo si el valor actual es igual al esperado.
current := cfg.Load().(Config)
cfg.CompareAndSwap(current, Config{MaxConns: 200})
```

Reglas:
- Tratar el valor almacenado como **inmutable**: nunca modificar después de `Store`.
- Todos los `Store` deben usar el mismo tipo concreto (no interfaces distintas).

---

## Cuándo usar cada primitiva

| Primitiva | Usa cuando… |
|---|---|
| `Mutex` | Necesitas exclusión mutua sobre cualquier sección crítica |
| `RWMutex` | Lecturas frecuentes, escrituras raras |
| `WaitGroup` | Esperar a que N goroutines terminen |
| `Once` | Inicialización lazy thread-safe, singleton |
| `Cond` | Un goroutine debe esperar a que otro cambie el estado |
| `Pool` | Objetos temporales costosos que se crean y descartan en loop |
| `sync.Map` | Cache o registro con escritura-una-vez y lectura-muchas |
| `atomic` | Contadores, flags y estados simples sin overhead de mutex |
| `atomic.Value` | Configuración o snapshot que se reemplaza atómicamente |
