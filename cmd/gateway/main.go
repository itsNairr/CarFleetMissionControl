package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"errors"

	"github.com/itsnairr/fleet-telemetry-engine/internal/worker"
)

func handleConnection(conn net.Conn, pool *worker.TelemetryWorkerPool) {
	defer conn.Close()

	fmt.Printf("New vehicle connected: %s\n", conn.RemoteAddr().String())

	conn.RemoteAddr().String()
	
}

func main() {
	sigChan := make(chan os.Signal, 1) //Used to catch the Ctrl+C or other termination signals
	//Buffer of 1 means that it can hold one signal without blocking the main thread. If we get a second signal before processing the first one, it would block. Not that it matters too much in this case, but it's good practice to know.
	
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	pool := worker.NewTelemetryWorkerPool(5, 100) //5 Workers with a shared queue of 100
	pool.Start()

	//Setup TCP connection
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Failed to bind to port 8080: %v\n", err)
		return
	}

	defer listener.Close() //Close at the end
	fmt.Println("Gateway TCP server listening on :8080...")


	go func() { //goroutine to not run on main thread
		for {
			conn, err := listener.Accept()
			if err != nil {
				// When we shutdown, listener.Close() is called, which causes Accept() to return net.ErrClosed.
				// This is a normal, clean shutdown—not a crash.
				if errors.Is(err, net.ErrClosed) {
					fmt.Println("TCP listener closed cleanly.")
					return
				}
				fmt.Printf("Accept error: %v\n", err)
				continue
			}

			// Handle this specific vehicle concurrently without blocking other cars!
			go handleConnection(conn, pool)
		}
	}()

	fmt.Println("Gateway running. Press Ctrl+C to shut down gracefully...")
	
	//Channel (frozen as there is not a val in sigChan)
	sig := <-sigChan //Main freezes until Ctrl+C
	fmt.Printf("\nReceived signal: %s. Initiating graceful shutdown...\n", sig)

	pool.Stop()

	fmt.Println("Shutdown complete. Exiting cleanly.")
}