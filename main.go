package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"rpc-load-balancer/internal/config"
	"rpc-load-balancer/internal/gateway"
	"syscall"
	"time"
)

const configFilename = "config.yaml"

func main() {
	log.Println("Starting RPC Gateway...")

	// Load configuration from YAML file
	if err := config.LoadConfig(configFilename); err != nil {
		log.Fatalf("Fatal: Failed to load configuration: %v", err)
	}

	// Initialize the gateway using the loaded config
	gw, err := gateway.NewGateway(&config.AppConfig)
	if err != nil {
		log.Fatalf("Fatal: Failed to initialize gateway: %v", err)
	}

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the periodic health checker
	gw.StartChecker(ctx)

	// Setup the HTTP server
	server := &http.Server{
		Addr:    config.AppConfig.GatewayPort, // Use port from config
		Handler: gw.ProxyHandler(),
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Gateway listening on http://localhost%s", config.AppConfig.GatewayPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fatal: Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %v. Shutting down server...", sig)

	// Signal the checker goroutine to stop
	cancel()

	// Shutdown the server gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped.")
}
