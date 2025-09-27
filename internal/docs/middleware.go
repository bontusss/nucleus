package docs

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SwaggerConfig holds configuration for Swagger UI
type SwaggerConfig struct {
	Title       string
	Description string
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Enabled     bool
}

// DefaultSwaggerConfig returns default Swagger configuration
func DefaultSwaggerConfig() SwaggerConfig {
	return SwaggerConfig{
		Title:       "Hotel ERP API",
		Description: "This is the Hotel ERP API server. It provides endpoints for managing hotel operations including authentication, point of sale, inventory, and more.",
		Version:     "1.0.0",
		Host:        "localhost:9000",
		BasePath:    "/api",
		Schemes:     []string{"http", "https"},
		Enabled:     true,
	}
}

// SetupSwagger configures and registers Swagger documentation routes
func SetupSwagger(r *gin.Engine, config SwaggerConfig) {
	if !config.Enabled {
		return
	}

	// Update swagger spec with config values
	updateSwaggerSpec(config)

	// Create docs group
	docs := r.Group("/docs")
	{
		// Swagger UI endpoint
		docs.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		// Alternative endpoints for easier access
		docs.GET("/", func(c *gin.Context) {
			c.Redirect(301, "/docs/swagger/index.html")
		})
		docs.GET("/api", func(c *gin.Context) {
			c.Redirect(301, "/docs/swagger/index.html")
		})
	}

	// Health endpoint for docs
	r.GET("/docs/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":       "healthy",
			"docs_enabled": config.Enabled,
			"swagger_url":  "/docs/swagger/index.html",
		})
	})
}

// SetupRedocly configures Redocly documentation (alternative to Swagger UI)
func SetupRedocly(r *gin.Engine, config SwaggerConfig) {
	if !config.Enabled {
		return
	}

	r.GET("/redoc", func(c *gin.Context) {
		html := generateRedocHTML(config)
		c.Header("Content-Type", "text/html")
		c.String(200, html)
	})
}

// updateSwaggerSpec updates the swagger specification with config values
func updateSwaggerSpec(config SwaggerConfig) {
	// This would typically update the global swagger spec
	// For now, we'll keep the hardcoded values in docs.go
	// In a production system, you'd want to make this more dynamic
}

// generateRedocHTML generates HTML for Redocly documentation
func generateRedocHTML(config SwaggerConfig) string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>` + config.Title + `</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body { margin: 0; padding: 0; }
    </style>
</head>
<body>
    <redoc spec-url="/docs/swagger/doc.json"></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@2.0.0/bundles/redoc.standalone.js"></script>
</body>
</html>`
}

// APIDocsMiddleware adds API documentation metadata to responses
func APIDocsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add API documentation headers
		c.Header("X-API-Version", "1.0.0")
		c.Header("X-API-Docs", "/docs/swagger/index.html")

		c.Next()
	}
}

// CORSForDocs enables CORS for documentation endpoints
func CORSForDocs() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
