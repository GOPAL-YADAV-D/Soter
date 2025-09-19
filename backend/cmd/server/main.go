package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/GOPAL-YADAV-D/Soter/graph"
	"github.com/GOPAL-YADAV-D/Soter/graph/generated"
	"github.com/GOPAL-YADAV-D/Soter/internal/config"
	"github.com/GOPAL-YADAV-D/Soter/internal/database"
	"github.com/GOPAL-YADAV-D/Soter/internal/handlers"
	"github.com/GOPAL-YADAV-D/Soter/internal/middleware"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Configure logging
	setupLogging(cfg.LogLevel)

	// Initialize database
	db, err := database.NewConnection(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Setup Gin router
	router := setupRouter(cfg, db)

	// Create HTTP server
	srv := &http.Server{
		Addr:    cfg.Host + ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logrus.WithField("address", srv.Addr).Info("Starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatal("Server forced to shutdown")
	}

	logrus.Info("Server exited")
}

func setupLogging(level string) {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.WithError(err).Warn("Invalid log level, defaulting to info")
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
}

func setupRouter(cfg *config.Config, db *database.DB) *gin.Engine {
	// Set Gin mode
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.PrometheusMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(gin.Recovery())

	// Health check endpoint
	healthHandler := handlers.NewHealthHandler(db)
	router.GET("/healthz", healthHandler.HealthCheck)

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// GraphQL endpoints
	resolver := &graph.Resolver{
		DB: db,
	}
	
	gqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	
	router.POST("/query", gin.WrapH(gqlHandler))
	router.GET("/playground", gin.WrapH(playground.Handler("GraphQL playground", "/query")))

	// API routes (future)
	api := router.Group("/api/v1")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	return router
}