package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/GOPAL-YADAV-D/Soter/internal/models"
)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db: db}
}

// CreateUploadSession creates a new upload session for tracking progress
func (r *FileRepository) CreateUploadSession(ctx context.Context, userID uuid.UUID, input models.UploadSessionInput) (*models.UploadSession, error) {
	sessionToken := generateSessionToken()

	session := &models.UploadSession{
		UserID:       userID,
		SessionToken: sessionToken,
		TotalFiles:   len(input.Files),
		TotalBytes:   input.TotalBytes,
		Status:       models.UploadStatusPending,
		StartedAt:    time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create upload session: %w", err)
	}

	return session, nil
}

// GetUploadSession retrieves an upload session by token
func (r *FileRepository) GetUploadSession(ctx context.Context, sessionToken string) (*models.UploadSession, error) {
	var session models.UploadSession
	if err := r.db.WithContext(ctx).Where("session_token = ?", sessionToken).First(&session).Error; err != nil {
		return nil, fmt.Errorf("upload session not found: %w", err)
	}
	return &session, nil
}

// UpdateUploadSessionProgress updates upload session progress
func (r *FileRepository) UpdateUploadSessionProgress(ctx context.Context, sessionID uuid.UUID, uploadedBytes int64, status string) error {
	updates := map[string]interface{}{
		"uploaded_bytes": uploadedBytes,
		"status":         status,
		"updated_at":     time.Now(),
	}

	if status == models.UploadStatusCompleted || status == models.UploadStatusFailed {
		updates["completed_at"] = time.Now()
	}

	return r.db.WithContext(ctx).Model(&models.UploadSession{}).
		Where("id = ?", sessionID).Updates(updates).Error
}

// CheckFileExists checks if a file with the given hash already exists
func (r *FileRepository) CheckFileExists(ctx context.Context, contentHash string) (*models.File, error) {
	var file models.File
	err := r.db.WithContext(ctx).Where("content_hash = ?", contentHash).First(&file).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // File doesn't exist
		}
		return nil, fmt.Errorf("error checking file existence: %w", err)
	}
	return &file, nil
}

// CreateFile creates a new file record
func (r *FileRepository) CreateFile(ctx context.Context, file *models.File) error {
	return r.db.WithContext(ctx).Create(file).Error
}

// CreateUserFileReference creates a reference from user to file (for deduplication)
func (r *FileRepository) CreateUserFileReference(ctx context.Context, userFile *models.UserFile) error {
	// Check if user already has a file with this name in this folder
	var existing models.UserFile
	err := r.db.WithContext(ctx).Where(
		"user_id = ? AND user_filename = ? AND folder_path = ? AND is_deleted = false",
		userFile.UserID, userFile.UserFilename, userFile.FolderPath,
	).First(&existing).Error

	if err == nil {
		return fmt.Errorf("file with name '%s' already exists in folder '%s'", userFile.UserFilename, userFile.FolderPath)
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("error checking existing file: %w", err)
	}

	// Create the new reference
	if err := r.db.WithContext(ctx).Create(userFile).Error; err != nil {
		return fmt.Errorf("failed to create user file reference: %w", err)
	}

	// Update user storage stats
	return r.UpdateUserStorageStats(ctx, userFile.UserID)
}

// GetUserFiles retrieves all files for a user with pagination
func (r *FileRepository) GetUserFiles(ctx context.Context, userID uuid.UUID, folderPath string, limit, offset int) ([]models.UserFile, error) {
	var userFiles []models.UserFile

	query := r.db.WithContext(ctx).
		Preload("File").
		Where("user_id = ? AND is_deleted = false", userID)

	if folderPath != "" {
		query = query.Where("folder_path = ?", folderPath)
	}

	err := query.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&userFiles).Error

	return userFiles, err
}

// GetUserFile retrieves a specific user file
func (r *FileRepository) GetUserFile(ctx context.Context, userID uuid.UUID, userFileID uuid.UUID) (*models.UserFile, error) {
	var userFile models.UserFile
	err := r.db.WithContext(ctx).
		Preload("File").
		Where("id = ? AND user_id = ? AND is_deleted = false", userFileID, userID).
		First(&userFile).Error

	if err != nil {
		return nil, fmt.Errorf("user file not found: %w", err)
	}

	return &userFile, nil
}

