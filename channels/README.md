# Channels en Go — Guía completa con ejemplos

## ¿Qué es un channel?

Un channel es un conducto tipado a través del cual las goroutines se comunican y
sincronizan. En Go se prefiere este modelo al de memoria compartida:

> *"Don't communicate by sharing memory; share memory by communicating."*

```go
ch := make(chan int)      // unbuffered: capacidad 0
ch := make(chan int, 10)  // buffered: capacidad 10
```

---

## Archivos del módulo

```
channels/
├── go.mod
├── main.go          — ejecuta todos los demos en orden
├── basic.go         — unbuffered, buffered, directional, close, range
├── select.go        — select, default, nil channel, timeout
├── pipeline.go      — pipeline, fan-out, fan-in (merge)
├── workerpool.go    — worker pool con jobs y results channels
├── semaphore.go     — semáforo de conteo con canal bufferizado
└── done.go          — done channel, or-done wrapper
```

---

## Cómo correrlo

```bash
go run .
```

---

## Ejemplos

### Unbuffered channel (`basic.go`)

Capacidad cero. El sender bloquea hasta que un receiver esté listo y viceversa.
Garantiza que ambos goroutines se encuentran en el punto de comunicación.

```go
ch := make(chan int) // capacity = 0

go func() {
    fmt.Println("sender: sending 42")
    ch <- 42 // bloquea hasta que alguien reciba
    fmt.Println("sender: send complete")
}()

v := <-ch // bloquea hasta que alguien envíe
fmt.Println("receiver: got", v)
```

---

### Buffered channel (`basic.go`)

El sender solo bloquea cuando el buffer está lleno; el receiver solo bloquea cuando
está vacío. Permite absorber ráfagas sin sincronización estricta.

```go
ch := make(chan string, 3) // capacity = 3

ch <- "a"  // no bloquea
ch <- "b"  // no bloquea
ch <- "c"  // no bloquea
// ch <- "d"  // BLOQUEARÍA: buffer lleno

fmt.Println("len:", len(ch), "cap:", cap(ch))
fmt.Println(<-ch, <-ch, <-ch) // a b c
```

---

### Directional channels (`basic.go`)

Tipos `chan<- T` (solo envío) y `<-chan T` (solo recepción). El compilador rechaza
usos incorrectos. Úsalos en firmas de funciones para documentar la intención.

```go
func produce(out chan<- int) { // solo puede enviar
    out <- 99
}

func consume(in <-chan int) { // solo puede recibir
    fmt.Println("got", <-in)
}

ch := make(chan int, 1)
produce(ch) // conversión implícita bidireccional → send-only
consume(ch) // conversión implícita bidireccional → receive-only
```

---

### Close + range (`basic.go`)

`close` señala que no vendrán más valores. `range` sobre un canal itera hasta que
esté cerrado y vacío. El comma-ok idiom distingue el zero value de un canal cerrado.

```go
ch := make(chan int, 5)

go func() {
    for i := 1; i <= 5; i++ {
        ch <- i
    }
    close(ch) // solo el sender cierra; enviar a un canal cerrado produce panic
}()

for v := range ch { // sale automáticamente cuando ch está cerrado y vacío
    fmt.Print(v, " ") // 1 2 3 4 5
}

v, ok := <-ch // ok=false → canal cerrado y vacío; v=0 (zero value)
```

---

### Select (`select.go`)

Multiplexar sobre múltiples operaciones de canal. Si más de un case está listo,
Go elige uno al azar (equitativo, no determinístico).

```go
ch1 := make(chan string, 1)
ch2 := make(chan string, 1)
ch1 <- "one"
ch2 <- "two"

select {
case v := <-ch1:
    fmt.Println("ch1:", v)
case v := <-ch2:
    fmt.Println("ch2:", v)
}
```

---

### Select: default — no bloquear (`select.go`)

