package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const BcryptDefaultCost = 12

// HashPassword generates a bcrypt hash of the given plain password string.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptDefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword verifies that a plain password matches its bcrypt hash.
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
