package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type AuthService struct {
	jwtSecret     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewAuthService(jwtSecret string) *AuthService {
	return &AuthService{
		jwtSecret:     []byte(jwtSecret),
		accessExpiry:  24 * time.Hour,     // 24 hours for access token
		refreshExpiry: 7 * 24 * time.Hour, // 7 days for refresh token
	}
}

// HashPassword hashes a password using bcrypt
func (a *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword verifies a password against its hash
func (a *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateTokenPair generates both access and refresh tokens
func (a *AuthService) GenerateTokenPair(userID, username, email string) (*TokenPair, error) {
	// Generate access token
	accessToken, err := a.generateToken(userID, username, email, a.accessExpiry)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := a.generateToken(userID, username, email, a.refreshExpiry)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    time.Now().Add(a.accessExpiry).Unix(),
	}, nil
}

// generateToken creates a JWT token
func (a *AuthService) generateToken(userID, username, email string, expiry time.Duration) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "soter",
			Subject:   userID,
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ValidateToken validates and parses a JWT token
func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check if token is expired
		if claims.ExpiresAt.Time.Before(time.Now()) {
			return nil, ErrTokenExpired
		}
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// RefreshAccessToken generates a new access token from a valid refresh token
func (a *AuthService) RefreshAccessToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := a.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	return a.GenerateTokenPair(claims.UserID, claims.Username, claims.Email)
}