El case `default` se ejecuta inmediatamente si ningún otro case está listo.
Permite try-send / try-receive sin goroutines.

```go
ch := make(chan int, 1)

// Try-send: envía solo si hay espacio ahora mismo.
select {
case ch <- 10:
    fmt.Println("sent 10")
default:
    fmt.Println("channel full, skipped")
}

// Try-receive: recibe solo si hay un valor ahora mismo.
select {
case v := <-ch:
    fmt.Println("received:", v)
default:
    fmt.Println("nothing to receive")
}
```

---

### Select: nil channel (`select.go`)

Un canal nil nunca está listo — su case es ignorado permanentemente por el scheduler.
Asignar `nil` a una variable de canal dentro de un loop desactiva ese case dinámicamente.

```go
var disabled chan string // nil
active := make(chan string, 1)
active <- "hello"

select {
case v := <-disabled: // nunca se selecciona
    fmt.Println("disabled:", v)
case v := <-active:
    fmt.Println("active:", v) // siempre este
}

// Patrón práctico: deshabilitar un case tras procesarlo.
for i := 0; i < 2; i++ {
    select {
    case v, ok := <-a:
        fmt.Println("a:", v)
        a = nil // deshabilitar: este case ya no competirá
    case v, ok := <-b:
        fmt.Println("b:", v)
        b = nil
    }
}
```

---

### Select: timeout (`select.go`)

El patrón canónico para acotar el tiempo de espera de un canal: race entre el
resultado y `time.After`.

```go
slow := make(chan string)

go func() {
    time.Sleep(200 * time.Millisecond)
    slow <- "result"
}()

select {
case v := <-slow:
    fmt.Println("got:", v)
case <-time.After(100 * time.Millisecond):
    fmt.Println("timeout")
}
```

---

### Pipeline (`pipeline.go`)

Serie de etapas conectadas por canales. Cada etapa es un goroutine que lee de su
entrada, transforma, y escribe en su salida. El pipeline es lazy: cada etapa solo
corre cuando la siguiente pide. Cada etapa cierra su canal de salida con `defer close(out)`.

```
generate(2..9) → filterPrimes → square → print
// output: 4 9 25 49
```

```go
// Cada etapa sigue la misma forma:
// recibe <-chan, devuelve <-chan, cierra con defer.
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

// Composición:
naturals := generate(2, 3, 4, 5, 6, 7, 8, 9)
primes   := filterPrimes(naturals)
squared  := square(primes)

for v := range squared {
    fmt.Printf("%d ", v) // 4 9 25 49
}
```

---

### Fan-out (`pipeline.go`)

Distribuir un canal de entrada entre N workers para procesar en paralelo.
Todos leen del mismo canal; el scheduler reparte el trabajo.

```go
jobs := generate(1, 2, 3, 4, 5, 6, 7, 8)

const numWorkers = 3
results := make([]<-chan int, numWorkers)
for i := 0; i < numWorkers; i++ {
    results[i] = squareWorker(i, jobs) // todos leen del mismo canal jobs
}

for v := range merge(results...) {
    fmt.Printf("%d ", v)
}
```

---

### Fan-in / merge (`pipeline.go`)

Combinar N canales en uno. Un goroutine por canal de entrada reenvía valores al
canal de salida compartido. Cierra el output cuando todos los inputs se agotan.

```go
func merge(cs ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup

    wg.Add(len(cs))
    for _, ch := range cs {
        go func(ch <-chan int) {
            defer wg.Done()
            for v := range ch {
                out <- v
            }
        }(ch)
    }

    go func() {
        wg.Wait()
        close(out)
    }()
    return out
}

// Uso:
for v := range merge(a, b, c) {
    fmt.Printf("%d ", v)
}
```

---

### Worker pool (`workerpool.go`)

N workers fijos consumen de un canal `jobs` y publican en `results`. Acota el
paralelismo (memoria, CPU) y ofrece backpressure: el productor bloquea si el pool
está saturado.

