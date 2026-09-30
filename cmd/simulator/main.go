package main

import (
	"encoding/binary"
	"fmt"
	"google.golang.org/protobuf/proto"
	"math/rand"
	"net"
	"sync"
	"time"

	pb "github.com/itsnairr/fleet-telemetry-engine/internal/protocol"
)

func main() {
	numVehicles := 20
	var wg sync.WaitGroup

	for i := 1; i <= numVehicles; i++ {
		wg.Add(1)
		vin := fmt.Sprintf("SIM-%04d", i)
		go simulateVehicle(vin, "localhost:8080", &wg)
	}

	wg.Wait()

}

func simulateVehicle(vin string, serverAddr string, wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.Dial("tcp", serverAddr) //connect
	if err != nil {
		fmt.Printf("Failed to connect to gateway: %v\n", err)
		return
	}
	defer conn.Close()

	for {
		speed := float32(rand.Float32() * 120.0) // 0 to 120 km/h
		gear := pb.Gear_GEAR_DRIVE
		if speed < 1.0 {
			gear = pb.Gear_GEAR_PARK
		}

		msg := &pb.VehicleTelemetry{
			Vin:         vin,
			TimestampMs: time.Now().UnixMilli(),
			SpeedKmh:    speed,
			Gear:        gear,
			VehicleMode: "Standard",

			// Random GPS wandering near San Francisco
			Location: &pb.GPSLocation{
				Latitude:  37.7749 + (rand.Float64()-0.5)*0.01,
				Longitude: -122.4194 + (rand.Float64()-0.5)*0.01,
				Speed:     float64(speed),
			},

			// Realistic EV Battery Telemetry
			BatteryState: &pb.BatteryState{
				StateOfCharge:      float32(50.0 + rand.Float32()*45.0),  // 50% - 95%
				PackVoltage:        float32(380.0 + rand.Float32()*20.0), // ~400V architecture
				PackCurrent:        float32(20.0 + rand.Float32()*150.0), // 20A - 170A draw
				PackTemperature:    float32(28.0 + rand.Float32()*8.0),   // 28°C - 36°C
				MaxCellTemperature: float32(30.0 + rand.Float32()*10.0),
				PackStatus:         pb.BatteryState_BATTERY_STATUS_DISCHARGING,
			},

			// 4-Corner Tire Pressures (~2.8 bar / ~41 psi)
			TirePressure: &pb.TirePressure{
				FrontLeftBar:  2.8 + (rand.Float32()-0.5)*0.1,
				FrontRightBar: 2.8 + (rand.Float32()-0.5)*0.1,
				RearLeftBar:   2.8 + (rand.Float32()-0.5)*0.1,
				RearRightBar:  2.8 + (rand.Float32()-0.5)*0.1,
			},
		}

		data, err := proto.Marshal(msg)
		if err != nil {
			fmt.Println("Marshal error:", err)
			return
		}

		// Create a 4-byte header and write the length into it:
		header := make([]byte, 4)
		binary.BigEndian.PutUint32(header, uint32(len(data)))

		// Send the 4-byte header, then the protobuf data:
		conn.Write(header)
		conn.Write(data)

		time.Sleep(1 * time.Second)
	}

}
