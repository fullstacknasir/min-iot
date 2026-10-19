package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
	"miniot/apps/api/internal/security"
)

type UserHandler struct {
	repo *repository.AuthRepository
}

func NewUserHandler(repo *repository.AuthRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// ListUsers returns all users in the caller's organization.
func (h *UserHandler) ListUsers(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	users, err := h.repo.FindUsersByOrganization(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": len(users)})
}

type CreateOrgUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required"`
}

// CreateUser adds a new user with a specific role to the caller's organization.
func (h *UserHandler) CreateUser(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	if orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}

	var req CreateOrgUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !models.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role specified"})
		return
	}

	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure password"})
		return
	}

	user := &models.User{
		OrganizationID: orgID,
		Email:          req.Email,
		PasswordHash:   passwordHash,
		Role:           req.Role,
	}

	if err := h.repo.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateUserRole updates the role of a user in caller's organization.
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	userID := c.Param("id")

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !models.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role specified"})
		return
	}

	err := h.repo.UpdateUserRole(c.Request.Context(), userID, orgID, req.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found in organization"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user role updated", "id": userID, "role": req.Role})
}

// DeleteUser removes a user from caller's organization.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	orgID := middleware.GetOrganizationID(c)
	userID := c.Param("id")

	callerUserID := middleware.GetUserID(c)
	if callerUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own user account"})
		return
	}

	err := h.repo.DeleteUser(c.Request.Context(), userID, orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found in organization"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user successfully removed", "id": userID})
}
