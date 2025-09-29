package main

import (
	"fmt"
	"log"
	db "nucleus/db/sqlc"
	_ "nucleus/docs/swagger"
	"nucleus/internal/auth"
	"nucleus/internal/config"
	"nucleus/internal/core/business"
	logs "nucleus/internal/core/ilogs"
	"nucleus/internal/core/inventory"
	"nucleus/internal/core/store"
	"nucleus/internal/docs"
	"nucleus/internal/middleware"
	"nucleus/internal/pos"
	"nucleus/internal/server"
	"nucleus/pkg/database"
	"nucleus/pkg/monitoring/logging"
	"nucleus/pkg/ratelimit"
	"nucleus/pkg/redis"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

// @title Nucleus ERP API
// @version 1.0.0
// @description Nucleus ERP is an open-source, API-first business suite designed for African businesses and beyond.
// @description It provides a modern, modular, and developer-friendly platform for managing core business operations.
// @description It also provides secure, scalable, and extensible endpoints for managing users, businesses, stores, inventory, suppliers, keys, and webhooks. Features include JWT authentication, API key support, rate limiting, activity logging, and comprehensive API documentation.
// @description
// @description ### Key Features
// @description - **Authentication & Authorization**: Secure user management with JWT-based access control.
// @description - **Business Management**: Create and manage businesses, branches, and organizational structures.
// @description - **Point of Sale (POS)**: Process sales, payments, and receipts with support for multi-branch operations.
// @description - **Inventory Management**: Track stock levels, suppliers, purchases, and transfers.
// @description - **Finance & Taxation**: Manage taxes, VAT, and financial records.
// @description - **Extensible via Webhooks & Events**: Trigger custom workflows (e.g., when a sale or inventory update occurs).
// @description - **Audit Logging**: Automatic logging of key user and system activities for compliance.
// @description
// @description ### Target Users
// @description - Small to medium businesses in Africa looking for ERP solutions tailored to their workflows.
// @description - Developers and integrators building custom business apps on top of Nucleus API.
// @description - Organizations needing a modular, open-source ERP that can be extended with plugins.
// @description
// @description ### Usage Notes
// @description - All requests must include a valid JWT token in the `Authorization` header.
// @description - API follows RESTful design and returns JSON responses.
// @description - File uploads (e.g., business logos) must be sent via multipart/form-data.
// @description
//
// @description  ## Authentication
// @description  - **JWT Token:** Obtain a token by POSTing to `/api/v1/auth/login` with valid credentials. Use the returned token in the `Authorization` header as `Bearer <token>`.
// @description  - **API Key:** Admins can generate API keys via the `/api/v1/key/generate` endpoint (requires authentication). Use the API key in the `Authorization` header as `ApiKey <key>`.
// @description  - See [API Docs](https://github.com/bontusss/nucleus) for more details.
// @description
//
// @securityDefinitions.apikey BearerAuth
// @in            header
// @name          Authorization
// @description   JWT Authorization header using the Bearer scheme. Example: "Authorization: Bearer {token}"
//
// @securityDefinitions.apikey ApiKeyAuth
// @in            header
// @name          Authorization
// @description   API Key header using the ApiKey scheme. Example: "Authorization: ApiKey {key}"
// @termsOfService https://usenucleus.com/terms
//
// @contact.name Nucleus ERP API Support
// @contact.url  https://github.com/bontusss/nucleus
// @contact.email support@usenucleus.com
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:7000
// @BasePath /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Authorization header using the Bearer scheme. Example: "Authorization: Bearer {token}"
func main() {
	start := time.Now()
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

	// Load config
	fmt.Println("Loading config...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load database
	log.Printf("Connecting to postgres database at %s", cfg.DatabaseURL)
	dbs, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	m, err := migrate.New(
		"file://db/migrations",
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatalf("Unable to instantiate the database schema migrator - %v", err)
	}

	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			log.Fatalf("Unable to migrate up to the latest database schema - %v", err)
		}
	}

	// Initialize sqlc
	log.Println("Setting up database queries")
	queries := db.New(dbs)

	// Initialize redis
	// Log Redis connection details (remove in production)
	log.Printf("Connecting to Redis at %s:%s", cfg.RedisHost, cfg.RedisPort)
	rConfig := redis.RedisConfig{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       0,
	}
	redisClient, err := redis.NewRedis(rConfig)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	rs := redisClient.RawClient()

	// Initialize rate limiter
	rateLimiter := ratelimit.NewRateLimit(rs)

	// Initialiaze services
	authSvc := auth.NewService(
		queries,
		cfg.JWTSecret,
		cfg.JWTRefreshSecret,
		time.Duration(cfg.JWTExpiry)*time.Minute,
		time.Duration(cfg.JWTRefreshExpiry)*time.Hour,
		redisClient,
		rs,
		cfg.LoginRateLimit,
		cfg.LoginRateWindow,
		cfg.LoginBlockDuration,
		cfg.IPRateLimit,
		dbs,
		logging.NewLogger(cfg),
	)

	r := gin.Default()

	// Apply global IP rate limiting middleware
	r.Use(ratelimit.IPRateLimitMiddleware(rateLimiter, cfg.IPRateLimit, time.Minute))

	// Register request logging middleware (stdout + file)
	r.Use(middleware.NewRequestLogger("tmp/logs/logs.json", cfg))

	// Setup API documentation
	docsConfig := docs.DefaultSwaggerConfig()
	docsConfig.Host = "localhost:" + cfg.Port
	docsConfig.Enabled = true

	// Add CORS for docs
	r.Use(docs.CORSForDocs())

	// Add API docs middleware
	r.Use(docs.APIDocsMiddleware())

	r.Static("/images", "./images")

	// Setup Swagger documentation
	docs.SetupSwagger(r, docsConfig)

	// Setup Redocly documentation (alternative)
	docs.SetupRedocly(r, docsConfig)

	// register routes
	v1 := r.Group("/api/v1")

	logger := logging.NewLogger(cfg)

	// public routes
	authHandler := auth.NewHandler(authSvc, cfg, logger, cfg.GinMode)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/register", authHandler.RegisterAdmin)
	v1.POST("/auth/verify-email", authHandler.VerifyEmail)
	v1.POST("/auth/forgot-password", authHandler.ForgotPassword)
	v1.POST("/auth/reset-password", authHandler.ResetPassword)

	// secured routes (JWT required)
	secured := v1.Group("")
	secured.Use(auth.AuthMiiddleware(authSvc))
	secured.POST("/auth/logout", authHandler.Logout)
	secured.POST("/auth/refresh", authHandler.Refresh)

	// Admin auth routes
	adminHandler := auth.NewAdminHandler(authSvc)
	adminHandler.RegisterAdminRoutes(secured, authSvc)

	// Core business setup
	businessService := business.NewBusiness(queries, dbs)
	coreHandler := business.NewBusinessHandler(businessService, cfg, logger)
	coreHandler.RegisterRoutes(secured, authSvc)

	// Logs routes
	logService := logs.NewLogs(dbs, queries)
	logsHandler := logs.NewLogsHandler(logService, logger)
	logsHandler.RegisterRoutes(secured, authSvc)

	// Store routes
	storeService := store.NewStore(dbs, queries)
	storeHandler := store.NewHandler(storeService, logger)
	storeHandler.RegisterRoutes(secured, authSvc)

	// Inventory
	inventoryService := inventory.NewInventory(queries, dbs)
	inventoryHandler := inventory.NewInventoryHandler(inventoryService, logger)
	inventoryHandler.RegisterRoutes(secured, authSvc)

	// POS routes
	pos.RegisterRoutes(secured, authSvc)

	// Create server with graceful shutdown
	serverConfig := server.Config{
		Port:            cfg.Port,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}

	srv := server.New(r, dbs, serverConfig)

	// Health godoc
	// @Summary Health check
	// @Description Check the health status of the API server
	// @Tags Health
	// @Produce json
	// @Success 200 {object} map[string]string "Service is healthy"
	// @Failure 500 {object} map[string]string "Service is unhealthy"
	// @Router /api/v1/health [get]
	v1.GET("/health", func(c *gin.Context) {
		if err := srv.Health(); err != nil {
			c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"name":     "Nucleus ERP API",
			"status":   "ok",
			"database": "ok",
			"redis":    "ok",
			"version":  cfg.ApiVersion,
			"uptime_s": int(time.Since(start).Seconds()),
			"time":     time.Now().UTC(),
		})
	})

	// Start server with graceful shutdown
	log.Printf("Starting Nucleus server version %s...", cfg.ApiVersion)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
