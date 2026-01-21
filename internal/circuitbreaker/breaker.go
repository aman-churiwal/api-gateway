package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrCircuitOpen is returned when circuit is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// Implements the circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	lastStateChange time.Time

	// Configuration
	maxFailures     int           // Number of failures before opening
	timeout         time.Duration // How long to stay open
	halfOpenSuccess int           // Successes needed in half-open to close
}

type Config struct {
	MaxFailures     int           // Default: 5
	Timeout         time.Duration // Default: 30 seconds
	HalfOpenSuccess int           // Default: 1
}

func New(cfg Config) *CircuitBreaker {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenSuccess <= 0 {
		cfg.HalfOpenSuccess = 1
	}

	return &CircuitBreaker{
		state:           StateClosed,
		maxFailures:     cfg.MaxFailures,
		timeout:         cfg.Timeout,
		halfOpenSuccess: cfg.HalfOpenSuccess,
		lastStateChange: time.Now(),
	}
}

// Executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	// Check if we should transition from Open to Half-Open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.setState(StateHalfOpen)
			cb.successCount = 0
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}

	cb.mu.Unlock()

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// Handles a failed request
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		// In half-open, any failure opens the circuit
		cb.setState(StateOpen)
		cb.successCount = 0
	} else if cb.failureCount >= cb.maxFailures {
		// Too many failures, open the circuit
		cb.setState(StateOpen)
	}
}

// Handles a successful request
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.halfOpenSuccess {
			// Enough successes in half-open, close the circuit
			cb.setState(StateClosed)
			cb.failureCount = 0
		}
	case StateClosed:
		// Reset failure count on success in closed state
		cb.failureCount = 0
	default:
		return
	}
}

// Changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState State) {
	if cb.state != newState {
		cb.state = newState
		cb.lastStateChange = time.Now()
	}
}

// Returns the current state
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()
}

// Returns current circuit breaker metrics
func (cb *CircuitBreaker) Metrics() Metrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Metrics{
		State:           cb.state,
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailureTime: cb.lastFailureTime,
		LastStateChange: cb.lastStateChange,
	}
}

// Holds circuit breaker metrics
type Metrics struct {
	State           State
	FailureCount    int
	SuccessCount    int
	LastFailureTime time.Time
	LastStateChange time.Time
}
