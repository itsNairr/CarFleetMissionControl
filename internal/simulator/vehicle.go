package simulator

import (
	"time"
	pb "github.com/itsnairr/fleet-telemetry-engine/internal/protocol"
)

type Scenario string

const (
	ScenarioHighwayCruising     Scenario = "HIGHWAY_CRUISING"
	ScenarioUrbanCommute        Scenario = "URBAN_COMMUTE"
	ScenarioSupercharging       Scenario = "SUPERCHARGING"
	ScenarioACCharging          Scenario = "AC_CHARGING"
	ScenarioParkedSentry        Scenario = "PARKED_SENTRY"
	ScenarioOffRoadAdventure    Scenario = "OFF_ROAD"
	ScenarioHeavyTowing         Scenario = "HEAVY_TOWING"
	ScenarioV2LWorksite         Scenario = "V2L_WORKSITE"
	ScenarioDeliveryStopCycle   Scenario = "DELIVERY_CYCLE"
	ScenarioWinterColdSoak      Scenario = "WINTER_COLD_SOAK"
	ScenarioAuxBatterySag       Scenario = "AUX_BATTERY_SAG"
	ScenarioThermalHazard       Scenario = "THERMAL_HAZARD"
	ScenarioTirePuncture        Scenario = "TIRE_PUNCTURE"
	ScenarioLowBatteryTurtle    Scenario = "TURTLE_MODE"
)

type SimulatedVehicle struct {
	Vin          string
	Model        pb.VehicleModel
	Scenario     Scenario
	CurrentSpeed float32
	BatterySoC   float32 // 0.0 to 100.0%
	OdometerKm   float32
	Latitude     float64
	Longitude    float64
	TireFLBar    float32
	TireFRBar    float32
	TireRLBar    float32
	TireRRBar    float32
	Aux12VVolts  float32
	IsLocked     bool
	FrunkOpen    bool
	TailgateOpen bool
}

// Creates the simulated car and returns the address
func NewSimulatedVehicle(vin string, model pb.VehicleModel, scenario Scenario) *SimulatedVehicle {
	v := &SimulatedVehicle{
		Vin:          vin,
		Model:        model,
		Scenario:     scenario,
		BatterySoC:   80.0,
		CurrentSpeed: 0.0,
		OdometerKm:   12450.0,
		Latitude:     37.7749, // San Francisco Bay Area / Fremont baseline
		Longitude:    -122.4194,
		TireFLBar:    2.4, // Standard 35 PSI ~ 2.4 bar
		TireFRBar:    2.4,
		TireRLBar:    2.4,
		TireRRBar:    2.4,
		Aux12VVolts:  13.8,
		IsLocked:     true,
	}

	// Tailor starting conditions to the active scenario
	switch scenario {
	case ScenarioHighwayCruising:
		v.CurrentSpeed = 110.0
		v.BatterySoC = 75.0
	case ScenarioUrbanCommute:
		v.CurrentSpeed = 45.0
		v.BatterySoC = 60.0
	case ScenarioSupercharging:
		v.BatterySoC = 18.0
	case ScenarioLowBatteryTurtle:
		v.BatterySoC = 3.0
		v.CurrentSpeed = 25.0
	case ScenarioTirePuncture:
		v.CurrentSpeed = 65.0
		v.TireFLBar = 1.9 // Pressure actively leaking
	case ScenarioDeliveryStopCycle:
		v.FrunkOpen = false
		v.TailgateOpen = true // Amazon EDV unloading
		v.IsLocked = false
	case ScenarioAuxBatterySag:
		v.Aux12VVolts = 10.8 // Dying low-voltage battery
	}

	return v
}

// Tick advances the vehicle state by 1 second.
func (v *SimulatedVehicle) Tick() {
	// 1. Scenario-specific physics & battery behavior
	switch v.Scenario {
	case ScenarioHighwayCruising:
		v.CurrentSpeed = 110.0
		v.BatterySoC -= 0.006 // ~20 kWh/100km consumption
	case ScenarioHeavyTowing:
		v.CurrentSpeed = 95.0
		v.BatterySoC -= 0.014 // Heavy aerodynamic drag & load
	case ScenarioUrbanCommute:
		// Slight variation around city speeds
		if v.CurrentSpeed < 45.0 {
			v.CurrentSpeed += 5.0
		} else {
			v.CurrentSpeed = 30.0
		}
		v.BatterySoC -= 0.004
	case ScenarioSupercharging:
		v.CurrentSpeed = 0.0
		if v.BatterySoC < 80.0 {
			v.BatterySoC += 0.08 // Rapid DC fast charge (~200 kW)
		} else if v.BatterySoC < 100.0 {
			v.BatterySoC += 0.02 // Taper curve
		}
	case ScenarioACCharging:
		v.CurrentSpeed = 0.0
		if v.BatterySoC < 100.0 {
			v.BatterySoC += 0.003 // ~11 kW Level 2 AC charge
		}
	case ScenarioLowBatteryTurtle:
		v.CurrentSpeed = 25.0 // Power-limited turtle mode
		if v.BatterySoC > 0.5 {
			v.BatterySoC -= 0.001
		}
	case ScenarioTirePuncture:
		v.CurrentSpeed = 50.0
		v.BatterySoC -= 0.007
		if v.TireFLBar > 0.8 {
			v.TireFLBar -= 0.02 // Gradual tire deflation
		}
	case ScenarioAuxBatterySag:
		v.CurrentSpeed = 0.0
		if v.Aux12VVolts > 9.0 {
			v.Aux12VVolts -= 0.01 // Parasitic drain / dead alternator/DCDC
		}
	default:
		// Parked / idle scenarios
		v.CurrentSpeed = 0.0
		v.BatterySoC -= 0.0002 // Phantom standby drain
	}

	// Clamp Battery SoC between 0% and 100%
	if v.BatterySoC < 0 {
		v.BatterySoC = 0
		v.CurrentSpeed = 0
	} else if v.BatterySoC > 100 {
		v.BatterySoC = 100
	}

	// Accumulate odometer and advance GPS if moving
	if v.CurrentSpeed > 0 {
		distKm := v.CurrentSpeed / 3600.0
		v.OdometerKm += distKm
		// Heading roughly North-East: 1 km ~ 0.009 degrees latitude
		v.Latitude += float64(distKm) * 0.006
		v.Longitude += float64(distKm) * 0.006
	}
}

