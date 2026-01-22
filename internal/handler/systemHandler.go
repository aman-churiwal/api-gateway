package handler

import (
	"net/http"

	"github.com/aman-churiwal/api-gateway/internal/proxy"
	"github.com/gin-gonic/gin"
)

// Handles system-related endpoints
type SystemHandler struct {
	proxies map[string]*proxy.Proxy
}

func NewSystemHandler(proxies map[string]*proxy.Proxy) *SystemHandler {
	return &SystemHandler{
		proxies: proxies,
	}
}

// Returns the status of all circuit breakers
func (h *SystemHandler) CircuitBreakerStatus(c *gin.Context) {
	statuses := make(map[string]interface{})

	for path, proxyInstance := range h.proxies {
		metrics := proxyInstance.CircuitBreakerMetrics()

		statuses[path] = gin.H{
			"state":             metrics.State.String(),
			"failure_count":     metrics.FailureCount,
			"success_count":     metrics.SuccessCount,
			"last_failure_time": metrics.LastFailureTime,
			"last_state_change": metrics.LastStateChange,
		}
	}

	c.JSON(http.StatusOK, statuses)
}

// Manually resets a circuit breaker
func (h *SystemHandler) ResetCircuitBreaker(c *gin.Context) {
	service := c.Param("service")

	proxyInstance, exists := h.proxies[service]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Service not found",
		})
		return
	}

	proxyInstance.ResetCircuitBreaker()

	c.JSON(http.StatusOK, gin.H{
		"message": "Circuit breaker reset successfully",
		"service": service,
	})
}

// Returns health status of all backend targets
func (h *SystemHandler) ServiceHealthStatus(c *gin.Context) {
	healthStatuses := make(map[string]interface{})

	for path, proxyInstance := range h.proxies {
		targetStatuses := proxyInstance.GetHealthStatus()
		healthyTargets := proxyInstance.GetHealthyTargets()
		allTargets := proxyInstance.GetAllTargets()
		overallHealth := proxyInstance.OverallHealth()

		statuses := make([]gin.H, 0)
		for _, status := range targetStatuses {
			statuses = append(statuses, gin.H{
				"target":        status.Target,
				"is_healthy":    status.IsHealthy,
				"last_check":    status.LastCheck,
				"last_success":  status.LastSuccess,
				"last_failure":  status.LastFailure,
				"failure_count": status.FailureCount,
			})
		}

		healthStatuses[path] = gin.H{
			"overall_health":  overallHealth.String(),
			"healthy_count":   len(healthyTargets),
			"total_count":     len(allTargets),
			"healthy_targets": healthyTargets,
			"all_targets":     allTargets,
			"target_status":   statuses,
		}
	}

	c.JSON(http.StatusOK, healthStatuses)
}
