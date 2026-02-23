# Goroutines — Guía completa con ejemplos

## ¿Qué es una goroutine?

Una goroutine es una función que se ejecuta de forma concurrente con otras goroutines
dentro del mismo proceso. Son el bloque fundamental de concurrencia en Go.

```go
go f()         // lanza f en una nueva goroutine
go func() {}() // función anónima
```

A diferencia de los threads del SO, las goroutines son:

| Propiedad | Thread (OS) | Goroutine |
|---|---|---|
| Stack inicial | 1–8 MB fijo | ~2 KB, crece dinámicamente |
| Gestión | Kernel | Runtime de Go (M:N scheduler) |
| Costo de creación | Alto (~microsegundos) | Bajo (~nanosegundos) |
| Cambio de contexto | Costoso (syscall) | Barato (en userspace) |
| Cantidad típica | Miles | Millones |

---

## Archivos del módulo

```
goroutines/
├── go.mod
├── main.go       — ejecuta todos los demos en orden
├── basics.go     — formas de lanzar goroutines
├── closure.go    — bug de captura de closure y sus fixes
├── lifecycle.go  — GOMAXPROCS, NumGoroutine, Gosched, stack growth
├── leak.go       — goroutine leaks y cómo prevenirlos
├── panic.go      — panic/recover en goroutines y patrón safeGo
└── patterns.go   — fire-and-forget, first-wins, bounded concurrency
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

### Formas de lanzar goroutines (`basics.go`)

`go` no bloquea: main continúa inmediatamente después de cada `go`. Sin `wg.Wait()`
el programa puede terminar antes de que las goroutines corran.

```go
func demoBasics() {
	var wg sync.WaitGroup

	// 1. Named function — clearest form; gives the goroutine a readable stack trace.
	wg.Add(1)
	go greet("Alice", &wg)

	// 2. Anonymous function — inline, useful for short closures.
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("  anonymous goroutine running")
	}()

	// 3. Method on a value — goroutines can call methods too.
	wg.Add(1)
	w := worker{id: 7}
	go w.run(&wg)

	// Main continues immediately after each `go` statement.
	// Without wg.Wait() the program might exit before the goroutines run.
	fmt.Println("  main: all goroutines launched")
	wg.Wait()
	fmt.Println("  main: all goroutines done")
}

func greet(name string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("  hello from goroutine, %s\n", name)
}

type worker struct{ id int }

