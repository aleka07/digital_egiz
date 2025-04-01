package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digital-egiz/backend/internal/api"
	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to the configuration directory")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := utils.NewLogger(&cfg.Log)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signals
		logger.Info("Received signal, initiating shutdown", zap.String("signal", sig.String()))
		cancel()
	}()

	// Initialize database
	database, err := db.NewDatabase(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	logger.Info("Database initialized")

	// Initialize service provider
	serviceProvider := services.NewServiceProvider(logger, cfg, database)
	if err := serviceProvider.Initialize(ctx); err != nil {
		logger.Fatal("Failed to initialize services", zap.Error(err))
	}
	logger.Info("Service provider initialized")

	// Create API router
	router := api.NewRouter(logger, cfg, database, serviceProvider)

	// Create HTTP server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", zap.String("address", serverAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Wait for cancellation signal
	<-ctx.Done()
	logger.Info("Shutting down server")

	// Create a timeout context for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown services
	if err := serviceProvider.Shutdown(); err != nil {
		logger.Error("Error during service shutdown", zap.Error(err))
	}

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}
