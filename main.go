package main

import (
	"fmt"
	db "herp/db/sqlc"
	_ "herp/docs/swagger"
	"herp/internal/auth"
	"herp/internal/config"
	"herp/internal/core/business"
	"herp/internal/core/ilogs"
	"herp/internal/core/inventory"
	"herp/internal/core/store"
	"herp/internal/docs"
	"herp/internal/middleware"
	"herp/internal/pos"
	"herp/internal/server"
	"herp/pkg/database"
	"herp/pkg/monitoring/logging"
	"herp/pkg/ratelimit"
	"herp/pkg/redis"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

// @title Hotel ERP API
// @version 1.0.0
// @description This is the Hotel ERP API server. It provides endpoints for managing hotel operations including authentication, point of sale, inventory, and more.
// @termsOfService http://swagger.io/terms/

// @contact.name Hotel ERP API Support
// @contact.email support@herp.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:7000
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Authorization header using the Bearer scheme. Example: "Authorization: Bearer {token}"

func main() {
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

	// Recovery middleware to ensure panics in /api return JSON
	r.Use(func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				path := c.Request.URL.Path
				if strings.HasPrefix(path, "/api/") {
					c.JSON(500, gin.H{
						"error": fmt.Sprintf("internal server error: %v", rec),
					})
					c.Abort()
					return
				}
				panic(rec) // let Gin’s default recovery handle non-API
			}
		}()
		c.Next()
	})

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


	// Serve Nuxt static assets (JS/CSS/images)
	r.Static("/_nuxt", "../public/_nuxt")
	r.StaticFile("/favicon.ico", "../public/favicon.ico")

	// Serve other static assets (like images in /public)
	r.Static("/assets", "../public/assets") // optional if you have assets

	// Catch-all: serve index.html for all other routes (SPA mode)
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"error": "API route not found"})
			return
		}
		c.File("../public/index.html")
	})

	// Create server with graceful shutdown
	serverConfig := server.Config{
		Port:            cfg.Port,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}

	srv := server.New(r, dbs, serverConfig)

	// Add health check endpoint
	// @Summary Health check
	// @Description Check the health status of the API server
	// @Tags health
	// @Produce json
	// @Success 200 {object} map[string]string "Service is healthy"
	// @Failure 500 {object} map[string]string "Service is unhealthy"
	// @Router /health [get]
	r.GET("/health", func(c *gin.Context) {
		if err := srv.Health(); err != nil {
			c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Start server with graceful shutdown
	log.Printf("Starting Hotel ERP server version %s...", cfg.ApiVersion)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
