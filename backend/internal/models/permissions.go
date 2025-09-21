package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Group represents a permission group within an organization (Linux-style)
type Group struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organizationId"`
	Name           string    `gorm:"size:50;not null" json:"name"`
	Description    string    `gorm:"type:text" json:"description"`
	Permissions    int       `gorm:"default:755" json:"permissions"` // Linux-style permissions (rwxrwxrwx)
	IsSystemGroup  bool      `gorm:"default:false" json:"isSystemGroup"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`

	// Relationships
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Users        []User        `gorm:"many2many:user_groups;" json:"users,omitempty"`
}

// UserGroup represents the many-to-many relationship between users and groups
type UserGroup struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	GroupID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"groupId"`
	AssignedAt time.Time  `json:"assignedAt"`
	AssignedBy *uuid.UUID `gorm:"type:uuid" json:"assignedBy"`

	// Relationships
	User  *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Group *Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// FileGroupPermission represents file-level permissions for specific groups
type FileGroupPermission struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FileID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"fileId"`
	GroupID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"groupId"`
	Permissions int        `gorm:"default:644" json:"permissions"` // Linux-style permissions
	GrantedBy   *uuid.UUID `gorm:"type:uuid" json:"grantedBy"`
	GrantedAt   time.Time  `json:"grantedAt"`

	// Relationships
	File  *File  `gorm:"foreignKey:FileID" json:"file,omitempty"`
	Group *Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// OrganizationStorageStats represents detailed storage statistics for an organization
type OrganizationStorageStats struct {
	ID                     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID         uuid.UUID `gorm:"type:uuid;not null;unique" json:"organizationId"`
	AllocatedSpaceMB       int       `gorm:"not null" json:"allocatedSpaceMb"`
	UsedSpaceMB            int       `gorm:"default:0" json:"usedSpaceMb"`
	FileCount              int       `gorm:"default:0" json:"fileCount"`
	UniqueFileCount        int       `gorm:"default:0" json:"uniqueFileCount"`
	DeduplicationSavingsMB int       `gorm:"default:0" json:"deduplicationSavingsMb"`
	LastCalculated         time.Time `json:"lastCalculated"`

	// Relationships
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

// Permission represents Linux-style permission values
type Permission int

const (
	// File permissions (octal values)
	PermissionNone          Permission = 0 // ---
	PermissionExecute       Permission = 1 // --x
	PermissionWrite         Permission = 2 // -w-
	PermissionWriteExec     Permission = 3 // -wx
	PermissionRead          Permission = 4 // r--
	PermissionReadExec      Permission = 5 // r-x
	PermissionReadWrite     Permission = 6 // rw-
	PermissionReadWriteExec Permission = 7 // rwx

	// Common permission combinations
	PermissionOwnerFull      Permission = 700 // rwx------
	PermissionOwnerReadWrite Permission = 600 // rw-------
	PermissionGroupRead      Permission = 644 // rw-r--r--
	PermissionGroupWrite     Permission = 664 // rw-rw-r--
	PermissionPublicRead     Permission = 755 // rwxr-xr-x
	PermissionPublicWrite    Permission = 777 // rwxrwxrwx
)

// HasPermission checks if the given permission level allows the requested operation
func (p Permission) HasPermission(requested Permission) bool {
	return int(p)&int(requested) == int(requested)
}

// CanRead checks if the permission allows read access
func (p Permission) CanRead() bool {
	return p.HasPermission(PermissionRead)
}

// CanWrite checks if the permission allows write access
func (p Permission) CanWrite() bool {
	return p.HasPermission(PermissionWrite)
}

// CanExecute checks if the permission allows execute access
func (p Permission) CanExecute() bool {
	return p.HasPermission(PermissionExecute)
}

// GetOwnerPermissions extracts owner permissions from a 3-digit permission
func (p Permission) GetOwnerPermissions() Permission {
	return Permission((int(p) / 100) % 10)
}

// GetGroupPermissions extracts group permissions from a 3-digit permission
func (p Permission) GetGroupPermissions() Permission {
	return Permission((int(p) / 10) % 10)
}

// GetOtherPermissions extracts other permissions from a 3-digit permission
func (p Permission) GetOtherPermissions() Permission {
	return Permission(int(p) % 10)
}

// FilePermissionLevel represents the level at which permission is being checked
type FilePermissionLevel int

const (
	PermissionLevelOwner FilePermissionLevel = iota
	PermissionLevelGroup
	PermissionLevelOther
)

// UserPermissionContext represents the context for checking user permissions
type UserPermissionContext struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	GroupIDs       []uuid.UUID
	IsOwner        bool
	Role           UserRole
}

// FilePermissionCheck represents a permission check result
type FilePermissionCheck struct {
	// Simple boolean permissions for UI
	CanRead     bool `json:"canRead"`
	CanWrite    bool `json:"canWrite"`
	CanDownload bool `json:"canDownload"`
	CanDelete   bool `json:"canDelete"`
	CanShare    bool `json:"canShare"`

	// Detailed Linux-style permissions
	Owner  PermissionSet `json:"owner"`
	Group  PermissionSet `json:"group"`
	Others PermissionSet `json:"others"`
	Octal  string        `json:"octal"` // e.g., "644"
}

// PermissionSet represents read/write/execute permissions for a role
type PermissionSet struct {
	Read    bool `json:"read"`
	Write   bool `json:"write"`
	Execute bool `json:"execute"`
}

// ParseLinuxPermissions converts octal permission (e.g., 644) to structured permissions
func ParseLinuxPermissions(permissions int, userID uuid.UUID, fileOwnerID *uuid.UUID, groupID *uuid.UUID, userGroupIDs []uuid.UUID) FilePermissionCheck {
	// Convert to 3-digit octal
	octal := fmt.Sprintf("%03o", permissions)

	// Parse each digit (owner, group, others)
	ownerPerms := (permissions >> 6) & 7
	groupPerms := (permissions >> 3) & 7
	otherPerms := permissions & 7

	result := FilePermissionCheck{
		Owner: PermissionSet{
			Read:    (ownerPerms & 4) != 0,
			Write:   (ownerPerms & 2) != 0,
			Execute: (ownerPerms & 1) != 0,
		},
		Group: PermissionSet{
			Read:    (groupPerms & 4) != 0,
			Write:   (groupPerms & 2) != 0,
			Execute: (groupPerms & 1) != 0,
		},
		Others: PermissionSet{
			Read:    (otherPerms & 4) != 0,
			Write:   (otherPerms & 2) != 0,
			Execute: (otherPerms & 1) != 0,
		},
		Octal: octal,
	}

	// Determine which permission set applies to this user
	var userPerms PermissionSet
	if fileOwnerID != nil && *fileOwnerID == userID {
		// User is the owner
		userPerms = result.Owner
	} else if groupID != nil {
		// Check if user is in the file's group
		userInGroup := false
		for _, ugid := range userGroupIDs {
			if ugid == *groupID {
				userInGroup = true
				break
			}
		}
		if userInGroup {
			userPerms = result.Group
		} else {
			userPerms = result.Others
		}
	} else {
		// Default to others permissions
		userPerms = result.Others
	}

	// Set simple boolean permissions based on user's effective permissions
	result.CanRead = userPerms.Read
	result.CanWrite = userPerms.Write
	result.CanDownload = userPerms.Read // Download requires read permission
	result.CanDelete = userPerms.Write  // Delete typically requires write permission
	result.CanShare = userPerms.Read    // Share requires read permission

	return result
}

// GroupType represents predefined system groups
type GroupType string

const (
	GroupTypeAdmin  GroupType = "admin"
	GroupTypeUsers  GroupType = "users"
	GroupTypeGuests GroupType = "guests"
)

// CreateGroupRequest represents a request to create a new group
type CreateGroupRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=50"`
	Description string `json:"description,omitempty"`
	Permissions int    `json:"permissions" validate:"min=0,max=777"`
}

