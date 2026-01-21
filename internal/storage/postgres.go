package storage

import (
	"fmt"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/models"
	"golang.org/x/net/context"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Postgres struct {
	DB *gorm.DB
}

// dsn - Data Source Name
func NewPostgres(dsn string) (*Postgres, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &Postgres{DB: db}, nil
}

func (p *Postgres) Ping(ctx context.Context) error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.PingContext(ctx)
}

func (p *Postgres) AutoMigrate() error {
	return p.DB.AutoMigrate(
		&models.APIKey{},
		&models.RateLimitTier{},
	)
}

func (p *Postgres) Close() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func (p *Postgres) Transaction(fn func(*gorm.DB) error) error {
	return p.DB.Transaction(fn)
}
