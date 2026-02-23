# timers

Ejemplos de `time.Timer`, `time.Ticker` y los patrones más comunes de temporización en Go.

## Ejecutar

```bash
go run .
```

## Estructura

| Archivo | Contenido |
|---------|-----------|
| `timer.go` | `NewTimer`, `Stop`, `Reset`, `AfterFunc` |
| `ticker.go` | `NewTicker`, `Ticker.Reset`, `time.Tick` |
| `timeafter.go` | `time.After`, timeout en select, riesgo de fuga |
| `patterns.go` | debounce, rate limiter, retry backoff, tarea periódica |

---

## time.Timer — disparo único

Un `Timer` dispara **una sola vez** tras la duración indicada.
El valor recibido en `.C` es el instante en que disparó.

```go
// timer.go
func demoTimer() {
    timer := time.NewTimer(80 * time.Millisecond)

    fmt.Println("  waiting for timer...")
    t := <-timer.C // blocks until the timer fires
    fmt.Printf("  fired at %s\n", t.Format("15:04:05.000"))
}
```

---

## time.Timer.Stop — cancelar antes de disparar

`Stop()` devuelve `true` si canceló el timer a tiempo.
Si devuelve `false`, el tick **ya está en el canal** y hay que drenarlo.

```go
// timer.go
func demoTimerStop() {
    timer := time.NewTimer(200 * time.Millisecond)

    stopped := timer.Stop()
    fmt.Printf("  Stop() returned: %v (true = cancelled in time)\n", stopped)

    if !stopped {
        // Drain to avoid a ghost tick reaching a future select.
        <-timer.C
        fmt.Println("  drained ghost tick")
    }

    select {
    case <-timer.C:
        fmt.Println("  unexpected tick")
    case <-time.After(300 * time.Millisecond):
        fmt.Println("  confirmed: no tick after Stop()")
    }
}
```

---

## time.Timer.Reset — reusar el timer

Secuencia segura para Go < 1.23: **stop → drain → reset**.

```go
// timer.go
func demoTimerReset() {
    timer := time.NewTimer(500 * time.Millisecond)

    // Stop and drain before resetting to avoid a stale tick.
    if !timer.Stop() {
        select {
        case <-timer.C:
        default:
        }
    }
    timer.Reset(60 * time.Millisecond) // new shorter duration

    t := <-timer.C
    fmt.Printf("  reset timer fired at %s\n", t.Format("15:04:05.000"))
}
```

---

## time.AfterFunc — callback en goroutine

Ejecuta una función en su propia goroutine tras el delay.
No expone canal; el `*Timer` devuelto sigue siendo cancelable con `Stop()`.

```go
// timer.go
func demoAfterFunc() {
    done := make(chan struct{})

    t := time.AfterFunc(60*time.Millisecond, func() {
        fmt.Printf("  AfterFunc callback at %s\n", time.Now().Format("15:04:05.000"))
        close(done)
    })

    fmt.Printf("  AfterFunc scheduled at %s\n", time.Now().Format("15:04:05.000"))
    <-done

    t.Stop() // safe no-op after callback has run
}
```

---

## time.NewTicker — disparo periódico

Un `Ticker` dispara **indefinidamente** al intervalo dado.
**Siempre llama a `Stop()`** — un ticker no detenido mantiene una goroutine
interna activa durante toda la vida del programa.

```go
// ticker.go
func demoTicker() {
    ticker := time.NewTicker(40 * time.Millisecond)
    defer ticker.Stop() // critical: free the internal goroutine

    deadline := time.After(160 * time.Millisecond)

    for {
        select {
        case t := <-ticker.C:
            fmt.Printf("  tick at %s\n", t.Format("15:04:05.000"))
        case <-deadline:
            fmt.Println("  deadline reached, stopping ticker")
            return
        }
    }
}
```

---

## time.Ticker.Reset — cambiar el intervalo en caliente

`Reset` detiene el ticker actual y lo reinicia con la nueva duración.
El canal NO se drena automáticamente; ticks previos pueden seguir ahí.

```go
// ticker.go
func demoTickerReset() {
    ticker := time.NewTicker(20 * time.Millisecond)
    defer ticker.Stop()

    fmt.Println("  phase 1: 20 ms interval")
    for i := 0; i < 3; i++ {
        t := <-ticker.C
        fmt.Printf("    tick %d at %s\n", i+1, t.Format("15:04:05.000"))
    }

    ticker.Reset(70 * time.Millisecond) // switch to a slower interval
    fmt.Println("  phase 2: 70 ms interval")
    for i := 0; i < 3; i++ {
        t := <-ticker.C
        fmt.Printf("    tick %d at %s\n", i+1, t.Format("15:04:05.000"))
    }
}
```

---

## time.Tick — atajo periódico (solo en programas de larga vida)

`time.Tick` devuelve únicamente el canal, sin exponer el `Ticker`.
**No hay forma de llamar `Stop()`**: el ticker vive para siempre.
Úsalo solo en `main` o en loops de muy larga vida.

```go
// ticker.go
func demoTimeTick() {
    fmt.Println("  time.Tick is safe here because main() runs only once")

    ch := time.Tick(50 * time.Millisecond) // leaks if called repeatedly
    deadline := time.After(160 * time.Millisecond)

    for {
        select {
        case t := <-ch:
            fmt.Printf("  time.Tick at %s\n", t.Format("15:04:05.000"))
        case <-deadline:
            fmt.Println("  done")
            return
        }
    }
}
```

