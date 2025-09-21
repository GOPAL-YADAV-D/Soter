package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GOPAL-YADAV-D/Soter/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// QuotaInfo represents quota information for a user
type QuotaInfo struct {
	UserID      uuid.UUID `json:"user_id"`
	UsedBytes   int64     `json:"used_bytes"`
	QuotaBytes  int64     `json:"quota_bytes"`
	FileCount   int       `json:"file_count"`
	LastChecked time.Time `json:"last_checked"`
}

// QuotaService manages storage quotas and usage tracking
type QuotaService struct {
	userRepo   *repository.UserRepository
	fileRepo   *repository.FileRepository
	quotaCache map[uuid.UUID]*QuotaInfo
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// NewQuotaService creates a new quota management service
func NewQuotaService(userRepo *repository.UserRepository, fileRepo *repository.FileRepository) *QuotaService {
	qs := &QuotaService{
		userRepo:   userRepo,
		fileRepo:   fileRepo,
		quotaCache: make(map[uuid.UUID]*QuotaInfo),
		cacheTTL:   5 * time.Minute,
	}

	// Start background quota monitoring
	go qs.backgroundMonitoring()

	return qs
}

// CheckQuota checks if a user can upload additional bytes
func (qs *QuotaService) CheckQuota(ctx context.Context, userID uuid.UUID, additionalBytes int64) (*QuotaInfo, error) {
	quotaInfo, err := qs.getUserQuota(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user quota: %w", err)
	}

	// Check if adding these bytes would exceed quota
	if quotaInfo.UsedBytes+additionalBytes > quotaInfo.QuotaBytes {
		return quotaInfo, fmt.Errorf("quota exceeded: would use %d bytes, quota is %d bytes",
			quotaInfo.UsedBytes+additionalBytes, quotaInfo.QuotaBytes)
	}

	return quotaInfo, nil
}

// UpdateQuotaUsage updates the quota usage after a successful upload
func (qs *QuotaService) UpdateQuotaUsage(ctx context.Context, userID uuid.UUID, bytesAdded int64) error {
	// Update cache
	qs.cacheMutex.Lock()
	if quotaInfo, exists := qs.quotaCache[userID]; exists {
		quotaInfo.UsedBytes += bytesAdded
		quotaInfo.FileCount++
		quotaInfo.LastChecked = time.Now()
	}
	qs.cacheMutex.Unlock()

	// In a production system, you would update the database here
	// For now, we'll just log the update
	logrus.Infof("Updated quota usage for user %s: added %d bytes", userID, bytesAdded)

	return nil
}

// getUserQuota gets quota information for a user (from cache or database)
func (qs *QuotaService) getUserQuota(ctx context.Context, userID uuid.UUID) (*QuotaInfo, error) {
	// Check cache first
	qs.cacheMutex.RLock()
	if quotaInfo, exists := qs.quotaCache[userID]; exists {
		if time.Since(quotaInfo.LastChecked) < qs.cacheTTL {
			qs.cacheMutex.RUnlock()
			return quotaInfo, nil
		}
	}
	qs.cacheMutex.RUnlock()

	// Cache miss or expired, calculate fresh data
	// Calculate current usage by counting user files
	userFiles, err := qs.fileRepo.GetUserFiles(ctx, userID, "", 1000, 0) // Get up to 1000 files
	if err != nil {
		return nil, fmt.Errorf("failed to get user files: %w", err)
	}

	var totalSize int64
	for _, userFile := range userFiles {
		// We need to get the actual file to get its size
		file, err := qs.fileRepo.GetByID(ctx, userFile.FileID)
		if err != nil {
			logrus.Warnf("Failed to get file %s: %v", userFile.FileID, err)
			continue
		}
		totalSize += file.FileSize
	}

	// Default quota (could be retrieved from user settings or organization settings)
	defaultQuota := int64(10 * 1024 * 1024 * 1024) // 10GB default

	quotaInfo := &QuotaInfo{
		UserID:      userID,
		UsedBytes:   totalSize,
		QuotaBytes:  defaultQuota,
		FileCount:   len(userFiles),
		LastChecked: time.Now(),
	}

	// Update cache
	qs.cacheMutex.Lock()
	qs.quotaCache[userID] = quotaInfo
	qs.cacheMutex.Unlock()

	logrus.Debugf("Calculated quota for user %s: %d/%d bytes (%d files)",
		userID, quotaInfo.UsedBytes, quotaInfo.QuotaBytes, quotaInfo.FileCount)

	return quotaInfo, nil
}

// GetQuotaInfo returns current quota information for a user
func (qs *QuotaService) GetQuotaInfo(ctx context.Context, userID uuid.UUID) (*QuotaInfo, error) {
	return qs.getUserQuota(ctx, userID)
}

// backgroundMonitoring runs periodic quota checks and cleanup
func (qs *QuotaService) backgroundMonitoring() {
	ticker := time.NewTicker(15 * time.Minute) // Check every 15 minutes
	defer ticker.Stop()

	for range ticker.C {
		qs.cleanupCache()
		qs.monitorQuotaUsage()
	}
}

// cleanupCache removes expired entries from the quota cache
func (qs *QuotaService) cleanupCache() {
	qs.cacheMutex.Lock()
	defer qs.cacheMutex.Unlock()

	now := time.Now()
	for userID, quotaInfo := range qs.quotaCache {
		if now.Sub(quotaInfo.LastChecked) > qs.cacheTTL*2 {
			delete(qs.quotaCache, userID)
		}
	}

	logrus.Debugf("Quota cache cleanup completed: %d entries remaining", len(qs.quotaCache))
}

// monitorQuotaUsage checks for users approaching their quota limits
func (qs *QuotaService) monitorQuotaUsage() {
	qs.cacheMutex.RLock()
	highUsageUsers := make([]uuid.UUID, 0)

	for userID, quotaInfo := range qs.quotaCache {
		usagePercent := float64(quotaInfo.UsedBytes) / float64(quotaInfo.QuotaBytes) * 100
		if usagePercent > 80 { // Alert at 80% usage
			highUsageUsers = append(highUsageUsers, userID)
		}
	}
	qs.cacheMutex.RUnlock()

	// Log high usage alerts
	for _, userID := range highUsageUsers {
		if quotaInfo, exists := qs.quotaCache[userID]; exists {
			usagePercent := float64(quotaInfo.UsedBytes) / float64(quotaInfo.QuotaBytes) * 100
			logrus.Warnf("User %s is using %.1f%% of their storage quota (%d/%d bytes)",
				userID, usagePercent, quotaInfo.UsedBytes, quotaInfo.QuotaBytes)
		}
	}
}

// IsQuotaExceeded checks if a user has exceeded their quota
func (qs *QuotaService) IsQuotaExceeded(ctx context.Context, userID uuid.UUID) (bool, error) {
	quotaInfo, err := qs.getUserQuota(ctx, userID)
	if err != nil {
		return false, err
	}

	return quotaInfo.UsedBytes > quotaInfo.QuotaBytes, nil
}

// GetQuotaUtilization returns the percentage of quota used
func (qs *QuotaService) GetQuotaUtilization(ctx context.Context, userID uuid.UUID) (float64, error) {
	quotaInfo, err := qs.getUserQuota(ctx, userID)
	if err != nil {
		return 0, err
	}

	if quotaInfo.QuotaBytes == 0 {
		return 0, nil
	}

	return float64(quotaInfo.UsedBytes) / float64(quotaInfo.QuotaBytes) * 100, nil
}
