# Interfaces

Sample code illustrating how to use interfaces in Go.

## Concepts covered

- **Interface declaration** — defining a contract with method signatures (`Shape`, `Stringer`)
- **Interface composition** — embedding multiple interfaces into one (`Describer`)
- **Implicit satisfaction** — types satisfy interfaces automatically, no `implements` keyword required
- **Polymorphism** — functions that accept an interface work with any conforming type
- **Restricted interfaces** — accepting only types that satisfy a composed interface
- **Type assertion** — extracting the concrete type from an interface value (`s.(Circle)`)
- **Type switch** — branching on the runtime type of an interface value
- **Slice of interfaces** — storing mixed types together and aggregating over them
- **Nil interface** — the zero value of an interface is `nil`

## Run

```bash
go run main.go
```
