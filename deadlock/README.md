# deadlock

Ejemplos de todos los estados de bloqueo que aparecen en un goroutine dump,
más el deadlock clásico AB por orden inconsistente de locks.

## Ejecutar

```bash
go run .          # muestra todos los estados y termina con exit 1
```

Para inspeccionar estados en vivo antes del crash:
```bash
# Terminal 1
go run .

# Terminal 2
go tool pprof http://localhost:6062/debug/pprof/goroutine?debug=2
```

## Estructura

| Archivo | Contenido |
|---------|-----------|
| `dump.go` | `dumpGoroutines()` — captura todos los goroutines con `runtime.Stack` |
| `channel.go` | `[chan receive]`, `[chan send]`, `[select]` |
| `io.go` | `[IO wait]` — socket TCP real (net.Pipe() no sirve) |
| `running.go` | `[running]` / `[runnable]` — busy loop |
| `mutex.go` | `[semacquire]` / `[sync.Mutex.Lock]` + deadlock AB final |

---

## Tabla de estados de goroutine

| Estado | Cuándo aparece |
|--------|---------------|
| `[running]` | El goroutine se está ejecutando en un OS thread en este momento |
| `[runnable]` | Listo para ejecutar, esperando un OS thread libre |
| `[chan receive]` | Bloqueado en `v := <-ch` (canal sin sender) |
| `[chan send]` | Bloqueado en `ch <- v` (canal sin receiver) |
| `[select]` | Bloqueado en `select` con todos los cases bloqueados |
| `[semacquire]` / `[sync.Mutex.Lock]` | Esperando adquirir un mutex / semáforo |
| `[IO wait]` | Bloqueado en el poller del OS esperando datos de red o I/O |
| `[sleep]` | Dentro de `time.Sleep` |
| `[syscall]` | Ejecutando una syscall bloqueante del OS |

> `[semacquire]` es el nombre histórico (Go < 1.22).
> Go ≥ 1.22 usa etiquetas más descriptivas: `[sync.Mutex.Lock]`, `[sync.WaitGroup.Wait]`, etc.

---

## Cómo leer un goroutine dump

`runtime.Stack(buf, true)` — o el panic del runtime — produce bloques como este:

```
goroutine 7 [chan receive]:          ← ID y estado de bloqueo
main.demoChanReceive.func1()         ← frame superior (dónde está bloqueado)
    /path/to/channel.go:22 +0x70
created by main.demoChanReceive in goroutine 1
    /path/to/channel.go:20 +0x6c
```

El estado entre corchetes es lo primero que hay que leer para diagnosticar.

```go
// dump.go
func dumpGoroutines() {
	buf := make([]byte, 256*1024)
	n := runtime.Stack(buf, true)
	raw := strings.TrimSpace(string(buf[:n]))

	fmt.Println()
	for _, block := range strings.Split(raw, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")

		// Line 0: "goroutine N [state, X minutes]:"
		fmt.Printf("  %s\n", lines[0])

		// Show up to 4 stack frames for context.
		limit := min(len(lines)-1, 4)
		for i := 1; i <= limit; i++ {
			fmt.Printf("  %s\n", lines[i])
		}
		if len(lines)-1 > limit {
			fmt.Printf("  ... (+%d lines)\n", len(lines)-1-limit)
		}
		fmt.Println()
	}
}
```

---

## [chan receive] — bloqueado esperando recibir

El goroutine está en `v := <-ch` y ningún otro goroutine enviará nunca al canal.
Aparece directamente como `[chan receive]` (no como `[select]`).

```go
// channel.go
func demoChanReceive() {
	ch := make(chan int) // unbuffered — no sender will ever write

	go func() {
		fmt.Println("  goroutine: blocking on  v := <-ch  (no sender)")
		v := <-ch // ← blocked here, shows as [chan receive]
		fmt.Println("  goroutine: received", v) // unreachable
	}()

	time.Sleep(80 * time.Millisecond) // let the goroutine reach the block
	dumpGoroutines()
	// goroutine is intentionally leaked — it will appear in the final crash dump
}
```

Dump resultante:
```
goroutine 6 [chan receive]:
main.demoChanReceive.func1()
    .../channel.go:22 +0x70
created by main.demoChanReceive in goroutine 1
    .../channel.go:20 +0x6c
```

---

## [chan send] — bloqueado esperando enviar

El goroutine está en `ch <- v` y ningún otro goroutine recibirá nunca del canal.

