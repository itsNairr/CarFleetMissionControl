package worker

import (
	"fmt"
	"sync"
	"time"

	pb "github.com/itsnairr/fleet-telemetry-engine/internal/protocol"
)

type TelemetryWorkerPool struct {
	numWorkers int
	jobQueue   chan *pb.VehicleTelemetry
	workerWg   sync.WaitGroup
}

// Essentially a python __init__ function to create a telemetry worker pool
func NewTelemetryWorkerPool(numWorkers int, queueCapacity int) *TelemetryWorkerPool {
	return &TelemetryWorkerPool{
		numWorkers: numWorkers,
		jobQueue:   make(chan *pb.VehicleTelemetry, queueCapacity),
		//All numWorkers worker goroutines are actively listening to that single shared jobQueue channel
	}
}

func (p *TelemetryWorkerPool) worker(workerID int) {
	defer p.workerWg.Done()

	// range over a channel automatically stops when the channel is closed
	for telemetry := range p.jobQueue {
		fmt.Printf("[Worker %d] Processing VIN: %s | Speed: %.1f km/h\n", workerID, telemetry.GetVin(), telemetry.GetSpeedKmh())
		// Simulate some processing time
		time.Sleep(5 * time.Millisecond)
	}

}

func (p *TelemetryWorkerPool) Start() {
	fmt.Printf("Starting %d workers...\n", p.numWorkers)
	for i := 1; i <= p.numWorkers; i++ {
		p.workerWg.Add(1)
		go p.worker(i)
	}
}

func (p *TelemetryWorkerPool) Stop() {
	fmt.Println("Stopping all workers...")
	close(p.jobQueue)
	p.workerWg.Wait()
	fmt.Println("All workers stopped")
}

func (p *TelemetryWorkerPool) Enqueue(telemetry *pb.VehicleTelemetry) bool {
	select {
	case p.jobQueue <- telemetry: //Attempt to push the telemetry into the job queue, <- means sending
		return true //Message successfully enqueued
	default: //If the job queue is full, the default case is executed
		fmt.Printf("Queue full! Dropping message for VIN: %s\n", telemetry.GetVin()) //Print that the message was dropped
		return false //Message dropped
	}
}