func (w worker) run(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("  worker%d running\n", w.id)
}
```

---

### Bug de captura de closure (`closure.go`)

Una closure captura la *variable*, no el *valor*. Si el loop modifica `i` antes de
que las goroutines corran, todas leen el valor final.

```go
func demoClosure() {
	var wg sync.WaitGroup

	// ── BUG (illustrative — run with go1.21 or earlier to observe) ──────────
	// Each goroutine captures the address of i. When they execute, i is
	// already 5. All goroutines may print "5".
	//
	// buggy := make(chan int, 5)
	// for i := 0; i < 5; i++ {
	//     go func() { buggy <- i }()  // ← captures &i, not the value
	// }

	// ── Fix 1: pass i as an argument (works on all Go versions) ─────────────
	// The value of i is copied into the parameter n at call time.
	fmt.Println("  fix 1 — pass as argument:")
	results := make(chan int, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) { // n is a fresh copy of i for each iteration
			defer wg.Done()
			results <- n
		}(i) // i evaluated here, at launch time
	}
	wg.Wait()
	close(results)
	for v := range results {
		fmt.Printf("  %d", v)
	}
	fmt.Println()

	// ── Fix 2: shadow the variable (Go < 1.22 idiom) ────────────────────────
	// `i := i` creates a new variable in the inner scope that holds the
	// current value. The closure captures the inner i, not the loop variable.
	fmt.Println("  fix 2 — shadow the variable:")
	results2 := make(chan int, 5)
	for i := 0; i < 5; i++ {
		i := i // new variable per iteration
		wg.Add(1)
		go func() {
			defer wg.Done()
			results2 <- i
		}()
	}
	wg.Wait()
	close(results2)
	for v := range results2 {
		fmt.Printf("  %d", v)
	}
	fmt.Println()

	// ── Subtlety: closures that legitimately share a variable ────────────────
	// Not every closure capture is a bug. If you *want* goroutines to observe
	// updates to a shared variable, capturing by reference is correct —
	// just protect the access with a mutex or atomic.
	var counter int
	var mu sync.Mutex
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++ // intentional shared write
			mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Println("  shared counter:", counter)
}
```

> Go 1.22 cambió la semántica de las variables de loop: cada iteración tiene su
> propia variable, haciendo el fix 2 innecesario en toolchains modernos.
> Fix 1 es preferible por claridad y compatibilidad universal.

---

### Lifecycle & runtime (`lifecycle.go`)

#### GOMAXPROCS — hilos de SO

Go usa un scheduler M:N: M goroutines se multiplexan sobre N threads de SO (P).
`GOMAXPROCS` controla cuántos threads ejecutan código Go simultáneamente.
El default desde Go 1.5 es `runtime.NumCPU()`.

```go
func demoGOMAXPROCS() {
	prev := runtime.GOMAXPROCS(0) // 0 = query without changing
	fmt.Printf("  GOMAXPROCS: %d  (NumCPU: %d)\n", prev, runtime.NumCPU())

	// Temporarily limit to 1 OS thread.
	runtime.GOMAXPROCS(1)
	fmt.Printf("  set to 1, now: %d\n", runtime.GOMAXPROCS(0))
	runtime.GOMAXPROCS(prev) // restore
}
```

#### NumGoroutine — detectar leaks

```go
func demoNumGoroutine() {
	before := runtime.NumGoroutine()
	fmt.Printf("  goroutines before: %d\n", before)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
		}()
	}

	fmt.Printf("  goroutines during: %d\n", runtime.NumGoroutine())
	wg.Wait()
	fmt.Printf("  goroutines after:  %d\n", runtime.NumGoroutine())
}
```

Si el conteo sigue creciendo después de `Wait`, hay goroutines que no terminaron.

#### Gosched — ceder el procesador

```go
func demoGosched() {
	var wg sync.WaitGroup
	printed := make(chan string, 10)

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			printed <- fmt.Sprintf("goroutine%d: before yield", id)
			runtime.Gosched() // yield: let others run
			printed <- fmt.Sprintf("goroutine%d: after yield", id)
		}(i)
	}

	wg.Wait()
	close(printed)
	for msg := range printed {
		fmt.Println(" ", msg)
	}
}
```

Rara vez necesario: el scheduler es preemptivo desde Go 1.14.

#### Stack dinámico

```go
func demoStackGrowth() {
	done := make(chan int)
	go func() {
		// Each recursive call adds a frame; Go grows the stack transparently.
		done <- deepRecurse(10000)
	}()
	fmt.Printf("  deep recursion result: %d\n", <-done)
}

func deepRecurse(n int) int {
	if n == 0 {
		return 0
	}
	return 1 + deepRecurse(n-1) // triggers stack growth
}
```

Las goroutines comienzan con ~2 KB de stack. Go lo crece automáticamente, por eso
es viable correr millones de goroutines mientras los threads de SO requieren 1–8 MB fijo.

---

### Goroutine leaks (`leak.go`)

Un leak ocurre cuando una goroutine queda bloqueada para siempre. El conteo de
`runtime.NumGoroutine()` crece indefinidamente — señal de alerta en producción.

#### Leak por send bloqueado

```go
func demoLeakSend() {
	leak := func() {
		ch := make(chan int) // unbuffered; nobody will read after this returns
		go func() {
			fmt.Println("  leaking goroutine: trying to send...")
			ch <- 42 // blocks forever — LEAK
		}()
		// function returns; ch goes out of scope but the goroutine is stuck
	}

	leak()
	time.Sleep(20 * time.Millisecond)
	// runtime.NumGoroutine() creció en +1 y no volverá a bajar
}
```

#### Leak por receive bloqueado

```go
func demoLeakReceive() {
	leak := func() {
		ch := make(chan int)
		go func() {
			fmt.Println("  leaking goroutine: waiting to receive...")
			<-ch // blocks forever — nobody will send — LEAK
		}()
		// Forgot to close(ch) or send a value; goroutine is stuck.
	}

	leak()
}
```

#### Fix: context + select

Siempre dar a la goroutine una salida alternativa vía `ctx.Done()`.

```go
func demoLeakFixed() {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Fixed send: goroutine exits via ctx.Done() if nobody reads.
	fixedSend := func(ctx context.Context) {
		ch := make(chan int, 1) // buffered: send won't block if buffer has space
		go func() {
			select {
			case ch <- 42:
				fmt.Println("  fixed send: value sent")
			case <-ctx.Done():
				fmt.Println("  fixed send: context cancelled, goroutine exiting")
			}
		}()
	}

	// Fixed receive: goroutine exits via ctx.Done() if nobody sends.
	fixedReceive := func(ctx context.Context) {
		ch := make(chan int)
		go func() {
			select {
			case v := <-ch:
				fmt.Println("  fixed receive: got", v)
			case <-ctx.Done():
				fmt.Println("  fixed receive: context cancelled, goroutine exiting")
			}
		}()
	}

	fixedSend(ctx)
	fixedReceive(ctx)

	<-ctx.Done()
	// runtime.NumGoroutine() delta = 0: ambas goroutines salieron limpiamente
}
```

---

### Panic & recover (`panic.go`)

Un panic en una goroutine **derriba todo el proceso**, no solo esa goroutine.
`recover()` solo funciona dentro de la misma goroutine donde ocurrió el panic,
llamado directamente desde una función diferida.

#### recover en la misma goroutine

```go
func demoPanicRecoverSameGoroutine() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("  recovered in same goroutine:", r)
		}
	}()

	fmt.Println("  about to panic...")
	panic("something went wrong") // caught by the deferred recover above
}
```

#### Patrón `safeGo`

Wrapper que lanza una goroutine con recover incorporado y reenvía el panic
como `error` a un canal. Útil en servidores donde un panic no debe derribar todo.

```go
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

