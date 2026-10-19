package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"miniot/apps/api/internal/config"
	"miniot/apps/api/internal/database"
	"miniot/apps/api/internal/handlers"
	"miniot/apps/api/internal/middleware"
	"miniot/apps/api/internal/models"
	"miniot/apps/api/internal/repository"
)

func main() {
	cfg := config.LoadConfig()
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	dbClients, err := database.InitDatabase(cfg)
	if err != nil {
		log.Printf("[WARN] Failed to connect to DB at startup: %v (will retry on requests)", err)
	}

	if dbClients != nil && dbClients.Pool != nil {
		if err := database.RunMigrations(context.Background(), dbClients.Pool); err != nil {
			log.Fatalf("[FATAL] Failed executing database migrations: %v", err)
		}
	}

	router := setupRouter(cfg, dbClients)
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("[minIoT API] Server listening on http://%s", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func setupRouter(cfg *config.Config, dbClients *database.Clients) *gin.Engine {
	middleware.InitLogger(cfg.Environment)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.StructuredLogger())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	registerPublicRoutes(router, dbClients, cfg)
	registerProtectedRoutes(router, dbClients, cfg)
	return router
}

func registerPublicRoutes(router *gin.Engine, dbClients *database.Clients, cfg *config.Config) {
	healthHandler := handlers.NewHealthHandler(dbClients)
	router.GET("/health", healthHandler.Check)
	router.GET("/docs", handlers.ServeDocsUI)
	router.GET("/docs/openapi.yaml", handlers.ServeOpenAPISpec)

	if dbClients == nil || dbClients.Pool == nil {
		return
	}

	authRepo := repository.NewAuthRepository(dbClients.Pool)
	authHandler := handlers.NewAuthHandler(authRepo, cfg)
	rateLimiter := middleware.LoginRateLimiter(dbClients.Redis, cfg.LoginRateLimitAttempts, cfg.LoginRateLimitWindow)

	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", rateLimiter, authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
		authGroup.POST("/logout", authHandler.Logout)
	}

	deviceRepo := repository.NewDeviceRepository(dbClients.Pool)
	emqxHandler := handlers.NewEMQXHandler(deviceRepo)

	emqxGroup := router.Group("/api/v1/emqx")
	{
		emqxGroup.POST("/auth", emqxHandler.HandleAuth)
		emqxGroup.POST("/acl", emqxHandler.HandleACL)
		emqxGroup.POST("/events", emqxHandler.HandleEvents)
		emqxGroup.POST("/webhook", emqxHandler.HandleEvents)
	}
}

func registerProtectedRoutes(router *gin.Engine, dbClients *database.Clients, cfg *config.Config) {
	if dbClients == nil || dbClients.Pool == nil {
		return
	}

	deviceRepo := repository.NewDeviceRepository(dbClients.Pool)
	telemetryRepo := repository.NewTelemetryRepository(dbClients.Pool, dbClients.Redis)
	authRepo := repository.NewAuthRepository(dbClients.Pool)

	deviceHandler := handlers.NewDeviceHandler(deviceRepo)
	telemetryHandler := handlers.NewTelemetryHandler(telemetryRepo)
	userHandler := handlers.NewUserHandler(authRepo)

	api := router.Group("/api/v1")
	if dbClients.Redis != nil {
		api.Use(middleware.APIRateLimiter(dbClients.Redis, 120, time.Minute))
	}
	api.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	registerDeviceRoutes(api, deviceHandler)
	registerTelemetryRoutes(api, telemetryHandler)
	registerUserRoutes(api, userHandler)
}

func registerUserRoutes(api *gin.RouterGroup, handler *handlers.UserHandler) {
	users := api.Group("/users")
	users.Use(middleware.RequireRole(models.RoleAdmin))
	{
		users.GET("", handler.ListUsers)
		users.POST("", handler.CreateUser)
		users.PATCH("/:id/role", handler.UpdateUserRole)
		users.DELETE("/:id", handler.DeleteUser)
	}
}


func registerDeviceRoutes(api *gin.RouterGroup, handler *handlers.DeviceHandler) {
	api.GET("/devices", handler.ListDevices)
	api.GET("/devices/:id", handler.GetDevice)
	api.GET("/devices/:id/credentials", handler.GetDeviceCredentials)

	manage := api.Group("/devices")
	manage.Use(middleware.RequireRole(models.RoleAdmin, models.RoleOperator))
	{
		manage.POST("", handler.CreateDevice)
		manage.PUT("/:id", handler.UpdateDevice)
		manage.DELETE("/:id", handler.DeleteDevice)
		manage.POST("/:id/credentials", handler.ProvisionDevice)
		manage.POST("/:id/enable", handler.EnableDevice)
		manage.POST("/:id/disable", handler.DisableDevice)
		manage.POST("/:id/revoke", handler.RevokeDevice)
	}
}

func registerTelemetryRoutes(api *gin.RouterGroup, handler *handlers.TelemetryHandler) {
	api.GET("/devices/:id/telemetry/latest", handler.GetLatestTelemetry)
	api.GET("/devices/:id/telemetry/history", handler.GetHistory)
	api.GET("/devices/:id/telemetry/aggregates", handler.GetAggregates)
	api.GET("/devices/:id/telemetry/live", handler.StreamLiveTelemetry)
	api.POST("/telemetry", middleware.RequireRole(models.RoleAdmin, models.RoleOperator), handler.IngestTelemetry)
}
