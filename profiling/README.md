# profiling

Herramientas de profiling y benchmarking de Go que aparecen en entrevistas técnicas.

## Ejecutar

```bash
go run .                          # genera cpu.prof, mem.prof, goroutine.prof, block.prof, mutex.prof
go test -bench=. -benchmem        # corre los benchmarks de bench_test.go
```

## Estructura

| Archivo | Contenido |
|---------|-----------|
| `cpu.go` | `pprof.StartCPUProfile` / `StopCPUProfile`, workload, `go tool pprof` |
| `memory.go` | `pprof.WriteHeapProfile`, comparación de patrones de allocación |
| `profiles.go` | `pprof.Lookup` — goroutine, block, mutex; `SetBlockProfileRate`, `SetMutexProfileFraction` |
| `http_pprof.go` | `import _ "net/http/pprof"` — endpoints siempre activos para producción |
| `bench_test.go` | `testing.B` — `b.N`, `b.ResetTimer`, `b.ReportAllocs`, `b.RunParallel`, sub-benchmarks |

---

## CPU profiling

```go
import "runtime/pprof"

f, _ := os.Create("cpu.prof")
defer f.Close()

pprof.StartCPUProfile(f)  // muestrea el PC de cada goroutine ~100 veces/segundo
// ... código a perfilar ...
pprof.StopCPUProfile()    // flush + cierra — SIEMPRE llamar antes de leer el archivo
```

```bash
go tool pprof cpu.prof
(pprof) top             # funciones más costosas (tiempo acumulado incluyendo callees)
(pprof) top -flat       # tiempo propio (solo el frame, sin callees)
(pprof) list sortWork   # fuente anotada con línea a línea
(pprof) web             # flame graph en el browser (requiere graphviz)
(pprof) png > out.png   # exportar como imagen
```

---

## Memory profiling

```go
import "runtime/pprof"

runtime.GC()               // GC antes → inuse_space muestra solo objetos vivos
pprof.WriteHeapProfile(f)  // snapshot del heap
```

```bash
go tool pprof mem.prof
(pprof) top -inuse_space    # bytes actualmente en uso → diagnosticar memory leaks
(pprof) top -alloc_space    # bytes totales alguna vez allocados → presión GC
(pprof) top -alloc_objects  # conteo de allocaciones totales
```

### Tipos de datos en el perfil de heap

| Tipo | Qué mide | Para qué sirve |
|------|----------|----------------|
| `inuse_space` | Bytes vivos ahora | Detectar leaks |
| `inuse_objects` | Objetos vivos ahora | Detectar leaks |
| `alloc_space` | Total alguna vez allocado | Presión sobre el GC |
| `alloc_objects` | Total de allocaciones | Presión sobre el GC |

### Reducir allocaciones

```
string +=       (200 iters) → 200  allocs  (crea nueva string en cada iteración)
strings.Builder (200 iters) →   7  allocs  (crece el buffer internamente)
Builder + Grow  (200 iters) →   2  allocs  (una sola allocación inicial)

append (sin cap, 1000 ints) →  13  allocs  (múltiples reallocs ~2× cada vez)
make([]int, 0, 1000)        →   2  allocs  (una sola allocación)
```

---

## Named profiles — pprof.Lookup

```go
// Listar todos los perfiles disponibles
for _, p := range pprof.Profiles() {
    fmt.Println(p.Name(), p.Count())
}

// Escribir un perfil a un archivo
p := pprof.Lookup("goroutine")
p.WriteTo(f, 0)   // debug=0: binario (para go tool pprof)
p.WriteTo(w, 1)   // debug=1: texto legible
p.WriteTo(w, 2)   // debug=2: texto verbose con labels
```

| Perfil | Qué captura | Requiere |
|--------|-------------|----------|
| `goroutine` | Stack traces de todas las goroutines actuales | — |
| `heap` | Allocaciones vivas en el heap | — |
| `allocs` | Todas las allocaciones (incluyendo liberadas), muestreadas | — |
| `block` | Goroutines bloqueadas en primitivos de sync | `runtime.SetBlockProfileRate(1)` |
| `mutex` | Holders de mutexes con contención | `runtime.SetMutexProfileFraction(1)` |
| `threadcreate` | Creación de OS threads | — |

```go
// Block y mutex están OFF por defecto (para evitar overhead)
// Activar ANTES del código que quieres perfilar:
runtime.SetBlockProfileRate(1)       // 1 = capturar cada evento
runtime.SetMutexProfileFraction(1)   // 1 = capturar cada contención
```

---

## HTTP pprof — endpoints de producción

```go
import _ "net/http/pprof"  // blank import registra rutas en http.DefaultServeMux
```

