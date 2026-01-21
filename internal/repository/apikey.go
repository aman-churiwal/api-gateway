package repository

import (
	"context"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/models"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIKeyRepository struct {
	db *storage.Postgres
}

func NewAPIKeyRepository(db *storage.Postgres) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	return r.db.DB.WithContext(ctx).Create(apiKey).Error
}

func (r *APIKeyRepository) FindByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := r.db.DB.WithContext(ctx).
		Where("key_hash = ? AND is_active = ?", hash, true).
		First(&apiKey).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &apiKey, err
}

func (r *APIKeyRepository) FindByID(ctx context.Context, id string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := r.db.DB.WithContext(ctx).
		Where("id = ?", id).
		First(&apiKey).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &apiKey, err
}

func (r *APIKeyRepository) List(ctx context.Context) ([]models.APIKey, error) {
	var keys []models.APIKey
	err := r.db.DB.WithContext(ctx).
		Order("created_at DESC").
		Find(&keys).Error

	return keys, err
}

func (r *APIKeyRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.db.DB.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.DB.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

func (r *APIKeyRepository) Delete(ctx context.Context, id string) error {
	return r.db.DB.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.APIKey{}).Error
}

func (r *APIKeyRepository) CountByTier(ctx context.Context, tier string) (int64, error) {
	var count int64
	err := r.db.DB.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("tier = ? AND is_active = ?", tier, true).
		Count(&count).Error

	return count, err
}
