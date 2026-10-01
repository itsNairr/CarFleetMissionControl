package session

import (
	"fmt"
	"net"
	"sync"
)

// Map cars to sockets based off of VIN
type SessionRegistry struct {
	mu       sync.RWMutex
	sessions map[string]net.Conn
}

// NewSessionRegistry initializes a thread-safe vehicle session 
func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{
		sessions: make(map[string]net.Conn),
	}
}

// Register vehicle to socket
func (r *SessionRegistry) Register(vin string, conn net.Conn) {
	r.mu.Lock() //Lock the map so no other car can chnage the sessions map or else it will corrupt
	defer r.mu.Unlock() //Unlock
	r.sessions[vin] = conn //Add the car to the map
	fmt.Printf("[Session] Registered %s | Active Sockets: %d\n", vin, len(r.sessions))
}

// Unregister removes a vehicle socket when the connection closes
func (r *SessionRegistry) Unregister(vin string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.sessions[vin];
	if exists {
		delete(r.sessions, vin)
		fmt.Printf("[Session] Deregistered %s | Active Sockets: %d\n", vin, len(r.sessions))
	}
}

// Get returns the open net.Conn for a specific vehicle, if connected
func (r *SessionRegistry) Get(vin string) (net.Conn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, exists := r.sessions[vin]
	return conn, exists
}

// Count returns the number of active connected vehicles
func (r *SessionRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}