// DeleteUserFile soft deletes a user file reference
func (r *FileRepository) DeleteUserFile(ctx context.Context, userID uuid.UUID, userFileID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&models.UserFile{}).
		Where("id = ? AND user_id = ?", userFileID, userID).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("failed to delete user file: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user file not found")
	}

	// Update user storage stats
	return r.UpdateUserStorageStats(ctx, userID)
}

// GetUserStorageStats retrieves storage statistics for a user
func (r *FileRepository) GetUserStorageStats(ctx context.Context, userID uuid.UUID) (*models.UserStorageStats, error) {
	var stats models.UserStorageStats
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create initial stats if they don't exist
			stats = models.UserStorageStats{
				UserID:         userID,
				LastCalculated: time.Now(),
			}
			if createErr := r.db.WithContext(ctx).Create(&stats).Error; createErr != nil {
				return nil, fmt.Errorf("failed to create user storage stats: %w", createErr)
			}
		} else {
			return nil, fmt.Errorf("error getting user storage stats: %w", err)
		}
	}
	return &stats, nil
}

// UpdateUserStorageStats recalculates and updates user storage statistics
func (r *FileRepository) UpdateUserStorageStats(ctx context.Context, userID uuid.UUID) error {
	// Use raw SQL to call the database function
	return r.db.WithContext(ctx).Exec("SELECT update_user_storage_stats(?)", userID).Error
}

// CalculateContentHash calculates SHA-256 hash of file content
func (r *FileRepository) CalculateContentHash(reader io.Reader) (string, int64, error) {
	hash := sha256.New()
	size, err := io.Copy(hash, reader)
	if err != nil {
		return "", 0, fmt.Errorf("failed to calculate hash: %w", err)
	}

	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return hashString, size, nil
}

// CheckDeduplication checks if file can be deduplicated and returns result
func (r *FileRepository) CheckDeduplication(ctx context.Context, contentHash string, fileSize int64) (*models.DeduplicationResult, error) {
	existingFile, err := r.CheckFileExists(ctx, contentHash)
	if err != nil {
		return nil, fmt.Errorf("error checking deduplication: %w", err)
	}

	result := &models.DeduplicationResult{
		SavingsBytes: 0,
	}

	if existingFile != nil {
		result.IsExisting = true
		result.ExistingFileID = &existingFile.ID
		result.SavingsBytes = fileSize
		result.StoragePath = existingFile.StoragePath
	} else {
		result.IsExisting = false
		result.StoragePath = generateStoragePath(contentHash)
	}

	return result, nil
}

// ValidateFileUpload performs security validation on uploaded files
func (r *FileRepository) ValidateFileUpload(ctx context.Context, filename, declaredMimeType string, content io.Reader) (*models.FileValidationResult, error) {
	result := &models.FileValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Calculate content hash and size
	contentHash, fileSize, err := r.CalculateContentHash(content)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to process file content: %v", err))
		return result, nil
	}

	result.ContentHash = contentHash
	result.FileSize = fileSize

	// Validate file size
	if fileSize > models.MaxFileSize {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("File size (%d bytes) exceeds maximum allowed size (%d bytes)", fileSize, models.MaxFileSize))
	}

	if fileSize == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "File is empty")
	}

	// Validate filename
	if strings.TrimSpace(filename) == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "Filename cannot be empty")
	}

	// Check for dangerous file extensions
	ext := strings.ToLower(filepath.Ext(filename))
	dangerousExts := []string{".exe", ".bat", ".cmd", ".com", ".pif", ".scr", ".vbs", ".js", ".jar", ".sh", ".ps1"}
	for _, dangerousExt := range dangerousExts {
		if ext == dangerousExt {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("File type '%s' is not allowed for security reasons", ext))
			break
		}
	}

	// Detect actual MIME type (this would need a proper library like libmagic)
	detectedMimeType := detectMimeType(filename, declaredMimeType)
	result.DetectedMimeType = detectedMimeType

	// Validate MIME type consistency
	if declaredMimeType != "" && detectedMimeType != declaredMimeType {
		// For now, just warn - in production you'd want more sophisticated detection
		result.Warnings = append(result.Warnings, fmt.Sprintf("Declared MIME type (%s) differs from detected type (%s)", declaredMimeType, detectedMimeType))
	}

	return result, nil
}