---

## time.After — atajo de un solo disparo

`time.After(d)` es azúcar sintáctica para `time.NewTimer(d).C`.
El timer subyacente se libera **solo cuando el canal dispara**.

> ⚠ **Fuga en bucles**: si el `select` toma otra rama antes de que dispare
> el timer, el timer **no** es recolectado por el GC hasta que expire.
> En bucles ajustados, usa `time.NewTimer` y reutilízalo.

```go
// timeafter.go
func demoTimeAfter() {
    fmt.Println("  waiting 60 ms via time.After...")
    t := <-time.After(60 * time.Millisecond)
    fmt.Printf("  received at %s\n", t.Format("15:04:05.000"))
}
```

---

## Patrón: timeout en select

La forma canónica de competir una operación lenta contra un deadline.

```go
// timeafter.go
func demoTimeout() {
    result := make(chan string)

    go func() {
        time.Sleep(200 * time.Millisecond) // slow operation
        result <- "data"
    }()

    select {
    case v := <-result:
        fmt.Println("  got result:", v)
    case <-time.After(100 * time.Millisecond):
        fmt.Println("  timed out waiting for result")
    }
}
```

---

## Patrón: debounce

Ignora eventos rápidos y actúa solo tras un período de silencio.
Cada nuevo evento reinicia el timer; la acción se ejecuta una sola vez.

```go
// patterns.go
func demoDebounce() {
    debounce := 120 * time.Millisecond
    timer := time.NewTimer(debounce)
    defer timer.Stop()

    // ...receive events on eventCh...
    for {
        select {
        case e, ok := <-eventCh:
            if !ok {
                eventCh = nil
                continue
            }
            fmt.Printf("  received %s — resetting timer\n", e)
            // Reset the debounce timer on each event.
            if !timer.Stop() {
                select {
                case <-timer.C:
                default:
                }
            }
            timer.Reset(debounce)

        case <-timer.C:
            fmt.Println("  debounced action fired")
            if eventCh == nil {
                return
            }
        }
    }
}
```

---

## Patrón: rate limiter

Procesa a lo sumo una solicitud por tick de un `Ticker`.

```go
// patterns.go
func demoRateLimit() {
    requests := make(chan int, 8)
    for i := 1; i <= 8; i++ {
        requests <- i
    }
    close(requests)

    // Allow one request every 50 ms.
    limiter := time.NewTicker(50 * time.Millisecond)
    defer limiter.Stop()

    for req := range requests {
        <-limiter.C // wait for the next token
        fmt.Printf("    request %d processed at %s\n", req, time.Now().Format("15:04:05.000"))
    }
}
```

---

## Patrón: retry con exponential backoff

El delay se duplica en cada fallo, con jitter aleatorio para evitar
el "thundering herd" cuando muchos clientes reintentann a la vez.

```go
// patterns.go
func demoRetryBackoff() {
    const (
        maxAttempts = 5
        baseDelay   = 20 * time.Millisecond
        maxDelay    = 200 * time.Millisecond
        failUntil   = 3
    )
    attempt := 0
    delay := baseDelay

    for {
        attempt++
        // ... run operation ...
        if attempt < failUntil {
            jitter := time.Duration(rand.Int63n(int64(delay / 2)))
            wait := delay + jitter
            if wait > maxDelay {
                wait = maxDelay
            }
            timer := time.NewTimer(wait)
            <-timer.C
            timer.Stop()
            delay *= 2
        } else {
            fmt.Println("  success")
            return
        }
    }
}
```

---

## Patrón: tarea periódica cancelable

Combina un `Ticker` con un canal `done` (o `context.Done()`) para detener
la tarea limpiamente desde fuera.

```go
// patterns.go
func demoPeriodic() {
    done := make(chan struct{})
    ticker := time.NewTicker(60 * time.Millisecond)

    go func() {
        time.Sleep(250 * time.Millisecond)
        close(done)
    }()

    count := 0
    for {
        select {
        case t := <-ticker.C:
            count++
            fmt.Printf("    tick %d at %s\n", count, t.Format("15:04:05.000"))
        case <-done:
            ticker.Stop()
            fmt.Printf("  cancelled after %d ticks\n", count)
            return
        }
    }
}
```

---

## Tabla de referencia rápida

| API | Tipo | Descripción | ¿Cancelable? |
|-----|------|-------------|:------------:|
| `time.NewTimer(d)` | one-shot | dispara una vez tras `d` | `Stop()` |
| `time.AfterFunc(d, f)` | one-shot | llama `f` en goroutine tras `d` | `Stop()` |
| `time.After(d)` | one-shot | atajo para `NewTimer(d).C` | no (fuga en bucles) |
| `time.NewTicker(d)` | repeating | dispara cada `d` | `Stop()` |
| `time.Tick(d)` | repeating | atajo para `NewTicker(d).C` | **nunca** (solo en main) |

## Reglas clave

1. **Siempre `Stop()` los timers y tickers** que ya no necesites.
2. **Stop + drain antes de Reset** (Go < 1.23):
   ```go
   if !timer.Stop() {
       select { case <-timer.C: default: }
   }
   timer.Reset(newDuration)
   ```
3. **No uses `time.After` en bucles ajustados** — usa `NewTimer` y reutilízalo.
4. **No uses `time.Tick` dentro de funciones** que se llamen más de una vez.
