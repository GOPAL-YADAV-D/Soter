package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// LocalStorageService implements storage operations using local filesystem
type LocalStorageService struct {
	basePath string
}

// NewLocalStorageService creates a new local storage service
func NewLocalStorageService(basePath string) (*LocalStorageService, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorageService{
		basePath: basePath,
	}, nil
}

// UploadFile uploads a file to local storage
func (s *LocalStorageService) UploadFile(ctx context.Context, storagePath string, content io.Reader, contentLength int64, contentType string) error {
	fullPath := filepath.Join(s.basePath, storagePath)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content to file
	_, err = io.Copy(file, content)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	logrus.Debugf("Successfully uploaded file to local storage: %s", fullPath)
	return nil
}

// DownloadFile downloads a file from local storage
func (s *LocalStorageService) DownloadFile(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, storagePath)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", storagePath)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// DeleteFile deletes a file from local storage
func (s *LocalStorageService) DeleteFile(ctx context.Context, storagePath string) error {
	fullPath := filepath.Join(s.basePath, storagePath)

	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logrus.Debugf("Successfully deleted file from local storage: %s", fullPath)
	return nil
}

// GenerateDownloadURL creates a temporary download URL for local files
// For local storage, this returns a relative path that the server can serve
func (s *LocalStorageService) GenerateDownloadURL(ctx context.Context, storagePath string, expiry time.Duration) (string, error) {
	fullPath := filepath.Join(s.basePath, storagePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", storagePath)
	}

	// For local storage, return a path that can be served by the application
	// In a real implementation, you might want to generate a temporary token
	return fmt.Sprintf("/api/files/download/%s", storagePath), nil
}

// GetFileInfo retrieves metadata about a local file
func (s *LocalStorageService) GetFileInfo(ctx context.Context, storagePath string) (*FileInfo, error) {
	fullPath := filepath.Join(s.basePath, storagePath)

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	info := &FileInfo{
		Size:         fileInfo.Size(),
		ContentType:  "application/octet-stream", // Default for local files
		LastModified: fileInfo.ModTime(),
		ETag:         fmt.Sprintf("%d-%d", fileInfo.Size(), fileInfo.ModTime().Unix()),
	}

	return info, nil
}
