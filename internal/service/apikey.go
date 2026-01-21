package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/models"
	"github.com/aman-churiwal/api-gateway/internal/repository"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/google/uuid"
)

type APIKeyService struct {
	db         *storage.Postgres
	repository *repository.APIKeyRepository
	redis      *storage.RedisClient
}

func NewAPIKeyService(db *storage.Postgres, repo *repository.APIKeyRepository, redis *storage.RedisClient) *APIKeyService {
	return &APIKeyService{
		db:         db,
		repository: repo,
		redis:      redis,
	}
}

func (s *APIKeyService) Create(ctx context.Context, name, createdBy, tier string) (string, error) {
	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Creating key with prefix
	key := "gw_" + base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	// Save to database
	apiKey := models.APIKey{
		KeyHash:   keyHash,
		Name:      name,
		CreatedBy: createdBy,
		Tier:      tier,
		IsActive:  true,
	}

	if err := s.repository.Create(ctx, &apiKey); err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}

	// Return plain key (only time it's visible)
	return key, nil
}

func (s *APIKeyService) Validate(ctx context.Context, key string) (*models.APIKey, error) {
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	// Check cache first
	cacheKey := fmt.Sprintf("apikey:cache:%s", keyHash)
	cached, err := s.redis.Get(ctx, cacheKey)

	if err == nil && cached != "" {
		// Cache hit
		var apiKey models.APIKey
		if err := json.Unmarshal([]byte(cached), &apiKey); err == nil {
			return &apiKey, nil
		}
	}

	// Cache miss - query database
	apiKey, err := s.repository.FindByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	if apiKey == nil {
		return nil, nil
	}

	// Cache the result
	apiKeyJSON, _ := json.Marshal(apiKey)
	s.redis.Set(ctx, cacheKey, apiKeyJSON, 5*time.Minute)

	return apiKey, nil
}

func (s *APIKeyService) Get(ctx context.Context, id string) (*models.APIKey, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *APIKeyService) List(ctx context.Context) ([]models.APIKey, error) {
	return s.repository.List(ctx)
}

func (s *APIKeyService) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	// Invalidate cache if tier or is_active is updated
	if _, hasTier := updates["tier"]; hasTier {
		s.invalidateCache(ctx, id)
	}
	if _, hasActive := updates["is_active"]; hasActive {
		s.invalidateCache(ctx, id)
	}

	return s.repository.Update(ctx, id, updates)
}

func (s *APIKeyService) Delete(ctx context.Context, id string) error {
	// Invalidate cache
	s.invalidateCache(ctx, id)

	return s.repository.Delete(ctx, id)
}

func (s *APIKeyService) UpdateLastUsed(ctx context.Context, id uuid.UUID) {
	// Update asynchronously - don't block request
	s.repository.UpdateLastUsed(ctx, id)
}

func (s *APIKeyService) invalidateCache(ctx context.Context, id string) {
	// Get the key to find its hash
	apiKey, err := s.repository.FindByID(ctx, id)
	if err != nil || apiKey == nil {
		return
	}

	cacheKey := fmt.Sprintf("apikey:cache:%s", apiKey.KeyHash)
	s.redis.Set(ctx, cacheKey, "", 0) // Delete by setting empty with no TTL
}
