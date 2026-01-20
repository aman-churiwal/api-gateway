package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/config"
	"github.com/aman-churiwal/api-gateway/internal/middleware"
	"github.com/aman-churiwal/api-gateway/internal/proxy"
	"github.com/aman-churiwal/api-gateway/internal/ratelimit"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router     *gin.Engine
	config     *config.Config
	redis      *storage.RedisClient
	proxies    map[string]*proxy.Proxy
	httpServer *http.Server
}

func New(cfg *config.Config, redis *storage.RedisClient) *Server {
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	s := &Server{
		router:  router,
		config:  cfg,
		redis:   redis,
		proxies: make(map[string]*proxy.Proxy),
	}

	// Initialize proxies for each configured service
	s.initializeProxies()

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	return s
}

func (s *Server) initializeProxies() {
	for _, svc := range s.config.Services {
		// Use the first target
		if len(svc.Targets) == 0 {
			log.Printf("Warning: Service %s has no targets configured", svc.Path)
			continue
		}

		p, err := proxy.New(svc.Targets[0])
		if err != nil {
			log.Printf("Failed to creater proxy for %s: %v", svc.Path, err)
			continue
		}

		s.proxies[svc.Path] = p
		log.Printf("Initialized proxy for %s -> %s", svc.Path, svc.Targets[0])
	}
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Recovery())
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.Logger())
	s.router.Use(middleware.CORS())

	rateLimiter := s.createRateLimiter()
	if rateLimiter != nil {
		s.router.Use(middleware.RateLimiter(rateLimiter))
	}
}

func (s *Server) createRateLimiter() *ratelimit.FixedWindowLimiter {
	limit := 100
	window := time.Minute

	if len(s.config.RateLimiters) > 0 {
		tier := s.config.RateLimiters[0]
		limit = tier.RequestsPerMinute
	}

	return ratelimit.NewFixedWindow(s.redis, limit, window)
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthCheck)

	s.setupProxyRoutes()

	admin := s.router.Group("/admin")
	{
		admin.GET("/status", s.adminStatus)
	}
}

func (s *Server) setupProxyRoutes() {
	for path, proxyInstance := range s.proxies {
		proxyPath := path
		p := proxyInstance

		s.router.Any(proxyPath+"/*proxyPath", func(c *gin.Context) {
			p.Handle(c)
		})

		s.router.Any(proxyPath, func(c *gin.Context) {
			p.Handle(c)
		})

		log.Printf("Registered proxy route: %s", proxyPath)
	}
}

func (s *Server) healthCheck(c *gin.Context) {
	redisHealthy := true

	if err := s.redis.Ping(c.Request.Context()); err != nil {
		redisHealthy = false
		log.Printf("Redis health check failed: %v", err)
	}

	status := "healthy"
	statusCode := http.StatusOK

	if !redisHealthy {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"service":   "api-gateway",
		"version":   "1.0.0",
		"timestamp": time.Now().Unix(),
		"checks": gin.H{
			"redis": redisHealthy,
		},
	})
}

func (s *Server) adminStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"gateway":  "running",
		"services": len(s.config.Services),
		"uptime":   time.Since(startTime).Seconds(),
	})
}

func (s *Server) Run(addr string) error {
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	log.Printf("Starting API Gateway on %s", addr)
	log.Printf("Environment: %s", s.config.Server.Environment)

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

var startTime = time.Now()
