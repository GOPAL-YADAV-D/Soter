package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/GOPAL-YADAV-D/Soter/internal/auth"
	"github.com/GOPAL-YADAV-D/Soter/internal/config"
	"github.com/GOPAL-YADAV-D/Soter/internal/handlers"
	"github.com/GOPAL-YADAV-D/Soter/internal/middleware"
	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/GOPAL-YADAV-D/Soter/internal/repository"
	"github.com/GOPAL-YADAV-D/Soter/internal/services"
)

func setupRoutes(
	authHandler *handlers.AuthHandler,
	orgHandler *handlers.OrganizationHandler,
	fileHandler *handlers.FileHandler,
	authService *auth.AuthService,
	rateLimiter *middleware.RateLimiter,
	csrfProtection *middleware.CSRFProtection,
	auditService *services.AuditService,
) *gin.Engine {
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Add security middleware
	r.Use(csrfProtection.SecureHeaders())
	r.Use(auditService.AuditMiddleware())

	// Configure CORS for React frontend with enhanced security
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check routes (no rate limiting)
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "Secure File Vault API",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "Secure File Vault API",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// CSRF token endpoint
	r.GET("/csrf-token", csrfProtection.GetCSRFTokenHandler())

	// API routes with rate limiting
	api := r.Group("/api/v1")
	api.Use(rateLimiter.RateLimitMiddleware())

	// Authentication routes (public, with CSRF protection)
	auth := api.Group("/auth")
	auth.Use(csrfProtection.CSRFProtection())
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", authHandler.Logout)
	}

	// Protected routes (require authentication + CSRF protection)
	protected := api.Group("/")
	protected.Use(authMiddleware(authService))
	protected.Use(csrfProtection.CSRFProtection())

	// User profile routes
	protected.GET("/profile", authHandler.GetUserProfile)

	// Organization routes
	org := protected.Group("/organization")
	{
		org.GET("/info", orgHandler.GetOrganizationInfo)
		org.GET("/storage", orgHandler.GetStorageUsage)
		org.GET("/list", orgHandler.ListOrganizations)
		org.POST("/groups", orgHandler.CreateGroup)
	}

	// File management routes
	files := protected.Group("/files")
	{
		files.POST("/upload-session", fileHandler.CreateUploadSession)
		files.POST("/upload/:sessionToken", fileHandler.UploadFile)
		files.POST("/upload-session/:sessionToken/complete", fileHandler.CompleteUploadSession)
		files.GET("/upload-session/:sessionToken/progress", fileHandler.GetUploadProgress)
		files.GET("", fileHandler.GetFiles)
		files.GET("/", fileHandler.GetFiles)
		files.GET("/:fileId", fileHandler.GetFileMetadata)
		files.GET("/:fileId/download", fileHandler.DownloadFile)
		files.DELETE("/:fileId", fileHandler.DeleteFile)
	}

	return r
}

// parseIntEnv parses an environment variable as int with default value
func parseIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// authMiddleware validates JWT tokens and sets user context
func authMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Parse user ID as UUID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		// Set user context
		c.Set("userID", userID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		c.Next()
	}
}

func main() {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Database connection
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.UserOrganization{},
		&models.Group{},
		&models.UserGroup{},
		&models.File{},
		&models.UserFile{},
		&models.FileGroupPermission{},
		&models.OrganizationStorageStats{},
		&models.UploadSession{},
		&models.UserStorageStats{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	orgRepo := repository.NewOrganizationRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	fileRepo := repository.NewFileRepository(db)

	// Initialize services
	authService := auth.NewAuthService(os.Getenv("JWT_SECRET"))

	// Initialize configuration
	cfg := &config.Config{
		RateLimitRPS:   parseIntEnv("RATE_LIMIT_RPS", 2),
		RateLimitBurst: parseIntEnv("RATE_LIMIT_BURST", 5),
		CSRFSecret:     getEnvOrDefault("CSRF_SECRET", "csrf-secret-change-in-production"),
	}

	// Initialize middleware services
	rateLimiter := middleware.NewRateLimiter(cfg)
	csrfProtection := middleware.NewCSRFProtection(cfg)
	auditService := services.NewAuditService(db)
	// quotaService := services.NewQuotaService(userRepo, fileRepo) // TODO: Integrate with file upload service

	storageService, err := services.NewStorageService(cfg)
	if err != nil {
		log.Fatal("Failed to initialize storage service:", err)
	}

	validationService := services.NewFileValidationService(cfg)
	uploadService := services.NewFileUploadService(
		fileRepo,
		storageService,
		validationService,
		userRepo,
	)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, orgRepo, groupRepo, authService)
	orgHandler := handlers.NewOrganizationHandler(orgRepo, userRepo, groupRepo)
	fileHandler := handlers.NewFileHandler(
		fileRepo,
		userRepo,
		orgRepo,
		groupRepo,
		uploadService,
		validationService,
		storageService,
	)

	// Setup routes
	r := setupRoutes(authHandler, orgHandler, fileHandler, authService, rateLimiter, csrfProtection, auditService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📊 Dashboard URL: http://localhost:%s/health", port)
	log.Printf("🔧 Environment: %s", gin.Mode())

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
