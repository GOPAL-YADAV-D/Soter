package models

import (
	"time"

	"github.com/google/uuid"
)

// File represents a unique file in storage (for deduplication)
type File struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ContentHash      string     `gorm:"uniqueIndex:files_content_hash_key;size:64;not null" json:"contentHash"`
	Filename         string     `gorm:"size:255;not null" json:"filename"`
	OriginalMimeType string     `gorm:"size:100" json:"originalMimeType"`
	DetectedMimeType string     `gorm:"size:100" json:"detectedMimeType"`
	FileSize         int64      `gorm:"not null" json:"fileSize"`
	StoragePath      string     `gorm:"size:500;not null" json:"storagePath"`
	IsEncrypted      bool       `gorm:"default:false" json:"isEncrypted"`
	OwnerID          *uuid.UUID `gorm:"type:uuid;index" json:"ownerId"`
	PrimaryGroupID   *uuid.UUID `gorm:"type:uuid;index" json:"primaryGroupId"`
	FilePermissions  int        `gorm:"default:644" json:"filePermissions"` // Linux-style permissions
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`

	// Relationships
	Owner            *User                 `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	PrimaryGroup     *Group                `gorm:"foreignKey:PrimaryGroupID" json:"primaryGroup,omitempty"`
	UserFiles        []UserFile            `gorm:"foreignKey:FileID" json:"userFiles,omitempty"`
	GroupPermissions []FileGroupPermission `gorm:"foreignKey:FileID" json:"groupPermissions,omitempty"`
}

// UserFile represents a user's reference to a file (many-to-many for deduplication)
type UserFile struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	FileID            uuid.UUID  `gorm:"type:uuid;not null;index" json:"fileId"`
	UserFilename      string     `gorm:"size:255;not null" json:"userFilename"`
	FolderPath        string     `gorm:"size:500;default:'/'" json:"folderPath"`
	IsDeleted         bool       `gorm:"default:false" json:"isDeleted"`
	DownloadCount     int        `gorm:"default:0" json:"downloadCount"`
	LastAccessed      *time.Time `json:"lastAccessed"`
	AccessPermissions int        `gorm:"default:644" json:"accessPermissions"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	File *File `gorm:"foreignKey:FileID" json:"file,omitempty"`
}

// UserStorageStats tracks storage savings per user
type UserStorageStats struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID             uuid.UUID `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
	TotalFiles         int       `json:"total_files" gorm:"default:0"`
	UniqueFiles        int       `json:"unique_files" gorm:"default:0"`
	TotalSizeBytes     int64     `json:"total_size_bytes" gorm:"default:0"`
	ActualStorageBytes int64     `json:"actual_storage_bytes" gorm:"default:0"`
	SavingsBytes       int64     `json:"savings_bytes" gorm:"default:0"`
	SavingsPercentage  float64   `json:"savings_percentage" gorm:"type:decimal(5,2);default:0.00"`
	LastCalculated     time.Time `json:"last_calculated" gorm:"default:CURRENT_TIMESTAMP"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// UploadSession tracks file upload progress
type UploadSession struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	SessionToken   string     `json:"session_token" gorm:"type:varchar(255);uniqueIndex;not null"`
	TotalFiles     int        `json:"total_files" gorm:"not null"`
	CompletedFiles int        `json:"completed_files" gorm:"default:0"`
	FailedFiles    int        `json:"failed_files" gorm:"default:0"`
	TotalBytes     int64      `json:"total_bytes" gorm:"not null"`
	UploadedBytes  int64      `json:"uploaded_bytes" gorm:"default:0"`
	Status         string     `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending, in_progress, completed, failed
	StartedAt      time.Time  `json:"started_at" gorm:"default:CURRENT_TIMESTAMP"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relationships
	User       User        `json:"user,omitempty" gorm:"foreignKey:UserID"`
	FileChunks []FileChunk `json:"file_chunks,omitempty" gorm:"foreignKey:UploadSessionID"`
}

// FileChunk for resumable uploads (optional for large files)
type FileChunk struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UploadSessionID uuid.UUID `json:"upload_session_id" gorm:"type:uuid;not null"`
	FileHash        string    `json:"file_hash" gorm:"type:varchar(64);not null"`
	ChunkIndex      int       `json:"chunk_index" gorm:"not null"`
	ChunkSize       int       `json:"chunk_size" gorm:"not null"`
	ChunkHash       string    `json:"chunk_hash" gorm:"type:varchar(64);not null"`
	StoragePath     string    `json:"storage_path" gorm:"type:varchar(500);not null"`
	Status          string    `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending, uploaded, verified
	CreatedAt       time.Time `json:"created_at"`

	// Relationships
	UploadSession UploadSession `json:"upload_session,omitempty" gorm:"foreignKey:UploadSessionID"`
}

// FileUploadInput represents input for file upload
type FileUploadInput struct {
	Filename    string `json:"filename" validate:"required"`
	MimeType    string `json:"mime_type" validate:"required"`
	FileSize    int64  `json:"file_size" validate:"required,min=1"`
	FolderPath  string `json:"folder_path,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
}

// UploadSessionInput represents input for creating upload session
type UploadSessionInput struct {
	Files      []FileUploadInput `json:"files" validate:"required,min=1"`
	TotalBytes int64             `json:"total_bytes" validate:"required,min=1"`
}

// UploadProgress represents upload progress for frontend
type UploadProgress struct {
	SessionID       uuid.UUID `json:"session_id"`
	SessionToken    string    `json:"session_token"`
	TotalFiles      int       `json:"total_files"`
	CompletedFiles  int       `json:"completed_files"`
	FailedFiles     int       `json:"failed_files"`
	TotalBytes      int64     `json:"total_bytes"`
	UploadedBytes   int64     `json:"uploaded_bytes"`
	Status          string    `json:"status"`
	ProgressPercent float64   `json:"progress_percent"`
}

// FileValidationResult represents file validation outcome
type FileValidationResult struct {
	IsValid          bool     `json:"is_valid"`
	DetectedMimeType string   `json:"detected_mime_type"`
	ContentHash      string   `json:"content_hash"`
	FileSize         int64    `json:"file_size"`
	Errors           []string `json:"errors,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
}

// DeduplicationResult represents the result of deduplication check
type DeduplicationResult struct {
	IsExisting     bool       `json:"is_existing"`
	ExistingFileID *uuid.UUID `json:"existing_file_id,omitempty"`
	SavingsBytes   int64      `json:"savings_bytes"`
	StoragePath    string     `json:"storage_path"`
}

// Constants for file validation
const (
	MaxFileSize     = 100 * 1024 * 1024 * 1024 // 100GB
	MaxFilesPerUser = 10000
	ChunkSize       = 5 * 1024 * 1024 // 5MB chunks for resumable uploads
)

// Status constants
const (
	UploadStatusPending    = "pending"
	UploadStatusInProgress = "in_progress"
	UploadStatusCompleted  = "completed"
	UploadStatusFailed     = "failed"

	ChunkStatusPending  = "pending"
	ChunkStatusUploaded = "uploaded"
	ChunkStatusVerified = "verified"
)
