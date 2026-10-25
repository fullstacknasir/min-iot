package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"miniot/apps/api/internal/models"
)

const AuthQueryTimeout = 3 * time.Second

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

// CreateOrganization inserts a new organization into the database.
func (r *AuthRepository) CreateOrganization(ctx context.Context, name string) (*models.Organization, error) {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	org := &models.Organization{
		ID:        "org-" + uuid.New().String()[:8],
		Name:      name,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	query := `INSERT INTO organizations (id, name, created_at, updated_at) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(queryCtx, query, org.ID, org.Name, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert organization: %w", err)
	}
	return org, nil
}

// FindOrganizationByID retrieves an organization record by its identifier.
func (r *AuthRepository) FindOrganizationByID(ctx context.Context, id string) (*models.Organization, error) {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	var org models.Organization
	query := `SELECT id, name, created_at, updated_at FROM organizations WHERE id = $1`
	err := r.pool.QueryRow(queryCtx, query, id).Scan(&org.ID, &org.Name, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// CreateUser inserts a user under an organization.
func (r *AuthRepository) CreateUser(ctx context.Context, user *models.User) error {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	if user.ID == "" {
		user.ID = "usr-" + uuid.New().String()[:8]
	}

	query := `INSERT INTO users (id, organization_id, email, name, password_hash, role, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(queryCtx, query, user.ID, user.OrganizationID, user.Email, user.Name, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt)
	return err
}

// FindUserByEmail retrieves a user by their email address.
func (r *AuthRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	var user models.User
	query := `SELECT id, organization_id, email, name, password_hash, role, created_at, updated_at 
	          FROM users WHERE email = $1`
	err := r.pool.QueryRow(queryCtx, query, email).Scan(
		&user.ID, &user.OrganizationID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserByID retrieves a user by their unique identifier.
func (r *AuthRepository) FindUserByID(ctx context.Context, id string) (*models.User, error) {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	var user models.User
	query := `SELECT id, organization_id, email, name, password_hash, role, created_at, updated_at 
	          FROM users WHERE id = $1`
	err := r.pool.QueryRow(queryCtx, query, id).Scan(
		&user.ID, &user.OrganizationID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// StoreRefreshToken saves a new refresh token hash for user session management.
func (r *AuthRepository) StoreRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	if token.ID == "" {
		token.ID = "tok-" + uuid.New().String()[:8]
	}
	token.CreatedAt = time.Now().UTC()

	query := `INSERT INTO refresh_tokens (id, user_id, token_hash, revoked, expires_at, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.pool.Exec(queryCtx, query, token.ID, token.UserID, token.TokenHash, token.Revoked, token.ExpiresAt, token.CreatedAt)
	return err
}

// FindRefreshTokenByHash retrieves a refresh token record using its SHA-256 hash.
func (r *AuthRepository) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	var token models.RefreshToken
	query := `SELECT id, user_id, token_hash, revoked, expires_at, created_at 
	          FROM refresh_tokens WHERE token_hash = $1`
	err := r.pool.QueryRow(queryCtx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.Revoked, &token.ExpiresAt, &token.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// RevokeRefreshToken marks a specific refresh token as revoked.
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	query := `UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1`
	tag, err := r.pool.Exec(queryCtx, query, tokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// RevokeAllUserRefreshTokens marks all active refresh tokens for a user as revoked.
func (r *AuthRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	query := `UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1`
	_, err := r.pool.Exec(queryCtx, query, userID)
	return err
}

// FindUsersByOrganization retrieves all users belonging to an organization.
// FindUsersByOrganization retrieves users belonging to an organization with optional pagination.
func (r *AuthRepository) FindUsersByOrganization(ctx context.Context, orgID string, page, limit int) ([]models.User, int, error) {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	var total int
	countQuery := `SELECT COUNT(*) FROM users WHERE organization_id = $1`
	if err := r.pool.QueryRow(queryCtx, countQuery, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed counting users: %w", err)
	}

	var rows pgx.Rows
	var err error

	if limit > 0 && page > 0 {
		offset := (page - 1) * limit
		query := `SELECT id, organization_id, email, name, password_hash, role, created_at, updated_at 
		          FROM users WHERE organization_id = $1 ORDER BY created_at ASC 
		          LIMIT $2 OFFSET $3`
		rows, err = r.pool.Query(queryCtx, query, orgID, limit, offset)
	} else {
		query := `SELECT id, organization_id, email, name, password_hash, role, created_at, updated_at 
		          FROM users WHERE organization_id = $1 ORDER BY created_at ASC`
		rows, err = r.pool.Query(queryCtx, query, orgID)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.OrganizationID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// UpdateUserProfile modifies user profile attributes (name, email).
func (r *AuthRepository) UpdateUserProfile(ctx context.Context, userID, name, email string) (*models.User, error) {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	now := time.Now().UTC()
	query := `UPDATE users SET name = $1, email = $2, updated_at = $3 
	          WHERE id = $4 
	          RETURNING id, organization_id, email, name, password_hash, role, created_at, updated_at`
	var user models.User
	err := r.pool.QueryRow(queryCtx, query, name, email, now, userID).Scan(
		&user.ID, &user.OrganizationID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUserPassword updates the hashed password for a user.
func (r *AuthRepository) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	now := time.Now().UTC()
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(queryCtx, query, passwordHash, now, userID)
	return err
}

// UpdateUserRole updates the role for a user within an organization.
func (r *AuthRepository) UpdateUserRole(ctx context.Context, userID, orgID, role string) error {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	now := time.Now().UTC()
	query := `UPDATE users SET role = $1, updated_at = $2 WHERE id = $3 AND organization_id = $4`
	tag, err := r.pool.Exec(queryCtx, query, role, now, userID, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteUser deletes a user from an organization.
func (r *AuthRepository) DeleteUser(ctx context.Context, userID, orgID string) error {
	queryCtx, cancel := context.WithTimeout(ctx, AuthQueryTimeout)
	defer cancel()

	query := `DELETE FROM users WHERE id = $1 AND organization_id = $2`
	tag, err := r.pool.Exec(queryCtx, query, userID, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

