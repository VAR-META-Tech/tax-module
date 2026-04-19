package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"tax-module/internal/config"
	"tax-module/internal/handler"
	"tax-module/internal/integration"
	"tax-module/internal/logger"
	"tax-module/internal/repository"
	"tax-module/internal/repository/postgres"
	"tax-module/internal/service"
	"tax-module/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Set Gin mode before creating router
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	log := logger.New(cfg.Log)

	// Database
	ctx := context.Background()
	dbPool, err := repository.NewPostgresPool(ctx, cfg.Database, &log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer dbPool.Close()

	// Services
	invoiceRepo := postgres.NewInvoiceRepo(dbPool, &log)
	tokenRepo := postgres.NewAccessTokenRepo(dbPool, &log)

	// Viettel SInvoice integration
	viettelClient := integration.NewViettelClient(cfg.Viettel, tokenRepo, &log)
	viettelPublisher := integration.NewViettelPublisher(viettelClient, cfg.Viettel, cfg.Seller, &log)

	// MISA MeInvoice integration
	misaClient := integration.NewMISAClient(cfg.MISA, tokenRepo, &log)
	misaPublisher := integration.NewMISAPublisher(misaClient, cfg.MISA, &log)

	// Multi-provider dispatcher — routes by invoice.Provider
	publisher := integration.NewDispatchingPublisher(cfg.Provider.Default, viettelPublisher, misaPublisher)

	// Worker pool
	workerPool := worker.NewPool(cfg.Worker, publisher, invoiceRepo, &log)
	workerPool.Start(ctx)
	enqueuer := worker.NewAdapter(workerPool)

	invoiceSvc := service.NewInvoiceService(invoiceRepo, publisher, enqueuer, &log)

	router := handler.NewRouter(&log, dbPool, invoiceSvc, cfg.Provider.Default)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info().Msgf("Listening to [:%d]...", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced shutdown")
	}

	// TODO: Shutdown worker pool (Part 7)
	workerPool.Shutdown()

	log.Info().Msg("Server is gracefully stopped")
}
