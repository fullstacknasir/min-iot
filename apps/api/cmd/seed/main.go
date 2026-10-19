package main

import (
	"context"
	"log"
	"time"

	"miniot/apps/api/internal/config"
	"miniot/apps/api/internal/database"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
	"miniot/apps/api/internal/security"
)

const (
	SeedTimeout       = 15 * time.Second
	DefaultOrgID      = "org-default-001"
	DefaultOrgName    = "Acme IoT Corporation"
	DefaultAdminEmail = "admin@miniot.io"
	DefaultAdminPass  = "AdminPassword123!"
)

func main() {
	cfg := config.LoadConfig()
	dbClients, err := database.InitDatabase(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer dbClients.Pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), SeedTimeout)
	defer cancel()

	if err := database.RunMigrations(ctx, dbClients.Pool); err != nil {
		log.Fatalf("Failed running migrations: %v", err)
	}

	authRepo := repository.NewAuthRepository(dbClients.Pool)

	seedOrganization(ctx, dbClients)
	seedAdminUser(ctx, authRepo)
	bindExistingDevicesToOrg(ctx, dbClients)

	log.Println("==================================================")
	log.Println("✔ Database seed completed successfully!")
	log.Printf("• Organization : %s (ID: %s)", DefaultOrgName, DefaultOrgID)
	log.Printf("• Admin User   : %s", DefaultAdminEmail)
	log.Printf("• Password     : %s", DefaultAdminPass)
	log.Printf("• Role         : %s", models.RoleAdmin)
	log.Println("==================================================")
}

func seedOrganization(ctx context.Context, clients *database.Clients) {
	now := time.Now().UTC()
	query := `INSERT INTO organizations (id, name, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4) 
	          ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at`
	_, err := clients.Pool.Exec(ctx, query, DefaultOrgID, DefaultOrgName, now, now)
	if err != nil {
		log.Fatalf("Failed seeding organization: %v", err)
	}
	log.Printf("[Seed] Provisioned organization: %s", DefaultOrgName)
}

func seedAdminUser(ctx context.Context, repo *repository.AuthRepository) {
	existingUser, err := repo.FindUserByEmail(ctx, DefaultAdminEmail)
	if err == nil && existingUser != nil {
		log.Printf("[Seed] Admin user '%s' already exists", DefaultAdminEmail)
		return
	}

	passHash, err := security.HashPassword(DefaultAdminPass)
	if err != nil {
		log.Fatalf("Password hashing failed: %v", err)
	}

	admin := &models.User{
		ID:             "usr-admin-001",
		OrganizationID: DefaultOrgID,
		Email:          DefaultAdminEmail,
		PasswordHash:   passHash,
		Role:           models.RoleAdmin,
	}

	if err := repo.CreateUser(ctx, admin); err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}
	log.Printf("[Seed] Created admin user: %s", DefaultAdminEmail)
}

func bindExistingDevicesToOrg(ctx context.Context, clients *database.Clients) {
	query := `UPDATE devices SET organization_id = $1 WHERE organization_id IS NULL`
	_, _ = clients.Pool.Exec(ctx, query, DefaultOrgID)
}