// SearchUserFiles searches files by filename or content
func (r *FileRepository) SearchUserFiles(ctx context.Context, userID uuid.UUID, query string, limit, offset int) ([]models.UserFile, error) {
	var userFiles []models.UserFile

	searchQuery := "%" + strings.ToLower(query) + "%"

	err := r.db.WithContext(ctx).
		Preload("File").
		Where("user_id = ? AND is_deleted = false AND LOWER(user_filename) LIKE ?", userID, searchQuery).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&userFiles).Error

	return userFiles, err
}

// GetFilesByFolder retrieves files in a specific folder
func (r *FileRepository) GetFilesByFolder(ctx context.Context, userID uuid.UUID, folderPath string) ([]models.UserFile, error) {
	var userFiles []models.UserFile

	err := r.db.WithContext(ctx).
		Preload("File").
		Where("user_id = ? AND folder_path = ? AND is_deleted = false", userID, folderPath).
		Order("user_filename ASC").
		Find(&userFiles).Error

	return userFiles, err
}

// Helper functions

func generateSessionToken() string {
	return uuid.New().String() + "-" + fmt.Sprintf("%d", time.Now().Unix())
}

func generateStoragePath(contentHash string) string {
	// Create a hierarchical path based on hash for better distribution
	return fmt.Sprintf("files/%s/%s/%s", contentHash[:2], contentHash[2:4], contentHash)
}

// Basic MIME type detection (in production, use a proper library)
func detectMimeType(filename, declaredMimeType string) string {
	ext := filepath.Ext(filename)
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		return mimeType
	}

	if declaredMimeType != "" {
		return declaredMimeType
	}

	return "application/octet-stream"
}

