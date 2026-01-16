package server

import (
	"log"

	"github.com/aman-churiwal/api-gateway/internal/config"
	"github.com/aman-churiwal/api-gateway/internal/proxy"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router  *gin.Engine
	config  *config.Config
	proxies map[string]*proxy.Proxy
}

func New(cfg *config.Config) *Server {
	router := gin.Default()

	s := &Server{
		router:  router,
		config:  cfg,
		proxies: make(map[string]*proxy.Proxy),
	}

	for _, svc := range cfg.Services {
		p, err := proxy.New(svc.Target)
		if err != nil {
			log.Printf("Failed to create proxy for %s: %v", svc.Path, err)
		}
		s.proxies[svc.Path] = p
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthCheck)

	for path, proxy := range s.proxies {
		s.router.Any(path+"/*proxyPath", proxy.Handle)
	}
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "api-gateway",
	})
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
