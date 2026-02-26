# defer

`defer` is Go's mechanism for scheduling a function call to run just before the surrounding function returns. It is the #1 source of "what does this print?" trick questions in technical interviews.

```bash
go run .
```

---

## Files

| File | Topic |
|------|-------|
| `basics.go` | LIFO order, argument evaluation, closure capture |
| `returns.go` | Named vs anonymous returns, error wrapping, transactions |
| `loops.go` | Resource-leak gotcha and three fixes |
| `panic.go` | Panic rules, recover(), safeDiv, safeGo |

---

## Rule 1 — LIFO: last deferred runs first

`defer` is a stack. Each `defer` pushes onto it; on return, the stack is popped.

```go
defer fmt.Println("1") // runs last
defer fmt.Println("2")
defer fmt.Println("3") // runs first
// Output: 3, 2, 1
```

---

## Rule 2 — Arguments are evaluated NOW

At the `defer` statement, all arguments are evaluated and stored immediately.
Changes to those variables afterward have no effect.

```go
x := 0
defer fmt.Println(x) // captures x=0 right here
x = 100
// defer prints: 0
```

**Contrast with closures**, which read the variable when they execute:

```go
x := 0
defer func() { fmt.Println(x) }() // reads x at execution time
x = 100
// defer prints: 100
```

### Classic trick question

```go
for i := range 3 {
    defer fmt.Printf("%d\n", i) // arg eval: stores 0, 1, 2; LIFO → prints 2, 1, 0
}
```

```go
for i := 0; i < 3; i++ {
    defer func() { fmt.Printf("%d\n", i) }() // closure captures &i; after loop i=3 → prints 3, 3, 3
}
```

Fix with shadowing (pre-Go 1.22 idiom):

```go
for i := 0; i < 3; i++ {
    i := i // new variable per iteration
    defer func() { fmt.Printf("%d\n", i) }() // prints 2, 1, 0
}
```

---

## Named returns — defer can change what the caller sees

`return X` is actually three steps:
1. Assign X to the return slot (named → uses the named var; anonymous → copies X)
2. Run deferred functions
3. Return to caller

```go
func anonymousReturn() int {
    x := 5
    defer func() { x *= 2 }() // modifies local x — return slot already holds 5
    return x                   // caller gets: 5
}

func namedReturn() (result int) {
    defer func() { result *= 2 }() // modifies the actual return slot
    result = 5
    return     // caller gets: 10
}

func namedReturnExplicit() (result int) {
    defer func() { result *= 2 }()
    return 5   // assigns result=5, THEN defer runs → caller gets: 10
}
```

### Practical: wrap every error with context

```go
func findRecord(id int) (err error) {
    defer func() {
        if err != nil {
            err = fmt.Errorf("findRecord(%d): %w", id, err)
        }
    }()
    // every early return gets wrapped automatically
}
```

### Practical: transaction rollback / commit

```go
func withTx(fn func(*tx) error) (err error) {
    t, err := beginTx()
    if err != nil { return err }
    defer func() {
        if err != nil {
            t.Rollback()
        } else {
            err = t.Commit()
        }
    }()
    return fn(t)
}
```

---

## defer in loops — the resource-leak gotcha

`defer` does NOT run at the end of each loop iteration.
It runs when the **enclosing function** returns.

```go
// WRONG — all N resources stay open until processWrong returns
func processWrong(n int) {
    for i := range n {
        r := openRes(i)
        defer r.Close() // fires at function exit, NOT loop iteration
    }
}
```

### Fix 1: extract to a helper function (most idiomatic)

```go
func processFixed(n int) {
    for i := range n { processOne(i) }
}

func processOne(id int) {
    r := openRes(id)
    defer r.Close() // fires when processOne returns → end of each iteration
}
```

### Fix 2: immediately-invoked function literal

```go
for i := range n {
    func() {
        r := openRes(i)
        defer r.Close() // fires when closure returns
    }()
}
```

### Fix 3: explicit close (no defer)

```go
for i := range n {
    r := openRes(i)
    // ... use r ...
    r.Close() // explicit — must be called on every exit path
}
```

Use Fix 3 only when the body is short and has no early returns.

---

## Panic & recover — four rules

### Rule 1: defer runs during panic unwinding

Deferred functions still execute before the goroutine dies.

```go
func f() (result string) {
    defer func() {
        result = "defer ran" // this executes even though f panicked
    }()
    panic("oh no")
}
```

### Rule 2: recover() stops panic ONLY from a direct defer

```go
// WORKS
defer func() {
    if r := recover(); r != nil { /* panic stopped */ }
}()

// DOES NOT WORK — recover() inside a helper returns nil
defer recoverHelper()  // helper's recover() sees nil; panic continues
```

### Rule 3: execution resumes after the deferred func, not after panic()

```go
func f() (result string) {
    defer func() {
        if r := recover(); r != nil {
            result = "recovered"
            // execution continues HERE, then f returns
            // it does NOT jump back to where panic() was called
        }
    }()
    panic("oops")
    // code here is unreachable
}
```

### Rule 4: os.Exit() bypasses defer

```go
defer file.Close()
log.Fatal(err)   // calls os.Exit(1) — defer DOES NOT run, file is NOT closed
```

Never use `log.Fatal`/`log.Fatalf`/`log.Fatalln` when you have important cleanup in defers.
Use `log.Print` + `return` (or `os.Exit` at the top level only) instead.

---

## Practical patterns

### Re-panic for unexpected panics

```go
func safeDiv(a, b int) (result int, err error) {
    defer func() {
        if r := recover(); r != nil {
            switch e := r.(type) {
            case runtime.Error:
                panic(e) // re-panic: programming error, do not swallow
            default:
                err = fmt.Errorf("recovered: %v", r)
            }
        }
    }()
    if b == 0 { panic("division by zero") }
    return a / b, nil
}
```

### safeGo — goroutine with built-in recovery

A goroutine that panics crashes the **whole program**. There is no way to recover from another goroutine — each goroutine must recover itself.

```go
func safeGo(fn func()) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                fmt.Printf("safeGo recovered: %v\n", r)
            }
        }()
        fn()
    }()
}
```

---

## Interview cheat-sheet

| Question | Answer |
|----------|--------|
| When does defer run? | When the enclosing function returns (normally or via panic) |
| Order of multiple defers? | LIFO — last registered runs first |
| When are defer arguments evaluated? | At the `defer` statement, not when it executes |
| Can defer in a loop close each iteration's resource? | No — use a helper function or immediately-invoked closure |
| Can defer modify a named return? | Yes — defer sees the named return variable |
| Can defer modify an anonymous return? | No — the return slot already holds the copy |
| Does recover() work from a helper? | No — must be called directly inside a deferred function |
| Does os.Exit run defers? | No — defers are skipped entirely |
| Can you recover from another goroutine's panic? | No — each goroutine must recover itself |
