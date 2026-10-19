package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"miniot/apps/api/internal/models"
)

const (
	RefreshTokenByteLength = 32
	JWTIssuer              = "miniot-auth"
)

type Claims struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
	Email          string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateAccessToken signs a short-lived JWT with user and organization claims.
func GenerateAccessToken(secret string, user *models.User, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		Role:           user.Role,
		Email:          user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    JWTIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}
	return signedToken, nil
}

// ValidateAccessToken parses and verifies a signed JWT string.
func ValidateAccessToken(secret, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// GenerateRefreshToken generates a secure random token and returns the raw string and its SHA-256 hash.
func GenerateRefreshToken() (rawToken string, tokenHash string, err error) {
	bytes := make([]byte, RefreshTokenByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}

	rawToken = hex.EncodeToString(bytes)
	tokenHash = HashRefreshToken(rawToken)
	return rawToken, tokenHash, nil
}

// HashRefreshToken computes the SHA-256 hex digest of a refresh token.
func HashRefreshToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

// GenerateDeviceSecret creates a 32-byte cryptographically random token with dsec_ prefix.
func GenerateDeviceSecret() (rawToken string, tokenHash string, err error) {
	bytes := make([]byte, RefreshTokenByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed generating device secret: %w", err)
	}

	rawToken = "dsec_" + hex.EncodeToString(bytes)
	tokenHash = HashRefreshToken(rawToken)
	return rawToken, tokenHash, nil
}
