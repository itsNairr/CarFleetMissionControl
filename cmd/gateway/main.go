package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/itsnairr/fleet-telemetry-engine/internal/protocol"
	"github.com/itsnairr/fleet-telemetry-engine/internal/worker"
	"google.golang.org/protobuf/proto"
)

func main() {
	sigChan := make(chan os.Signal, 1) //Used to catch the Ctrl+C or other termination signals
	//Buffer of 1 means that it can hold one signal without blocking the main thread. If we get a second signal before processing the first one, it would block. Not that it matters too much in this case, but it's good practice to know.
	
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	pool := worker.NewTelemetryWorkerPool(5, 100) //5 Workers with a shared queue of 100
	pool.Start()

	stopProducer := make(chan struct{}) //Used to signal the producer to stop

	go func() {
		counter := 1 //Using counter to simulate VIN numbers, and speed
		for {
			select {
			case <-stopProducer: //If the producer is stopped, return
				return
			default:
				msg := &pb.VehicleTelemetry{
					Vin:         fmt.Sprintf("VIN-%04d", counter),
					TimestampMs: time.Now().UnixMilli(),
					Location: &pb.GPSLocation{
						Latitude:  12.9716,
						Longitude: 77.5946,
					},
					VehicleMode: "DRIVE",
					Gear:        pb.Gear_GEAR_DRIVE,
					SpeedKmh:    float32(20 + (counter%50)*2),
				}

				// Simulate wire serialization & deserialization
				data, err := proto.Marshal(msg)
				if err == nil {
					var received pb.VehicleTelemetry
					if err := proto.Unmarshal(data, &received); err == nil {
						if ok := pool.Enqueue(&received); !ok {
							fmt.Printf("[ALERT] Queue full! Dropped packet for VIN: %s\n", received.GetVin())
						}
					}
				}

				counter++
				time.Sleep(10 * time.Millisecond) //Cooldown
			}
		}
	}()

	fmt.Println("Gateway running. Press Ctrl+C to shut down gracefully...")

	sig := <-sigChan
	fmt.Printf("\nReceived signal: %s. Initiating graceful shutdown...\n", sig)

	close(stopProducer)

	pool.Stop()

	fmt.Println("Shutdown complete. Exiting cleanly.")
}