```go
// channel.go
func demoChanSend() {
	ch := make(chan int) // unbuffered — no receiver will ever read

	go func() {
		fmt.Println("  goroutine: blocking on  ch <- 42  (no receiver)")
		ch <- 42 // ← blocked here, shows as [chan send]
		fmt.Println("  goroutine: sent (unreachable)")
	}()

	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
	// goroutine intentionally leaked
}
```

Dump resultante:
```
goroutine 7 [chan send]:
main.demoChanSend.func1()
    .../channel.go:45 +0x6c
created by main.demoChanSend in goroutine 1
    .../channel.go:43 +0x6c
```

---

## [select] — bloqueado en select con todos los cases bloqueados

El goroutine entró via `select` — aunque sólo haya un case, el runtime lo
etiqueta `[select]` en lugar de `[chan receive]`.

```go
// channel.go
func demoSelect() {
	ch1 := make(chan int)    // no sender
	ch2 := make(chan string) // no sender

	go func() {
		fmt.Println("  goroutine: blocking in select (both channels empty, no senders)")
		select {
		case v := <-ch1: // ← blocked
			fmt.Println("  goroutine: got int", v)
		case s := <-ch2: // ← blocked
			fmt.Println("  goroutine: got string", s)
		}
		// shows as [select], not [chan receive]
	}()

	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()
	// goroutine intentionally leaked
}
```

Dump resultante:
```
goroutine 8 [select]:
main.demoSelect.func1()
    .../channel.go:70 +0xb4
created by main.demoSelect in goroutine 1
    .../channel.go:68 +0x90
```

---

## [IO wait] — bloqueado en I/O de red

El goroutine está dentro del poller del OS esperando datos de un socket TCP
que nunca llegan. Requiere un socket real — `net.Pipe()` usa canales internamente
y aparece como `[select]`.

```go
// io.go
func demoIOWait() {
	// Start a TCP server that accepts a connection but never writes to it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  error setting up listener:", err)
		return
	}

	serverStop := make(chan struct{})
	serverDone := make(chan struct{})

	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return // listener was closed before accepting
		}
		defer conn.Close()
		// Hold the connection open (but never write) until signalled.
		<-serverStop
	}()

	// Client connects and immediately tries to Read — server never writes.
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		fmt.Println("  error connecting:", err)
		ln.Close()
		return
	}

	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		buf := make([]byte, 1)
		fmt.Printf("  goroutine: blocking on net.Conn.Read (server never writes)\n")
		_, err := conn.Read(buf) // ← blocked here inside OS poller, shows as [IO wait]
		if err != nil {
			fmt.Println("  goroutine: unblocked with error:", err)
		}
	}()

	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()

	// Cleanup: signal server to stop, close client conn, wait for both goroutines.
	close(serverStop)
	conn.Close()
	ln.Close()
	<-clientDone
	<-serverDone
}
```

Dump resultante:
```
goroutine 9 [IO wait]:
internal/poll.runtime_pollWait(0x..., 0x72)
internal/poll.(*pollDesc).waitRead(...)
internal/poll.(*netFD).Read(...)
net.(*conn).Read(...)
main.demoIOWait.func3()
    .../io.go:63 +0x...
```

---

## [running] / [runnable] — goroutine activo

`[running]` significa que el goroutine se estaba ejecutando en un OS thread
en el momento exacto del snapshot. El goroutine que llama a `runtime.Stack`
siempre aparece como `[running]`. Un busy loop aparece como `[running]` si
está en un OS thread en ese instante, o `[runnable]` si fue preemptado.

```go
// running.go
func demoRunning() {
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		fmt.Println("  goroutine: busy loop — appears as [running] or [runnable]")
		i := 0
		for {
			select {
			case <-stop:
				fmt.Printf("  goroutine: stopped after %d iterations\n", i)
				return
			default:
				i++
				// Gosched yields to other goroutines so the loop does not
				// starve them. At the moment of the snapshot this goroutine
				// may or may not hold an OS thread — hence [running] vs [runnable].
				runtime.Gosched()
			}
		}
	}()

	time.Sleep(80 * time.Millisecond)
	// dumpGoroutines itself shows [running] for the calling goroutine.
	dumpGoroutines()

	// IMPORTANT: wait for the goroutine to fully exit before continuing.
	// If it is still [runnable] when the final deadlock demo runs, the
	// runtime deadlock detector will not fire (a runnable goroutine means
	// the program can still make progress).
	close(stop)
	<-done
}
```

Dump resultante:
```
goroutine 1 [running]:               ← quien llama a dumpGoroutines()
main.dumpGoroutines()
    .../dump.go:28 +0x44

goroutine 5 [runnable]:              ← busy loop preemptado en el snapshot
runtime.Gosched(...)
    .../proc.go:342
main.demoRunning.func1()
    .../running.go:47 +0xb4
```

