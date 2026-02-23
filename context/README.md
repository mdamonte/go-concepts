# context.Context — Guía completa con ejemplos

## ¿Qué es `context.Context`?

Es una interfaz del paquete estándar que permite pasar tres cosas a través de una
cadena de llamadas sin modificar las firmas de cada función:

1. **Señal de cancelación** — para abortar trabajo en progreso.
2. **Deadline / timeout** — para imponer límites de tiempo.
3. **Valores request-scoped** — IDs de request, tokens de auth, trace IDs.

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool)  // ¿hay un límite de tiempo?
    Done()      <-chan struct{}                // cerrado cuando se cancela
    Err()       error                         // Canceled | DeadlineExceeded
    Value(key any) any                        // valor asociado a la clave
}
```

La regla de oro: **el primer parámetro de toda función que haga I/O debe ser `context.Context`**.

---

## Árbol de contextos

Cada contexto deriva de un padre. Cancelar el padre cancela **todos** sus descendientes.
Cancelar un hijo **no** afecta al padre ni a los hermanos.

```
Background
└── WithCancel (parent)
    ├── WithCancel (child1)        ← cancelar child1 afecta solo a él y sus hijos
    │   └── WithCancel (grandchild)
    └── WithTimeout (child2, 10s)  ← sigue corriendo
```

---

## Constructores

| Constructor | Cuándo usarlo |
|---|---|
| `context.Background()` | Raíz del árbol: `main`, tests, handlers de alto nivel |
| `context.TODO()` | Placeholder hasta cablear el contexto real |
| `WithCancel(parent)` | Cancelación manual bajo demanda |
| `WithTimeout(parent, d)` | Presupuesto de tiempo relativo (e.g. 500 ms) |
| `WithDeadline(parent, t)` | SLA de reloj absoluto (e.g. "antes de las 14:00:05") |
| `WithValue(parent, k, v)` | Datos request-scoped (IDs, tokens, loggers) |
| `WithCancelCause(parent)` | Como WithCancel, pero con motivo específico (Go 1.20) |
| `WithTimeoutCause(parent, d, err)` | Como WithTimeout, adjunta causa al expirar (Go 1.21) |
| `WithDeadlineCause(parent, t, err)` | Como WithDeadline, adjunta causa al expirar (Go 1.21) |

---

## Archivos del módulo

```
context/
├── go.mod
├── main.go            — ejecuta todos los demos en orden
├── background_todo.go — Background() y TODO()
├── cancel.go          — WithCancel: parar un goroutine a demanda
├── timeout.go         — WithTimeout: presupuesto de tiempo relativo
├── deadline.go        — WithDeadline: límite de tiempo absoluto
├── value.go           — WithValue: datos request-scoped + patrón de clave tipada
├── cause.go           — WithCancelCause / WithTimeoutCause / WithDeadlineCause
├── propagation.go     — cascada de cancelación en un árbol de contextos
└── http.go            — context con HTTP server y client
```

---

## Cómo correrlo

```bash
go run .
```

---

## Detalles por caso de uso

### `WithCancel`

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel() // SIEMPRE defer; libera recursos aunque no canceles manualmente

go func() {
    select {
    case <-doWork():
    case <-ctx.Done():  // ctx.Done() es un canal cerrado al cancelar
        return
    }
}()

cancel() // señal a todos los goroutines que observan este ctx
```

### `WithTimeout` vs `WithDeadline`

```go
// Relativo: "dame hasta 500 ms"
ctx, cancel := context.WithTimeout(parent, 500*time.Millisecond)

// Absoluto: "termina antes de las 14:00:05.000"
ctx, cancel := context.WithDeadline(parent, time.Date(...))

// Ambos exponen el tiempo límite:
if deadline, ok := ctx.Deadline(); ok {
    fmt.Println("time left:", time.Until(deadline))
}
```

Cuando el tiempo expira, `ctx.Err()` devuelve `context.DeadlineExceeded`.

### `WithValue` — patrón de clave tipada

```go
// MAL: usar string como clave → colisiones entre paquetes
ctx = context.WithValue(ctx, "userID", 42)

// BIEN: tipo propio no exportado
type ctxKey string
const keyUserID ctxKey = "userID"
ctx = context.WithValue(ctx, keyUserID, 42)

// Recuperar con type assertion
if id, ok := ctx.Value(keyUserID).(int); ok {
    // ...
}
```

### HTTP server & client

El paquete `net/http` integra context de forma nativa en ambos lados.

#### Cliente — `http.NewRequestWithContext`

```go
// ✅ Correcto: el request lleva el context y puede ser cancelado.
ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
defer cancel()

req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err := http.DefaultClient.Do(req)
// Si el timeout expira antes → err contiene "context deadline exceeded"
```

Nunca usar `http.NewRequest` en producción: no tiene forma de cancelarse.

#### Servidor — `r.Context()`

Cada request HTTP llega con su propio context. El runtime lo cancela
automáticamente cuando el cliente se desconecta.

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // cancelado si el cliente se va

    // Pasar ctx a cada llamada downstream (DB, gRPC, otro HTTP).
    result, err := db.QueryContext(ctx, "SELECT ...")
    if err != nil {
        // Si el cliente desconectó, ctx.Err() == context.Canceled
        return
    }
}
```

#### Los cuatro escenarios de `http.go`

| Demo | Escenario |
|---|---|
| `demoClientTimeout` | Timeout en request saliente; éxito vs. `DeadlineExceeded` |
| `demoClientCancel` | Abortar un request en vuelo desde otra goroutine |
| `demoServerDisconnect` | Handler detecta desconexión del cliente y aborta trabajo costoso |
| `demoServerPropagation` | `r.Context()` se pasa al request downstream — toda la cadena se cancela junta |

La propagación es el patrón más importante: si el cliente original cancela,
el frontend cancela su llamada al "DB" sin esperar la respuesta.

```
Cliente → Frontend handler → DB service
   ↓ cancela
   ↓──────────────────────────↓ ctx.Done() se cierra en toda la cadena
```

---

### `WithCancelCause` — el "por qué" de la cancelación

```go
ctx, cancel := context.WithCancelCause(parent)
cancel(ErrRateLimit)         // adjunta el motivo

ctx.Err()              // → context.Canceled  (siempre)
context.Cause(ctx)     // → ErrRateLimit       (el motivo real)
```

---

## Reglas y antipatrones

| Regla | Motivo |
|---|---|
| Siempre `defer cancel()` | Evita goroutine leaks del timer interno |
| No guardar el contexto en structs | Debe fluir explícitamente por los parámetros |
| No pasar `nil` como contexto | Usa `context.TODO()` o `context.Background()` |
| No usar strings como claves de valor | Colisiones entre paquetes |
| No meter parámetros opcionales en el contexto | Para eso están los parámetros de la función |
| Chequear `ctx.Err()` antes de trabajo costoso | Abortar rápido si ya está cancelado |