// Creates the vehicle and immedietly returns on the address
func (v *SimulatedVehicle) ToTelemetry() *pb.VehicleTelemetry { 
	nowMs := time.Now().UnixMilli()

	// Determine Gear & Driving Dynamics
	gear := pb.Gear_GEAR_PARK
	if v.CurrentSpeed > 0 {
		gear = pb.Gear_GEAR_DRIVE
	}

	// Charging Subsystem
	chargingState := &pb.ChargingState{
		State: pb.ChargingState_CHARGE_STATE_DISCONNECTED,
	}
	if v.Scenario == ScenarioSupercharging {
		chargingState.State = pb.ChargingState_CHARGE_STATE_CHARGING
		chargingState.ChargerType = pb.ChargingState_CHARGER_TYPE_DC_FAST
		chargingState.ChargingPowerKw = 210.0
		chargingState.ChargePortDoorOpen = true
		chargingState.ChargePortLatchEngaged = true
	} else if v.Scenario == ScenarioACCharging {
		chargingState.State = pb.ChargingState_CHARGE_STATE_CHARGING
		chargingState.ChargerType = pb.ChargingState_CHARGER_TYPE_AC_LEVEL_2
		chargingState.ChargingPowerKw = 11.5
		chargingState.ChargePortDoorOpen = true
		chargingState.ChargePortLatchEngaged = true
	}

	// Battery Subsystem (Pack Voltage ~400V nominal)
	packVoltage := float32(392.0)
	packCurrent := float32(0.0)
	if v.CurrentSpeed > 0 {
		packCurrent = v.CurrentSpeed * 2.8 // Draw current proportional to speed
	} else if chargingState.State == pb.ChargingState_CHARGE_STATE_CHARGING {
		packCurrent = -(chargingState.ChargingPowerKw * 1000.0 / packVoltage)
	}

	battery := &pb.BatteryState{
		StateOfCharge:           v.BatterySoC,
		PackVoltage:             packVoltage,
		PackCurrent:             packCurrent,
		PackPower:               (packVoltage * packCurrent) / 1000.0,
		PackHealth:              98.5,
		PackStatus:              pb.BatteryState_BATTERY_STATUS_OK,
		LowVoltageBatteryVolts: v.Aux12VVolts,
		MinCellVoltage:          3.82,
		MaxCellVoltage:          3.85,
	}

	// Cabin & Closures
	cabin := &pb.CabinState{
		InsideTempCelsius:  21.5,
		OutsideTempCelsius: 18.0,
		ClimateOn:          true,
		IsLocked:           v.IsLocked,
		FrunkOpen:          v.FrunkOpen,
		DriverDoorOpen:     false,
	}

	//Build Base Telemetry Frame
	telemetry := &pb.VehicleTelemetry{
		Vin:              v.Vin,
		Model:            v.Model,
		TimestampMs:      nowMs,
		Gear:             gear,
		SpeedKmh:         v.CurrentSpeed,
		OdometerKm:       v.OdometerKm,
		RemainingRangeKm: v.BatterySoC * 4.6, // ~460 km range at 100%
		VehicleMode:      string(v.Scenario),
		Location: &pb.GPSLocation{
			Latitude:  v.Latitude,
			Longitude: v.Longitude,
			Speed:     float64(v.CurrentSpeed),
		},
		BatteryState:  battery,
		ChargingState: chargingState,
		TirePressure: &pb.TirePressure{
			FrontLeftBar:  v.TireFLBar,
			FrontRightBar: v.TireFRBar,
			RearLeftBar:   v.TireRLBar,
			RearRightBar:  v.TireRRBar,
		},
		CabinState: cabin,
	}

	// 6. Truck-Specific Telemetry (Populated only for electric pickups)
	if v.Model == pb.VehicleModel_VEHICLE_MODEL_RIVIAN_R1T || v.Model == pb.VehicleModel_VEHICLE_MODEL_TESLA_CYBERTRUCK {
		telemetry.TruckState = &pb.TruckState{
			TailgateOpen:             v.TailgateOpen,
			TowModeActive:            v.Scenario == ScenarioHeavyTowing,
			TrailerConnected:         v.Scenario == ScenarioHeavyTowing,
			EstimatedTrailerWeightKg: 2800.0,
			BedOutletsActive:         v.Scenario == ScenarioV2LWorksite,
			BedOutletsPowerKw:        2.4,
		}
	}

	//Active Diagnostic Trouble Codes (DTCs)
	if v.TireFLBar < 1.7 {
		telemetry.ActiveAlertCodes = append(telemetry.ActiveAlertCodes, "BMS_w035_TirePressureLow")
	}
	if v.Aux12VVolts < 11.0 {
		telemetry.ActiveAlertCodes = append(telemetry.ActiveAlertCodes, "VCFRONT_a182_AuxBatterySag")
	}
	if v.BatterySoC < 5.0 {
		telemetry.ActiveAlertCodes = append(telemetry.ActiveAlertCodes, "DI_w032_LowSocTurtle")
	}

	return telemetry
}
