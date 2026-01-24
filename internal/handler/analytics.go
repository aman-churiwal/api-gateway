package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnalyticsHandler struct {
	service *service.AnalyticsService
}

func NewAnalyticsHandler(service *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

// Handles GET /admin/analytics
func (h *AnalyticsHandler) GetSummary(c *gin.Context) {
	// Parse time range
	from, to, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	summary, err := h.service.GetSummary(ctx, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *AnalyticsHandler) GetTimeSeries(c *gin.Context) {
	// Parse time range
	from, to, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	timeSeriesData, err := h.service.GetTimeSeriesData(ctx, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, timeSeriesData)
}

// Handles GET /admin/analytics/keys/:id
func (h *AnalyticsHandler) GetAPIKeyStats(c *gin.Context) {
	idStr := c.Param("id")
	apiKeyID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	// Parse time range
	from, to, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	stats, err := h.service.GetAPIKeyStats(ctx, apiKeyID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Handles GET /admin/logs
func (h *AnalyticsHandler) GetLogs(c *gin.Context) {
	// Parse time range
	from, to, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse pagination
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse status code filter (optional)
	var statusCode *int
	if statusStr := c.Query("status"); statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			statusCode = &s
		}
	}

	ctx := c.Request.Context()
	logs, err := h.service.GetLogs(ctx, from, to, statusCode, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"limit":  limit,
		"offset": offset,
	})
}

// Parses 'from' and 'to' query parameters
func parseTimeRange(c *gin.Context) (time.Time, time.Time, error) {
	// Default: last 24 hours
	to := time.Now()
	from := to.Add(-24 * time.Hour)

	if fromStr := c.Query("from"); fromStr != "" {
		parsedFrom, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			// Try Unix timestamp
			if timestamp, err := strconv.ParseInt(fromStr, 10, 64); err == nil {
				parsedFrom = time.Unix(timestamp, 0)
			} else {
				return time.Time{}, time.Time{}, err
			}
		}
		from = parsedFrom
	}

	if toStr := c.Query("to"); toStr != "" {
		parsedTo, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			// Try Unix timestamp
			if timestamp, err := strconv.ParseInt(toStr, 10, 64); err == nil {
				parsedTo = time.Unix(timestamp, 0)
			} else {
				return time.Time{}, time.Time{}, err
			}
		}
		to = parsedTo
	}

	return from, to, nil
}
