package database

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"miniot/apps/api/internal/security"
)

const (
	DefaultOrgID      = "org-default-001"
	DefaultOrgName    = "Acme IoT Corporation"
	DefaultAdminEmail = "admin@miniot.io"
	DefaultAdminPass  = "AdminPassword123!"
)

// AutoSeed ensures the default organization and admin user exist on startup.
func AutoSeed(ctx context.Context, pool *pgxpool.Pool) error {
	var userCount int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return err
	}

	if userCount > 0 {
		return nil
	}

	now := time.Now().UTC()
	orgQuery := `INSERT INTO organizations (id, name, created_at, updated_at) 
	             VALUES ($1, $2, $3, $4) 
	             ON CONFLICT (id) DO NOTHING`
	if _, err := pool.Exec(ctx, orgQuery, DefaultOrgID, DefaultOrgName, now, now); err != nil {
		return err
	}

	passHash, err := security.HashPassword(DefaultAdminPass)
	if err != nil {
		return err
	}

	adminQuery := `INSERT INTO users (id, organization_id, email, password_hash, role, created_at, updated_at)
	               VALUES ($1, $2, $3, $4, $5, $6, $7)
	               ON CONFLICT (email) DO NOTHING`
	if _, err := pool.Exec(ctx, adminQuery, "usr-admin-001", DefaultOrgID, DefaultAdminEmail, passHash, "admin", now, now); err != nil {
		return err
	}

	deviceQuery := `UPDATE devices SET organization_id = $1 WHERE organization_id IS NULL`
	_, _ = pool.Exec(ctx, deviceQuery, DefaultOrgID)

	log.Printf("[AutoSeed] Successfully seeded default organization and admin user (%s)", DefaultAdminEmail)
	return nil
}
