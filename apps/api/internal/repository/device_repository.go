package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/security"
)

const (
	QueryTimeout = 3 * time.Second
)

var ErrDeviceNotFound = errors.New("device not found in organization")

type DeviceRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceRepository(pool *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{pool: pool}
}

// FindAll retrieves devices scoped to the user's organization and user assignment with optional pagination.
func (r *DeviceRepository) FindAll(ctx context.Context, organizationID, userID, role string, allDevices bool, page, limit int) ([]models.Device, int, error) {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var total int
	isAdminScope := role == models.RoleAdmin || allDevices

	if isAdminScope {
		countQuery := `SELECT COUNT(*) FROM devices WHERE organization_id = $1`
		if err := r.pool.QueryRow(queryCtx, countQuery, organizationID).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("failed counting devices: %w", err)
		}
	} else {
		countQuery := `SELECT COUNT(*) FROM devices d 
		               WHERE d.organization_id = $1 
		                 AND (d.assigned_user_id = $2 OR (NOT EXISTS (SELECT 1 FROM devices WHERE organization_id = $1 AND assigned_user_id = $2) AND d.assigned_user_id IS NULL))`
		if err := r.pool.QueryRow(queryCtx, countQuery, organizationID, userID).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("failed counting devices: %w", err)
		}
	}

	var rows pgx.Rows
	var err error

	if limit > 0 && page > 0 {
		offset := (page - 1) * limit
		if isAdminScope {
			query := `SELECT d.id, d.organization_id, d.name, d.device_type, d.status, d.metadata, d.last_seen, 
			                 d.assigned_user_id, u.email as assigned_user_email, d.created_at, d.updated_at 
			          FROM devices d 
			          LEFT JOIN users u ON d.assigned_user_id = u.id 
			          WHERE d.organization_id = $1 
			          ORDER BY d.created_at DESC 
			          LIMIT $2 OFFSET $3`
			rows, err = r.pool.Query(queryCtx, query, organizationID, limit, offset)
		} else {
			query := `SELECT d.id, d.organization_id, d.name, d.device_type, d.status, d.metadata, d.last_seen, 
			                 d.assigned_user_id, u.email as assigned_user_email, d.created_at, d.updated_at 
			          FROM devices d 
			          LEFT JOIN users u ON d.assigned_user_id = u.id 
			          WHERE d.organization_id = $1 
			            AND (d.assigned_user_id = $2 OR (NOT EXISTS (SELECT 1 FROM devices WHERE organization_id = $1 AND assigned_user_id = $2) AND d.assigned_user_id IS NULL))
			          ORDER BY d.created_at DESC 
			          LIMIT $3 OFFSET $4`
			rows, err = r.pool.Query(queryCtx, query, organizationID, userID, limit, offset)
		}
	} else {
		if isAdminScope {
			query := `SELECT d.id, d.organization_id, d.name, d.device_type, d.status, d.metadata, d.last_seen, 
			                 d.assigned_user_id, u.email as assigned_user_email, d.created_at, d.updated_at 
			          FROM devices d 
			          LEFT JOIN users u ON d.assigned_user_id = u.id 
			          WHERE d.organization_id = $1 
			          ORDER BY d.created_at DESC`
			rows, err = r.pool.Query(queryCtx, query, organizationID)
		} else {
			query := `SELECT d.id, d.organization_id, d.name, d.device_type, d.status, d.metadata, d.last_seen, 
			                 d.assigned_user_id, u.email as assigned_user_email, d.created_at, d.updated_at 
			          FROM devices d 
			          LEFT JOIN users u ON d.assigned_user_id = u.id 
			          WHERE d.organization_id = $1 
			            AND (d.assigned_user_id = $2 OR (NOT EXISTS (SELECT 1 FROM devices WHERE organization_id = $1 AND assigned_user_id = $2) AND d.assigned_user_id IS NULL))
			          ORDER BY d.created_at DESC`
			rows, err = r.pool.Query(queryCtx, query, organizationID, userID)
		}
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed querying devices: %w", err)
	}
	defer rows.Close()

	devices := make([]models.Device, 0)
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, 0, err
		}
		devices = append(devices, *d)
	}
	return devices, total, nil
}

