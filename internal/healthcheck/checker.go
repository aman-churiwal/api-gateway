package healthcheck

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// Performs health checks on backend targets
type Checker struct {
	mu             sync.RWMutex
	targets        []string
	healthStatus   map[string]*Status
	healthyTargets []string
	endpoint       string
	interval       time.Duration
	timeout        time.Duration
	maxFailures    int
	stopChan       chan struct{}
	running        bool
}

// Holds health checker configuration
type Config struct {
	Targets     []string
	Endpoint    string        // Health check endpoint (e.g., "/health")
	Interval    time.Duration // How often to check (default: 10s)
	Timeout     time.Duration // Request timeout (default: 5s)
	MaxFailures int           // Failures before marking unhealthy (default: 3)
}

func NewChecker(cfg *Config) *Checker {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "/health"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 3
	}

	checker := &Checker{
		targets:        cfg.Targets,
		healthStatus:   make(map[string]*Status),
		healthyTargets: make([]string, 0),
		endpoint:       cfg.Endpoint,
		interval:       cfg.Interval,
		timeout:        cfg.Timeout,
		maxFailures:    cfg.MaxFailures,
		stopChan:       make(chan struct{}),
	}

	// Initialize status for all targets
	for _, target := range cfg.Targets {
		checker.healthStatus[target] = &Status{
			Target:    target,
			IsHealthy: true, // Assume healthy initially
			LastCheck: time.Now(),
		}
	}

	return checker
}

// Begins periodic health checks
func (c *Checker) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	log.Printf("Starting health checks for %d targets (interval: %v)", len(c.targets), c.interval)

	// Run initial check immediately
	c.checkAll()

	// Start periodic checks
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.checkAll()
			case <-c.stopChan:
				return
			}
		}
	}()
}

// Stops the health checker
func (c *Checker) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		close(c.stopChan)
		c.running = false
		log.Printf("Health checker stopped")
	}
}

// Performs health check on all targets
func (c *Checker) checkAll() {
	var wg sync.WaitGroup

	for _, target := range c.targets {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			c.checkTarget(t)
		}(target)
	}

	wg.Wait()
	c.updateHealthyTargets()
}

// Performs health check on a single target
func (c *Checker) checkTarget(target string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	url := target + c.endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.recordFailure(target)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.recordFailure(target)
		return
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx as healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		c.recordSuccess(target)
	} else {
		c.recordFailure(target)
	}
}

// Records a successful health check
func (c *Checker) recordSuccess(target string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	status := c.healthStatus[target]
	status.LastCheck = time.Now()
	status.LastSuccess = time.Now()
	status.FailureCount = 0

	if !status.IsHealthy {
		log.Printf("Target :%s is now healthy", target)
		status.IsHealthy = true
	}
}

// Records a failed health check
func (c *Checker) recordFailure(target string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	status := c.healthStatus[target]
	status.LastCheck = time.Now()
	status.LastFailure = time.Now()
	status.FailureCount++

	if status.IsHealthy && status.FailureCount >= c.maxFailures {
		log.Printf("Target %s is now unhealthy (failures: %d)", target, status.FailureCount)
		status.IsHealthy = false
	}
}

// Updates the list of healthy targets
func (c *Checker) updateHealthyTargets() {
	c.mu.Lock()
	defer c.mu.Unlock()

	healthy := make([]string, 0)
	for _, target := range c.targets {
		if c.healthStatus[target].IsHealthy {
			healthy = append(healthy, target)
		}
	}

	c.healthyTargets = healthy
}

// Returns only healthy targets
func (c *Checker) GetHealthyTargets() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return copy to prevent external modification
	targets := make([]string, len(c.healthyTargets))
	copy(targets, c.healthyTargets)

	return targets
}

// Returns all targets regardless of health
func (c *Checker) GetAllTargets() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	targets := make([]string, len(c.targets))
	copy(targets, c.targets)

	return targets
}

// Return the health status of a specific target
func (c *Checker) GetStatus(target string) *Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if status, exists := c.healthStatus[target]; exists {
		// Return copy
		statusCopy := *status
		return &statusCopy
	}

	return nil
}

// Returns health status of all target
func (c *Checker) GetAllStatus() map[string]*Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statusMap := make(map[string]*Status)
	for target, status := range c.healthStatus {
		statusCopy := *status
		statusMap[target] = &statusCopy
	}

	return statusMap
}

// Returns the overall health status
func (c *Checker) OverallHealth() HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalTargets := len(c.targets)
	healthyCount := len(c.healthyTargets)

	if healthyCount == 0 {
		return Unhealthy
	}
	if healthyCount < totalTargets {
		return Degraded
	}

	return Healthy
}