```
jobs ──┬── worker1 ──┐
       ├── worker2 ──┼── results
       └── worker3 ──┘
```

```go
jobs    := make(chan job,    numJobs)
results := make(chan result, numJobs)

var wg sync.WaitGroup
for id := 1; id <= numWorkers; id++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        for j := range jobs { // sale cuando jobs se cierre y vacíe
            results <- result{jobID: j.id, output: j.value * j.value}
        }
    }(id)
}

for i := 1; i <= numJobs; i++ {
    jobs <- job{id: i, value: i * 10}
}
close(jobs) // workers salen del range tras drenar

go func() { wg.Wait(); close(results) }()

for r := range results {
    fmt.Printf("job %d → %d\n", r.jobID, r.output)
}
```

---

### Semaphore (`semaphore.go`)

Un canal bufferizado de capacidad N actúa como semáforo de conteo: limita cuántas
goroutines corren simultáneamente sin necesidad de un pool explícito.

```go
sem := make(chan struct{}, 3) // máximo 3 goroutines a la vez

for i := 1; i <= 9; i++ {
    go func(id int) {
        sem <- struct{}{}        // acquire: bloquea si ya hay 3 corriendo
        defer func() { <-sem }() // release: siempre en defer

        fmt.Printf("task%d running\n", id)
        time.Sleep(30 * time.Millisecond)
    }(i)
}
```

Diferencia con worker pool: las goroutines se crean bajo demanda; el semáforo
solo controla cuántas corren a la vez.

---

### Done channel (`done.go`)

Canal usado solo como señal de broadcast sin datos. Cerrarlo despierta a **todos**
los goroutines bloqueados en él simultáneamente — a diferencia de enviar un valor,
que despierta solo a uno. Es la base de `context.Context`.

```go
done := make(chan struct{})

for i := 1; i <= 3; i++ {
    go func(id int) {
        <-done // bloquea hasta que done se cierre
        fmt.Printf("goroutine%d: done!\n", id)
    }(i)
}

time.Sleep(60 * time.Millisecond)
close(done) // despierta los 3 goroutines a la vez
```

---

### Or-done wrapper (`done.go`)

Cuando el producer nunca cierra su canal, un `range` corriente bloquearía para
siempre. `orDone` envuelve el canal para que el consumidor pueda parar limpiamente
sin goroutine leaks.

```go
func orDone(done <-chan struct{}, in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for {
            select {
            case <-done:
                return
            case v, ok := <-in:
                if !ok {
                    return
                }
                select {
                case out <- v:
                case <-done:
                    return
                }
            }
        }
    }()
    return out
}

// Uso: iterar de forma segura aunque el producer no cierre su canal.
for v := range orDone(done, values) {
    if v >= 4 {
        close(done) // señal al producer
        break
    }
}
```

---

## Tabla de operaciones y comportamiento

| Operación | Canal nil | Canal abierto | Canal cerrado |
|---|---|---|---|
| `ch <- v` | bloquea para siempre | bloquea si lleno | **panic** |
| `v := <-ch` | bloquea para siempre | bloquea si vacío | zero value (inmediato) |
| `close(ch)` | **panic** | ok | **panic** |
| `len(ch)` | 0 | nº de elementos | nº restantes |
| `cap(ch)` | 0 | capacidad | capacidad |

---

## Reglas prácticas

| Regla | Motivo |
|---|---|
| Solo el sender cierra el canal | El receiver no sabe cuándo el sender terminó |
| No cerrar si hay múltiples senders | Usar `sync.WaitGroup` + goroutine que cierra |
| Preferir canales unidireccionales en firmas | Documenta y restringe el uso correcto |
| No usar canales para transferir ownership de mutexes | Complica el razonamiento; usa `sync` |
| Usar `done` o `context` para cancelar goroutines | Evita goroutine leaks |