// Uso:
errs := make(chan error, 3)
for i := 1; i <= 3; i++ {
	wg.Add(1)
	safeGo(&wg, errs, func(id int) func() {
		return func() {
			if id == 2 {
				panic(fmt.Sprintf("goroutine%d exploded", id))
			}
			fmt.Printf("  goroutine%d finished ok\n", id)
		}
	}(i))
}
wg.Wait()
close(errs)
for err := range errs {
	fmt.Println("  caught:", err) // panic: goroutine2 exploded
}
```

#### Panic en goroutine separada

```go
func demoPanicInGoroutine() {
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				// Must recover here — recovery in main() would be too late.
				fmt.Println("  goroutine recovered its own panic:", r)
			}
		}()

		time.Sleep(10 * time.Millisecond)
		panic("goroutine-level panic")
	}()

	<-done
	fmt.Println("  main: goroutine finished (panic was handled inside it)")
}
```

---

### Fire and forget (`patterns.go`)

Tarea en background sin resultado. Siempre proveer una salida vía context o done
channel para evitar leaks.

```go
func demoFireAndForget() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Background heartbeat — fires and is forgotten by main.
	go func(ctx context.Context) {
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				fmt.Println("  heartbeat tick")
			case <-ctx.Done():
				fmt.Println("  heartbeat stopped")
				return
			}
		}
	}(ctx)

	time.Sleep(70 * time.Millisecond)
	cancel() // stop the background goroutine
	time.Sleep(10 * time.Millisecond)
}
```

---

### First response wins (`patterns.go`)

Lanzar N goroutines con la misma tarea; usar el primer resultado. Las demás se
cancelan vía context para que no queden leakeadas.

```go
func demoFirstWins() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type response struct {
		worker int
		value  string
	}

	ch := make(chan response, 3) // buffered so slow goroutines can still send and exit

	latencies := []time.Duration{60, 20, 40} // worker2 wins
	for i, lat := range latencies {
		go func(id int, latency time.Duration) {
			select {
			case <-time.After(latency):
				ch <- response{worker: id, value: fmt.Sprintf("result-from-worker%d", id)}
			case <-ctx.Done():
				// context was cancelled before we finished; exit cleanly
			}
		}(i+1, lat)
	}

	first := <-ch
	cancel() // cancel the remaining goroutines
	fmt.Printf("  first response: worker%d → %s\n", first.worker, first.value)
}
```

---

### Bounded concurrency (`patterns.go`)

Lanzar muchas goroutines pero limitar cuántas corren simultáneamente con un semáforo.
Previene el thundering herd y el agotamiento de recursos.

```go
func demoBounded() {
	const total = 12
	const maxConcurrent = 3

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for i := 1; i <= total; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sem <- struct{}{}        // acquire: bloquea si ya hay 3 corriendo
			defer func() { <-sem }() // release

			fmt.Printf("  task%02d running\n", id)
			time.Sleep(15 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	// peak concurrency: 3 (max allowed: 3)
}
```

---

## Reglas prácticas

| Regla | Motivo |
|---|---|
| Toda goroutine necesita una salida conocida | Evitar leaks |
| Pasar contexto o done channel, no variables globales | Cancelación limpia y explícita |
| Pasar variables de loop como argumentos | Evitar el bug de captura de closure |
| `recover()` dentro de la goroutine que puede hacer panic | Es el único lugar donde funciona |
| No asumir orden de ejecución entre goroutines | El scheduler es no-determinístico |
| Medir con `runtime.NumGoroutine()` antes y después | Detectar leaks en tests |
