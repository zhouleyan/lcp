package iam

import "golang.org/x/crypto/bcrypt"

const bcryptDefaultCost = 10

// PasswordService handles password hashing and verification using bcrypt.
type PasswordService struct {
	cost int
}

// NewPasswordService creates a PasswordService with the default bcrypt cost.
func NewPasswordService() *PasswordService {
	return &PasswordService{cost: bcryptDefaultCost}
}

// Hash generates a bcrypt hash of the password.
func (ps *PasswordService) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), ps.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Verify compares a password against a bcrypt hash.
func (ps *PasswordService) Verify(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
