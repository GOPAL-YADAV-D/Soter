package repository

import (
	"fmt"
	"time"

	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user *models.User) (*models.User, error) {
	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpdateLastLogin(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login", now).Error
}

func (r *UserRepository) AssignUserToOrganization(userOrg *models.UserOrganization) error {
	return r.db.Create(userOrg).Error
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) CreateOrganizationWithUser(
	orgName, orgDescription, userName, username, email, passwordHash string,
) (*models.Organization, error) {
	var org models.Organization
	var user models.User

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Create organization
		var description *string
		if orgDescription != "" {
			description = &orgDescription
		}

		org = models.Organization{
			ID:          uuid.New(),
			Name:        orgName,
			Description: description,
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := tx.Create(&org).Error; err != nil {
			return fmt.Errorf("failed to create organization: %w", err)
		}

		user = models.User{
			ID:             uuid.New(),
			Name:           userName,
			Username:       username,
			Email:          email,
			PasswordHash:   passwordHash,
			IsActive:       true,
			StorageQuotaMB: 1000,
			OrganizationID: org.ID,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := tx.Create(&user).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		if err := tx.Model(&org).Update("created_by_user_id", user.ID).Error; err != nil {
			return fmt.Errorf("failed to update organization creator: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &org, nil
}
