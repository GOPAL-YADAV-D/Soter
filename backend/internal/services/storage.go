package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// StorageService handles file operations (simplified for local development)
// In production, this would integrate with Azure Blob Storage
type StorageService struct {
	basePath string
}

// NewStorageService creates a new storage service
func NewStorageService(basePath string) (*StorageService, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &StorageService{
		basePath: basePath,
	}, nil
}

// UploadFile uploads a file to storage
func (s *StorageService) UploadFile(ctx context.Context, storagePath string, content io.Reader, contentLength int64, contentType string) error {
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

	return nil
}

// DownloadFile downloads a file from storage
func (s *StorageService) DownloadFile(ctx context.Context, storagePath string) (io.ReadCloser, int64, error) {
	fullPath := filepath.Join(s.basePath, storagePath)

	// Get file info
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get file info: %w", err)
	}

	// Open file
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}

	return file, info.Size(), nil
}

// DeleteFile deletes a file from storage
func (s *StorageService) DeleteFile(ctx context.Context, storagePath string) error {
	fullPath := filepath.Join(s.basePath, storagePath)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// FileExists checks if a file exists in storage
func (s *StorageService) FileExists(ctx context.Context, storagePath string) (bool, error) {
	fullPath := filepath.Join(s.basePath, storagePath)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// GetFileInfo retrieves file metadata from storage
func (s *StorageService) GetFileInfo(ctx context.Context, storagePath string) (*FileInfo, error) {
	fullPath := filepath.Join(s.basePath, storagePath)

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &FileInfo{
		StoragePath:  storagePath,
		Size:         info.Size(),
		ContentType:  "application/octet-stream", // Would be detected properly in production
		LastModified: info.ModTime(),
		ETag:         fmt.Sprintf("\"%d\"", info.ModTime().Unix()),
	}, nil
}

// GenerateDownloadURL generates a URL for downloading a file
func (s *StorageService) GenerateDownloadURL(ctx context.Context, storagePath string, expiryMinutes int) (string, error) {
	// For local development, return a simple file path
	// In production, this would generate a proper SAS token
	return fmt.Sprintf("/api/v1/files/download/%s", storagePath), nil
}

// ListFiles lists files in storage with prefix
func (s *StorageService) ListFiles(ctx context.Context, prefix string, maxResults int32) ([]string, error) {
	prefixPath := filepath.Join(s.basePath, prefix)
	baseDir := s.basePath
	if prefix != "" {
		baseDir = filepath.Dir(prefixPath)
	}

	var files []string
	count := int32(0)

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(s.basePath, path)
			if err != nil {
				return err
			}

			// Convert to forward slashes for consistency
			relPath = strings.ReplaceAll(relPath, "\\", "/")

			if prefix == "" || strings.HasPrefix(relPath, prefix) {
				files = append(files, relPath)
				count++

				if maxResults > 0 && count >= maxResults {
					return filepath.SkipDir
				}
			}
		}

		return nil
	})

	return files, err
}

// CopyFile creates a copy of a file
func (s *StorageService) CopyFile(ctx context.Context, sourceStoragePath, destStoragePath string) error {
	sourcePath := filepath.Join(s.basePath, sourceStoragePath)
	destPath := filepath.Join(s.basePath, destStoragePath)

	// Ensure destination directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

// FileInfo represents file metadata
type FileInfo struct {
	StoragePath  string            `json:"storage_path"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	LastModified interface{}       `json:"last_modified"`
	ETag         string            `json:"etag"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}
