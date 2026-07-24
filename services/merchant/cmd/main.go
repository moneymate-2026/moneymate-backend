package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/moneymate-2026/moneymate-backend/services/merchant/config"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/internal/app"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to Load Config: %v", err)
	}

	merchantApp, err := app.Build(cfg)
	if err != nil {
		log.Fatalf("Failed to build app: %v", err)
	}
	defer merchantApp.Close()

	go func() {
		if err := merchantApp.Run(); err != nil {
			log.Fatalf("App run failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Println("Shutdown signal received, gracefully shutting down...")
	merchantApp.Close()
	log.Println("Merchant service stopped cleanly ✅")
}
