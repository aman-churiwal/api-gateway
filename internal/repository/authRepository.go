package repository

import (
	"context"

	"github.com/aman-churiwal/api-gateway/internal/models"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"gorm.io/gorm"
)

type AuthRepository struct {
	db *storage.Postgres
}

func NewUserRepository(db *storage.Postgres) *AuthRepository {
	return &AuthRepository{db: db}
}

// Inserts a new user into the database
func (r *AuthRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.DB.WithContext(ctx).Create(user).Error
}

// Retrieves user by email
func (r *AuthRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.DB.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &user, err
}

// Retrieves user by id
func (r *AuthRepository) FindById(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.db.DB.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &user, err
}

// Retrieves all users
func (r *AuthRepository) List(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := r.db.DB.WithContext(ctx).
		Order("created_at DESC").
		Find(&users).Error

	return users, err
}
