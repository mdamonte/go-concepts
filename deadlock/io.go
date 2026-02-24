package main

import (
	"fmt"
	"net"
	"time"
)

// demoIOWait shows goroutine state [IO wait]:
// the goroutine is blocked inside the OS network poller waiting for data
// that never arrives on a real TCP socket.
//
// Important: net.Pipe() does NOT produce [IO wait] — it is implemented with
// Go-level channels and shows as [select]. A real OS socket is required.
//
// Goroutine dump entry:
//
//	goroutine N [IO wait]:
//	internal/poll.runtime_pollWait(0x..., 0x72)
//	internal/poll.(*pollDesc).waitRead(...)
//	internal/poll.(*netFD).Read(...)
//	net.(*conn).Read(...)
//	main.demoIOWait.func3()
//
// You also see [IO wait] on goroutines blocked in:
//   - os.File.Read on a non-blocking fd (pipe, tty, named pipe)
//   - net.Conn.Write when the send buffer is full
func demoIOWait() {
	// Start a TCP server that accepts a connection but never writes to it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  error setting up listener:", err)
		return
	}

	serverStop := make(chan struct{})
	serverDone := make(chan struct{})

	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return // listener was closed before accepting
		}
		defer conn.Close()
		// Hold the connection open (but never write) until signalled.
		<-serverStop
	}()

	// Client connects and immediately tries to Read — server never writes.
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		fmt.Println("  error connecting:", err)
		ln.Close()
		return
	}

	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		buf := make([]byte, 1)
		fmt.Printf("  goroutine: blocking on net.Conn.Read (server never writes)\n")
		_, err := conn.Read(buf) // ← blocked here inside OS poller, shows as [IO wait]
		if err != nil {
			fmt.Println("  goroutine: unblocked with error:", err)
		}
	}()

	time.Sleep(80 * time.Millisecond)
	dumpGoroutines()

	// Cleanup: signal server to stop, close client conn, wait for both goroutines.
	close(serverStop)
	conn.Close()
	ln.Close()
	<-clientDone
	<-serverDone
}
