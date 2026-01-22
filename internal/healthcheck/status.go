package healthcheck

import "time"

type Status struct {
	Target       string
	IsHealthy    bool
	LastCheck    time.Time
	LastSuccess  time.Time
	LastFailure  time.Time
	FailureCount int
}

// Represents overall health of a service
type HealthStatus int

const (
	Healthy HealthStatus = iota
	Degraded
	Unhealthy
)

func (h HealthStatus) String() string {
	switch h {
	case Healthy:
		return "healthy"
	case Degraded:
		return "degraded"
	case Unhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}
