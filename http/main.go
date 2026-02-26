package main

import "fmt"

// Each demo covers one aspect of net/http that appears in technical interviews.
//
// Run:
//
//	go run .
func main() {
	section("Server — Handler, HandlerFunc, ServeMux, Go 1.22 routing")
	demoServer()

	section("Middleware — Logger, Auth, Recovery, Chain")
	demoMiddleware()

	section("Client — custom client, timeout, status codes, context cancellation")
	demoClient()

	section("Graceful shutdown — drain in-flight requests before stopping")
	demoShutdown()

	section("httptest — NewRecorder (unit) vs NewServer (integration)")
	demoRecorder()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