```go
// Patrón típico de producción:
go func() {
    // Puerto separado, solo localhost — NUNCA exponer públicamente
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

| Endpoint | Uso |
|----------|-----|
| `/debug/pprof/` | Índice con todos los perfiles |
| `/debug/pprof/profile?seconds=30` | Perfil CPU de 30s (bloquea hasta completar) |
| `/debug/pprof/heap` | Heap actual |
| `/debug/pprof/goroutine?debug=2` | Todas las goroutines con stacks completos |
| `/debug/pprof/block` | Goroutines bloqueadas en sync |
| `/debug/pprof/mutex` | Contención de mutexes |
| `/debug/pprof/trace?seconds=5` | Trace de ejecución de 5s |

```bash
# Descargar y analizar directamente:
go tool pprof http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Abrir UI web con flame graph:
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

---

## Benchmarks — testing.B

```go
// Estructura básica — b.N es ajustado por el runner hasta ~1 segundo
func BenchmarkFoo(b *testing.B) {
    for range b.N {   // Go 1.22: range b.N en lugar de for i := 0; i < b.N; i++
        // código a medir
    }
}
```

### b.ResetTimer — excluir setup del tiempo medido

```go
func BenchmarkWithSetup(b *testing.B) {
    data := loadLargeDataset()  // setup costoso — NO debe contar
    b.ResetTimer()              // ← reset aquí; lo anterior queda excluido

    for range b.N {
        process(data)           // solo esto se mide
    }
}
```

### b.StopTimer / b.StartTimer — setup por iteración

```go
func BenchmarkWithPerIterSetup(b *testing.B) {
    for range b.N {
        b.StopTimer()
        data := buildInput()   // setup por iteración, no medido
        b.StartTimer()
        process(data)          // solo esto se mide
    }
}
```

### b.ReportAllocs y -benchmem

```go
func BenchmarkSprintf(b *testing.B) {
    b.ReportAllocs()  // equivalente a -benchmem solo para este benchmark
    for range b.N {
        _ = fmt.Sprintf("key=%d", 42)
    }
}
```

```bash
# Salida con -benchmem:
# BenchmarkStringConcat-8    234300    6123 ns/op    5680 B/op    100 allocs/op
#                                      ↑ tiempo      ↑ bytes      ↑ allocs
#                                        por op        por op       por op
```

### b.RunParallel — carga concurrente

```go
func BenchmarkParallel(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            // cada goroutine ejecuta esto de forma independiente
        }
    })
}
```

### Sub-benchmarks

```go
func BenchmarkSlice(b *testing.B) {
    b.Run("append_no_cap", func(b *testing.B) { ... })
    b.Run("make_with_cap", func(b *testing.B) { ... })
    b.Run("make_with_len", func(b *testing.B) { ... })
}
```

```bash
go test -bench=BenchmarkSlice/.* -benchmem   # solo sub-benchmarks de Slice
```

### Comparar dos versiones con benchstat

```bash
go test -bench=. -count=10 > old.txt
# hacer cambios
go test -bench=. -count=10 > new.txt
benchstat old.txt new.txt
# → muestra delta estadístico con p-value
```

### Flags de benchmarking

```bash
go test -bench=.                    # todos los benchmarks
go test -bench=BenchmarkFoo         # benchmark específico (regex)
go test -bench=. -benchmem          # + bytes/op y allocs/op
go test -bench=. -count=5           # repetir 5 veces (reduce ruido)
go test -bench=. -benchtime=5s      # correr durante 5s en lugar de 1s
go test -bench=. -cpu=1,2,4,8       # probar con distintos GOMAXPROCS
go test -bench=. -cpuprofile=cpu.prof    # + perfil CPU
go test -bench=. -memprofile=mem.prof    # + perfil memoria
```

---

## Reglas clave

1. **CPU profile** muestrea ~100 Hz — la función que más aparece consume más CPU.
2. **`-flat` vs cumulative**: flat = tiempo propio; cumulative = incluye callees. Empezar con flat para encontrar el hotspot real.
3. **`runtime.GC()` antes de `WriteHeapProfile`** → `inuse_space` muestra solo objetos vivos.
4. **Block y mutex están OFF** (`rate=0`) — activar solo cuando vayas a perfilar, para evitar overhead.
5. **`b.ResetTimer()` siempre** cuando el benchmark tiene setup no trivial.
6. **`-count=5` o más** para benchmarks estables — una sola corrida puede tener ruido.
7. **`import _ "net/http/pprof"`** — NUNCA en un puerto público; solo `localhost` o detrás de auth.
8. **`go tool pprof -http=:8080`** abre una UI web con flame graphs sin necesidad de graphviz.
