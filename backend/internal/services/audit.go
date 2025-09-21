package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// AuditService handles audit logging for security and compliance
type AuditService struct {
	db *gorm.DB
}

// AuditEvent represents an auditable action in the system
type AuditEvent struct {
	ID             uuid.UUID              `json:"id"`
	UserID         *uuid.UUID             `json:"user_id,omitempty"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty"`
	Action         string                 `json:"action"`
	ResourceType   string                 `json:"resource_type"`
	ResourceID     *uuid.UUID             `json:"resource_id,omitempty"`
	IPAddress      string                 `json:"ip_address"`
	UserAgent      string                 `json:"user_agent"`
	RequestID      string                 `json:"request_id,omitempty"`
	Details        map[string]interface{} `json:"details,omitempty"`
	Status         string                 `json:"status"` // success, failure, error
	Timestamp      time.Time              `json:"timestamp"`
}

// NewAuditService creates a new audit logging service
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{
		db: db,
	}
}

// LogEvent logs an audit event to the database
func (as *AuditService) LogEvent(ctx context.Context, event *AuditEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Set ID if not provided
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	// Convert details to JSON
	var detailsJSON []byte
	if event.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(event.Details)
		if err != nil {
			logrus.Errorf("Failed to marshal audit event details: %v", err)
			detailsJSON = []byte("{}")
		}
	}

	// Create audit log record
	auditLog := map[string]interface{}{
		"id":              event.ID,
		"user_id":         event.UserID,
		"organization_id": event.OrganizationID,
		"action":          event.Action,
		"resource_type":   event.ResourceType,
		"resource_id":     event.ResourceID,
		"ip_address":      event.IPAddress,
		"user_agent":      event.UserAgent,
		"request_id":      event.RequestID,
		"details":         string(detailsJSON),
		"status":          event.Status,
		"created_at":      event.Timestamp,
	}

	// Insert into database
	err := as.db.WithContext(ctx).Table("audit_logs").Create(auditLog).Error
	if err != nil {
		logrus.Errorf("Failed to insert audit log: %v", err)
		return fmt.Errorf("failed to log audit event: %w", err)
	}

	// Also log to application logs for immediate visibility
	logrus.WithFields(logrus.Fields{
		"audit_id":      event.ID,
		"user_id":       event.UserID,
		"action":        event.Action,
		"resource_type": event.ResourceType,
		"resource_id":   event.ResourceID,
		"ip_address":    event.IPAddress,
		"status":        event.Status,
	}).Info("Audit event logged")

	return nil
}

// LogFileUpload logs file upload events
func (as *AuditService) LogFileUpload(ctx context.Context, userID uuid.UUID, fileID uuid.UUID, filename string, fileSize int64, ipAddress, userAgent string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}

	event := &AuditEvent{
		UserID:       &userID,
		Action:       "file_upload",
		ResourceType: "file",
		ResourceID:   &fileID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       status,
		Details: map[string]interface{}{
			"filename":  filename,
			"file_size": fileSize,
		},
	}

	as.LogEvent(ctx, event)
}

// LogFileDownload logs file download events
func (as *AuditService) LogFileDownload(ctx context.Context, userID uuid.UUID, fileID uuid.UUID, filename string, ipAddress, userAgent string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}

	event := &AuditEvent{
		UserID:       &userID,
		Action:       "file_download",
		ResourceType: "file",
		ResourceID:   &fileID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       status,
		Details: map[string]interface{}{
			"filename": filename,
		},
	}

	as.LogEvent(ctx, event)
}

// LogFileDelete logs file deletion events
func (as *AuditService) LogFileDelete(ctx context.Context, userID uuid.UUID, fileID uuid.UUID, filename string, ipAddress, userAgent string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}

	event := &AuditEvent{
		UserID:       &userID,
		Action:       "file_delete",
		ResourceType: "file",
		ResourceID:   &fileID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       status,
		Details: map[string]interface{}{
			"filename": filename,
		},
	}

	as.LogEvent(ctx, event)
}

// LogUserLogin logs user authentication events
func (as *AuditService) LogUserLogin(ctx context.Context, userID *uuid.UUID, username, ipAddress, userAgent string, success bool, failureReason string) {
	status := "success"
	details := map[string]interface{}{
		"username": username,
	}

	if !success {
		status = "failure"
		details["failure_reason"] = failureReason
	}

	event := &AuditEvent{
		UserID:       userID,
		Action:       "user_login",
		ResourceType: "user",
		ResourceID:   userID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       status,
		Details:      details,
	}

	as.LogEvent(ctx, event)
}

// LogUserLogout logs user logout events
func (as *AuditService) LogUserLogout(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string) {
	event := &AuditEvent{
		UserID:       &userID,
		Action:       "user_logout",
		ResourceType: "user",
		ResourceID:   &userID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       "success",
	}

	as.LogEvent(ctx, event)
}

