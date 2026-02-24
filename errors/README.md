# errors

Manejo de errores idiomático en Go: errores centinela, tipos custom,
wrapping con `%w`, `errors.Is/As`, métodos `Is()/As()` personalizados,
`errors.Join` y patrones de producción.

## Ejecutar

```bash
go run .
```

## Estructura

| Archivo | Contenido |
|---------|-----------|
| `sentinel.go` | `errors.New`, `errors.Is`, errores centinela |
| `types.go` | Tipos custom, `errors.As` |
| `wrapping.go` | `fmt.Errorf %w`, cadena de Unwrap, `%v` vs `%w` |
| `custom_is_as.go` | Métodos `Is()` y `As()` personalizados |
| `join.go` | `errors.Join`, colectar errores múltiples |
| `patterns.go` | `OpError`, errores opacos, panic vs error |

---

## Errores centinela y errors.Is

Un error centinela es una variable de paquete que representa una condición
conocida. Los llamadores comparan con `errors.Is`, que **recorre toda la
cadena de Unwrap** — no es una comparación `==` simple.

```go
// sentinel.go
var (
	ErrNotFound   = errors.New("not found")
	ErrPermission = errors.New("permission denied")
	ErrTimeout    = errors.New("operation timed out")
)

func findUser(id int) (string, error) {
	switch id {
	case 1:
		return "alice", nil
	case 2:
		return "", ErrPermission
	default:
		return "", fmt.Errorf("findUser %d: %w", id, ErrNotFound)
	}
}

func demoSentinel() {
	ids := []int{1, 2, 99}
	for _, id := range ids {
		name, err := findUser(id)
		if err == nil {
			fmt.Printf("  id=%d → user=%q\n", id, name)
			continue
		}
		switch {
		case errors.Is(err, ErrNotFound):
			fmt.Printf("  id=%d → not found (wrapped: %v)\n", id, err)
		case errors.Is(err, ErrPermission):
			fmt.Printf("  id=%d → access denied\n", id)
		default:
			fmt.Printf("  id=%d → unexpected error: %v\n", id, err)
		}
	}

	// errors.Is recorre cadenas de cualquier profundidad.
	wrapped := fmt.Errorf("service layer: %w", fmt.Errorf("repo layer: %w", ErrNotFound))
	fmt.Println("  errors.Is(wrapped, ErrNotFound):", errors.Is(wrapped, ErrNotFound)) // true
}
```

---

## Tipos de error custom y errors.As

Usa un tipo custom cuando el error necesita **campos que el llamador pueda
inspeccionar programáticamente**. `errors.As` extrae el tipo concreto de
la cadena de Unwrap — es el equivalente tipado de `errors.Is`.

```go
// types.go
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %q: %s", e.Field, e.Message)
}

type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}

func fetchResource(path string) error {
	if path == "/secret" {
		return fmt.Errorf("fetch %q: %w", path, &HTTPError{Code: http.StatusForbidden, Message: "forbidden"})
	}
	return nil
}

func demoCustomType() {
	inputs := []string{"", "abc", "25"}
	for _, input := range inputs {
		_, err := parseAge(input)
		if err == nil {
			fmt.Printf("  parseAge(%q) → ok\n", input)
			continue
		}
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			fmt.Printf("  parseAge(%q) → field=%q msg=%q\n", input, valErr.Field, valErr.Message)
		}
	}

	paths := []string{"/public", "/secret", "/missing"}
	for _, path := range paths {
		err := fetchResource(path)
		if err == nil {
			fmt.Printf("  fetch(%q) → ok\n", path)
			continue
		}
		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			// errors.As unwrapped la cadena para encontrar *HTTPError.
			fmt.Printf("  fetch(%q) → code=%d msg=%q\n", path, httpErr.Code, httpErr.Message)
		}
	}
}
```

---

## Wrapping con %w y cadena de Unwrap

`fmt.Errorf("%w", err)` envuelve el error y lo hace accesible via `errors.Unwrap`.
Usa `%v` (en lugar de `%w`) cuando la causa es un detalle interno que los
llamadores **no deben** inspeccionar.

