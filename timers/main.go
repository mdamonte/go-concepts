package main

import "fmt"

func main() {
	section("time.NewTimer — disparo único")
	demoTimer()

	section("time.Timer.Stop — cancelar antes de disparar")
	demoTimerStop()

	section("time.Timer.Reset — reusar el timer")
	demoTimerReset()

	section("time.AfterFunc — ejecutar función tras un delay")
	demoAfterFunc()

	section("time.NewTicker — disparo periódico")
	demoTicker()

	section("time.Ticker.Reset — cambiar el intervalo en caliente")
	demoTickerReset()

	section("time.After — shortcut de un solo disparo")
	demoTimeAfter()

	section("time.Tick — shortcut periódico (solo en programas de larga vida)")
	demoTimeTick()

	section("Patrón: timeout en select")
	demoTimeout()

	section("Patrón: debounce")
	demoDebounce()

	section("Patrón: rate limiter")
	demoRateLimit()

	section("Patrón: retry con exponential backoff")
	demoRetryBackoff()

	section("Patrón: tarea periódica cancelable")
	demoPeriodic()
}

func section(title string) {
	fmt.Printf("\n━━━ %s ━━━\n", title)
}