// LogPermissionChange logs permission changes
func (as *AuditService) LogPermissionChange(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID, action string, ipAddress, userAgent string, details map[string]interface{}) {
	event := &AuditEvent{
		UserID:       &userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       "success",
		Details:      details,
	}

	as.LogEvent(ctx, event)
}

// LogSecurityEvent logs security-related events
func (as *AuditService) LogSecurityEvent(ctx context.Context, userID *uuid.UUID, action, description, ipAddress, userAgent string, severity string) {
	details := map[string]interface{}{
		"description": description,
		"severity":    severity,
	}

	event := &AuditEvent{
		UserID:       userID,
		Action:       action,
		ResourceType: "security",
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       "success",
		Details:      details,
	}

	as.LogEvent(ctx, event)
}

// GetAuditLogs retrieves audit logs with filtering and pagination
func (as *AuditService) GetAuditLogs(ctx context.Context, filters AuditLogFilters, limit, offset int) ([]*AuditEvent, int64, error) {
	query := as.db.WithContext(ctx).Table("audit_logs")

	// Apply filters
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.ResourceType != "" {
		query = query.Where("resource_type = ?", filters.ResourceType)
	}
	if filters.IPAddress != "" {
		query = query.Where("ip_address = ?", filters.IPAddress)
	}
	if !filters.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if !filters.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	// Get total count
	var total int64
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get paginated results
	var rows []map[string]interface{}
	err = query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve audit logs: %w", err)
	}

	// Convert to AuditEvent structs
	events := make([]*AuditEvent, len(rows))
	for i, row := range rows {
		event := &AuditEvent{}

		// Map fields from database row
		if id, ok := row["id"]; ok {
			if uuidVal, err := uuid.Parse(fmt.Sprintf("%v", id)); err == nil {
				event.ID = uuidVal
			}
		}

		if userID, ok := row["user_id"]; ok && userID != nil {
			if uuidVal, err := uuid.Parse(fmt.Sprintf("%v", userID)); err == nil {
				event.UserID = &uuidVal
			}
		}

		event.Action = fmt.Sprintf("%v", row["action"])
		event.ResourceType = fmt.Sprintf("%v", row["resource_type"])
		event.IPAddress = fmt.Sprintf("%v", row["ip_address"])
		event.UserAgent = fmt.Sprintf("%v", row["user_agent"])
		event.Status = fmt.Sprintf("%v", row["status"])

		if createdAt, ok := row["created_at"]; ok {
			if timeVal, ok := createdAt.(time.Time); ok {
				event.Timestamp = timeVal
			}
		}

		// Parse details JSON
		if details, ok := row["details"]; ok && details != nil {
			detailsStr := fmt.Sprintf("%v", details)
			var detailsMap map[string]interface{}
			if err := json.Unmarshal([]byte(detailsStr), &detailsMap); err == nil {
				event.Details = detailsMap
			}
		}

		events[i] = event
	}

	return events, total, nil
}

// AuditLogFilters represents filters for audit log queries
type AuditLogFilters struct {
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	Action       string     `json:"action,omitempty"`
	ResourceType string     `json:"resource_type,omitempty"`
	IPAddress    string     `json:"ip_address,omitempty"`
	StartTime    time.Time  `json:"start_time,omitempty"`
	EndTime      time.Time  `json:"end_time,omitempty"`
}

// AuditMiddleware creates middleware to automatically log HTTP requests
func (as *AuditService) AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip audit logging for health checks and metrics
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()

		// Process request
		c.Next()

		// Log the request after processing
		go func() {
			duration := time.Since(start)

			var userID *uuid.UUID
			if userIDInterface, exists := c.Get("userID"); exists {
				if uid, ok := userIDInterface.(uuid.UUID); ok {
					userID = &uid
				}
			}

			// Determine action based on HTTP method and path
			action := fmt.Sprintf("http_%s_%s", strings.ToLower(c.Request.Method), c.Request.URL.Path)

			// Simplify action for common patterns
			if strings.Contains(c.Request.URL.Path, "/files/") {
				switch c.Request.Method {
				case "POST":
					action = "file_upload"
				case "GET":
					action = "file_download"
				case "DELETE":
					action = "file_delete"
				}
			}

			status := "success"
			if c.Writer.Status() >= 400 {
				status = "failure"
			}

			details := map[string]interface{}{
				"method":        c.Request.Method,
				"path":          c.Request.URL.Path,
				"status_code":   c.Writer.Status(),
				"duration_ms":   duration.Milliseconds(),
				"response_size": c.Writer.Size(),
			}

			event := &AuditEvent{
				UserID:       userID,
				Action:       action,
				ResourceType: "http_request",
				IPAddress:    c.ClientIP(),
				UserAgent:    c.Request.UserAgent(),
				RequestID:    c.GetString("request_id"),
				Status:       status,
				Details:      details,
			}

			as.LogEvent(context.Background(), event)
		}()
	}
}
