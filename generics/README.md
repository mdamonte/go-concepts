# generics

Ejemplos de todos los aspectos de generics en Go 1.18+ que aparecen en entrevistas técnicas.

## Ejecutar

```bash
go run .
```

## Estructura

| Archivo | Contenido |
|---------|-----------|
| `constraints.go` | `any`, `comparable`, `~T`, uniones, constraints con métodos |
| `functions.go` | `Map`, `Filter`, `Reduce`, `Contains`, `Keys/Values`, `Must` |
| `datastructs.go` | `Stack[T]`, `Queue[T]`, `Set[T comparable]` |
| `patterns.go` | Inferencia, múltiples parámetros, zero value, `Result[T]`, limitaciones |

---

## Tabla de constraints

| Constraint | Significado | Ejemplo |
|-----------|-------------|---------|
| `any` | Cualquier tipo (`interface{}`) | Contenedores genéricos |
| `comparable` | Soporta `==` y `!=` | Claves de map, búsqueda |
| `~T` | Underlying type es T | Tipos definidos sobre primitivos |
| Unión `A \| B` | T debe ser uno de los listados | `Min`, `Max`, aritmética |
| Interfaz con método | T debe implementar el método | `Stringer`, `io.Reader` |

---

## Constraints

```go
// any — sin restricción de operaciones
func Identity[T any](v T) T { return v }

// comparable — permite == y !=
func Equal[T comparable](a, b T) bool { return a == b }

// Ordered — union de todos los tipos ordenados
type Ordered interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
        ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
        ~float32 | ~float64 | ~string
}
func Min[T Ordered](a, b T) T { if a < b { return a }; return b }

// ~T — "cualquier tipo cuyo underlying type sea T"
// Sin ~: solo acepta el tipo nombrado exacto.
// Con ~: acepta también tipos definidos como `type Celsius float64`.
type Temperature interface{ ~float64 }
func AbsDiff[T Temperature](a, b T) T { ... }

// Método como constraint
type Stringer interface{ String() string }
func PrintAll[T Stringer](items []T) { ... }

// Union de tipos numéricos
type Number interface {
    ~int | ~int32 | ~int64 | ~float32 | ~float64
}
func Sum[T Number](s []T) T { ... }
```

---

## Funciones genéricas

```go
// Map — transforma cada elemento (T puede diferir de U)
func Map[T, U any](s []T, f func(T) U) []U

// Filter — retiene elementos según predicado
func Filter[T any](s []T, f func(T) bool) []T

// Reduce — acumula de izquierda a derecha
func Reduce[T, U any](s []T, init U, f func(U, T) U) U

// Contains — búsqueda lineal (requiere comparable)
func Contains[T comparable](s []T, v T) bool

// Keys / Values — extraen keys o values de un map
func Keys[K comparable, V any](m map[K]V) []K
func Values[K comparable, V any](m map[K]V) []V

// Must — desenvuelve (value, error), panic si err != nil
func Must[T any](v T, err error) T
```

Uso:
```go
nums := []int{1, 2, 3, 4, 5}

squares := Map(nums, func(n int) string { return fmt.Sprintf("%d²=%d", n, n*n) })
evens   := Filter(nums, func(n int) bool { return n%2 == 0 })
sum     := Reduce(nums, 0, func(acc, n int) int { return acc + n })

Contains(nums, 3)  // true
Contains(nums, 9)  // false
```

---

## Estructuras de datos

```go
// Stack[T] — LIFO
type Stack[T any] struct{ items []T }
func (s *Stack[T]) Push(v T)
func (s *Stack[T]) Pop() (T, bool)   // (zero, false) si vacío
func (s *Stack[T]) Peek() (T, bool)
func (s *Stack[T]) Len() int

// Queue[T] — FIFO
type Queue[T any] struct{ items []T }
func (q *Queue[T]) Enqueue(v T)
func (q *Queue[T]) Dequeue() (T, bool)

// Set[T comparable] — colección de valores únicos
type Set[T comparable] struct{ m map[T]struct{} }
func NewSet[T comparable](vals ...T) *Set[T]
func (s *Set[T]) Add(v T)
func (s *Set[T]) Contains(v T) bool
func (s *Set[T]) Union(other *Set[T]) *Set[T]
func (s *Set[T]) Intersection(other *Set[T]) *Set[T]
func (s *Set[T]) Difference(other *Set[T]) *Set[T]
```

