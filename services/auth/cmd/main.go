package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/moneymate-2026/moneymate-backend/auth/config"
	"github.com/moneymate-2026/moneymate-backend/auth/internal/app"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to Load Config: %v", err)
	}
	application, err := app.Build(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	go func() {
		addr := "0.0.0.0:" + cfg.Server.HTTPAddr

		if cfg.Server.HTTPAddr == "" {
			addr = "0.0.0.0:8081" 
		}
		log.Printf("Auth service starting on %s", addr)

		if err := application.Server.Listen(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Println("Shutdown signal received, gracefully shutting down...")

	if err := application.Server.Shutdown(); err != nil {
		log.Printf("Error shutting down Fiber server: %v", err)
	}
	application.Close()

	log.Println("Auth service stopped cleanly ✅")
}
