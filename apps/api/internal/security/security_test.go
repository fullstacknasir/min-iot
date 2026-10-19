package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"miniot/apps/api/internal/models"
)

const (
	TestSecret   = "test_super_secret_signing_key_12345"
	TestPassword = "SuperSecretPassword123!"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword(TestPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, TestPassword, hash)

	assert.True(t, CheckPassword(hash, TestPassword), "Valid password should match hash")
	assert.False(t, CheckPassword(hash, "WrongPassword"), "Invalid password should not match hash")
}

func TestAccessTokenGenerationAndValidation(t *testing.T) {
	user := &models.User{
		ID:             "usr-test-123",
		OrganizationID: "org-test-456",
		Email:          "developer@test.io",
		Role:           models.RoleAdmin,
	}

	token, err := GenerateAccessToken(TestSecret, user, 15*time.Minute)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ValidateAccessToken(TestSecret, token)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.OrganizationID, claims.OrganizationID)
	assert.Equal(t, user.Role, claims.Role)
	assert.Equal(t, user.Email, claims.Email)

	// Validate with wrong secret
	_, err = ValidateAccessToken("wrong_secret_key", token)
	assert.Error(t, err, "Validation should fail with invalid secret")
}

func TestExpiredAccessToken(t *testing.T) {
	user := &models.User{
		ID:             "usr-expired",
		OrganizationID: "org-expired",
		Email:          "expired@test.io",
		Role:           models.RoleViewer,
	}

	// Generate already expired token
	token, err := GenerateAccessToken(TestSecret, user, -1*time.Minute)
	assert.NoError(t, err)

	_, err = ValidateAccessToken(TestSecret, token)
	assert.Error(t, err, "Expired token must be rejected")
}

func TestRefreshTokenGenerationAndHashing(t *testing.T) {
	rawToken, tokenHash, err := GenerateRefreshToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, rawToken)
	assert.NotEmpty(t, tokenHash)

	recalculatedHash := HashRefreshToken(rawToken)
	assert.Equal(t, tokenHash, recalculatedHash, "HashRefreshToken must be deterministic")
}
