package handlers

import (
	"net/http"
	"strconv"

	"github.com/GOPAL-YADAV-D/Soter/internal/auth"
	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/GOPAL-YADAV-D/Soter/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	userRepo    *repository.UserRepository
	orgRepo     *repository.OrganizationRepository
	groupRepo   *repository.GroupRepository
	authService *auth.AuthService
}

func NewAuthHandler(
	userRepo *repository.UserRepository,
	orgRepo *repository.OrganizationRepository,
	groupRepo *repository.GroupRepository,
	authService *auth.AuthService,
) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		orgRepo:     orgRepo,
		groupRepo:   groupRepo,
		authService: authService,
	}
}

// Register handles user registration with organization creation or joining
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Name                    string  `json:"name" binding:"required"`
		Username                string  `json:"username" binding:"required"`
		Email                   string  `json:"email" binding:"required,email"`
		Password                string  `json:"password" binding:"required,min=6"`
		OrganizationName        *string `json:"organizationName,omitempty"`
		OrganizationDescription *string `json:"organizationDescription,omitempty"`
		OrganizationID          *string `json:"organizationId,omitempty"`
		AllocatedSpaceMB        *int    `json:"allocatedSpaceMb,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	existingUser, _ := h.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	existingUser, _ = h.userRepo.GetByUsername(req.Username)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
		return
	}

	// Hash password
	hashedPassword, err := h.authService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var organization *models.Organization
	var isNewOrg bool

	// Handle organization creation or joining
	if req.OrganizationID != nil {
		// Join existing organization
		orgID, err := uuid.Parse(*req.OrganizationID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
			return
		}

		organization, err = h.orgRepo.GetByID(c.Request.Context(), orgID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
			return
		}
		isNewOrg = false
	} else if req.OrganizationName != nil {
		// Create new organization
		allocatedSpace := 100 // Default 100MB
		if req.AllocatedSpaceMB != nil {
			allocatedSpace = *req.AllocatedSpaceMB
		}

		newOrg := &models.Organization{
			Name:             *req.OrganizationName,
			Description:      req.OrganizationDescription,
			AllocatedSpaceMB: allocatedSpace,
		}

		// Create user first to get ID for organization creation
		user := &models.User{
			Name:         req.Name,
			Username:     req.Username,
			Email:        req.Email,
			PasswordHash: hashedPassword,
		}

		createdUser, err := h.userRepo.Create(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		// Create organization with user as creator
		organization, err = h.orgRepo.Create(c.Request.Context(), newOrg, createdUser.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization"})
			return
		}

		// Generate tokens and respond
		tokenPair, err := h.authService.GenerateTokenPair(createdUser.ID.String(), createdUser.Username, createdUser.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"user":         createdUser,
			"organization": organization,
			"tokenPair":    tokenPair,
			"message":      "Registration successful",
		})
		return
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either organizationName or organizationId must be provided"})
		return
	}

	// Create user and assign to existing organization
	user := &models.User{
		Name:           req.Name,
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   hashedPassword,
		OrganizationID: organization.ID,
	}

	createdUser, err := h.userRepo.Create(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Assign user to organization
	userOrg := &models.UserOrganization{
		UserID:         createdUser.ID,
		OrganizationID: organization.ID,
		Role:           models.RoleMember, // Default role when joining
	}

	if err := h.userRepo.AssignUserToOrganization(userOrg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign user to organization"})
		return
	}

	// Assign user to default 'users' group
	usersGroup, err := h.groupRepo.GetByOrganizationAndName(c.Request.Context(), organization.ID, "users")
	if err == nil {
		h.groupRepo.AssignUserToGroup(c.Request.Context(), createdUser.ID, usersGroup.ID, createdUser.ID)
	}

	// Generate tokens
	tokenPair, err := h.authService.GenerateTokenPair(createdUser.ID.String(), createdUser.Username, createdUser.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":         createdUser,
		"organization": organization,
		"tokenPair":    tokenPair,
		"isNewOrg":     isNewOrg,
		"message":      "Registration successful",
	})
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !h.authService.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Get user's organization
	organization, err := h.orgRepo.GetByID(c.Request.Context(), user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve organization"})
		return
	}

	// Update last login
	h.userRepo.UpdateLastLogin(user.ID)

	tokenPair, err := h.authService.GenerateTokenPair(user.ID.String(), user.Username, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"organization": organization,
		"tokenPair":    tokenPair,
		"message":      "Login successful",
	})
}

// GetUserProfile returns the current user's profile and organization info
func (h *AuthHandler) GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.userRepo.GetByID(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	organization, err := h.orgRepo.GetByID(c.Request.Context(), user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve organization"})
		return
	}

	// Get user's groups
	groups, err := h.groupRepo.GetUserGroups(c.Request.Context(), user.ID, organization.ID)
	if err != nil {
		groups = []models.Group{} // Don't fail if groups can't be retrieved
	}

	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"organization": organization,
		"groups":       groups,
	})
}

// Logout handles user logout (invalidate tokens if needed)
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a stateless JWT system, logout is typically handled client-side
	// by removing the tokens. In a stateful system, you would invalidate
	// the refresh token here.
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate refresh token and get user info
	claims, err := h.authService.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Generate new token pair
	tokenPair, err := h.authService.GenerateTokenPair(user.ID.String(), user.Username, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokenPair": tokenPair,
		"message":   "Token refreshed successfully",
	})
}

type OrganizationHandler struct {
	orgRepo   *repository.OrganizationRepository
	userRepo  *repository.UserRepository
	groupRepo *repository.GroupRepository
}

func NewOrganizationHandler(
	orgRepo *repository.OrganizationRepository,
	userRepo *repository.UserRepository,
	groupRepo *repository.GroupRepository,
) *OrganizationHandler {
	return &OrganizationHandler{
		orgRepo:   orgRepo,
		userRepo:  userRepo,
		groupRepo: groupRepo,
	}
}

// GetStorageUsage returns storage usage statistics for the user's organization
func (h *OrganizationHandler) GetStorageUsage(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.userRepo.GetByID(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	storageUsage, err := h.orgRepo.GetStorageUsageSummary(c.Request.Context(), user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve storage usage"})
		return
	}

	c.JSON(http.StatusOK, storageUsage)
}

// GetOrganizationInfo returns information about the user's organization
func (h *OrganizationHandler) GetOrganizationInfo(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, err := h.userRepo.GetByID(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	organization, err := h.orgRepo.GetByID(c.Request.Context(), user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve organization"})
		return
	}

	// Get organization groups
	groups, err := h.groupRepo.GetByOrganization(c.Request.Context(), organization.ID)
	if err != nil {
		groups = []models.Group{} // Don't fail if groups can't be retrieved
	}

	c.JSON(http.StatusOK, gin.H{
		"organization": organization,
		"groups":       groups,
	})
}

// ListOrganizations returns a list of organizations (for joining)
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	// Add pagination
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

	// This would need to be implemented in the repository
	// For now, return a simple response
	c.JSON(http.StatusOK, gin.H{
		"organizations": []gin.H{},
		"page":          page,
		"limit":         limit,
		"total":         0,
		"message":       "Organization listing not yet implemented",
	})
}

// CreateGroup creates a new group in the user's organization
func (h *OrganizationHandler) CreateGroup(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.GetByID(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if user has admin role or is in admin group
	// This would need proper permission checking

	group := &models.Group{
		OrganizationID: user.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		Permissions:    req.Permissions,
		IsSystemGroup:  false,
	}

	createdGroup, err := h.groupRepo.Create(c.Request.Context(), group)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"group":   createdGroup,
		"message": "Group created successfully",
	})
}
