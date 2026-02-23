package main

import (
	"context"
	"fmt"
)

// demoBackgroundTODO shows the two root (parentless) context constructors.
func demoBackgroundTODO() {
	// context.Background() is the root of every context tree.
	// Use it in main(), tests, and top-level server handlers — anywhere
	// you are starting a fresh operation with no incoming context.
	bg := context.Background()
	fmt.Println("Background:", bg)

	// context.TODO() is semantically identical to Background() but signals
	// intent: "I know a context belongs here; I haven't wired it up yet."
	// Treat it as a TODO comment for context plumbing. Static analysis tools
	// (e.g. staticcheck) can flag TODO contexts in production paths.
	todo := context.TODO()
	fmt.Println("TODO:      ", todo)
}
