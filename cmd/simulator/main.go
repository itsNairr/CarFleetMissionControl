package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	pb "github.com/itsnairr/fleet-telemetry-engine/internal/protocol"
	"github.com/itsnairr/fleet-telemetry-engine/internal/simulator"
)


func main() {
	// A sample of real-world fleet vehicles
	fleetConfigs := []struct {
		model    pb.VehicleModel
		scenario simulator.Scenario
	}{
		// {pb.VehicleModel_VEHICLE_MODEL_TESLA_CYBERTRUCK, simulator.ScenarioHeavyTowing},
		// {pb.VehicleModel_VEHICLE_MODEL_RIVIAN_R1T, simulator.ScenarioOffRoadAdventure},
		// {pb.VehicleModel_VEHICLE_MODEL_TESLA_MODEL_3, simulator.ScenarioHighwayCruising},
		{pb.VehicleModel_VEHICLE_MODEL_TESLA_MODEL_Y, simulator.ScenarioSupercharging},
		// {pb.VehicleModel_VEHICLE_MODEL_RIVIAN_EDV, simulator.ScenarioDeliveryStopCycle},
		// {pb.VehicleModel_VEHICLE_MODEL_TESLA_MODEL_S, simulator.ScenarioTirePuncture},
		// {pb.VehicleModel_VEHICLE_MODEL_RIVIAN_R1S, simulator.ScenarioWinterColdSoak},
		// {pb.VehicleModel_VEHICLE_MODEL_TESLA_MODEL_X, simulator.ScenarioAuxBatterySag},
		// {pb.VehicleModel_VEHICLE_MODEL_TESLA_MODEL_3, simulator.ScenarioUrbanCommute},
		// {pb.VehicleModel_VEHICLE_MODEL_RIVIAN_R1T, simulator.ScenarioV2LWorksite},
	}

	var wg sync.WaitGroup

	for i, cfg := range fleetConfigs {
		wg.Add(1)
		vin := fmt.Sprintf("SIM-%04d", i+1)
		vehicle := simulator.NewSimulatedVehicle(vin, cfg.model, cfg.scenario)
		go simulateVehicle(vehicle, "localhost:8080", &wg)
	}

	wg.Wait()
}


func simulateVehicle(v *simulator.SimulatedVehicle, serverAddr string, wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("[%s] Failed to connect to gateway: %v\n", v.Vin, err)
		return
	}
	defer conn.Close()

	// Initialize a 1 Hz ticker (rhythmic 1-second ticks)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop() // Cleans up the timer resource when connection terminates

	// Loop blocks on ticker channel instead of sleeping at the bottom
	for range ticker.C {
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