---

## [semacquire] / [sync.Mutex.Lock] — bloqueado en mutex

El goroutine está esperando adquirir un mutex que otro goroutine sostiene.
Go < 1.22 lo etiqueta `[semacquire]`; Go ≥ 1.22 usa `[sync.Mutex.Lock]`.

```go
// mutex.go
func demoSemacquire() {
	var mu sync.Mutex
	holderReady := make(chan struct{})
	releaseHolder := make(chan struct{})
	waiterDone := make(chan struct{})

	// Goroutine A: acquires the lock and holds it until signalled.
	go func() {
		mu.Lock()
		fmt.Println("  holder: acquired mu — will hold it")
		close(holderReady)
		<-releaseHolder
		mu.Unlock()
		fmt.Println("  holder: released mu")
	}()

	<-holderReady // wait until A holds the lock

	// Goroutine B: tries to acquire the same lock → blocks in [semacquire].
	go func() {
		defer close(waiterDone)
		fmt.Println("  waiter: blocking on mu.Lock() (held by holder)")
		mu.Lock() // ← blocked here, shows as [semacquire]
		fmt.Println("  waiter: acquired mu")
		mu.Unlock()
	}()

	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()

	// Cleanup: release so the waiter can proceed.
	close(releaseHolder)
	<-waiterDone
}
```

Dump resultante (Go 1.22+):
```
goroutine 7 [sync.Mutex.Lock]:
sync.runtime_SemacquireMutex(0x..., 0x0, 0x1)
sync.(*Mutex).lockSlow(...)
sync.(*Mutex).Lock(...)
main.demoSemacquire.func2()
    .../mutex.go:51 +0x...
```

---

## Deadlock AB — orden inconsistente de locks

El deadlock más común en producción: dos goroutines adquieren los mismos
locks en orden inverso. El runtime lo detecta cuando **todos** los goroutines
están dormidos y no puede hacerse ningún progreso.

```go
// mutex.go
var (
	muA sync.Mutex
	muB sync.Mutex
)

// goroutine1 locks A then waits for B.
func goroutine1(wg *sync.WaitGroup) {
	defer wg.Done()
	muA.Lock()
	fmt.Println("  goroutine1: locked A")
	time.Sleep(50 * time.Millisecond) // give goroutine2 time to lock B

	fmt.Println("  goroutine1: waiting for B...") // blocks here forever
	muB.Lock()
	defer muB.Unlock()
	defer muA.Unlock()
	fmt.Println("  goroutine1: locked both (unreachable)")
}

// goroutine2 locks B then waits for A.
func goroutine2(wg *sync.WaitGroup) {
	defer wg.Done()
	muB.Lock()
	fmt.Println("  goroutine2: locked B")
	time.Sleep(50 * time.Millisecond) // give goroutine1 time to lock A

	fmt.Println("  goroutine2: waiting for A...") // blocks here forever
	muA.Lock()
	defer muA.Unlock()
	defer muB.Unlock()
	fmt.Println("  goroutine2: locked both (unreachable)")
}
```

En un programa sin código de red, el runtime emitiría:
```
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [sync.Mutex.Lock]:         ← main bloqueado en wg.Wait()
sync.runtime_SemacquireMutex(...)
...

goroutine 18 [sync.Mutex.Lock]:        ← goroutine1 esperando muB
sync.(*Mutex).lockSlow(...)
main.goroutine1(...)

goroutine 19 [sync.Mutex.Lock]:        ← goroutine2 esperando muA
sync.(*Mutex).lockSlow(...)
main.goroutine2(...)
```

> **Nota macOS/Linux**: después de usar `net.Listen`, el poller kqueue/epoll
> permanece activo y suprime el detector de deadlock del runtime. El demo
> imprime el dump manualmente y sale con `exit 1` para simular el crash.

---

## Reglas clave

1. **Lee el estado entre corchetes primero** — es el diagnóstico más rápido.
2. **`[chan receive]` vs `[select]`**: mismo canal, distinto estado según si
   el goroutine entró con `<-ch` directo o vía `select`.
3. **`net.Pipe()` muestra `[select]`**, no `[IO wait]` — usa sockets TCP reales.
4. **Un goroutine `[runnable]` impide al detector de deadlock** dispararse —
   el runtime lo interpreta como "el programa puede hacer progreso".
5. **Fix del deadlock AB**: establece un orden canónico global de locks
   (`muA` siempre antes que `muB`) y respétalo en todos los goroutines.
