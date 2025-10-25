package auth

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// PasswordVerifier handles password verification
type PasswordVerifier struct {
	adminPasswordHash string
}

// NewPasswordVerifier creates a new password verifier
func NewPasswordVerifier() (*PasswordVerifier, error) {
	adminHash := os.Getenv("ADMIN_PASSWORD_BCRYPT_HASH")
	if adminHash == "" {
		return nil, fmt.Errorf("ADMIN_PASSWORD_BCRYPT_HASH environment variable is required")
	}

	return &PasswordVerifier{
		adminPasswordHash: adminHash,
	}, nil
}

// VerifyAdminPassword verifies the admin password
func (v *PasswordVerifier) VerifyAdminPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(v.adminPasswordHash), []byte(password))
}

// VerifyUserPassword verifies a user's password against a bcrypt hash
func (v *PasswordVerifier) VerifyUserPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// HashPassword generates a bcrypt hash for a password (used for utilities)
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
