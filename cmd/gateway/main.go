package main

import (
	"log"

	"github.com/aman-churiwal/api-gateway/internal/config"
	"github.com/aman-churiwal/api-gateway/internal/server"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}
	srv := server.New(cfg)
	if err := srv.Run(":" + cfg.Port); err != nil {
		log.Fatal("Server failed: ", err)
	}
}
