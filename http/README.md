# http

Patrones de `net/http` que aparecen en entrevistas técnicas de backend en Go.

## Ejecutar

```bash
go run .
```

## Estructura

| Archivo | Contenido |
|---------|-----------|
| `server.go` | `Handler`, `HandlerFunc`, `ServeMux`, routing Go 1.22 (`{id}`, método) |
| `middleware.go` | Logger, Auth, Recovery, patrón `Chain` |
| `client.go` | `http.Client` con timeout, status codes, cancelación con context |
| `shutdown.go` | Graceful shutdown — drenar requests en vuelo antes de parar |
| `recorder.go` | `httptest.NewRecorder` (unit) vs `httptest.NewServer` (integración) |

---

## Server — Handler, ServeMux

```go
// http.Handler — la interfaz central
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// http.HandlerFunc — adapter: convierte una función en un Handler
mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "hello")
})

// Struct-based handler — cuando el handler necesita estado
type greetHandler struct{ greeting string }
func (h greetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) { ... }
mux.Handle("GET /greet", greetHandler{greeting: "Hello"})
```

### ServeMux — Go 1.22

```go
mux := http.NewServeMux() // no usar http.DefaultServeMux en producción

// Método + ruta (Go 1.22+)
mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")           // extrae {id} del path
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"id": id})
})

// Wildcard catch-all {path...}
mux.HandleFunc("/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
    path := r.PathValue("path")       // captura todo después de /files/
    fmt.Fprintf(w, "file: %s", path)
})

// Respuesta JSON
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)    // llamar ANTES de escribir el body
json.NewEncoder(w).Encode(payload)

// Error con status code
http.Error(w, "not found", http.StatusNotFound)

// Leer body JSON
json.NewDecoder(r.Body).Decode(&payload)
defer r.Body.Close()
```

**Precedencia**: el patrón más específico gana. `"GET /users/me"` tiene prioridad sobre `"GET /users/{id}"`.

---

## Middleware

```go
// Tipo canónico de middleware
type Middleware func(http.Handler) http.Handler

// Wrapping pattern
func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)          // llama al siguiente en la cadena
        log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
    })
}

// Capturar el status code — ResponseWriter no expone WriteHeader por defecto
type responseRecorder struct {
    http.ResponseWriter
    status int
}
func (r *responseRecorder) WriteHeader(code int) {
    r.status = code
    r.ResponseWriter.WriteHeader(code)
}

// Auth via closure — el token se captura en construcción
func Auth(validToken string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
            if got != validToken {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return  // NO llamar a next
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Recovery — atrapa panics y devuelve 500
func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                http.Error(w, "internal server error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// Chain — aplica middlewares de derecha a izquierda; el primero listado ejecuta primero
// Chain(h, mw1, mw2, mw3) ≡ mw1(mw2(mw3(h)))
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
    for i := len(mws) - 1; i >= 0; i-- {
        h = mws[i](h)
    }
    return h
}

// Uso
mux.Handle("GET /protected",
    Chain(handler, Logger, Auth("secret"), Recovery),
)
```

---

## Client

```go
// NUNCA usar http.DefaultClient en producción — no tiene timeout
// Un servidor que no responde bloquea la goroutine para siempre

client := &http.Client{Timeout: 5 * time.Second}

// Non-2xx NO devuelve error — siempre verificar el status code
resp, err := client.Get(url)
if err != nil {
    return err // error de transporte (DNS, TLS, timeout de red)
}
defer resp.Body.Close()                    // siempre cerrar
if resp.StatusCode >= 400 {
    body, _ := io.ReadAll(resp.Body)
    return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
}

// Context cancellation — usar NewRequestWithContext, no req.WithContext
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
resp, err = client.Do(req)

// Drenar body aunque no lo uses — permite reutilizar la conexión TCP
io.Copy(io.Discard, resp.Body)
resp.Body.Close()
```

---

## Graceful shutdown

```go
srv := &http.Server{Addr: ":8080", Handler: mux}

// En producción: esperar señal del OS
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

go func() {
    if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
        log.Fatal(err)  // error real — ErrServerClosed es esperado
    }
}()

<-ctx.Done()   // bloqueado hasta SIGINT / SIGTERM

stop()         // dejar de recibir señales

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Shutdown: deja de aceptar nuevas conexiones,
// espera que los handlers activos terminen (hasta shutdownCtx).
if err := srv.Shutdown(shutdownCtx); err != nil {
    log.Fatal("shutdown error:", err)
}
```

**Clave**: `ListenAndServe` devuelve `http.ErrServerClosed` al finalizar `Shutdown` — esto **no es un error**, es la señal de que el cierre fue limpio.

---

## httptest

```go
// NewRecorder — test unitario, sin red, sin OS resources
req := httptest.NewRequest("GET", "/users/42", nil)
w   := httptest.NewRecorder()
handler.ServeHTTP(w, req)

w.Code                     // status code
w.Body.String()            // body
w.Result().Header.Get("X") // headers

// NewServer — test de integración con TCP real
// Necesario cuando el código bajo test es un http.Client
srv := httptest.NewServer(handler)
defer srv.Close()
resp, _ := http.Get(srv.URL + "/users/42")
```

### Table-driven handler test

```go
func TestUserHandler(t *testing.T) {
    tests := []struct {
        name   string
        method string
        path   string
        body   string
        want   int
    }{
        {"get existing",      "GET",    "/users/1", "",             200},
        {"post valid",        "POST",   "/users",   `{"name":"X"}`, 201},
        {"post invalid json", "POST",   "/users",   `bad`,          400},
        {"method not allowed","DELETE", "/users/1", "",             405},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            req := httptest.NewRequest(tc.method, tc.path,
                strings.NewReader(tc.body))
            w := httptest.NewRecorder()
            mux.ServeHTTP(w, req)
            if w.Code != tc.want {
                t.Errorf("want %d, got %d", tc.want, w.Code)
            }
        })
    }
}
```

---

## Reglas clave

1. **`http.Handler`** es la interfaz central — cualquier tipo con `ServeHTTP` lo satisface.
2. **`http.HandlerFunc`** es el adaptador para funciones — evita crear structs innecesarios.
3. **`http.NewServeMux()`** en lugar de `http.DefaultServeMux` — terceros pueden registrar rutas accidentalmente.
4. **Go 1.22**: `"GET /users/{id}"` y `r.PathValue("id")` — routing con método y wildcards sin librería externa.
5. **`http.Client` siempre con `Timeout`** — `DefaultClient` sin timeout puede colgar para siempre.
6. **Non-2xx no es error del cliente** — verificar `resp.StatusCode` explícitamente.
7. **Siempre `defer resp.Body.Close()`** y drenar el body — necesario para reutilizar conexiones TCP.
8. **`NewRequestWithContext`** en lugar de `req.WithContext` — la versión con context desde construcción.
9. **`http.ErrServerClosed`** es esperado al hacer `Shutdown` — no tratarlo como error.
10. **`httptest.NewRecorder`** para unit tests de handlers; **`httptest.NewServer`** para probar clients.
