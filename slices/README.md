# slices

Internals y gotchas de slices en Go. La mayoría de las "trick questions" de entrevista
sobre slices vienen de no entender que un slice es una vista sobre un array compartido.

## Ejecutar

```bash
go run .
```

## Estructura

| Archivo | Contenido |
|---------|-----------|
| `internals.go` | Header `{ptr, len, cap}`, backing array compartido, pass-by-value |
| `append.go` | Crecimiento, in-place vs realloc, gotcha del subslice, `s[low:high:max]` |
| `operations.go` | `copy`, delete, insert, filter in-place, reverse, dedup, stdlib `slices` |
| `nil.go` | nil vs empty, JSON, `reflect.DeepEqual`, `==` sólo contra nil |

---

## Internals — `{ptr, len, cap}`

Un slice es una struct de 3 palabras (24 bytes en 64-bit):

```
+──────────+─────+─────+
│  ptr     │ len │ cap │   ← header (vive en el stack)
+──────────+─────+─────+
     │
     ▼
[0][1][2][3][4][5]         ← backing array (vive en el heap)
```

- **ptr** → primer elemento visible a través de este slice
- **len** → elementos accesibles con `s[i]`
- **cap** → elementos desde ptr hasta el final del backing array

```go
// Dos slices comparten el mismo backing array
a := []int{1, 2, 3, 4, 5}
b := a[1:4]   // b = [2 3 4], comparte el array de a
              // cap(b) = 4, no 3 — llega hasta el final de a

b[0] = 99     // escribe en a[1] — ambos lo ven
fmt.Println(a) // [1 99 3 4 5]
fmt.Println(b) // [99 3 4]
```

### Pass-by-value: el header se copia, el array no

```go
func modifyElement(s []int, i, v int) {
    s[i] = v   // ✓ escribe en el backing array compartido — caller lo ve
}

func appendInside(s []int) {
    s = append(s, 999) // ✗ modifica una copia del header — caller NO lo ve
}

// Para que append sea visible: retornar el slice
func appendAndReturn(s []int, v int) []int {
    return append(s, v)
}
s = appendAndReturn(s, 999) // caller debe reasignar
```

---

## Append — cuándo copia, cuándo no

```
len < cap  → escribe en el backing array existente (sin allocación)
len == cap → aloca nuevo array más grande, copia todo, luego escribe
```

**Factor de crecimiento**: ~2× para slices pequeños, ~1.25× a partir de ~256 elementos.
El valor exacto está en `runtime/slice.go` y puede cambiar entre versiones.

```go
// Pre-alocar cuando el tamaño es conocido (evita realocaciones)
result := make([]int, 0, n)
for _, v := range input {
    result = append(result, transform(v))
}
```

### El gotcha más común: append a un subslice

```go
orig := []int{1, 2, 3, 4, 5}
sub  := orig[1:3]   // sub = [2 3]; cap(sub) = 4  ← llega hasta orig[4]

sub = append(sub, 99)  // cap=4 > len=2 → escribe en orig[3], no aloca
fmt.Println(orig)       // [1 2 3 99 5]  ← orig[3] fue sobreescrito silenciosamente!
```

### Fix: slice de 3 índices `s[low:high:max]`

```go
// cap(s[low:high:max]) = max - low
safe := orig[1:3:3]   // cap = 3-1 = 2 = len → append DEBE alocar nuevo array

safe = append(safe, 99)
fmt.Println(orig)  // [1 2 3 4 5]  ← intacto
fmt.Println(safe)  // [2 3 99]     ← nuevo backing array
```

---

## Operaciones

### copy

```go
// copy(dst, src) copia min(len(dst), len(src)) elementos
// Retorna el número de elementos copiados
// Seguro con slices que se solapan (mismo backing array)

n := copy(dst, src)

// Desplazar izquierda usando overlapping copy
copy(s[i:], s[i+1:])
s = s[:len(s)-1]
```

### Delete

```go
// O(n) — preserva orden
s = append(s[:i], s[i+1:]...)

// O(1) — cambia orden (swap con el último)
s[i] = s[len(s)-1]
s = s[:len(s)-1]
```

### Insert

```go
s = append(s, 0)         // hacer espacio (puede alocar)
copy(s[i+1:], s[i:])     // desplazar derecha (copy maneja overlap)
s[i] = v
```

### Filter in-place (zero allocation)

```go
// Reutiliza el backing array: escribe los elementos seleccionados al frente.
// No usar original después — tiene datos obsoletos más allá de result.
result := s[:0]
for _, v := range s {
    if keep(v) {
        result = append(result, v)
    }
}
```

### stdlib `slices` package (Go 1.21+)

```go
import "slices"

slices.Sort(s)
slices.Contains(s, v)
slices.Index(s, v)          // -1 si no existe
slices.Delete(s, i, j)      // elimina s[i:j]
slices.Compact(sorted)      // dedup de slice ordenado
slices.Equal(a, b)          // comparación elemento a elemento
slices.Reverse(s)           // in-place
```

---

## Nil vs empty

```go
var s []int        // nil slice  : s == nil → true,  len=0, cap=0
s := []int{}       // empty slice: s == nil → false, len=0, cap=0
s := make([]int,0) // empty slice: s == nil → false, len=0, cap=0
```

Para `range`, `len`, `cap`, `append`: **idénticos**.

### JSON — diferencia crítica

```go
type Resp struct {
    Items []int `json:"items"`
}
json.Marshal(Resp{Items: nil})      // → {"items":null}
json.Marshal(Resp{Items: []int{}})  // → {"items":[]}
```

### reflect.DeepEqual

```go
reflect.DeepEqual([]int(nil), []int{})   // false — nil ≠ empty
reflect.DeepEqual([]int{}, []int{})      // true
```

### == sólo funciona contra nil

```go
s := []int{1, 2, 3}
s == nil           // ✓ único uso válido de == con slices
// s == []int{1,2,3}  ← compile error: invalid operation

// Para comparar dos slices: slices.Equal o reflect.DeepEqual
slices.Equal(s, []int{1,2,3}) // true (Go 1.21+)
```

### nil map vs nil slice

```go
append(nilSlice, 1)     // ✓ seguro — retorna nuevo slice
nilMap["key"] = "val"   // ✗ panic: assignment to entry in nil map
```

---

## Reglas clave

1. **Un slice es un header** `{ptr, len, cap}` — pasarlo a una función copia el header, no el array.
2. **Modificar elementos** a través de un slice es visible al caller; **append no lo es** (a menos que se retorne).
3. **`cap(sub)` llega hasta el final del backing array** — append a un subslice puede sobreescribir datos del original silenciosamente.
4. **`s[low:high:max]`** fuerza `cap = max-low`, evitando que append comparta memoria con el original.
5. **nil slice es preferible** a `[]int{}` para "sin resultados" — excepto cuando necesitas JSON `[]`.
6. **`==` sólo se puede usar contra nil** — para comparar dos slices usa `slices.Equal` o `reflect.DeepEqual`.