// GetByContentHash retrieves a file by content hash
func (r *FileRepository) GetByContentHash(ctx context.Context, contentHash string) (*models.File, error) {
	var file models.File
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Preload("PrimaryGroup").
		First(&file, "content_hash = ?", contentHash).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// Create creates a new file record
func (r *FileRepository) Create(ctx context.Context, file *models.File) (*models.File, error) {
	err := r.db.WithContext(ctx).Create(file).Error
	return file, err
}

// GetByID retrieves a file by ID
func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.File, error) {
	var file models.File
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Preload("PrimaryGroup").
		Preload("GroupPermissions").
		First(&file, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetUserFilesWithPermissions retrieves files for a user with permission checking
func (r *FileRepository) GetUserFilesWithPermissions(ctx context.Context, userID, orgID uuid.UUID) ([]models.FileMetadata, error) {
	var results []models.FileMetadata

	// Get user's groups for permission checking
	groupRepo := NewGroupRepository(r.db)
	userGroups, err := groupRepo.GetUserGroups(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	groupIDs := make([]uuid.UUID, len(userGroups))
	for i, group := range userGroups {
		groupIDs[i] = group.ID
	}

	// Query files with permissions
	query := `
		SELECT DISTINCT
			uf.id,
			uf.user_filename,
			f.filename as original_name,
			f.file_size,
			f.detected_mime_type as content_type,
			uf.created_at as uploaded_at,
			uf.download_count,
			uf.last_accessed,
			uf.folder_path,
			EXISTS(
				SELECT 1 FROM user_files uf2 
				JOIN files f2 ON uf2.file_id = f2.id 
				WHERE f2.content_hash = f.content_hash AND uf2.user_id != uf.user_id
			) as is_deduped,
			f.owner_id,
			f.primary_group_id,
			f.file_permissions
		FROM user_files uf
		JOIN files f ON uf.file_id = f.id
		JOIN users u ON uf.user_id = u.id
		WHERE u.organization_id = ? 
			AND uf.is_deleted = false
			AND (
				f.owner_id = ? OR
				f.primary_group_id IN (?) OR
				EXISTS (
					SELECT 1 FROM file_group_permissions fgp
					WHERE fgp.file_id = f.id AND fgp.group_id IN (?)
				)
			)
		ORDER BY uf.created_at DESC
	`

	rows, err := r.db.WithContext(ctx).Raw(query, orgID, userID, groupIDs, groupIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var metadata models.FileMetadata
		var ownerID, primaryGroupID *uuid.UUID
		var filePermissions int

		err := rows.Scan(
			&metadata.ID,
			&metadata.UserFilename,
			&metadata.OriginalName,
			&metadata.FileSize,
			&metadata.ContentType,
			&metadata.UploadedAt,
			&metadata.DownloadCount,
			&metadata.LastAccessed,
			&metadata.FolderPath,
			&metadata.IsDeduped,
			&ownerID,
			&primaryGroupID,
			&filePermissions,
		)
		if err != nil {
			return nil, err
		}

		// Check permissions for this file
		permissions := r.calculateUserFilePermissions(ctx, userID, groupIDs, ownerID, primaryGroupID, filePermissions)
		metadata.Permissions = permissions

		results = append(results, metadata)
	}

	return results, nil
}

// calculateUserFilePermissions calculates effective permissions for a user on a file
func (r *FileRepository) calculateUserFilePermissions(
	ctx context.Context,
	userID uuid.UUID,
	userGroupIDs []uuid.UUID,
	fileOwnerID, filePrimaryGroupID *uuid.UUID,
	filePermissions int,
) models.FilePermissionCheck {
	perm := models.Permission(filePermissions)

	var effectivePermission models.Permission

	// Check if user is owner
	if fileOwnerID != nil && *fileOwnerID == userID {
		effectivePermission = perm.GetOwnerPermissions()
	} else {
		// Check group permissions
		if filePrimaryGroupID != nil {
			for _, groupID := range userGroupIDs {
				if groupID == *filePrimaryGroupID {
					effectivePermission = perm.GetGroupPermissions()
					break
				}
			}
		}

		// If no group match, use other permissions
		if effectivePermission == 0 {
			effectivePermission = perm.GetOtherPermissions()
		}
	}

	return models.FilePermissionCheck{
		CanRead:     effectivePermission.CanRead(),
		CanWrite:    effectivePermission.CanWrite(),
		CanDownload: effectivePermission.CanRead(),  // Download requires read permission
		CanDelete:   effectivePermission.CanWrite(), // Delete requires write permission
		CanShare:    effectivePermission.CanRead(),  // Share requires read permission
	}
}

// CreateUserFile creates a user file reference
func (r *FileRepository) CreateUserFile(ctx context.Context, userFile *models.UserFile) (*models.UserFile, error) {
	err := r.db.WithContext(ctx).Create(userFile).Error
	return userFile, err
}

// UpdateDownloadCount increments the download count for a user file
func (r *FileRepository) UpdateDownloadCount(ctx context.Context, userFileID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.UserFile{}).
		Where("id = ?", userFileID).
		Updates(map[string]interface{}{
			"download_count": gorm.Expr("download_count + 1"),
			"last_accessed":  "NOW()",
		}).Error
}

// SetFileGroupPermission sets specific permissions for a group on a file
func (r *FileRepository) SetFileGroupPermission(ctx context.Context, perm *models.FileGroupPermission) error {
	return r.db.WithContext(ctx).
		Create(perm).Error
}

// GetFileGroupPermissions retrieves all group permissions for a file
func (r *FileRepository) GetFileGroupPermissions(ctx context.Context, fileID uuid.UUID) ([]models.FileGroupPermission, error) {
	var permissions []models.FileGroupPermission
	err := r.db.WithContext(ctx).
		Preload("Group").
		Where("file_id = ?", fileID).
		Find(&permissions).Error
	return permissions, err
}

// SoftDeleteUserFile marks a user file as deleted
func (r *FileRepository) SoftDeleteUserFile(ctx context.Context, userFileID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.UserFile{}).
		Where("id = ?", userFileID).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": "NOW()",
		}).Error
}
