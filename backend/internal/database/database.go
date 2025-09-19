package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/GOPAL-YADAV-D/Soter/internal/config"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type DB struct {
	*sql.DB
}

func NewConnection(cfg *config.Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logrus.Info("Successfully connected to PostgreSQL database")
	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck checks if the database is responding
func (db *DB) HealthCheck() error {
	ctx, cancel := getContextWithTimeout(5 * time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

// Helper function to create context with timeout
func getContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}