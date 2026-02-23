# Stack vs Heap en Go

## Conceptos clave

### Stack (pila)

Cada goroutine tiene su propio stack. Es una región de memoria contigua que crece
y se contrae automáticamente con las llamadas a funciones. Cuando una función retorna,
su _frame_ desaparece y toda la memoria que reservó se libera **instantáneamente**,
sin intervención del garbage collector.

- Asignación O(1): solo desplazar un puntero.
- Sin fragmentación.
- En Go, el stack empieza pequeño (~2 KB) y crece dinámicamente si se necesita más espacio.

### Heap (montón)

Región de memoria compartida entre todas las goroutines. El runtime asigna aquí los
valores cuya vida útil supera al frame donde fueron creados. El **garbage collector**
es el responsable de liberar esta memoria, lo que introduce latencia y presión de GC.

---

## Escape Analysis: ¿quién decide dónde va cada variable?

El compilador de Go ejecuta un análisis estático llamado **escape analysis** en tiempo
de compilación. Si puede probar que una variable no _escapa_ del frame donde fue creada,
la deja en el stack. Si no puede garantizarlo, la mueve al heap.

Las causas más comunes por las que una variable **escapa al heap** son:

| Causa | Ejemplo |
|---|---|
| Retornar su dirección | `return &x` |
| Ser capturada por un closure | `func() { x++ }` |
| Ser asignada a una interfaz | `var a any = x` |
| Ser demasiado grande para el stack | arrays gigantes |
| Tamaño desconocido en compilación | `make([]int, n)` con n variable |

---

## El programa

```
stack-vs-heap/
├── go.mod
├── main.go        — cuatro ejemplos comentados de stack y heap
└── alloc_test.go  — benchmarks que miden el impacto en rendimiento
```

### Casos ilustrados en `main.go`

| Función | Dónde vive | Por qué |
|---|---|---|
| `returnValue()` | **stack** | se devuelve una copia; nadie guarda la dirección |
| `sumArray()` | **stack** | array de tamaño fijo conocido en compilación |
| `returnPointer()` | **heap** | `&x` escapa: su vida supera al frame |
| `closureCapture()` | **heap** | `x` es capturada por el closure retornado |
| `interfaceBox()` | **heap** | el valor concreto se _boxea_ para satisfacer `any` |
| `makeSlice()` | **heap** | tamaño variable; slice puede crecer |

---

## Cómo correrlo

```bash
go run main.go
```

## Ver el escape analysis del compilador

```bash
go build -gcflags="-m" .
```

La flag `-m` imprime las decisiones del compilador. Busca líneas como:

```
./main.go:42:2: x escapes to heap
./main.go:20:2: x does not escape
```

Para más detalle usa `-m=2`:

```bash
go build -gcflags="-m=2" .
```

## Benchmarks

```bash
go test -bench=. -benchmem .
```

`-benchmem` muestra columnas extra:

| Columna | Significado |
|---|---|
| `ns/op` | nanosegundos por operación |
| `B/op` | bytes asignados en heap por operación |
| `allocs/op` | número de asignaciones en heap por operación |

Ejemplo de salida esperada:

```
BenchmarkReturnValue-8     1000000000   0.3 ns/op      0 B/op   0 allocs/op
BenchmarkReturnPointer-8   200000000    6.0 ns/op      8 B/op   1 allocs/op
BenchmarkMakeSlice-8        50000000   28.0 ns/op    512 B/op   1 allocs/op
```

`ReturnValue` no hace ninguna asignación en heap (`0 B/op`, `0 allocs/op`).
`ReturnPointer` asigna 8 bytes (un `int` de 64 bits) cada vez que se llama.

---

## Regla práctica

> **Si no necesitas compartir la dirección de una variable, devuelve el valor.**
> El compilador es capaz de optimizar lo demás.

No hace falta evitar manualmente el heap en todo momento — el GC de Go es eficiente.
Pero entender escape analysis ayuda a escribir código de bajo nivel con menor latencia
y menos presión sobre el recolector.
