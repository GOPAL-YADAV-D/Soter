package services

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/GOPAL-YADAV-D/Soter/internal/repository"
)

// FileInput represents a file input for upload session creation
type FileInput struct {
	Filename    string `json:"filename"`
	MimeType    string `json:"mimeType"`
	FileSize    int64  `json:"fileSize"`
	FolderPath  string `json:"folderPath"`
	ContentHash string `json:"contentHash"`
}

// CreateUploadSessionRequest represents a request to create an upload session
type CreateUploadSessionRequest struct {
	UserID     uuid.UUID   `json:"userId"`
	Files      []FileInput `json:"files"`
	TotalBytes int64       `json:"totalBytes"`
}

// CreateUploadSessionResponse represents the response from creating an upload session
type CreateUploadSessionResponse struct {
	SessionToken   uuid.UUID `json:"sessionToken"`
	TotalFiles     int       `json:"totalFiles"`
	TotalBytes     int64     `json:"totalBytes"`
	DuplicateFiles int       `json:"duplicateFiles"`
}

// UploadFileRequest represents a request to upload a file
type UploadFileRequest struct {
	UserID          uuid.UUID  `json:"userId"`
	SessionToken    *uuid.UUID `json:"sessionToken,omitempty"`
	Filename        string     `json:"filename"`
	UserFilename    string     `json:"userFilename"`
	MimeType        string     `json:"mimeType"`
	FolderPath      string     `json:"folderPath"`
	Content         io.Reader  `json:"-"`
	FileSize        int64      `json:"fileSize"`
	OwnerID         *uuid.UUID `json:"ownerId,omitempty"`
	PrimaryGroupID  *uuid.UUID `json:"primaryGroupId,omitempty"`
	FilePermissions int        `json:"filePermissions"`
}

// UploadFileResponse represents the response from uploading a file
type UploadFileResponse struct {
	FileID       uuid.UUID `json:"fileId"`
	UserFileID   uuid.UUID `json:"userFileId"`
	IsExisting   bool      `json:"isExisting"`
	SavingsBytes int64     `json:"savingsBytes"`
	Warnings     []string  `json:"warnings,omitempty"`
}

// UploadProgressResponse represents upload progress information
type UploadProgressResponse struct {
	SessionToken   uuid.UUID `json:"sessionToken"`
	TotalFiles     int       `json:"totalFiles"`
	CompletedFiles int       `json:"completedFiles"`
	TotalBytes     int64     `json:"totalBytes"`
	UploadedBytes  int64     `json:"uploadedBytes"`
	Status         string    `json:"status"`
	Progress       float64   `json:"progress"`
}

// FileUploadService orchestrates file upload with deduplication
type FileUploadService struct {
	fileRepo          *repository.FileRepository
	storageService    *StorageService
	validationService *FileValidationService
	userRepo          *repository.UserRepository
}

// NewFileUploadService creates a new file upload service
func NewFileUploadService(
	fileRepo *repository.FileRepository,
	storageService *StorageService,
	validationService *FileValidationService,
	userRepo *repository.UserRepository,
) *FileUploadService {
	return &FileUploadService{
		fileRepo:          fileRepo,
		storageService:    storageService,
		validationService: validationService,
		userRepo:          userRepo,
	}
}

// CreateUploadSession creates a new upload session for tracking progress
func (s *FileUploadService) CreateUploadSession(ctx context.Context, req *CreateUploadSessionRequest) (*CreateUploadSessionResponse, error) {
	// Calculate total bytes and detect duplicates
	totalBytes := req.TotalBytes
	duplicateFiles := 0

	for _, fileInput := range req.Files {
		// Check if file already exists
		existing, _ := s.fileRepo.GetByContentHash(ctx, fileInput.ContentHash)
		if existing != nil {
			duplicateFiles++
		}
	}

	session := &models.UploadSession{
		UserID:     req.UserID,
		TotalFiles: len(req.Files),
		TotalBytes: totalBytes,
		Status:     "active",
	}

	// Generate session token
	sessionToken := uuid.New()

	return &CreateUploadSessionResponse{
		SessionToken:   sessionToken,
		TotalFiles:     session.TotalFiles,
		TotalBytes:     session.TotalBytes,
		DuplicateFiles: duplicateFiles,
	}, nil
}

// UploadFile processes an individual file upload
func (s *FileUploadService) UploadFile(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, error) {
	// 1. Validate file content and security
	validation, err := s.validationService.ValidateFile(ctx, req.Filename, req.MimeType, req.Content)
	if err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	if !validation.IsValid {
		return nil, fmt.Errorf("file validation failed: %v", validation.Errors)
	}

	// 2. Check for existing file (deduplication logic)
	existingFile, err := s.fileRepo.GetByContentHash(ctx, validation.ContentHash)

	var file *models.File
	var isExisting bool
	var savingsBytes int64

	if existingFile != nil {
		// File already exists - deduplication saves storage
		file = existingFile
		isExisting = true
		savingsBytes = validation.FileSize
	} else {
		// New file - upload to storage
		storagePath := s.generateStoragePath(validation.ContentHash)

		err = s.storageService.UploadFile(ctx, storagePath, req.Content,
			validation.FileSize, validation.DetectedMimeType)
		if err != nil {
			return nil, fmt.Errorf("failed to upload file to storage: %w", err)
		}

		// Create file record
		file = &models.File{
			ContentHash:      validation.ContentHash,
			Filename:         req.Filename,
			OriginalMimeType: req.MimeType,
			DetectedMimeType: validation.DetectedMimeType,
			FileSize:         validation.FileSize,
			StoragePath:      storagePath,
			OwnerID:          req.OwnerID,
			PrimaryGroupID:   req.PrimaryGroupID,
			FilePermissions:  req.FilePermissions,
		}

		file, err = s.fileRepo.Create(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to create file record: %w", err)
		}
	}

	// 3. Create user file reference
	userFile := &models.UserFile{
		UserID:       req.UserID,
		FileID:       file.ID,
		UserFilename: req.UserFilename,
		FolderPath:   req.FolderPath,
	}

	userFile, err = s.fileRepo.CreateUserFile(ctx, userFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create user file reference: %w", err)
	}

	return &UploadFileResponse{
		FileID:       file.ID,
		UserFileID:   userFile.ID,
		IsExisting:   isExisting,
		SavingsBytes: savingsBytes,
		Warnings:     validation.Warnings,
	}, nil
}

// CompleteUploadSession marks an upload session as complete
func (s *FileUploadService) CompleteUploadSession(ctx context.Context, sessionToken, userID uuid.UUID) error {
	// Implementation would mark session as complete in database
	// For now, just return success
	return nil
}

// GetUploadProgress returns the progress of an upload session
func (s *FileUploadService) GetUploadProgress(ctx context.Context, sessionToken, userID uuid.UUID) (*UploadProgressResponse, error) {
	// Implementation would fetch progress from database
	// For now, return a placeholder
	return &UploadProgressResponse{
		SessionToken:   sessionToken,
		TotalFiles:     0,
		CompletedFiles: 0,
		TotalBytes:     0,
		UploadedBytes:  0,
		Status:         "completed",
		Progress:       100.0,
	}, nil
}

// generateStoragePath generates a hierarchical path for file storage
func (s *FileUploadService) generateStoragePath(contentHash string) string {
	// Create hierarchical path: files/ab/cd/abcd1234567890...
	return fmt.Sprintf("files/%s/%s/%s",
		contentHash[:2],
		contentHash[2:4],
		contentHash)
}
