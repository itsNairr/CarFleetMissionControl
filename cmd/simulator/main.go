package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/itsnairr/fleet-telemetry-engine/internal/simulator"
)

func main() {
	numVehiclesFlag := flag.Int("n", 1, "Number of fleet vehicles to simulate")           //-n
	serverAddrFlag := flag.String("addr", "localhost:8080", "Gateway TCP server address") //-addr
	flag.Parse()

	count := *numVehiclesFlag
	if flag.NArg() > 0 {
		if parsed, err := strconv.Atoi(flag.Arg(0)); err == nil && parsed > 0 {
			count = parsed
		}
	}

	fmt.Printf("🚗 Initializing EV Fleet Simulator: %d vehicles\n", count)
	fmt.Printf("📍 Telemetry Distribution: California Roads & Metros\n")
	fmt.Printf("📡 Connecting to Gateway: %s\n\n", *serverAddrFlag)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		vin := fmt.Sprintf("SIM-%04d", i+1)

		// Probabilistic scenario assignment (mirroring real-world fleet distribution)
		scenario := simulator.SelectRandomScenario()
		// Select compatible vehicle model (EDV for delivery, pickups for towing/V2L, etc.)
		model := simulator.SelectCompatibleModel(scenario)

		vehicle := simulator.NewSimulatedVehicle(vin, model, scenario)

		// Subtle 5ms stagger to avoid TCP thundering herd on gateway connection accept
		time.Sleep(5 * time.Millisecond)
		go simulateVehicle(ctx, vehicle, *serverAddrFlag, &wg)
	}

	<-ctx.Done()
	fmt.Println("\n🛑 Shutdown signal received. Disconnecting fleet cleanly...")
	wg.Wait()
	fmt.Println("All simulated vehicles stopped. Exiting.")

}

func simulateVehicle(ctx context.Context, v *simulator.SimulatedVehicle, serverAddr string, wg *sync.WaitGroup) {
	defer wg.Done()

	backoff := 1 * time.Second
	maxBackoff := 10 * time.Second

	for {
		// Check if operator pressed Ctrl+C before trying to connect
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := net.Dial("tcp", serverAddr)
		if err != nil {
			fmt.Printf("[%s] 📡 Modem searching for gateway... retry in %v\n", v.Vin, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				// Exponential backoff
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
		}

		// Connected, Reset backoff
		backoff = 1 * time.Second
		fmt.Printf("[%s] 📶 Connected to gateway\n", v.Vin)

		// Stream telemetry until socket breaks or Ctrl+C is pressed
		streamTelemetry(ctx, conn, v)
		conn.Close()
	}
}

func streamTelemetry(ctx context.Context, conn net.Conn, v *simulator.SimulatedVehicle) {
	// Initialize a 1 Hz ticker
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Loop blocks on ticker channel
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			//Advance vehicle physics & snapshot telemetry
			v.Tick()
			msg := v.ToTelemetry()

			// Serialize Protobuf
			data, err := proto.Marshal(msg)
			if err != nil {
				fmt.Printf("[%s] Marshal error: %v\n", v.Vin, err)
				return
			}

			// 4-byte length-prefix framing
			header := make([]byte, 4)
			binary.BigEndian.PutUint32(header, uint32(len(data)))

			if _, err := conn.Write(header); err != nil {
				return
			}
			if _, err := conn.Write(data); err != nil {
				return
			}
		}
	}
}
