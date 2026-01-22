package loadbalancer

import "sync"

type LeastConnections struct {
	mu          sync.RWMutex
	connections map[string]int
}

func NewLeastConnections() *LeastConnections {
	return &LeastConnections{
		connections: make(map[string]int),
	}
}

// Returns the target with least connections
func (l *LeastConnections) Next(targets []string) string {
	if len(targets) == 0 {
		return ""
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var selected string
	minConn := int(^uint(0) >> 1) // Max int

	for _, target := range targets {
		conn := l.connections[target]
		if conn < minConn {
			minConn = conn
			selected = target
		}
	}

	// If no target selected (all have same connections), pick first
	if selected == "" {
		selected = targets[0]
	}

	return selected
}

// Increments the connection count for a target
func (l *LeastConnections) Increment(target string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.connections[target]++
}

// Decrements the connection count for a target
func (l *LeastConnections) Decrement(target string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.connections[target] > 0 {
		l.connections[target]--
	}
}

// Returns the strategy name
func (l *LeastConnections) Name() string {
	return "least_connections"
}