```go
// wrapping.go
func demoWrapping() {
	// Cadena de tres niveles: db → repo → service.
	dbErr := errors.New("connection refused")
	repoErr := fmt.Errorf("repo.FindUser: %w", dbErr)
	svcErr := fmt.Errorf("service.GetUser id=42: %w", repoErr)

	fmt.Println("  full error:", svcErr)
	// service.GetUser id=42: repo.FindUser: connection refused

	// errors.Unwrap pela exactamente una capa.
	fmt.Println("  Unwrap once:", errors.Unwrap(svcErr))
	fmt.Println("  Unwrap twice:", errors.Unwrap(errors.Unwrap(svcErr)))

	// errors.Is recorre toda la cadena.
	fmt.Println("  Is(dbErr):", errors.Is(svcErr, dbErr)) // true

	// %v rompe la cadena — errors.Is ya no puede encontrar la causa.
	opaque := fmt.Errorf("something went wrong: %v", dbErr) // %v, not %w
	fmt.Println("  Is(dbErr) through %v:", errors.Is(opaque, dbErr)) // false
}
```

Convención de mensaje: `"operación: causa"` — sin mayúscula inicial, sin punto final.

---

## errors.Is con método Is() personalizado

Implementar `Is(error) bool` permite que `errors.Is` compare por igualdad
**semántica** en lugar de identidad de puntero.

```go
// custom_is_as.go
type StatusError struct {
	Code    int
	Message string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("status %d: %s", e.Code, e.Message)
}

// Is hace que errors.Is(err, &StatusError{Code: 404}) coincida con cualquier
// StatusError con Code 404, ignorando el mensaje.
func (e *StatusError) Is(target error) bool {
	var t *StatusError
	if !errors.As(target, &t) {
		return false
	}
	return e.Code == t.Code
}

func demoCustomIs() {
	sentinel := &StatusError{Code: 404, Message: ""}

	err1 := &StatusError{Code: 404, Message: "user not found"}
	err2 := &StatusError{Code: 500, Message: "internal error"}
	wrapped := fmt.Errorf("handler: %w", &StatusError{Code: 404, Message: "post not found"})

	fmt.Println("  err1 Is 404 sentinel:", errors.Is(err1, sentinel))       // true
	fmt.Println("  err2 Is 404 sentinel:", errors.Is(err2, sentinel))       // false
	fmt.Println("  wrapped Is 404 sentinel:", errors.Is(wrapped, sentinel)) // true
}
```

---

## errors.As con método As() personalizado

Implementar `As(any) bool` permite que `errors.As` busque dentro de tipos
contenedor (colecciones, wrappers, etc.).

```go
// custom_is_as.go
type MultiError struct {
	Errors []error
}

func (m *MultiError) Error() string {
	return fmt.Sprintf("%d errors occurred", len(m.Errors))
}

// As busca en cada error contenido uno asignable a target.
func (m *MultiError) As(target any) bool {
	for _, err := range m.Errors {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

func demoCustomAs() {
	multi := &MultiError{
		Errors: []error{
			fmt.Errorf("first: %w", &ValidationError{Field: "name", Message: "too short"}),
			&StatusError{Code: 422, Message: "unprocessable"},
		},
	}

	var valErr *ValidationError
	if errors.As(multi, &valErr) {
		fmt.Printf("  found ValidationError: field=%q msg=%q\n", valErr.Field, valErr.Message)
	}

	var statusErr *StatusError
	if errors.As(multi, &statusErr) {
		fmt.Printf("  found StatusError: code=%d msg=%q\n", statusErr.Code, statusErr.Message)
	}
}
```

---

## errors.Join — múltiples errores (Go 1.20+)

`errors.Join` combina varios errores en uno. `errors.Is` y `errors.As`
funcionan contra cada error del conjunto.

```go
// join.go
func demoJoin() {
	err1 := errors.New("database timeout")
	err2 := errors.New("cache miss")
	err3 := errors.New("rate limit exceeded")

	joined := errors.Join(err1, err2, err3)
	fmt.Println("  joined error:")
	fmt.Println(" ", joined)
	// database timeout
	// cache miss
	// rate limit exceeded

	fmt.Println("  Is(err2):", errors.Is(joined, err2)) // true

	// nil inputs se ignoran; si todos son nil, retorna nil.
	noErr := errors.Join(nil, nil)
	fmt.Println("  Join(nil, nil):", noErr) // <nil>

	// Colectar errores de validación y combinarlos.
	var errs []error
	for _, f := range fields {
		if f.value == "" {
			errs = append(errs, &ValidationError{Field: f.name, Message: "must not be empty"})
		}
	}
	if combined := errors.Join(errs...); combined != nil {
		fmt.Println(" ", combined)
	}
}
```

---

## Patrón: error de operación con contexto

