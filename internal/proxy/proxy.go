package proxy

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/circuitbreaker"
	"github.com/aman-churiwal/api-gateway/internal/healthcheck"
	"github.com/aman-churiwal/api-gateway/internal/loadbalancer"
	"github.com/gin-gonic/gin"
)

type Proxy struct {
	targets        []string
	proxies        map[string]*httputil.ReverseProxy
	circuitBreaker *circuitbreaker.CircuitBreaker
	loadBalancer   loadbalancer.Strategy
	healthChecker  *healthcheck.Checker
}

type Config struct {
	Targets              []string
	LoadBalancerStrategy string
	CircuitBreaker       circuitbreaker.Config
	HealthCheck          healthcheck.Config
}

func New(targetURL string) (*Proxy, error) {
	return NewWithConfig(Config{
		Targets:              []string{targetURL},
		LoadBalancerStrategy: "round-robin",
		CircuitBreaker: circuitbreaker.Config{
			MaxFailures:     5,
			Timeout:         30 * time.Second,
			HalfOpenSuccess: 1,
		},
	})
}

// Creates a new Proxy with custom circuit breaker config
func NewWithConfig(cfg Config) (*Proxy, error) {
	if len(cfg.Targets) == 0 {
		return nil, errors.New("at least one target is required")
	}

	// Create circuit breaker
	cb := circuitbreaker.New(cfg.CircuitBreaker)

	// Create load balancer strategy
	lb, err := loadbalancer.NewStrategy(cfg.LoadBalancerStrategy)
	if err != nil {
		return nil, err
	}

	// Create reverse proxies for each target
	proxies := make(map[string]*httputil.ReverseProxy)
	for _, targetURL := range cfg.Targets {
		target, err := url.Parse(targetURL)
		if err != nil {
			return nil, err
		}

		proxies[targetURL] = httputil.NewSingleHostReverseProxy(target)
	}

	// Setup health check configurations
	if cfg.HealthCheck.Targets == nil {
		cfg.HealthCheck.Targets = cfg.Targets
	}

	// Create health checker
	hc := healthcheck.NewChecker(&cfg.HealthCheck)
	hc.Start()

	p := &Proxy{
		targets:        cfg.Targets,
		proxies:        proxies,
		circuitBreaker: cb,
		loadBalancer:   lb,
		healthChecker:  hc,
	}

	log.Printf("Proxy initialized with %d targets, strategy: %s", len(cfg.Targets), lb.Name())

	return p, nil
}

// Forwards the request to the backend
func (p *Proxy) Handle(c *gin.Context) {
	// Get healthy targets only
	healthyTargets := p.healthChecker.GetHealthyTargets()

	if len(healthyTargets) == 0 {
		log.Println("No healthy targets available")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No healthy backend servers available",
		})
		return
	}

	// Select target using load balancer
	selectedTarget := p.loadBalancer.Next(healthyTargets)

	if selectedTarget == "" {
		log.Println("Load balancer returned empty target")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Failed to select backend server",
		})
		return
	}

	// Get the proxy for this target
	targetProxy, exists := p.proxies[selectedTarget]
	if !exists {
		log.Printf("Proxy not found for target: %s", selectedTarget)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Internal server error",
		})
		return
	}

	// Track connections for least-connections strategy
	if lc, ok := p.loadBalancer.(*loadbalancer.LeastConnections); ok {
		lc.Increment(selectedTarget)
		defer lc.Decrement(selectedTarget)
	}

	// Parse target URL
	target, _ := url.Parse(selectedTarget)

	// Wrap the proxy call with circuit breaker
	err := p.circuitBreaker.Call(func() error {
		// Create a response recorder to capture status
		recorder := &responseRecorder{
			ResponseWriter: c.Writer,
			statusCode:     http.StatusOK,
		}

		// Preserve the original request
		req := c.Request

		// Modify request for proxying
		req.URL.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = target.Host

		// Add X-Forwarded-For header
		if clientIP := c.ClientIP(); clientIP != "" {
			req.Header.Set("X-Forwarded-For", clientIP)
		}

		// Add backend target header for debugging
		c.Header("X-Backend-Server", selectedTarget)

		// Replace writer with recorder
		c.Writer = recorder

		// Forward the request
		targetProxy.ServeHTTP(c.Writer, req)

		// Check if backend returned 5xx error
		if recorder.statusCode >= 500 {
			return errors.New("backend error")
		}

		return nil
	})

	if err != nil {
		if err == circuitbreaker.ErrCircuitOpen {
			log.Printf("Circuit breaker open for %s", selectedTarget)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporarily unavailable",
			})
			return
		}

		// Other errors are already handled by proxy
	}
}

// Returns the current circuit breaker state
func (p *Proxy) CircuitBreakerState() circuitbreaker.State {
	return p.circuitBreaker.State()
}

// Returns circuit breaker metrics
func (p *Proxy) CircuitBreakerMetrics() circuitbreaker.Metrics {
	return p.circuitBreaker.Metrics()
}

// Manually resets the circuit breaker
func (p *Proxy) ResetCircuitBreaker() {
	p.circuitBreaker.Reset()
}

// Returns health status of all targets
func (p *Proxy) GetHealthStatus() map[string]*healthcheck.Status {
	return p.healthChecker.GetAllStatus()
}

// Returns list of healthy targets
func (p *Proxy) GetHealthyTargets() []string {
	return p.healthChecker.GetHealthyTargets()
}

// Returns all targets
func (p *Proxy) GetAllTargets() []string {
	return p.healthChecker.GetAllTargets()
}

// Returns overall health status
func (p *Proxy) OverallHealth() healthcheck.HealthStatus {
	return p.healthChecker.OverallHealth()
}

// Stops the health checker
func (p *Proxy) Stop() {
	if p.healthChecker != nil {
		p.healthChecker.Stop()
	}
}

// Captures the response status code
type responseRecorder struct {
	gin.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	return r.ResponseWriter.Write(data)
}
