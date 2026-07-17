package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/moneymate-2026/moneymate-backend/auth/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("auth service exited with error: %v", err)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log.Printf("starting auth service — env=%s grpc=%s", cfg.Env, cfg.Server.GRPCAddr)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connectPostgres(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	redisClient, err := connectRedis(cfg)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	// c := buildContainer(cfg, pool, redisClient)

	grpcServer := grpc.NewServer()

	listener, err := net.Listen("tcp", cfg.Server.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("grpc server listening on %s", cfg.Server.GRPCAddr)
		if err := grpcServer.Serve(listener); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		log.Printf("server error: %v", err)
	}

	grpcServer.GracefulStop()
	log.Println("auth service stopped cleanly")
	return nil
}