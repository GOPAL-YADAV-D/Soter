package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"

	"github.com/GOPAL-YADAV-D/Soter/internal/models"
)

// AuthService handles authentication operations
type AuthService struct {
	db        *sql.DB
	jwtSecret []byte
}

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID         uuid.UUID        `json:"user_id"`
	OrganizationID uuid.UUID        `json:"organization_id"`
	Role           models.UserRole  `json:"role"`
	Email          string           `json:"email"`
	jwt.RegisteredClaims
}

// Argon2Config represents the configuration for Argon2 hashing
type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Config returns a secure default configuration for Argon2
func DefaultArgon2Config() *Argon2Config {
	return &Argon2Config{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// NewAuthService creates a new authentication service
func NewAuthService(db *sql.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: []byte(jwtSecret),
	}
}

// HashPassword hashes a password using Argon2id
func (a *AuthService) HashPassword(password string) (string, error) {
	config := DefaultArgon2Config()
	
	// Generate a random salt
	salt := make([]byte, config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	
	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)
	
	// Encode the hash and salt
	encodedHash := hex.EncodeToString(hash)
	encodedSalt := hex.EncodeToString(salt)
	
	// Return the formatted hash
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", 
		argon2.Version, config.Memory, config.Iterations, config.Parallelism, encodedSalt, encodedHash), nil
}

// VerifyPassword verifies a password against a hash
func (a *AuthService) VerifyPassword(password, hash string) (bool, error) {
	// Parse the hash to extract parameters
	var version int
	var memory, iterations uint32
	var parallelism uint8
	var salt, hashBytes []byte
	var err error
	
	_, err = fmt.Sscanf(hash, "$argon2id$v=%d$m=%d,t=%d,p=%d$%x$%x", 
		&version, &memory, &iterations, &parallelism, &salt, &hashBytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse hash: %w", err)
	}
	
	// Hash the provided password with the same parameters
	computedHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(hashBytes)))
	
	// Compare the hashes
	return hex.EncodeToString(computedHash) == hex.EncodeToString(hashBytes), nil
}

// GenerateTokenPair generates both access and refresh tokens
func (a *AuthService) GenerateTokenPair(userID, organizationID uuid.UUID, role models.UserRole, email string) (*models.TokenPair, error) {
	// Generate access token (short-lived)
	accessToken, err := a.generateAccessToken(userID, organizationID, role, email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	
	// Generate refresh token (long-lived)
	refreshToken, err := a.generateRefreshToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	
	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hour
	}, nil
}

// generateAccessToken generates a short-lived access token
func (a *AuthService) generateAccessToken(userID, organizationID uuid.UUID, role models.UserRole, email string) (string, error) {
	claims := JWTClaims{
		UserID:         userID,
		OrganizationID: organizationID,
		Role:           role,
		Email:          email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "soter-auth",
			Subject:   userID.String(),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// generateRefreshToken generates a long-lived refresh token and stores it in the database
func (a *AuthService) generateRefreshToken(userID uuid.UUID) (string, error) {
	// Generate a random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	
	token := hex.EncodeToString(tokenBytes)
	
	// Hash the token for storage
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	
	// Store the refresh token in the database
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`
	
	_, err := a.db.Exec(query, userID, tokenHash, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}
	
	return token, nil
}

// ValidateToken validates a JWT token and returns the claims
func (a *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token")
}

// RefreshAccessToken generates a new access token using a refresh token
func (a *AuthService) RefreshAccessToken(refreshToken string) (*models.TokenPair, error) {
	// Hash the provided refresh token
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])
	
	// Find the refresh token in the database
	query := `
		SELECT rt.user_id, rt.expires_at, u.email, uo.organization_id, uo.role
		FROM refresh_tokens rt
		JOIN users u ON rt.user_id = u.id
		JOIN user_organizations uo ON u.id = uo.user_id
		WHERE rt.token_hash = $1 AND rt.is_revoked = false AND rt.expires_at > NOW()
	`
	
	var userID uuid.UUID
	var expiresAt time.Time
	var email string
	var organizationID uuid.UUID
	var role models.UserRole
	
	err := a.db.QueryRow(query, tokenHash).Scan(&userID, &expiresAt, &email, &organizationID, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired refresh token")
		}
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}
	
	// Generate new token pair
	return a.GenerateTokenPair(userID, organizationID, role, email)
}

// RevokeRefreshToken revokes a refresh token
func (a *AuthService) RevokeRefreshToken(refreshToken string) error {
	// Hash the provided refresh token
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])
	
	// Revoke the token
	query := `
		UPDATE refresh_tokens 
		SET is_revoked = true, revoked_at = NOW()
		WHERE token_hash = $1 AND is_revoked = false
	`
	
	result, err := a.db.Exec(query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found or already revoked")
	}
	
	return nil
}

// CleanupExpiredTokens removes expired refresh tokens from the database
func (a *AuthService) CleanupExpiredTokens() error {
	query := `
		DELETE FROM refresh_tokens 
		WHERE expires_at < NOW() OR is_revoked = true
	`
	
	_, err := a.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	
	logrus.Info("Cleaned up expired refresh tokens")
	return nil
}

// Login authenticates a user and returns token pair
func (a *AuthService) Login(email, password string) (*models.TokenPair, *models.UserWithRole, error) {
	// Find user by email
	query := `
		SELECT u.id, u.name, u.username, u.email, u.password_hash, u.is_active,
		       uo.organization_id, uo.role, o.name as org_name, o.description as org_description
		FROM users u
		JOIN user_organizations uo ON u.id = uo.user_id
		JOIN organizations o ON uo.organization_id = o.id
		WHERE u.email = $1 AND u.is_active = true
	`
	
	var user models.User
	var userOrg models.UserOrganization
	var org models.Organization
	
	err := a.db.QueryRow(query, email).Scan(
		&user.ID, &user.Name, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsActive, &userOrg.OrganizationID, &userOrg.Role,
		&org.Name, &org.Description,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("invalid email or password")
		}
		return nil, nil, fmt.Errorf("failed to find user: %w", err)
	}
	
	// Verify password
	valid, err := a.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify password: %w", err)
	}
	
	if !valid {
		return nil, nil, fmt.Errorf("invalid email or password")
	}
	
	// Update last login
	updateQuery := `UPDATE users SET last_login = NOW() WHERE id = $1`
	_, err = a.db.Exec(updateQuery, user.ID)
	if err != nil {
		logrus.WithError(err).Warn("Failed to update last login")
	}
	
	// Generate token pair
	tokenPair, err := a.GenerateTokenPair(user.ID, userOrg.OrganizationID, userOrg.Role, user.Email)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}
	
	// Create user with role
	userWithRole := &models.UserWithRole{
		User:         user,
		Role:         userOrg.Role,
		Organization: org,
	}
	
	return tokenPair, userWithRole, nil
}

