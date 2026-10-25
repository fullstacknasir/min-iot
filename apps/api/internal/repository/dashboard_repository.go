package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"miniot/apps/api/internal/models"
)

var (
	ErrDashboardNotFound = errors.New("dashboard not found")
)

type DashboardRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

// CreateDashboard inserts a new dashboard scoped to an organization.
func (r *DashboardRepository) CreateDashboard(ctx context.Context, d *models.Dashboard) error {
	configJSON, err := d.Configuration.ToJSON()
	if err != nil {
		return fmt.Errorf("failed serializing dashboard configuration: %w", err)
	}

	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now

	// If marked as default, unset other defaults in organization first
	if d.IsDefault {
		_ = r.clearOrgDefaults(ctx, d.OrganizationID)
	}

	query := `INSERT INTO dashboards (
		id, organization_id, title, description, is_default, configuration, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = r.pool.Exec(ctx, query,
		d.ID, d.OrganizationID, d.Title, d.Description, d.IsDefault, configJSON, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

// GetDashboardByID retrieves a dashboard by ID within the organization.
func (r *DashboardRepository) GetDashboardByID(ctx context.Context, id, orgID string) (*models.Dashboard, error) {
	query := `SELECT id, organization_id, title, description, is_default, configuration, created_at, updated_at
	          FROM dashboards
	          WHERE id = $1 AND organization_id = $2`

	var d models.Dashboard
	var configBytes []byte

	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(
		&d.ID, &d.OrganizationID, &d.Title, &d.Description, &d.IsDefault, &configBytes, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDashboardNotFound
		}
		return nil, err
	}

	if len(configBytes) > 0 {
		_ = json.Unmarshal(configBytes, &d.Configuration)
	}
	d.Configuration.EnsureValidConfiguration()

	return &d, nil
}

// ListDashboards returns all dashboards for an organization ordered with default first.
func (r *DashboardRepository) ListDashboards(ctx context.Context, orgID string) ([]*models.Dashboard, error) {
	query := `SELECT id, organization_id, title, description, is_default, configuration, created_at, updated_at
	          FROM dashboards
	          WHERE organization_id = $1
	          ORDER BY is_default DESC, updated_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dashboards := make([]*models.Dashboard, 0)
	for rows.Next() {
		var d models.Dashboard
		var configBytes []byte

		if err := rows.Scan(
			&d.ID, &d.OrganizationID, &d.Title, &d.Description, &d.IsDefault, &configBytes, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(configBytes) > 0 {
			_ = json.Unmarshal(configBytes, &d.Configuration)
		}
		d.Configuration.EnsureValidConfiguration()
		dashboards = append(dashboards, &d)
	}
	return dashboards, rows.Err()
}

// UpdateDashboard updates an existing dashboard.
func (r *DashboardRepository) UpdateDashboard(ctx context.Context, d *models.Dashboard) error {
	configJSON, err := d.Configuration.ToJSON()
	if err != nil {
		return fmt.Errorf("failed serializing dashboard configuration: %w", err)
	}

	d.UpdatedAt = time.Now().UTC()

	if d.IsDefault {
		_ = r.clearOrgDefaults(ctx, d.OrganizationID)
	}

	query := `UPDATE dashboards
	          SET title = $1, description = $2, is_default = $3, configuration = $4, updated_at = $5
	          WHERE id = $6 AND organization_id = $7`

	cmd, err := r.pool.Exec(ctx, query,
		d.Title, d.Description, d.IsDefault, configJSON, d.UpdatedAt, d.ID, d.OrganizationID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrDashboardNotFound
	}
	return nil
}

// DeleteDashboard deletes a dashboard by ID within the organization.
func (r *DashboardRepository) DeleteDashboard(ctx context.Context, id, orgID string) error {
	query := `DELETE FROM dashboards WHERE id = $1 AND organization_id = $2`
	cmd, err := r.pool.Exec(ctx, query, id, orgID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrDashboardNotFound
	}
	return nil
}

// SetDefaultDashboard designates a dashboard as the organization's default.
func (r *DashboardRepository) SetDefaultDashboard(ctx context.Context, id, orgID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Unset current defaults
	_, err = tx.Exec(ctx, `UPDATE dashboards SET is_default = FALSE WHERE organization_id = $1`, orgID)
	if err != nil {
		return err
	}

	// Set target as default
	cmd, err := tx.Exec(ctx, `UPDATE dashboards SET is_default = TRUE, updated_at = NOW() WHERE id = $1 AND organization_id = $2`, id, orgID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrDashboardNotFound
	}

	return tx.Commit(ctx)
}

func (r *DashboardRepository) clearOrgDefaults(ctx context.Context, orgID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE dashboards SET is_default = FALSE WHERE organization_id = $1`, orgID)
	return err
}