El patrón de `net.OpError` / `os.PathError` de la stdlib: captura operación,
recurso y causa en campos separados. Implementa `Unwrap()` para mantener la
cadena accesible.

```go
// patterns.go
type OpError struct {
	Op   string // "read", "write", "connect", …
	Path string // identificador del recurso
	Err  error  // causa subyacente
}

func (e *OpError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
}

func (e *OpError) Unwrap() error { return e.Err }

var ErrConnectionRefused = errors.New("connection refused")

func openDB(dsn string) error {
	return &OpError{Op: "connect", Path: dsn, Err: ErrConnectionRefused}
}

func demoOpError() {
	err := openDB("postgres://localhost:5432/mydb")
	fmt.Println("  error:", err)
	// connect postgres://localhost:5432/mydb: connection refused

	fmt.Println("  Is(ErrConnectionRefused):", errors.Is(err, ErrConnectionRefused)) // true

	var opErr *OpError
	if errors.As(err, &opErr) {
		fmt.Printf("  op=%q path=%q\n", opErr.Op, opErr.Path)
	}
}
```

---

## Patrón: errores opacos vs exportados

```go
// patterns.go

// Exportado — los llamadores PUEDEN detectarlo con errors.Is.
var ErrInvalidInput = errors.New("invalid input")

// No exportado — los llamadores NO pueden inspeccionarlo.
// Útil para errores que son detalles de implementación.
var opaqueErr = errors.New("internal state corrupted")

func process(input string) error {
	if input == "" {
		return fmt.Errorf("process: %w", ErrInvalidInput) // %w → detectable
	}
	if input == "corrupt" {
		return fmt.Errorf("process: %v", opaqueErr) // %v → opaco
	}
	return nil
}

func demoOpaque() {
	cases := []string{"", "corrupt", "ok"}
	for _, input := range cases {
		err := process(input)
		if err == nil {
			fmt.Printf("  process(%q) → ok\n", input)
			continue
		}
		fmt.Printf("  process(%q) → %v\n", input, err)
		fmt.Printf("    Is(ErrInvalidInput): %v\n", errors.Is(err, ErrInvalidInput))
	}
	// process("") → Is(ErrInvalidInput): true
	// process("corrupt") → Is(ErrInvalidInput): false
}
```

---

## Patrón: panic vs error

```go
// patterns.go

// Recover en el boundary de la API: convierte panic en error.
safeDiv := func(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("safeDiv: panic: %v", r)
		}
	}()
	result = a / b // panic si b == 0
	return
}

v, err := safeDiv(10, 2) // v=5, err=nil
_, err = safeDiv(10, 0)  // err="safeDiv: panic: runtime error: integer divide by zero"

// Panic para errores de programación detectados en startup.
mustParseConfig := func(path string) string {
	if path == "" {
		panic("mustParseConfig: path must not be empty") // falla rápido
	}
	return path
}
```

Regla: **retorna errores** para condiciones esperadas y recuperables;
**panic** para invariantes del programa que nunca deberían ocurrir en código correcto.
Nunca dejes que un panic cruce la frontera de una API pública sin convertirlo en error.

---

## Tabla de referencia rápida

| Función | Comportamiento |
|---------|---------------|
| `errors.New("msg")` | Crea un error sin campos extra |
| `fmt.Errorf("ctx: %w", err)` | Envuelve `err` en la cadena |
| `fmt.Errorf("ctx: %v", err)` | Convierte a string — rompe la cadena |
| `errors.Is(err, target)` | Busca `target` en toda la cadena |
| `errors.As(err, &ptr)` | Extrae el primer valor del tipo de `ptr` |
| `errors.Unwrap(err)` | Pela una capa (`Unwrap() error`) |
| `errors.Join(errs...)` | Combina varios errores (Go 1.20+) |

## Reglas clave

1. **Usa `%w` para envolver**, `%v` para ocultar la causa intencionalmente.
2. **Usa `errors.Is/As`**, nunca `==` directo sobre errores envueltos.
3. **Implementa `Unwrap() error`** en tipos custom que envuelvan otro error.
4. **Implementa `Is()`** cuando la igualdad semántica difiere de la identidad de puntero.
5. **Implementa `As()`** en tipos contenedor para que `errors.As` busque dentro.
6. **Expón solo los errores centinela que los llamadores deben manejar**; usa unexported para detalles internos.
7. **Panic solo para invariantes de programación** — nunca para condiciones de runtime esperadas.
