package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/GOPAL-YADAV-D/Soter/internal/config"
	"github.com/sirupsen/logrus"
)

// StorageInterface defines the interface for storage operations
type StorageInterface interface {
	UploadFile(ctx context.Context, storagePath string, content io.Reader, contentLength int64, contentType string) error
	DownloadFile(ctx context.Context, storagePath string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, storagePath string) error
	GenerateDownloadURL(ctx context.Context, storagePath string, expiry time.Duration) (string, error)
	GetFileInfo(ctx context.Context, storagePath string) (*FileInfo, error)
}

// StorageService is a wrapper that delegates to either Azure or local storage
type StorageService struct {
	implementation StorageInterface
	environment    string
}

// NewStorageService creates a new storage service based on configuration
func NewStorageService(cfg *config.Config) (*StorageService, error) {
	var impl StorageInterface
	var err error

	switch cfg.StorageEnvironment {
	case "production":
		// Use Azure Blob Storage (simplified for now)
		simpleImpl, err := NewAzureStorageService(
			cfg.AzureStorageAccount,
			cfg.AzureStorageKey,
			cfg.AzureStorageContainer,
			cfg.AzureStorageEndpoint,
		)
		if err != nil {
			// Fallback to local storage if Azure fails
			logrus.Warnf("Azure storage initialization failed, falling back to local storage: %v", err)
			impl, err = NewLocalStorageService(cfg.LocalStoragePath)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize fallback local storage: %w", err)
			}
			logrus.Info("Using local filesystem storage as fallback")
		} else {
			impl = simpleImpl
			logrus.Info("Initialized Azure Blob Storage service (placeholder)")
		}

	case "local":
		fallthrough
	default:
		// Use local filesystem storage
		impl, err = NewLocalStorageService(cfg.LocalStoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize local storage: %w", err)
		}
		logrus.Info("Initialized local filesystem storage service")
	}

	return &StorageService{
		implementation: impl,
		environment:    cfg.StorageEnvironment,
	}, nil
}

// UploadFile uploads a file to the configured storage backend
func (s *StorageService) UploadFile(ctx context.Context, storagePath string, content io.Reader, contentLength int64, contentType string) error {
	return s.implementation.UploadFile(ctx, storagePath, content, contentLength, contentType)
}

// DownloadFile downloads a file from the configured storage backend
func (s *StorageService) DownloadFile(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	return s.implementation.DownloadFile(ctx, storagePath)
}

// DeleteFile deletes a file from the configured storage backend
func (s *StorageService) DeleteFile(ctx context.Context, storagePath string) error {
	return s.implementation.DeleteFile(ctx, storagePath)
}

// GenerateDownloadURL creates a secure, time-limited download URL
func (s *StorageService) GenerateDownloadURL(ctx context.Context, storagePath string, expiry time.Duration) (string, error) {
	return s.implementation.GenerateDownloadURL(ctx, storagePath, expiry)
}

// GetFileInfo retrieves metadata about a file
func (s *StorageService) GetFileInfo(ctx context.Context, storagePath string) (*FileInfo, error) {
	return s.implementation.GetFileInfo(ctx, storagePath)
}

// GetStorageEnvironment returns the current storage environment
func (s *StorageService) GetStorageEnvironment() string {
	return s.environment
}

// IsProduction returns true if using production storage (Azure)
func (s *StorageService) IsProduction() bool {
	return s.environment == "production"
}

// FileExists checks if a file exists in storage
func (s *StorageService) FileExists(ctx context.Context, storagePath string) (bool, error) {
	_, err := s.implementation.GetFileInfo(ctx, storagePath)
	if err != nil {
		return false, nil // File doesn't exist or error occurred
	}
	return true, nil
}
