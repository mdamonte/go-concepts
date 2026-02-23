package main

import "fmt"

func main() {
	section("context.Background / context.TODO")
	demoBackgroundTODO()

	section("context.WithCancel")
	demoCancel()

	section("context.WithTimeout")
	demoTimeout()

	section("context.WithDeadline")
	demoDeadline()

	section("context.WithValue")
	demoValue()

	section("context.WithCancelCause / WithTimeoutCause / WithDeadlineCause")
	demoCause()

	section("Propagation: parent cancels all children")
	demoPropagation()

	section("HTTP server & client")
	demoHTTP()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
