package loadbalancer

import (
	"math/rand"
	"time"
)

type Random struct {
	rng *rand.Rand
}

func NewRandom() *Random {
	return &Random{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Returns a random target
func (r *Random) Next(targets []string) string {
	if len(targets) == 0 {
		return ""
	}

	return targets[r.rng.Intn(len(targets))]
}

// Returns the strategy name
func (r *Random) Name() string {
	return "random"
}
