package main

import (
	"fmt"
	"time"

	pb "github.com/itsnairr/fleet-telemetry-engine/internal/protocol"
	"google.golang.org/protobuf/proto"
)

func main() {
	msg := &pb.VehicleTelemetry{
		Vin:           "12345678901234567",
		TimestampMs: time.Now().UnixMilli(),
		Location: &pb.GPSLocation{
			Latitude:  12.9716,
			Longitude: 77.5946,
			Speed:     0,
			Heading:   0,
		},
		VehicleMode: "DRIVE",
		Gear:        pb.Gear_GEAR_DRIVE,
		SpeedKmh:    10,
	}
	fmt.Println(msg)
	
	data, err := proto.Marshal(msg)
	if err != nil {
		fmt.Println("Failed to serialize:", err)
		return
	}

	fmt.Println("Raw Bytes:", data)
	fmt.Println("Raw Bytes length:", len(data))

	var received = &pb.VehicleTelemetry{}
	err = proto.Unmarshal(data, received)
	if err != nil {
		fmt.Println("Failed to deserialize:", err)
		return
	}

	fmt.Println("Recieved Vin", received.GetVin())
	fmt.Println("Recieved Timestamp", received.GetTimestampMs())
	fmt.Println("Recieved Location Lat", received.GetLocation().GetLatitude())
	fmt.Println("Recieved Location Long", received.GetLocation().GetLongitude())
	fmt.Println("Recieved Vehicle Mode", received.GetVehicleMode())
	fmt.Println("Recieved Gear", received.GetGear())
	fmt.Println("Recieved Speed", received.GetSpeedKmh())
}