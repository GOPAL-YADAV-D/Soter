package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the role of a user within an organization
type UserRole string

const (
	RoleAdmin  UserRole = "ADMIN"
	RoleMember UserRole = "MEMBER"
	RoleViewer UserRole = "VIEWER"
)

// User represents a user in the system
type User struct {
	ID             uuid.UUID  `json:"id" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name           string     `json:"name" gorm:"size:100;not null"`
	Username       string     `json:"username" gorm:"size:50;unique;not null"`
	Email          string     `json:"email" gorm:"size:255;unique;not null"`
	PasswordHash   string     `json:"-" gorm:"size:255;not null"` // Never expose in JSON
	IsActive       bool       `json:"is_active" gorm:"default:true"`
	LastLogin      *time.Time `json:"last_login"`
	StorageQuotaMB int        `json:"storage_quota_mb" gorm:"default:1000"`
	OrganizationID uuid.UUID  `json:"organization_id" gorm:"type:uuid"` // Primary organization
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relationships
	Organizations []Organization `gorm:"many2many:user_organizations;" json:"organizations,omitempty"`
	Groups        []Group        `gorm:"many2many:user_groups;" json:"groups,omitempty"`
	Files         []UserFile     `gorm:"foreignKey:UserID" json:"files,omitempty"`
}

// Organization represents an organization in the system
type Organization struct {
	ID               uuid.UUID  `json:"id" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name             string     `json:"name" gorm:"size:100;unique;not null"`
	Description      *string    `json:"description" gorm:"type:text"`
	CreatedByUserID  *uuid.UUID `json:"created_by_user_id" gorm:"type:uuid"`
	AllocatedSpaceMB int        `json:"allocated_space_mb" gorm:"default:100"`
	UsedSpaceMB      int        `json:"used_space_mb" gorm:"default:0"`
	IsActive         bool       `json:"is_active" gorm:"default:true"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Relationships
	Users  []User  `gorm:"many2many:user_organizations;" json:"users,omitempty"`
	Groups []Group `gorm:"foreignKey:OrganizationID" json:"groups,omitempty"`
}

// UserOrganization represents the relationship between users and organizations
type UserOrganization struct {
	ID             uuid.UUID `json:"id" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	UserID         uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;not null"`
	Role           UserRole  `json:"role" gorm:"type:varchar(20);default:'MEMBER'"`
	JoinedAt       time.Time `json:"joined_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"` // Never expose in JSON
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at" db:"revoked_at"`
	IsRevoked bool       `json:"is_revoked" db:"is_revoked"`
}

// UserWithRole represents a user with their role in a specific organization
type UserWithRole struct {
	User
	Role         UserRole     `json:"role"`
	Organization Organization `json:"organization"`
}

// TokenPair represents a pair of access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access token expiry in seconds
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Name                    string  `json:"name" validate:"required,min=2,max=100"`
	Username                string  `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email                   string  `json:"email" validate:"required,email"`
	Password                string  `json:"password" validate:"required,min=8,max=128"`
	OrganizationName        string  `json:"organization_name" validate:"required,min=2,max=100"`
	OrganizationDescription *string `json:"organization_description,omitempty"`
}

// AuthContext represents the authentication context
type AuthContext struct {
	UserID          uuid.UUID
	OrganizationID  uuid.UUID
	Role            UserRole
	IsAuthenticated bool
}

// IsValidRole checks if a role is valid
func (r UserRole) IsValid() bool {
	return r == RoleAdmin || r == RoleMember || r == RoleViewer
}

// HasPermission checks if a role has the required permission level
func (r UserRole) HasPermission(required UserRole) bool {
	// Define permission hierarchy: ADMIN > MEMBER > VIEWER
	roleLevels := map[UserRole]int{
		RoleAdmin:  3,
		RoleMember: 2,
		RoleViewer: 1,
	}

	userLevel, exists := roleLevels[r]
	if !exists {
		return false
	}

	requiredLevel, exists := roleLevels[required]
	if !exists {
		return false
	}

	return userLevel >= requiredLevel
}

// CanManageOrganization checks if the role can manage organization settings
func (r UserRole) CanManageOrganization() bool {
	return r == RoleAdmin
}

// CanUploadFiles checks if the role can upload files
func (r UserRole) CanUploadFiles() bool {
	return r == RoleAdmin || r == RoleMember
}

// CanViewFiles checks if the role can view files
func (r UserRole) CanViewFiles() bool {
	return r == RoleAdmin || r == RoleMember || r == RoleViewer
}
