package loadbalancer

import "sync"

type RoundRobin struct {
	mu      sync.Mutex
	current int
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{current: 0}
}

// Returns the next target in round-robin order
func (r *RoundRobin) Next(targets []string) string {
	if len(targets) == 0 {
		return ""
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	target := targets[r.current%len(targets)]
	r.current++

	return target
}

// Returns the strategy name
func (r *RoundRobin) Name() string {
	return "round_robin"
}
