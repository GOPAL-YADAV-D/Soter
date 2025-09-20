package repositories

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/GOPAL-YADAV-D/Soter/internal/models"
)

// UserRepository handles user-related database operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUserWithOrganization creates a new user and organization in a transaction
func (r *UserRepository) CreateUserWithOrganization(user *models.User, org *models.Organization) (*models.UserWithRole, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create organization first
	orgQuery := `
		INSERT INTO organizations (name, description, created_by_user_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	
	var orgID uuid.UUID
	err = tx.QueryRow(orgQuery, org.Name, org.Description, user.ID).Scan(
		&org.ID, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}
	
	org.ID = orgID

	// Create user
	userQuery := `
		INSERT INTO users (name, username, email, password_hash, storage_quota_mb)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	
	var userID uuid.UUID
	err = tx.QueryRow(userQuery, user.Name, user.Username, user.Email, user.PasswordHash, user.StorageQuotaMB).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	user.ID = userID

	// Update organization with created_by_user_id
	updateOrgQuery := `UPDATE organizations SET created_by_user_id = $1 WHERE id = $2`
	_, err = tx.Exec(updateOrgQuery, user.ID, org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization creator: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Create user with role (the trigger should have automatically created the user_organizations entry)
	userWithRole := &models.UserWithRole{
		User:         *user,
		Role:         models.RoleAdmin, // Creator is automatically admin
		Organization: *org,
	}

	logrus.WithFields(logrus.Fields{
		"user_id":         user.ID,
		"organization_id": org.ID,
		"email":          user.Email,
	}).Info("Created user with organization")

	return userWithRole, nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, name, username, email, password_hash, is_active, 
		       last_login, storage_quota_mb, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	
	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Name, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsActive, &user.LastLogin, &user.StorageQuotaMB,
		&user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(userID uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, name, username, email, password_hash, is_active,
		       last_login, storage_quota_mb, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	
	user := &models.User{}
	err := r.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Name, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsActive, &user.LastLogin, &user.StorageQuotaMB,
		&user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// GetUserWithRole retrieves a user with their role in a specific organization
func (r *UserRepository) GetUserWithRole(userID uuid.UUID) (*models.UserWithRole, error) {
	query := `
		SELECT u.id, u.name, u.username, u.email, u.is_active, u.last_login,
		       u.storage_quota_mb, u.created_at, u.updated_at,
		       uo.organization_id, uo.role,
		       o.name as org_name, o.description as org_description,
		       o.created_at as org_created_at, o.updated_at as org_updated_at
		FROM users u
		JOIN user_organizations uo ON u.id = uo.user_id
		JOIN organizations o ON uo.organization_id = o.id
		WHERE u.id = $1 AND u.is_active = true
	`
	
	userWithRole := &models.UserWithRole{}
	err := r.db.QueryRow(query, userID).Scan(
		&userWithRole.ID, &userWithRole.Name, &userWithRole.Username, &userWithRole.Email,
		&userWithRole.IsActive, &userWithRole.LastLogin, &userWithRole.StorageQuotaMB,
		&userWithRole.CreatedAt, &userWithRole.UpdatedAt,
		&userWithRole.Organization.ID, &userWithRole.Role,
		&userWithRole.Organization.Name, &userWithRole.Organization.Description,
		&userWithRole.Organization.CreatedAt, &userWithRole.Organization.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found or inactive")
		}
		return nil, fmt.Errorf("failed to get user with role: %w", err)
	}
	
	return userWithRole, nil
}

// CheckEmailExists checks if an email already exists
func (r *UserRepository) CheckEmailExists(email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	
	var exists bool
	err := r.db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	
	return exists, nil
}

// CheckUsernameExists checks if a username already exists
func (r *UserRepository) CheckUsernameExists(username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	
	var exists bool
	err := r.db.QueryRow(query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}
	
	return exists, nil
}

// CheckOrganizationNameExists checks if an organization name already exists
func (r *UserRepository) CheckOrganizationNameExists(name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM organizations WHERE name = $1)`
	
	var exists bool
	err := r.db.QueryRow(query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check organization name existence: %w", err)
	}
	
	return exists, nil
}

// UpdateUserLastLogin updates the user's last login timestamp
func (r *UserRepository) UpdateUserLastLogin(userID uuid.UUID) error {
	query := `UPDATE users SET last_login = NOW() WHERE id = $1`
	
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	
	return nil
}

// GetOrganizationMembers retrieves all members of an organization
func (r *UserRepository) GetOrganizationMembers(organizationID uuid.UUID) ([]*models.UserWithRole, error) {
	query := `
		SELECT u.id, u.name, u.username, u.email, u.is_active, u.last_login,
		       u.storage_quota_mb, u.created_at, u.updated_at,
		       uo.role,
		       o.name as org_name, o.description as org_description,
		       o.created_at as org_created_at, o.updated_at as org_updated_at
		FROM users u
		JOIN user_organizations uo ON u.id = uo.user_id
		JOIN organizations o ON uo.organization_id = o.id
		WHERE uo.organization_id = $1 AND u.is_active = true
		ORDER BY u.created_at ASC
	`
	
	rows, err := r.db.Query(query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query organization members: %w", err)
	}
	defer rows.Close()
	
	var members []*models.UserWithRole
	for rows.Next() {
		member := &models.UserWithRole{}
		err := rows.Scan(
			&member.ID, &member.Name, &member.Username, &member.Email,
			&member.IsActive, &member.LastLogin, &member.StorageQuotaMB,
			&member.CreatedAt, &member.UpdatedAt,
			&member.Role,
			&member.Organization.Name, &member.Organization.Description,
			&member.Organization.CreatedAt, &member.Organization.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		
		member.Organization.ID = organizationID
		members = append(members, member)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate members: %w", err)
	}
	
	return members, nil
}

