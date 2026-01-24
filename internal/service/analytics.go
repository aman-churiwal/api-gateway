package service

import (
	"context"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/repository"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/google/uuid"
)

type AnalyticsService struct {
	db         *storage.Postgres
	repository *repository.RequestLogRepository
}

func NewAnalyticsService(db *storage.Postgres, repo *repository.RequestLogRepository) *AnalyticsService {
	return &AnalyticsService{
		db:         db,
		repository: repo,
	}
}

// Holds analytics summary data
type AnalyticsSummary struct {
	TotalRequests   int64                    `json:"total_requests"`
	AvgResponseTime float64                  `json:"avg_response_time_ms"`
	P50ResponseTime int                      `json:"p50_response_time_ms"`
	P95ResponseTime int                      `json:"p95_response_time_ms"`
	P99ResponseTime int                      `json:"p99_response_time_ms"`
	ErrorRate       float64                  `json:"error_rate"`
	SuccessRate     float64                  `json:"success_rate"`
	ClientErrorRate float64                  `json:"client_error_rate"`
	ServerErrorRate float64                  `json:"server_error_rate"`
	TopEndpoints    []map[string]interface{} `json:"top_endpoints"`
}

// Holds time-series analytics data
type TimeSeriesData struct {
	Hour            time.Time `json:"hour"`
	Count           int64     `json:"count"`
	AvgResponseTime float64   `json:"avg_response_time"`
}

// Retrieves analytics summary for a time range
func (s *AnalyticsService) GetSummary(ctx context.Context, from, to time.Time) (*AnalyticsSummary, error) {
	summary := &AnalyticsSummary{}

	// Total requests
	totalRequests, err := s.repository.CountByTimeRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	summary.TotalRequests = totalRequests

	if totalRequests == 0 {
		return summary, nil
	}

	// Average response time
	avgResponseTime, err := s.repository.GetAverageResponseTime(ctx, from, to)
	if err != nil {
		return nil, err
	}
	summary.AvgResponseTime = avgResponseTime

	// P50, P95, P99 Response times
	p50, _ := s.repository.GetPercentile(ctx, from, to, 0.50)
	summary.P50ResponseTime = p50

	p95, _ := s.repository.GetPercentile(ctx, from, to, 0.95)
	summary.P95ResponseTime = p95

	p99, _ := s.repository.GetPercentile(ctx, from, to, 0.99)
	summary.P99ResponseTime = p99

	// Error counts
	clientErrors, err := s.repository.CountByStatusCodeRange(ctx, 400, 499, from, to)
	if err != nil {
		return nil, err
	}

	serverErrors, err := s.repository.CountByStatusCodeRange(ctx, 500, 599, from, to)
	if err != nil {
		return nil, err
	}

	// Calculate rates
	totalErrors := clientErrors + serverErrors
	summary.ErrorRate = (float64(totalErrors) / float64(totalRequests)) * 100
	summary.SuccessRate = 100 - summary.ErrorRate
	summary.ClientErrorRate = (float64(clientErrors) / float64(totalRequests)) * 100
	summary.ServerErrorRate = (float64(serverErrors) / float64(totalRequests)) * 100

	// Top Endpoints
	topEndpoints, err := s.repository.GetTopEndpoints(ctx, from, to, 10)
	if err != nil {
		return nil, err
	}
	summary.TopEndpoints = topEndpoints

	return summary, nil
}

// Retrieves time-series data
func (s *AnalyticsService) GetTimeSeriesData(ctx context.Context, from, to time.Time) ([]TimeSeriesData, error) {
	hourlyStatus, err := s.repository.GetHourlyStatus(ctx, from, to)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]TimeSeriesData, 0, len(hourlyStatus))
	for _, stat := range hourlyStatus {
		timeSeries = append(timeSeries, TimeSeriesData{
			Hour:            stat["hour"].(time.Time),
			Count:           stat["count"].(int64),
			AvgResponseTime: stat["avg_response_time"].(float64),
		})
	}

	return timeSeries, nil
}

// Retrieves analytics for a specific API key
func (s *AnalyticsService) GetAPIKeyStats(ctx context.Context, apiKeyID uuid.UUID, from, to time.Time) (*AnalyticsSummary, error) {
	// Similar to GetSummary but filtered by API key
	logs, err := s.repository.FindByAPIKey(ctx, apiKeyID, from, to, 10000, 0)
	if err != nil {
		return nil, err
	}

	if len(logs) == 0 {
		return &AnalyticsSummary{}, nil
	}

	// Calculate metrics from logs
	summary := &AnalyticsSummary{
		TotalRequests: int64(len(logs)),
	}

	var totalResponseTime int64
	var clientErrors, serverErrors int64

	for _, log := range logs {
		totalResponseTime += int64(log.ResponseTimeMs)

		if log.StatusCode >= 400 && log.StatusCode <= 499 {
			clientErrors++
		}
		if log.StatusCode >= 500 && log.StatusCode <= 599 {
			serverErrors++
		}

	}
	summary.AvgResponseTime = float64(totalResponseTime) / float64(summary.TotalRequests)

	totalErrors := clientErrors + serverErrors
	summary.ErrorRate = (float64(totalErrors) / float64(summary.TotalRequests)) * 100
	summary.SuccessRate = 100 - summary.ErrorRate
	summary.ClientErrorRate = (float64(clientErrors) / float64(summary.TotalRequests)) * 100
	summary.ServerErrorRate = (float64(serverErrors) / float64(summary.TotalRequests)) * 100

	return summary, nil
}

// Retrieves request log with pagination and filtering
func (s *AnalyticsService) GetLogs(ctx context.Context, from, to time.Time, statusCode *int, limit, offset int) ([]interface{}, error) {
	var logs []interface{}

	if statusCode != nil {
		logResults, err := s.repository.FindByStatusCode(ctx, *statusCode, from, to, limit, offset)
		if err != nil {
			return nil, err
		}
		for _, log := range logResults {
			logs = append(logs, log)
		}
	} else {
		logResults, err := s.repository.FindByTimeRange(ctx, from, to, limit, offset)
		if err != nil {
			return nil, err
		}
		for _, log := range logResults {
			logs = append(logs, log)
		}
	}

	return logs, nil
}

// Deletes logs older than specified retention period
func (s *AnalyticsService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	cutOffDate := time.Now().AddDate(0, 0, -retentionDays)
	return s.repository.DeleteOldLogs(ctx, cutOffDate)
}
