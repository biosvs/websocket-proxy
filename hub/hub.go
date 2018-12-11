package hub

import (
	"errors"
	"sync"

	conn "websocket-proxy/connection"
)

const (
	stateRun = iota
	stateStopped
)

// Hub collection of connections
type Hub struct {
	sync.Mutex
	state          int
	connections    map[int]*conn.Connection
	maxConnections int
}

// New create new hub
func New() *Hub {
	return &Hub{
		state:       stateRun,
		connections: make(map[int]*conn.Connection),
	}
}

// Stop stop all connections
func (h *Hub) Stop() {
	h.Lock()
	h.state = stateStopped
	h.Unlock()
}

// GetNextName get name for new connection
func (h *Hub) GetNextName() int {
	h.maxConnections++
	return h.maxConnections
}

// AddConnection add connection to hub
func (h *Hub) AddConnection(conn *conn.Connection) error {
	h.Lock()
	defer h.Unlock()

	if h.state == stateStopped {
		return errors.New("hub already stateStopped")
	}

	h.connections[conn.GetUniq()] = conn

	return nil
}

// DropConnection remove connection from hub
func (h *Hub) DropConnection(uniq int) {
	h.Lock()
	delete(h.connections, uniq)
	h.Unlock()
}

// CloseAllConnections close all connections and empty hub
func (h *Hub) CloseAllConnections() {
	h.Lock()
	for _, c := range h.connections {
		c.Close()
	}
	h.connections = map[int]*conn.Connection{}
	h.Unlock()
}
