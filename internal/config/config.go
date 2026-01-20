package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Server       ServerConfig    `json:"server"`
	Redis        RedisConfig     `json:"redis"`
	Services     []ServiceConfig `json:"services"`
	RateLimiters []RateLimiter   `json:"rate_limiters"`
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

type ServiceConfig struct {
	Path    string   `json:"path"`
	Targets []string `json:"targets"`
}

type RateLimiter struct {
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
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Server.Environment = env
	}
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}
}

func validate(cfg *Config) error {
	if cfg.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
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

	return nil
}

func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