---

## Patterns

### Inferencia de tipos
```go
// Go infiere el tipo cuando puede:
Double(21)       // Double[int]
Double(3.14)     // Double[float64]

// Sintaxis explícita siempre disponible:
Double[int64](7)
```

### Múltiples parámetros de tipo
```go
type Pair[A, B any] struct { First A; Second B }

p := NewPair("edad", 30)     // Pair[string, int]
p := NewPair(true, []int{1}) // Pair[bool, []int]
```

### Zero value de un parámetro de tipo
```go
func First[T any](s []T) (T, bool) {
    if len(s) == 0 {
        var zero T   // 0, "", false, nil — según T
        return zero, false
    }
    return s[0], true
}
```

### Result[T] — encapsular valor + error
```go
type Result[T any] struct { Value T; Err error }

func Ok[T any](v T) Result[T]      { return Result[T]{Value: v} }
func Err[T any](e error) Result[T] { return Result[T]{Err: e} }

func (r Result[T]) IsOk() bool { return r.Err == nil }
func (r Result[T]) Unwrap() T  { /* panic si Err != nil */ }

// Útil en pipelines y canales de resultados asíncronos
results := make(chan Result[User])
```

### GroupBy — comparable como clave de map
```go
func GroupBy[T any, K comparable](s []T, key func(T) K) map[K][]T

byLen := GroupBy(words, func(s string) int { return len(s) })
// map[1:[c] 2:[go] 3:[zig] 4:[rust java]]
```

### Type switch via `any(v)`
```go
// No se puede hacer v.(type) directamente cuando T no es interface.
// Workaround: convertir a any primero.
func Describe[T any](v T) string {
    switch x := any(v).(type) {
    case int:    return fmt.Sprintf("int(%d)", x)
    case string: return fmt.Sprintf("string(%q)", x)
    default:     return fmt.Sprintf("%T(%v)", x, x)
    }
}
```

---

## Limitaciones clave (preguntas de entrevista)

### 1. No se pueden definir métodos genéricos en tipos no genéricos
```go
// INVÁLIDO — los métodos no pueden introducir nuevos parámetros de tipo:
type MySlice []int
func (s MySlice) Map[U any](f func(int) U) []U { ... }  // ← ERROR

// Solución A — función top-level:
Map(mySlice, f)

// Solución B — hacer el tipo genérico:
type MySlice[T any] []T
func (s MySlice[T]) String() string { ... }  // ✓ usa el T del tipo
```

### 2. No se puede hacer type assertion directa sobre T
```go
// INVÁLIDO cuando T está restringido a una unión:
func f[T Number](v T) {
    _ = v.(int)  // ← ERROR: T no es una interfaz en este contexto
}

// Workaround:
_ = any(v).(int)  // runtime assertion, pierde seguridad estática
```

### 3. Una union constraint no puede usarse como tipo de variable
```go
type Number interface{ ~int | ~float64 }

var x Number  // ← ERROR: Number contiene elementos no-interface
              // Solo puede usarse como constraint de tipo parámetro
```

### 4. No hay specialization (a diferencia de C++ templates)
Go genera una sola implementación por constraint, no una por tipo concreto.
No hay optimizaciones específicas por tipo como en C++.

---

## Reglas clave

1. **`any` vs `comparable`**: usa `comparable` cuando necesites `==`, `!=`, o usar T como clave de map.
2. **`~T` es esencial** para que los tipos definidos sobre primitivos satisfagan constraints numéricas.
3. **Zero value** con `var zero T` — siempre válido para cualquier T.
4. **La inferencia funciona** cuando los tipos se pueden deducir de los argumentos; si no, especifica explícitamente.
5. **No hay métodos genéricos** — usa funciones top-level o tipos genéricos.
6. **`any(v).(type)` como escape hatch** — válido pero pierde garantías estáticas en compile time.
