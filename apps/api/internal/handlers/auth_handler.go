package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"miniot/apps/api/internal/config"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
	"miniot/apps/api/internal/security"
)

type AuthHandler struct {
	repo *repository.AuthRepository
	cfg  *config.Config
}

func NewAuthHandler(repo *repository.AuthRepository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{repo: repo, cfg: cfg}
}

// Register creates an organization (if new) and registers a new authenticated user.
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	org, role, err := h.resolveOrganizationAndRole(c, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure password"})
		return
	}

	user := &models.User{
		OrganizationID: org.ID,
		Email:          req.Email,
		PasswordHash:   passwordHash,
		Role:           role,
	}

	if err := h.repo.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
		return
	}

	response, err := h.issueTokens(c, user, org)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed issuing authentication tokens"})
		return
	}
	c.JSON(http.StatusCreated, response)
}

// Login authenticates credentials and returns short-lived JWT and rotatable refresh token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.repo.FindUserByEmail(c.Request.Context(), req.Email)
	if err != nil || !security.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	org, err := h.repo.FindOrganizationByID(c.Request.Context(), user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load organization context"})
		return
	}

	response, err := h.issueTokens(c, user, org)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed issuing authentication tokens"})
		return
	}
	c.JSON(http.StatusOK, response)
}

// Refresh implements secure refresh token rotation.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenHash := security.HashRefreshToken(req.RefreshToken)
	storedToken, err := h.repo.FindRefreshTokenByHash(c.Request.Context(), tokenHash)
	if err != nil || storedToken.Revoked || time.Now().UTC().After(storedToken.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	// Revoke current token before issuing new family pair (Rotation)
	_ = h.repo.RevokeRefreshToken(c.Request.Context(), tokenHash)

	user, err := h.repo.FindUserByID(c.Request.Context(), storedToken.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "associated user not found"})
		return
	}

	response, err := h.issueTokens(c, user, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate session tokens"})
		return
	}
	c.JSON(http.StatusOK, response)
}

// Logout revokes the specified refresh token session.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenHash := security.HashRefreshToken(req.RefreshToken)
	if err := h.repo.RevokeRefreshToken(c.Request.Context(), tokenHash); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully logged out"})
}

// GetProfile returns the profile of the currently authenticated user.
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.repo.FindUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	org, _ := h.repo.FindOrganizationByID(c.Request.Context(), user.OrganizationID)
	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"organization": org,
	})
}

// UpdateProfile updates the name, email, or password of the currently authenticated user.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.repo.FindUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	newName := user.Name
	if req.Name != nil {
		newName = *req.Name
	}

	newEmail := user.Email
	if req.Email != nil && *req.Email != "" {
		newEmail = *req.Email
	}

	if req.NewPassword != nil && *req.NewPassword != "" {
		if len(*req.NewPassword) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "new password must be at least 8 characters"})
			return
		}
		if req.CurrentPassword == nil || !security.CheckPassword(user.PasswordHash, *req.CurrentPassword) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "current password is incorrect"})
			return
		}
		newHash, err := security.HashPassword(*req.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		if err := h.repo.UpdateUserPassword(c.Request.Context(), userID, newHash); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
			return
		}
	}

	updatedUser, err := h.repo.UpdateUserProfile(c.Request.Context(), userID, newName, newEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	org, _ := h.repo.FindOrganizationByID(c.Request.Context(), updatedUser.OrganizationID)
	c.JSON(http.StatusOK, gin.H{
		"message":      "profile updated successfully",
		"user":         updatedUser,
		"organization": org,
	})
}

func (h *AuthHandler) resolveOrganizationAndRole(c *gin.Context, req *models.RegisterRequest) (*models.Organization, string, error) {
	if req.OrganizationID != "" {
		org, err := h.repo.FindOrganizationByID(c.Request.Context(), req.OrganizationID)
		if err != nil {
			return nil, "", errors.New("organization not found")
		}
		role := req.Role
		if role == "" {
			role = models.RoleViewer
		}
		if !models.IsValidRole(role) {
			return nil, "", errors.New("invalid role specified")
		}
		return org, role, nil
	}

	name := req.OrganizationName
	if name == "" {
		name = "Default Organization"
	}
	org, err := h.repo.CreateOrganization(c.Request.Context(), name)
	if err != nil {
		return nil, "", err
	}
	return org, models.RoleAdmin, nil
}

func (h *AuthHandler) issueTokens(c *gin.Context, user *models.User, org *models.Organization) (*models.AuthResponse, error) {
	accessToken, err := security.GenerateAccessToken(h.cfg.JWTSecret, user, h.cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	rawRefresh, refreshHash, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshTokenRecord := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		Revoked:   false,
		ExpiresAt: time.Now().UTC().Add(h.cfg.RefreshTokenTTL),
	}

	if err := h.repo.StoreRefreshToken(c.Request.Context(), refreshTokenRecord); err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(h.cfg.AccessTokenTTL.Seconds()),
		User:         user,
		Organization: org,
	}, nil
}
