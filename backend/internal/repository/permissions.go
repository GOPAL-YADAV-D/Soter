package repository

import (
	"context"
	"time"

	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizationRepository struct {
	db *gorm.DB
}

func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create creates a new organization with default groups and assigns creator as admin
func (r *OrganizationRepository) Create(ctx context.Context, org *models.Organization, creatorUserID uuid.UUID) (*models.Organization, error) {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return nil, err
	}

	// Set creator
	org.CreatedByUserID = &creatorUserID

	// Create organization
	if err := tx.Create(org).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Assign creator to organization as admin
	userOrg := &models.UserOrganization{
		ID:             uuid.New(),
		UserID:         creatorUserID,
		OrganizationID: org.ID,
		Role:           models.RoleAdmin,
		JoinedAt:       time.Now(),
	}

	if err := tx.Create(userOrg).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update user's primary organization
	if err := tx.Model(&models.User{}).Where("id = ?", creatorUserID).Update("organization_id", org.ID).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Create initial storage stats record
	storageStats := &models.OrganizationStorageStats{
		OrganizationID:         org.ID,
		AllocatedSpaceMB:       org.AllocatedSpaceMB,
		UsedSpaceMB:            0,
		FileCount:              0,
		UniqueFileCount:        0,
		DeduplicationSavingsMB: 0,
		LastCalculated:         time.Now(),
	}

	if err := tx.Create(storageStats).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return org, tx.Commit().Error
}

// GetByID retrieves an organization by ID
func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	err := r.db.WithContext(ctx).
		Preload("Groups").
		Preload("Users").
		First(&org, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetByName retrieves an organization by name
func (r *OrganizationRepository) GetByName(ctx context.Context, name string) (*models.Organization, error) {
	var org models.Organization
	err := r.db.WithContext(ctx).First(&org, "name = ? AND is_active = ?", name, true).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetStorageStats retrieves storage statistics for an organization
func (r *OrganizationRepository) GetStorageStats(ctx context.Context, orgID uuid.UUID) (*models.OrganizationStorageStats, error) {
	var stats models.OrganizationStorageStats
	err := r.db.WithContext(ctx).
		First(&stats, "organization_id = ?", orgID).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetStorageUsageSummary calculates and returns storage usage summary
func (r *OrganizationRepository) GetStorageUsageSummary(ctx context.Context, orgID uuid.UUID) (*models.StorageUsageSummary, error) {
	stats, err := r.GetStorageStats(ctx, orgID)
	if err != nil {
		return nil, err
	}

	availableSpace := stats.AllocatedSpaceMB - stats.UsedSpaceMB
	usagePercentage := float64(stats.UsedSpaceMB) / float64(stats.AllocatedSpaceMB) * 100

	var deduplicationPercent float64
	if stats.FileCount > 0 {
		logicalSize := stats.UsedSpaceMB + stats.DeduplicationSavingsMB
		if logicalSize > 0 {
			deduplicationPercent = float64(stats.DeduplicationSavingsMB) / float64(logicalSize) * 100
		}
	}

	return &models.StorageUsageSummary{
		AllocatedSpaceMB:       stats.AllocatedSpaceMB,
		UsedSpaceMB:            stats.UsedSpaceMB,
		AvailableSpaceMB:       availableSpace,
		UsagePercentage:        usagePercentage,
		FileCount:              stats.FileCount,
		UniqueFileCount:        stats.UniqueFileCount,
		DeduplicationSavingsMB: stats.DeduplicationSavingsMB,
		DeduplicationPercent:   deduplicationPercent,
		LastCalculated:         stats.LastCalculated,
	}, nil
}

// CheckStorageQuota checks if the organization has enough space for additional storage
func (r *OrganizationRepository) CheckStorageQuota(ctx context.Context, orgID uuid.UUID, additionalMB int) (bool, error) {
	stats, err := r.GetStorageStats(ctx, orgID)
	if err != nil {
		return false, err
	}

	return (stats.UsedSpaceMB + additionalMB) <= stats.AllocatedSpaceMB, nil
}

type GroupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// Create creates a new group
func (r *GroupRepository) Create(ctx context.Context, group *models.Group) (*models.Group, error) {
	err := r.db.WithContext(ctx).Create(group).Error
	return group, err
}

// GetByID retrieves a group by ID
func (r *GroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Group, error) {
	var group models.Group
	err := r.db.WithContext(ctx).
		Preload("Users").
		First(&group, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByOrganizationAndName retrieves a group by organization and name
func (r *GroupRepository) GetByOrganizationAndName(ctx context.Context, orgID uuid.UUID, name string) (*models.Group, error) {
	var group models.Group
	err := r.db.WithContext(ctx).
		First(&group, "organization_id = ? AND name = ?", orgID, name).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByOrganization retrieves all groups for an organization
func (r *GroupRepository) GetByOrganization(ctx context.Context, orgID uuid.UUID) ([]models.Group, error) {
	var groups []models.Group
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("is_system_group DESC, name ASC").
		Find(&groups).Error
	return groups, err
}

// GetUserGroups retrieves all groups a user belongs to
func (r *GroupRepository) GetUserGroups(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) ([]models.Group, error) {
	var groups []models.Group
	err := r.db.WithContext(ctx).
		Joins("JOIN user_groups ug ON groups.id = ug.group_id").
		Where("ug.user_id = ? AND groups.organization_id = ?", userID, orgID).
		Find(&groups).Error
	return groups, err
}

// AssignUserToGroup assigns a user to a group
func (r *GroupRepository) AssignUserToGroup(ctx context.Context, userID, groupID, assignedBy uuid.UUID) error {
	userGroup := &models.UserGroup{
		UserID:     userID,
		GroupID:    groupID,
		AssignedBy: &assignedBy,
	}
	return r.db.WithContext(ctx).Create(userGroup).Error
}

// RemoveUserFromGroup removes a user from a group
func (r *GroupRepository) RemoveUserFromGroup(ctx context.Context, userID, groupID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		Delete(&models.UserGroup{}).Error
}
