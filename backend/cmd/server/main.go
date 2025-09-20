package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/GOPAL-YADAV-D/Soter/internal/auth"
	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/GOPAL-YADAV-D/Soter/internal/repository"
)

type Server struct {
	db          *gorm.DB
	authService *auth.AuthService
	userRepo    *repository.UserRepository
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Name                    string `json:"name" binding:"required"`
	Username                string `json:"username" binding:"required"`
	Email                   string `json:"email" binding:"required,email"`
	Password                string `json:"password" binding:"required,min=6"`
	OrganizationName        string `json:"organizationName" binding:"required"`
	OrganizationDescription string `json:"organizationDescription"`
}

type AuthResponse struct {
	User      *models.User    `json:"user"`
	TokenPair *auth.TokenPair `json:"tokenPair"`
	Message   string          `json:"message"`
}

func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !s.authService.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	tokenPair, err := s.authService.GenerateTokenPair(user.ID.String(), user.Username, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:      user,
		TokenPair: tokenPair,
		Message:   "Login successful",
	})
}

func (s *Server) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	existingUser, _ := s.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	existingUser, _ = s.userRepo.GetByUsername(req.Username)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
		return
	}

	// Hash password
	hashedPassword, err := s.authService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create organization first
	org, err := s.userRepo.CreateOrganizationWithUser(
		req.OrganizationName,
		req.OrganizationDescription,
		req.Name,
		req.Username,
		req.Email,
		hashedPassword,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user and organization"})
		return
	}

	// Get the created user
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve created user"})
		return
	}

	// Generate tokens
	tokenPair, err := s.authService.GenerateTokenPair(user.ID.String(), user.Username, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	log.Printf("User registered successfully: %s (Organization: %s)", user.Email, org.Name)

	c.JSON(http.StatusCreated, AuthResponse{
		User:      user,
		TokenPair: tokenPair,
		Message:   "Registration successful",
	})
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "Secure File Vault API",
		"timestamp": time.Now().Format(time.RFC3339),
	})
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
	db.AutoMigrate(&models.User{}, &models.Organization{})

	// Initialize services
	authService := auth.NewAuthService(
		os.Getenv("JWT_SECRET"),
	)
	userRepo := repository.NewUserRepository(db)

	server := &Server{
		db:          db,
		authService: authService,
		userRepo:    userRepo,
	}

	// Setup Gin
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Routes
	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", server.login)
		api.POST("/auth/register", server.register)
	}

	r.GET("/healthz", server.health)
	r.GET("/health", server.health)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}
