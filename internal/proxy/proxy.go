package proxy

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/circuitbreaker"
	"github.com/gin-gonic/gin"
)

type Proxy struct {
	target         *url.URL
	proxy          *httputil.ReverseProxy
	circuitBreaker *circuitbreaker.CircuitBreaker
}

func New(targetURL string) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	// Create circuit breaker with default config
	cb := circuitbreaker.New(circuitbreaker.Config{
		MaxFailures:     5,
		Timeout:         30 * time.Second,
		HalfOpenSuccess: 1,
	})

	return &Proxy{
		target:         target,
		proxy:          httputil.NewSingleHostReverseProxy(target),
		circuitBreaker: cb,
	}, nil
}

// Creates a new Proxy with custom circuit breaker config
func NewWithConfig(targetURL string, cbConfig circuitbreaker.Config) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	cb := circuitbreaker.New(cbConfig)

	return &Proxy{
		target:         target,
		proxy:          httputil.NewSingleHostReverseProxy(target),
		circuitBreaker: cb,
	}, nil
}

// Forwards the request to the backend
func (p *Proxy) Handle(c *gin.Context) {
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
		req.URL.Host = p.target.Host
		req.URL.Scheme = p.target.Scheme
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = p.target.Host

		// Add X-Forwarded-For header
		if clientIP := c.ClientIP(); clientIP != "" {
			req.Header.Set("X-Forwarded-For", clientIP)
		}

		// Replace writer with recorder
		c.Writer = recorder

		// Forward the request
		p.proxy.ServeHTTP(c.Writer, req)

		// Check if backend returned 5xx error
		if recorder.statusCode >= 500 {
			return errors.New("backend error")
		}

		return nil
	})

	if err != nil {
		if err == circuitbreaker.ErrCircuitOpen {
			log.Printf("Circuit breaker open for %s", p.target.Host)
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