// FindByID retrieves a single device by ID scoped strictly to caller's organization.
func (r *DeviceRepository) FindByID(ctx context.Context, id, organizationID string) (*models.Device, error) {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	query := `SELECT d.id, d.organization_id, d.name, d.device_type, d.status, d.metadata, d.last_seen, 
	                 d.assigned_user_id, u.email as assigned_user_email, d.created_at, d.updated_at 
	          FROM devices d 
	          LEFT JOIN users u ON d.assigned_user_id = u.id 
	          WHERE d.id = $1 AND d.organization_id = $2`
	row := r.pool.QueryRow(queryCtx, query, id, organizationID)

	d, err := scanDeviceRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Create inserts a new device scoped to the organization.
func (r *DeviceRepository) Create(ctx context.Context, d *models.Device) error {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	metadataBytes, _ := json.Marshal(d.Metadata)
	if len(metadataBytes) == 0 {
		metadataBytes = []byte("{}")
	}

	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now
	d.Status = models.DeviceStatusOffline

	query := `INSERT INTO devices (id, organization_id, name, device_type, status, metadata, last_seen, assigned_user_id, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.pool.Exec(queryCtx, query, d.ID, d.OrganizationID, d.Name, d.DeviceType, d.Status, metadataBytes, d.LastSeen, d.AssignedUserID, d.CreatedAt, d.UpdatedAt)
	return err
}

// Update modifies device name, type, and metadata scoped to organization.
func (r *DeviceRepository) Update(ctx context.Context, id, organizationID string, req *models.UpdateDeviceRequest) (*models.Device, error) {
	existing, err := r.FindByID(ctx, id, organizationID)
	if err != nil {
		return nil, err
	}

	applyDeviceUpdates(existing, req)

	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	metadataBytes, _ := json.Marshal(existing.Metadata)
	query := `UPDATE devices SET name = $1, device_type = $2, metadata = $3, assigned_user_id = $4, updated_at = $5 
	          WHERE id = $6 AND organization_id = $7`
	_, err = r.pool.Exec(queryCtx, query, existing.Name, existing.DeviceType, metadataBytes, existing.AssignedUserID, existing.UpdatedAt, id, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed updating device: %w", err)
	}
	return r.FindByID(ctx, id, organizationID)
}

// Delete removes a device scoped to organization.
func (r *DeviceRepository) Delete(ctx context.Context, id, organizationID string) error {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	query := `DELETE FROM devices WHERE id = $1 AND organization_id = $2`
	tag, err := r.pool.Exec(queryCtx, query, id, organizationID)
	if err != nil {
		return fmt.Errorf("failed deleting device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// UpdateStatus changes a device's status (online, offline, disabled, revoked).
func (r *DeviceRepository) UpdateStatus(ctx context.Context, id, organizationID, newStatus string) (*models.Device, error) {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	now := time.Now().UTC()
	query := `UPDATE devices SET status = $1, updated_at = $2 WHERE id = $3 AND organization_id = $4 RETURNING id`
	var returnedID string
	err := r.pool.QueryRow(queryCtx, query, newStatus, now, id, organizationID).Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id, organizationID)
}

// StoreDeviceCredential saves a hashed per-device credential.
func (r *DeviceRepository) StoreDeviceCredential(ctx context.Context, cred *models.DeviceCredential) error {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	if cred.ID == "" {
		cred.ID = "cred-" + uuid.New().String()[:8]
	}
	now := time.Now().UTC()
	cred.CreatedAt = now
	cred.UpdatedAt = now

	// Revoke prior active credentials for this device
	_ = r.SetCredentialsStatus(ctx, cred.DeviceID, cred.OrganizationID, models.CredentialStatusRevoked)

	query := `INSERT INTO device_credentials (id, device_id, organization_id, credential_type, secret_hash, status, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(queryCtx, query, cred.ID, cred.DeviceID, cred.OrganizationID, cred.CredentialType, cred.SecretHash, cred.Status, cred.CreatedAt, cred.UpdatedAt)
	return err
}

// GetDeviceCredential returns the active credential metadata for a device without exposing hash.
func (r *DeviceRepository) GetDeviceCredential(ctx context.Context, deviceID, organizationID string) (*models.DeviceCredential, error) {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	query := `SELECT id, device_id, organization_id, credential_type, status, created_at, updated_at, revoked_at 
	          FROM device_credentials WHERE device_id = $1 AND organization_id = $2 ORDER BY created_at DESC LIMIT 1`
	var cred models.DeviceCredential
	err := r.pool.QueryRow(queryCtx, query, deviceID, organizationID).Scan(
		&cred.ID, &cred.DeviceID, &cred.OrganizationID, &cred.CredentialType, &cred.Status, &cred.CreatedAt, &cred.UpdatedAt, &cred.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// SetCredentialsStatus updates credential status (active, disabled, revoked) for a device.
func (r *DeviceRepository) SetCredentialsStatus(ctx context.Context, deviceID, organizationID, status string) error {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	now := time.Now().UTC()
	var revokedAt *time.Time
	if status == models.CredentialStatusRevoked {
		revokedAt = &now
	}

	query := `UPDATE device_credentials SET status = $1, revoked_at = COALESCE($2, revoked_at), updated_at = $3 
	          WHERE device_id = $4 AND organization_id = $5`
	_, err := r.pool.Exec(queryCtx, query, status, revokedAt, now, deviceID, organizationID)
	return err
}

func scanDeviceRow(row pgx.Row) (*models.Device, error) {
	var d models.Device
	var metadataBytes []byte
	err := row.Scan(&d.ID, &d.OrganizationID, &d.Name, &d.DeviceType, &d.Status, &metadataBytes, &d.LastSeen, &d.AssignedUserID, &d.AssignedUserEmail, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &d.Metadata)
	}
	return &d, nil
}

func applyDeviceUpdates(device *models.Device, req *models.UpdateDeviceRequest) {
	if req.Name != nil && *req.Name != "" {
		device.Name = *req.Name
	}
	if req.DeviceType != nil && *req.DeviceType != "" {
		device.DeviceType = *req.DeviceType
	}
	if req.Metadata != nil {
		device.Metadata = req.Metadata
	}
	if req.ClearAssignedUser {
		device.AssignedUserID = nil
	} else if req.AssignedUserID != nil {
		if *req.AssignedUserID == "" {
			device.AssignedUserID = nil
		} else {
			device.AssignedUserID = req.AssignedUserID
		}
	}
	device.UpdatedAt = time.Now().UTC()
}

// AuthenticateDevice validates incoming MQTT device credentials and rejects disabled/revoked units.
func (r *DeviceRepository) AuthenticateDevice(ctx context.Context, clientID, username, password string) (*models.Device, bool, error) {
	deviceID := resolveDeviceIdentifier(clientID, username)
	if deviceID == "" || password == "" {
		return nil, false, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var device models.Device
	err := r.pool.QueryRow(queryCtx, "SELECT id, organization_id, status FROM devices WHERE id = $1", deviceID).
		Scan(&device.ID, &device.OrganizationID, &device.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	if device.Status == models.DeviceStatusDisabled || device.Status == models.DeviceStatusRevoked {
		return nil, false, nil
	}

	tokenHash := security.HashRefreshToken(password)
	var validCredential bool
	credQuery := "SELECT EXISTS(SELECT 1 FROM device_credentials WHERE device_id = $1 AND secret_hash = $2 AND status = 'active')"
	if err := r.pool.QueryRow(queryCtx, credQuery, deviceID, tokenHash).Scan(&validCredential); err != nil || !validCredential {
		return nil, false, err
	}

	_ = r.RecordDeviceConnect(ctx, deviceID)
	return &device, true, nil
}

// AuthorizeDeviceAction verifies that a device only publishes to its telemetry topic and subscribes to its command topic.
func (r *DeviceRepository) AuthorizeDeviceAction(ctx context.Context, clientID, username, action, topic string) (bool, error) {
	deviceID := resolveDeviceIdentifier(clientID, username)
	if deviceID == "" {
		return false, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var status string
	err := r.pool.QueryRow(queryCtx, "SELECT status FROM devices WHERE id = $1", deviceID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) || status == models.DeviceStatusDisabled || status == models.DeviceStatusRevoked {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if action == models.EMQXActionPublish {
		expectedTopic := fmt.Sprintf("devices/%s/telemetry", deviceID)
		return topic == expectedTopic, nil
	}

	if action == models.EMQXActionSubscribe {
		return isPermittedCommandTopic(topic, deviceID), nil
	}

	return false, nil
}

// RecordDeviceConnect updates device status to online and timestamps last_seen.
func (r *DeviceRepository) RecordDeviceConnect(ctx context.Context, deviceID string) error {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	now := time.Now().UTC()
	query := `UPDATE devices SET status = $1, last_seen = $2, updated_at = $2 
	          WHERE id = $3 AND status != $4 AND status != $5`
	_, err := r.pool.Exec(queryCtx, query, models.DeviceStatusOnline, now, deviceID, models.DeviceStatusDisabled, models.DeviceStatusRevoked)
	return err
}

// RecordDeviceDisconnect updates device status to offline and timestamps last_seen.
func (r *DeviceRepository) RecordDeviceDisconnect(ctx context.Context, deviceID string) error {
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	now := time.Now().UTC()
	query := `UPDATE devices SET status = $1, last_seen = $2, updated_at = $2 
	          WHERE id = $3 AND status != $4 AND status != $5`
	_, err := r.pool.Exec(queryCtx, query, models.DeviceStatusOffline, now, deviceID, models.DeviceStatusDisabled, models.DeviceStatusRevoked)
	return err
}

func resolveDeviceIdentifier(clientID, username string) string {
	if clientID != "" {
		return clientID
	}
	return username
}

func isPermittedCommandTopic(topic, deviceID string) bool {
	exactCmdTopic := fmt.Sprintf("devices/%s/commands", deviceID)
	wildcardCmdTopic := fmt.Sprintf("devices/%s/commands/#", deviceID)
	singularCmdTopic := fmt.Sprintf("devices/%s/command", deviceID)

	return topic == exactCmdTopic || topic == wildcardCmdTopic || topic == singularCmdTopic
}
