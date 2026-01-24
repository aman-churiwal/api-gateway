package repository

import (
	"context"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/models"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/google/uuid"
)

type RequestLogRepository struct {
	db *storage.Postgres
}

func NewRequestLogRepository(db *storage.Postgres) *RequestLogRepository {
	return &RequestLogRepository{db: db}
}

// Inserts a new request log
func (r *RequestLogRepository) Create(ctx context.Context, log *models.RequestLog) error {
	return r.db.DB.WithContext(ctx).Create(log).Error
}

// Inserts multiple request logs (for batch insertion)
func (r *RequestLogRepository) CreateBatch(ctx context.Context, logs []*models.RequestLog) error {
	if len(logs) == 0 {
		return nil
	}

	return r.db.DB.WithContext(ctx).Create(&logs).Error
}

// Retrieves logs within a time range
func (r *RequestLogRepository) FindByTimeRange(ctx context.Context, from, to time.Time, limit, offset int) ([]models.RequestLog, error) {
	var logs []models.RequestLog

	err := r.db.DB.WithContext(ctx).
		Where("timestamp BETWEEN ? AND ?", from, to).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, err
}

// Retrieves logs for a specific API key
func (r *RequestLogRepository) FindByAPIKey(ctx context.Context, apiKeyID uuid.UUID, from, to time.Time, limit, offset int) ([]models.RequestLog, error) {
	var logs []models.RequestLog
	err := r.db.DB.WithContext(ctx).
		Where("api_key_id = ? AND timestamp BETWEEN ? AND ?", apiKeyID, from, to).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, err
}

// Retrieve logs with specific status code
func (r *RequestLogRepository) FindByStatusCode(ctx context.Context, statusCode int, from, to time.Time, limit, offset int) ([]models.RequestLog, error) {
	var logs []models.RequestLog

	err := r.db.DB.WithContext(ctx).
		Where("status_code = ? AND timestamp BETWEEN ? AND ?", statusCode, from, to).
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, err
}

// Counts logs in a time range
func (r *RequestLogRepository) CountByTimeRange(ctx context.Context, from, to time.Time) (int64, error) {
	var count int64

	err := r.db.DB.WithContext(ctx).
		Model(&models.RequestLog{}).
		Where("timestamp BETWEEN ? AND ?", from, to).
		Count(&count).Error

	return count, err
}

// Calculates average response time
func (r *RequestLogRepository) GetAverageResponseTime(ctx context.Context, from, to time.Time) (float64, error) {
	var avg float64

	err := r.db.DB.WithContext(ctx).
		Model(&models.RequestLog{}).
		Where("timestamp BETWEEN ? AND ?", from, to).
		Select("AVG(response_time_ms)").
		Scan(&avg).Error

	return avg, err
}

// Calculates response time percentile
func (r *RequestLogRepository) GetPercentile(ctx context.Context, from, to time.Time, percentile float64) (int, error) {
	// Calculate percentile using SQL
	var result int
	query := `
		SELECT PERCENTILE_CONT(?) WITHIN GROUP (ORDER BY response_time_ms)
		FROM request_logs
		WHERE timestamp BETWEEN ? AND ?
	`

	err := r.db.DB.WithContext(ctx).Raw(query, percentile, from, to).Scan(&result).Error
	return result, err
}

// Count logs by status code range (e.g., 4xx, 5xx)
func (r *RequestLogRepository) CountByStatusCodeRange(ctx context.Context, minStatusCode, maxStatusCode int, from, to time.Time) (int64, error) {
	var count int64

	err := r.db.DB.WithContext(ctx).
		Model(&models.RequestLog{}).
		Where("status_code BETWEEN ? AND ? AND timestamp BETWEEN ? AND ?", minStatusCode, maxStatusCode, from, to).
		Count(&count).Error

	return count, err
}

// Returns most frequently accessed endpoints
func (r *RequestLogRepository) GetTopEndpoints(ctx context.Context, from, to time.Time, limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	rows, err := r.db.DB.WithContext(ctx).
		Model(&models.RequestLog{}).
		Select("path, COUNT(*) as count").
		Where("timestamp BETWEEN ? AND ?", from, to).
		Group("path").
		Order("count DESC").
		Limit(limit).
		Rows()

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var path string
		var count int64

		if err := rows.Scan(&path, &count); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"path":  path,
			"count": count,
		})
	}

	return results, nil
}

// Returns the request count grouped by hour
func (r *RequestLogRepository) GetHourlyStatus(ctx context.Context, from, to time.Time) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	rows, err := r.db.DB.WithContext(ctx).
		Model(&models.RequestLog{}).
		Select("DATE_TRUNC('hour', timestamp) as hour, COUNT(*) as count, AVG(response_time_ms) as avg_response_time").
		Where("timestamp BETWEEN ? AND ?", from, to).
		Group("hour").
		Order("hour ASC").
		Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hour time.Time
		var count int64
		var avgResponseTime float64
		if err := rows.Scan(&hour, &count, &avgResponseTime); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"hour":              hour,
			"count":             count,
			"avg_response_time": avgResponseTime,
		})
	}

	return results, nil
}

// Deletes logs older than the specified time
func (r *RequestLogRepository) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.DB.WithContext(ctx).
		Where("timestamp < ?", before).
		Delete(&models.RequestLog{})

	return result.RowsAffected, result.Error
}
