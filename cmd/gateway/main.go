package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aman-churiwal/api-gateway/internal/config"
	"github.com/aman-churiwal/api-gateway/internal/server"
	"github.com/aman-churiwal/api-gateway/internal/storage"
	"github.com/joho/godotenv"
)

func main() {
	// Load env if it exists
	godotenv.Load()

	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	redis, err := storage.NewRedis(
		cfg.Redis.GetRedisAddr(),
		cfg.Redis.Password,
		cfg.Redis.DB,
	)

	if err != nil {
		log.Fatalf("Failed to connect to Reids: %v", err)
	}
	defer redis.Close()

	log.Println("Connected to redis successfully")

	// Create server
	srv := server.New(cfg, redis)

	go func() {
		addr := ":" + cfg.Server.Port
		if err := srv.Run(addr); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Sever Exited")
}
