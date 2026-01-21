package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Server         ServerConfig      `json:"server"`
	Redis          RedisConfig       `json:"redis"`
	Database       DatabaseConfig    `json:"database"`
	JWT            JWTConfig         `json:"jwt"`
	Services       []ServiceConfig   `json:"services"`
	RateLimitTiers []RateLimiterTier `json:"rate_limit_tiers"`
}

type ServerConfig struct {
	Port        string `json:"port"`
	Environment string `json:"environment"` // Development or production
}

type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

type JWTConfig struct {
	Secret      string `json:"secret"`
	ExpiryHours int    `json:"expiry_hours"`
}

type ServiceConfig struct {
	Path    string   `json:"path"`
	Targets []string `json:"targets"`
}

type RateLimiterTier struct {
	Name              string `json:"name"`
	RequestsPerMinute int    `json:"requests_per_minute"`
	RequestsPerHour   int    `json:"requests_per_hour"`
	Algorithm         string `json:"algorithm"`
}

func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	applyEnvOverrides(&config)

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func applyEnvOverrides(cfg *Config) {
	// Server overrides
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Server.Environment = env
	}

	// Redis overrides
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}

	// Database overrides
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.Database.Host = host
	}
	if user := os.Getenv("DB_USER"); user != "" {
		cfg.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		cfg.Database.DBName = dbname
	}

	// JWT overrides
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWT.Secret = secret
	}
}

func validate(cfg *Config) error {
	if cfg.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}

	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if len(cfg.Services) == 0 {
		return fmt.Errorf("at least one service must be configured")
	}

	for i, svc := range cfg.Services {
		if svc.Path == "" {
			return fmt.Errorf("service %d: path is required", i)
		}
		if len(svc.Targets) == 0 {
			return fmt.Errorf("service %d: at least one target is required", i)
		}
	}

	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if cfg.JWT.ExpiryHours <= 0 {
		cfg.JWT.ExpiryHours = 24 // Default to 24 hours
	}

	return nil
}

// Returns the Redis address in host:port format
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
