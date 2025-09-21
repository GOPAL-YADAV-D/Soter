package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/GOPAL-YADAV-D/Soter/internal/repository"
	"github.com/GOPAL-YADAV-D/Soter/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FileHandler struct {
	fileRepo          *repository.FileRepository
	userRepo          *repository.UserRepository
	orgRepo           *repository.OrganizationRepository
	groupRepo         *repository.GroupRepository
	fileUploadService *services.FileUploadService
	validationService *services.FileValidationService
}

func NewFileHandler(
	fileRepo *repository.FileRepository,
	userRepo *repository.UserRepository,
	orgRepo *repository.OrganizationRepository,
	groupRepo *repository.GroupRepository,
	uploadService *services.FileUploadService,
	validationService *services.FileValidationService,
) *FileHandler {
	return &FileHandler{
		fileRepo:          fileRepo,
		userRepo:          userRepo,
		orgRepo:           orgRepo,
		groupRepo:         groupRepo,
		fileUploadService: uploadService,
		validationService: validationService,
	}
}

// CreateUploadSession creates a new upload session for multiple files
func (h *FileHandler) CreateUploadSession(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Files      []services.FileInput `json:"files" binding:"required"`
		TotalBytes int64                `json:"totalBytes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user for organization context
	user, err := h.userRepo.GetByID(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check storage quota
	totalMB := int(req.TotalBytes / (1024 * 1024))
	hasQuota, err := h.orgRepo.CheckStorageQuota(c.Request.Context(), user.OrganizationID, totalMB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check storage quota"})
		return
	}

	if !hasQuota {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": "Storage quota exceeded",
			"code":  "STORAGE_QUOTA_EXCEEDED",
		})
		return
	}

	// Create upload session
	sessionReq := &services.CreateUploadSessionRequest{
		UserID:     userID.(uuid.UUID),
		Files:      req.Files,
		TotalBytes: req.TotalBytes,
	}

	session, err := h.fileUploadService.CreateUploadSession(c.Request.Context(), sessionReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload session"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// UploadFile handles individual file upload within a session
func (h *FileHandler) UploadFile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from request"})
		return
	}
	defer file.Close()

	sessionToken := c.Param("sessionToken")
	folderPath := c.PostForm("folderPath")
	userFilename := c.PostForm("userFilename")

	if sessionToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session token is required"})
		return
	}

	if folderPath == "" {
		folderPath = "/"
	}

	if userFilename == "" {
		userFilename = header.Filename
	}

	sessionUUID, err := uuid.Parse(sessionToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}

	// Get user for organization context
	user, err := h.userRepo.GetByID(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get user's default group for file permissions
	userGroups, err := h.groupRepo.GetUserGroups(c.Request.Context(), user.ID, user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user groups"})
		return
	}

	var primaryGroupID *uuid.UUID
	if len(userGroups) > 0 {
		primaryGroupID = &userGroups[0].ID // Use first group as primary
	}

	// Create upload request
	uploadReq := &services.UploadFileRequest{
		UserID:          userID.(uuid.UUID),
		SessionToken:    &sessionUUID,
		Filename:        header.Filename,
		UserFilename:    userFilename,
		MimeType:        header.Header.Get("Content-Type"),
		FolderPath:      folderPath,
		Content:         file,
		FileSize:        header.Size,
		OwnerID:         &user.ID,
		PrimaryGroupID:  primaryGroupID,
		FilePermissions: 644, // Default permissions: rw-r--r--
	}

	// Upload file
	response, err := h.fileUploadService.UploadFile(c.Request.Context(), uploadReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Upload failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CompleteUploadSession marks an upload session as complete
func (h *FileHandler) CompleteUploadSession(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	sessionTokenStr := c.Param("sessionToken")
	sessionToken, err := uuid.Parse(sessionTokenStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}

	err = h.fileUploadService.CompleteUploadSession(c.Request.Context(), sessionToken, userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete upload session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Upload session completed successfully"})
}

// GetFiles returns list of files for the user with permission checking
func (h *FileHandler) GetFiles(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get pagination parameters
	page := 1
	limit := 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Get user for organization context
	user, err := h.userRepo.GetByID(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get files with permissions
	files, err := h.fileRepo.GetUserFilesWithPermissions(c.Request.Context(), user.ID, user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve files"})
		return
	}

	// Simple pagination (in production, this should be done in the database)
	total := len(files)
	start := (page - 1) * limit
	end := start + limit

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedFiles := files[start:end]

	c.JSON(http.StatusOK, gin.H{
		"files": paginatedFiles,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + limit - 1) / limit,
		},
	})
}

// GetFileMetadata returns detailed metadata for a specific file
func (h *FileHandler) GetFileMetadata(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	fileIDStr := c.Param("fileId")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	// Get user file
	userFile, err := h.fileRepo.GetUserFile(c.Request.Context(), userID.(uuid.UUID), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Check if user has access to this file
	if userFile.UserID != userID.(uuid.UUID) {
		// TODO: Check group permissions
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get file details
	file, err := h.fileRepo.GetByID(c.Request.Context(), userFile.FileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve file details"})
		return
	}

	// Check if file is deduplicated
	// TODO: Implement deduplication check

	metadata := &models.FileMetadata{
		ID:            userFile.ID,
		UserFilename:  userFile.UserFilename,
		OriginalName:  file.Filename,
		FileSize:      file.FileSize,
		ContentType:   file.DetectedMimeType,
		UploadedAt:    userFile.CreatedAt,
		DownloadCount: userFile.DownloadCount,
		LastAccessed:  userFile.LastAccessed,
		FolderPath:    userFile.FolderPath,
		IsDeduped:     false, // TODO: Calculate
		Owner:         file.Owner,
		Permissions:   models.ParseLinuxPermissions(file.FilePermissions, userID.(uuid.UUID), file.OwnerID, file.PrimaryGroupID, []uuid.UUID{}), // TODO: Get user's group IDs

		// Additional fields for frontend compatibility
		Hash:           file.ContentHash,
		GroupName:      "",                     // TODO: Get primary group name
		DuplicateCount: 0,                      // TODO: Calculate actual duplicates
		IsOriginal:     true,                   // TODO: Determine if this is the original
		RelatedFiles:   []models.RelatedFile{}, // TODO: Get related files
		Tags:           []string{},             // TODO: Get actual tags
	}

	c.JSON(http.StatusOK, metadata)
}

// DownloadFile handles file downloads with permission checking
func (h *FileHandler) DownloadFile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	fileIDStr := c.Param("fileId")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	// Get user file
	userFile, err := h.fileRepo.GetUserFile(c.Request.Context(), userID.(uuid.UUID), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Check if user has access to this file
	if userFile.UserID != userID.(uuid.UUID) {
		// TODO: Check group permissions
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get file details
	file, err := h.fileRepo.GetByID(c.Request.Context(), userFile.FileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve file details"})
		return
	}

	// Update download count
	go h.fileRepo.UpdateDownloadCount(context.Background(), userFile.ID)

	// TODO: Implement actual file streaming from storage
	// For now, return file information
	c.JSON(http.StatusOK, gin.H{
		"message":     "Download would start here",
		"filename":    userFile.UserFilename,
		"contentType": file.DetectedMimeType,
		"fileSize":    file.FileSize,
		"storagePath": file.StoragePath,
	})
}

// DeleteFile handles file deletion (soft delete)
func (h *FileHandler) DeleteFile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	fileIDStr := c.Param("fileId")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	// Get user file
	userFile, err := h.fileRepo.GetUserFile(c.Request.Context(), userID.(uuid.UUID), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Check if user has access to this file
	if userFile.UserID != userID.(uuid.UUID) {
		// TODO: Check group permissions for delete access
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Soft delete the user file
	err = h.fileRepo.SoftDeleteUserFile(c.Request.Context(), userFile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// GetUploadProgress returns the progress of an upload session
func (h *FileHandler) GetUploadProgress(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	sessionTokenStr := c.Param("sessionToken")
	sessionToken, err := uuid.Parse(sessionTokenStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}

	progress, err := h.fileUploadService.GetUploadProgress(c.Request.Context(), sessionToken, userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload session not found"})
		return
	}

	c.JSON(http.StatusOK, progress)
}
