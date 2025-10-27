package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType represents the type of JWT token
type TokenType string

const (
	TokenTypePortal TokenType = "portal"
	TokenTypeAdmin  TokenType = "admin"
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id,omitempty"` // Empty for admin tokens
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	signingKey []byte
}

// NewJWTManager creates a new JWT manager
func NewJWTManager() (*JWTManager, error) {
	signingKey := os.Getenv("JWT_SIGNING_SECRET_KEY")
	if signingKey == "" {
		return nil, fmt.Errorf("JWT_SIGNING_SECRET_KEY environment variable is required")
	}

	// Validate key strength
	if err := validateJWTKey(signingKey); err != nil {
		return nil, fmt.Errorf("JWT signing key validation failed: %w", err)
	}

	return &JWTManager{
		signingKey: []byte(signingKey),
	}, nil
}

// validateJWTKey validates the minimum length of the JWT signing key
func validateJWTKey(key string) error {
	// Minimum length check (256 bits = 32 bytes for HS256)
	if len(key) < 32 {
		return fmt.Errorf("JWT signing key must be at least 32 characters long (got %d)", len(key))
	}
	return nil
}

// GeneratePortalToken generates a JWT token for a portal user
func (m *JWTManager) GeneratePortalToken(userID, sessionID string, expiresIn time.Duration) (string, error) {
	claims := JWTClaims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: TokenTypePortal,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.signingKey)
}

// GenerateAdminToken generates a JWT token for admin access
func (m *JWTManager) GenerateAdminToken(expiresIn time.Duration) (string, error) {
	claims := JWTClaims{
		UserID:    "admin",
		SessionID: "",
		TokenType: TokenTypeAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.signingKey)
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.signingKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
