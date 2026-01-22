package loadbalancer

import "fmt"

// Creates a load balancing strategy based on name
func NewStrategy(strategyName string) (Strategy, error) {
	switch strategyName {
	case "round-robin", "round_robin", "":
		return NewRoundRobin(), nil
	case "random":
		return NewRandom(), nil
	case "least-connection", "least_connections":
		return NewLeastConnections(), nil
	default:
		return nil, fmt.Errorf("unknown load balancing strategy: %s", strategyName)
	}
}
