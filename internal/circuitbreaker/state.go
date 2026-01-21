package circuitbreaker

type State int

const (
	// StateClosed - normal operation, requests pass through
	StateClosed State = iota

	// StateOpen - circuit is open, requests fail immediately
	StateOpen

	// StateHalfOpen - testing if service recovered, allow limited requests
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
