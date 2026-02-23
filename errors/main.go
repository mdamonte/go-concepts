package main

import "fmt"

func main() {
	section("Errores centinela y errors.Is")
	demoSentinel()

	section("Tipos de error custom y errors.As")
	demoCustomType()

	section("Wrapping con %%w y cadena de Unwrap")
	demoWrapping()

	section("errors.Is con método Is() personalizado")
	demoCustomIs()

	section("errors.As con método As() personalizado")
	demoCustomAs()

	section("errors.Join — múltiples errores (Go 1.20+)")
	demoJoin()

	section("Patrón: error de operación con contexto")
	demoOpError()

	section("Patrón: errores opacos vs exportados")
	demoOpaque()

	section("Patrón: panic vs error")
	demoPanicVsError()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
