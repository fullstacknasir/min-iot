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
	ErrCustomWidgetNotFound = errors.New("custom widget not found")
)

type CustomWidgetRepository struct {
	pool *pgxpool.Pool
}

func NewCustomWidgetRepository(pool *pgxpool.Pool) *CustomWidgetRepository {
	return &CustomWidgetRepository{pool: pool}
}

// CreateCustomWidget inserts a new custom widget definition.
func (r *CustomWidgetRepository) CreateCustomWidget(ctx context.Context, w *models.CustomWidget) error {
	schemaJSON, err := json.Marshal(w.SettingsSchema)
	if err != nil {
		return fmt.Errorf("failed serializing settings_schema: %w", err)
	}
	if len(schemaJSON) == 0 || string(schemaJSON) == "null" {
		schemaJSON = []byte("{}")
	}

	now := time.Now().UTC()
	w.CreatedAt = now
	w.UpdatedAt = now

	query := `INSERT INTO custom_widgets (
		id, organization_id, name, description, widget_type, template_html, style_css, controller_js, settings_schema, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = r.pool.Exec(ctx, query,
		w.ID, w.OrganizationID, w.Name, w.Description, w.WidgetType,
		w.TemplateHTML, w.StyleCSS, w.ControllerJS, schemaJSON, w.CreatedAt, w.UpdatedAt,
	)
	return err
}

// GetCustomWidgetByID retrieves a custom widget by its ID and organization.
func (r *CustomWidgetRepository) GetCustomWidgetByID(ctx context.Context, id, orgID string) (*models.CustomWidget, error) {
	query := `SELECT id, organization_id, name, description, widget_type, template_html, style_css, controller_js, settings_schema, created_at, updated_at
	          FROM custom_widgets
	          WHERE id = $1 AND organization_id = $2`

	var w models.CustomWidget
	var schemaBytes []byte

	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(
		&w.ID, &w.OrganizationID, &w.Name, &w.Description, &w.WidgetType,
		&w.TemplateHTML, &w.StyleCSS, &w.ControllerJS, &schemaBytes, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomWidgetNotFound
		}
		return nil, err
	}

	if len(schemaBytes) > 0 {
		_ = json.Unmarshal(schemaBytes, &w.SettingsSchema)
	}
	if w.SettingsSchema == nil {
		w.SettingsSchema = make(map[string]interface{})
	}

	return &w, nil
}

// ListCustomWidgets lists all custom widgets in an organization ordered by updated_at desc.
func (r *CustomWidgetRepository) ListCustomWidgets(ctx context.Context, orgID string) ([]*models.CustomWidget, error) {
	query := `SELECT id, organization_id, name, description, widget_type, template_html, style_css, controller_js, settings_schema, created_at, updated_at
	          FROM custom_widgets
	          WHERE organization_id = $1
	          ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	widgets := make([]*models.CustomWidget, 0)
	for rows.Next() {
		var w models.CustomWidget
		var schemaBytes []byte

		if err := rows.Scan(
			&w.ID, &w.OrganizationID, &w.Name, &w.Description, &w.WidgetType,
			&w.TemplateHTML, &w.StyleCSS, &w.ControllerJS, &schemaBytes, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if len(schemaBytes) > 0 {
			_ = json.Unmarshal(schemaBytes, &w.SettingsSchema)
		}
		if w.SettingsSchema == nil {
			w.SettingsSchema = make(map[string]interface{})
		}

		widgets = append(widgets, &w)
	}

	return widgets, rows.Err()
}

// UpdateCustomWidget updates an existing custom widget definition.
func (r *CustomWidgetRepository) UpdateCustomWidget(ctx context.Context, w *models.CustomWidget) error {
	schemaJSON, err := json.Marshal(w.SettingsSchema)
	if err != nil {
		return fmt.Errorf("failed serializing settings_schema: %w", err)
	}
	if len(schemaJSON) == 0 || string(schemaJSON) == "null" {
		schemaJSON = []byte("{}")
	}

	w.UpdatedAt = time.Now().UTC()

	query := `UPDATE custom_widgets
	          SET name = $1, description = $2, widget_type = $3, template_html = $4, style_css = $5, controller_js = $6, settings_schema = $7, updated_at = $8
	          WHERE id = $9 AND organization_id = $10`

	res, err := r.pool.Exec(ctx, query,
		w.Name, w.Description, w.WidgetType, w.TemplateHTML, w.StyleCSS, w.ControllerJS, schemaJSON, w.UpdatedAt,
		w.ID, w.OrganizationID,
	)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return ErrCustomWidgetNotFound
	}
	return nil
}

// DeleteCustomWidget removes a custom widget definition.
func (r *CustomWidgetRepository) DeleteCustomWidget(ctx context.Context, id, orgID string) error {
	query := `DELETE FROM custom_widgets WHERE id = $1 AND organization_id = $2`
	res, err := r.pool.Exec(ctx, query, id, orgID)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return ErrCustomWidgetNotFound
	}
	return nil
}