// UpdateGroupRequest represents a request to update a group
type UpdateGroupRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=50"`
	Description *string `json:"description,omitempty"`
	Permissions *int    `json:"permissions,omitempty" validate:"omitempty,min=0,max=777"`
}

// AssignUserToGroupRequest represents a request to assign a user to a group
type AssignUserToGroupRequest struct {
	UserID  uuid.UUID `json:"userId" validate:"required"`
	GroupID uuid.UUID `json:"groupId" validate:"required"`
}

// JoinOrganizationRequest represents a request to join an existing organization
type JoinOrganizationRequest struct {
	OrganizationID uuid.UUID `json:"organizationId" validate:"required"`
	Name           string    `json:"name" validate:"required,min=2,max=100"`
	Username       string    `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email          string    `json:"email" validate:"required,email"`
	Password       string    `json:"password" validate:"required,min=8,max=128"`
}

// CreateOrganizationRequest represents a request to create a new organization
type CreateOrganizationRequest struct {
	Name           string  `json:"name" validate:"required,min=2,max=100"`
	Description    *string `json:"description,omitempty"`
	AllocatedSpace int     `json:"allocatedSpaceMb" validate:"min=1,max=10000"` // Max 10GB for demo
}

// FileMetadata represents metadata for file display
type FileMetadata struct {
	ID            uuid.UUID           `json:"id"`
	UserFilename  string              `json:"userFilename"`
	OriginalName  string              `json:"originalName"`
	FileSize      int64               `json:"fileSize"`
	ContentType   string              `json:"contentType"`
	UploadedAt    time.Time           `json:"uploadedAt"`
	DownloadCount int                 `json:"downloadCount"`
	LastAccessed  *time.Time          `json:"lastAccessed,omitempty"`
	FolderPath    string              `json:"folderPath"`
	IsDeduped     bool                `json:"isDeduped"`
	Owner         *User               `json:"owner,omitempty"`
	Groups        []Group             `json:"groups,omitempty"`
	Permissions   FilePermissionCheck `json:"permissions"`
	Tags          []string            `json:"tags,omitempty"`

	// Additional fields for frontend compatibility
	Hash           string        `json:"hash,omitempty"`           // Content hash
	GroupName      string        `json:"groupName,omitempty"`      // Primary group name
	DuplicateCount int           `json:"duplicateCount,omitempty"` // Number of duplicates
	IsOriginal     bool          `json:"isOriginal,omitempty"`     // Is this the original file
	RelatedFiles   []RelatedFile `json:"relatedFiles,omitempty"`   // Related file info
}

// RelatedFile represents a related file entry
type RelatedFile struct {
	ID           string `json:"id"`
	UserFilename string `json:"userFilename"`
	UploadedBy   string `json:"uploadedBy"`
	UploaderName string `json:"uploaderName"`
	UploadedAt   string `json:"uploadedAt"`
}

// StorageUsageSummary represents storage usage information
type StorageUsageSummary struct {
	AllocatedSpaceMB       int       `json:"allocatedSpaceMb"`
	UsedSpaceMB            int       `json:"usedSpaceMb"`
	AvailableSpaceMB       int       `json:"availableSpaceMb"`
	UsagePercentage        float64   `json:"usagePercentage"`
	FileCount              int       `json:"fileCount"`
	UniqueFileCount        int       `json:"uniqueFileCount"`
	DeduplicationSavingsMB int       `json:"deduplicationSavingsMb"`
	DeduplicationPercent   float64   `json:"deduplicationPercent"`
	LastCalculated         time.Time `json:"lastCalculated"`
}
